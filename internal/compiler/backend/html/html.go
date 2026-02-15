package html

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
	"github.com/greynewell/schemaflux/internal/config"
	"github.com/greynewell/schemaflux/internal/entity"
	"github.com/greynewell/schemaflux/internal/output"
	"github.com/greynewell/schemaflux/internal/render"
	"github.com/greynewell/schemaflux/internal/schema"
	"github.com/greynewell/schemaflux/internal/taxonomy"
)

// Backend is the HTML static site backend.
type Backend struct{}

func (b *Backend) Name() string { return "HTML" }

// Emit renders the entire IR Program to HTML files in outputDir.
func (b *Backend) Emit(p *ir.Program, outputDir string) error {
	start := time.Now()

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	// Initialize render engine
	log.Printf("Loading templates from %s...", p.Config.Paths.Templates)
	engine, err := render.NewEngine(p.Config)
	if err != nil {
		return fmt.Errorf("initializing render engine: %w", err)
	}

	// Extract CSS/JS
	if err := extractAssets(engine, p.Config, outputDir); err != nil {
		return err
	}

	schemaGen := schema.NewGenerator(p.Site, p.Config.Schema)

	// Track sitemap entries
	var sitemapEntries []output.SitemapEntry
	var sitemapMu sync.Mutex
	today := time.Now().Format("2006-01-02")

	addSitemapEntry := func(path, priority, changefreq string) {
		sitemapMu.Lock()
		defer sitemapMu.Unlock()
		sitemapEntries = append(sitemapEntries, output.NewSitemapEntry(
			p.Site.BaseURL, path, today, priority, changefreq,
		))
	}

	// Collect taxonomies as []taxonomy.Taxonomy for render contexts
	allTaxonomies := make([]taxonomy.Taxonomy, len(p.Taxonomies))
	for i, tg := range p.Taxonomies {
		allTaxonomies[i] = tg.Taxonomy
	}

	// Build valid slugs map
	validSlugs := make(map[string]map[string]bool)
	for _, tg := range p.Taxonomies {
		validSlugs[tg.Taxonomy.Name] = tg.ValidSlugs
	}

	// Favorites already resolved by FavoritesPass — extract raw entities for templates.
	favorites := ir.RawEntitySlice(p.Favorites)

	// Render entity pages (concurrent, 32-goroutine pool)
	log.Printf("Rendering %d entity pages...", len(p.Entities))
	var entityErrors int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, 32)

	for _, re := range p.Entities {
		wg.Add(1)
		sem <- struct{}{}
		go func(re *ir.ResolvedEntity) {
			defer wg.Done()
			defer func() { <-sem }()

			err := renderEntityPage(re, p, engine, allTaxonomies, validSlugs, outputDir, addSitemapEntry)
			if err != nil {
				atomic.AddInt64(&entityErrors, 1)
				fmt.Fprintf(os.Stderr, "Warning: failed to render %s: %v\n", re.Slug, err)
			}
		}(re)
	}
	wg.Wait()
	if entityErrors > 0 {
		log.Printf("  %d entity pages had errors", entityErrors)
	}

	// Generate search index
	if len(p.Entities) > 0 {
		if err := generateSearchIndex(p, outputDir); err != nil {
			log.Printf("Warning: failed to generate search index: %v", err)
		}
	}

	// Build category entries for RSS
	categoryEntries := make(map[string][]*entity.Entity)
	for _, tg := range p.Taxonomies {
		if tg.Taxonomy.Name == p.Config.RSS.CategoryTaxonomy {
			for _, entry := range tg.Taxonomy.Entries {
				categoryEntries[entry.Slug] = entry.Entities
			}
		}
	}

	// Render taxonomy pages
	log.Printf("Rendering taxonomy pages...")
	for _, tg := range p.Taxonomies {
		if err := renderTaxonomyPages(tg, p, engine, schemaGen, allTaxonomies, outputDir, addSitemapEntry, today); err != nil {
			return fmt.Errorf("rendering taxonomy %s: %w", tg.Taxonomy.Name, err)
		}
	}

	// Render all-entities pages
	if p.Config.Templates.AllEntities != "" {
		log.Printf("Rendering all-entities pages...")
		if err := renderAllEntitiesPages(p, engine, schemaGen, allTaxonomies, outputDir, addSitemapEntry); err != nil {
			return fmt.Errorf("rendering all-entities pages: %w", err)
		}
	}

	// Render homepage
	log.Printf("Rendering homepage...")
	if err := renderHomepage(p, engine, schemaGen, allTaxonomies, favorites, outputDir); err != nil {
		return fmt.Errorf("rendering homepage: %w", err)
	}
	addSitemapEntry("/index.html", p.Config.Sitemap.Priorities["homepage"], p.Config.Sitemap.ChangeFreqs["homepage"])

	// Render static pages
	log.Printf("Rendering %d static page(s)...", len(p.Config.Templates.StaticPages))
	for path, tmpl := range p.Config.Templates.StaticPages {
		if err := renderStaticPage(path, tmpl, p, engine, allTaxonomies, favorites, outputDir); err != nil {
			log.Printf("Warning: failed to render static page %s: %v", path, err)
		}
	}

	// Generate sitemap
	log.Printf("Generating sitemap (%d entries)...", len(sitemapEntries))
	sitemapFiles := output.GenerateSitemapFiles(sitemapEntries, p.Site.BaseURL, p.Config.Sitemap.MaxURLsPerFile)
	for _, sf := range sitemapFiles {
		if err := os.WriteFile(filepath.Join(outputDir, sf.Filename), []byte(sf.Content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", sf.Filename, err)
		}
	}
	log.Printf("  Generated %d sitemap file(s)", len(sitemapFiles))

	// Generate RSS
	rawEntities := p.RawEntities()
	rssFeeds := output.GenerateRSSFeeds(rawEntities, p.Config, categoryEntries)
	for _, feed := range rssFeeds {
		feedPath := filepath.Join(outputDir, feed.RelativePath)
		if err := os.MkdirAll(filepath.Dir(feedPath), 0755); err != nil {
			return fmt.Errorf("creating dir for RSS %s: %w", feed.RelativePath, err)
		}
		if err := os.WriteFile(feedPath, []byte(feed.Content), 0644); err != nil {
			return fmt.Errorf("writing RSS %s: %w", feed.RelativePath, err)
		}
	}
	if len(rssFeeds) > 0 {
		log.Printf("Generated %d RSS feed(s)", len(rssFeeds))
	}

	// Generate robots.txt
	robotsContent := output.GenerateRobotsTxt(p.Config)
	if err := os.WriteFile(filepath.Join(outputDir, "robots.txt"), []byte(robotsContent), 0644); err != nil {
		return fmt.Errorf("writing robots.txt: %w", err)
	}

	// Generate llms.txt
	if p.Config.LlmsTxt.Enabled {
		llmsContent := output.GenerateLlmsTxt(p.Config, rawEntities, allTaxonomies)
		if err := os.WriteFile(filepath.Join(outputDir, "llms.txt"), []byte(llmsContent), 0644); err != nil {
			return fmt.Errorf("writing llms.txt: %w", err)
		}
	}

	// Generate manifest.json
	manifestContent := output.GenerateManifest(p.Config)
	if err := os.WriteFile(filepath.Join(outputDir, "manifest.json"), []byte(manifestContent), 0644); err != nil {
		return fmt.Errorf("writing manifest.json: %w", err)
	}

	// Write CNAME if configured
	if p.Site.CNAME != "" {
		if err := os.WriteFile(filepath.Join(outputDir, "CNAME"), []byte(p.Site.CNAME+"\n"), 0644); err != nil {
			return fmt.Errorf("writing CNAME: %w", err)
		}
	}

	// Copy static assets
	if p.Config.Paths.Static != "" {
		if err := copyDir(p.Config.Paths.Static, outputDir); err != nil {
			log.Printf("Warning: failed to copy static assets: %v", err)
		}
	}

	elapsed := time.Since(start)
	log.Printf("  HTML backend complete (%s)", elapsed.Round(time.Millisecond))
	return nil
}

func extractAssets(engine *render.Engine, cfg *config.Config, outputDir string) error {
	if cfg.Output.ExtractCSS != "" {
		cssContent, err := engine.RenderCSS()
		if err != nil {
			log.Printf("Warning: failed to render CSS: %v", err)
		} else if cssContent != "" {
			cssPath := filepath.Join(outputDir, cfg.Output.ExtractCSS)
			if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
				return fmt.Errorf("writing CSS: %w", err)
			}
		}
	}
	if cfg.Output.ExtractJS != "" {
		jsContent, err := engine.RenderJS()
		if err != nil {
			log.Printf("Warning: failed to render JS: %v", err)
		} else if jsContent != "" {
			jsPath := filepath.Join(outputDir, cfg.Output.ExtractJS)
			if err := os.WriteFile(jsPath, []byte(jsContent), 0644); err != nil {
				return fmt.Errorf("writing JS: %w", err)
			}
		}
	}
	return nil
}

func renderEntityPage(
	re *ir.ResolvedEntity,
	p *ir.Program,
	engine *render.Engine,
	allTaxonomies []taxonomy.Taxonomy,
	validSlugs map[string]map[string]bool,
	outputDir string,
	addSitemapEntry func(string, string, string),
) error {
	e := re.Raw

	// Write share image SVG (URL already computed by SchemaPass — no IR mutation)
	if re.ShareImageSVG != "" {
		writeShareSVG(outputDir, p.Site.BaseURL, re.Slug+".svg", re.ShareImageSVG)
	}

	// Resolve pairings to raw entities for template compat
	var pairings []*entity.Entity
	for _, paired := range re.Pairings {
		pairings = append(pairings, paired.Raw)
	}

	// Resolve related to raw entities for template compat
	var related []*entity.Entity
	for _, rel := range re.Related {
		related = append(related, rel.Raw)
	}

	// Chart data JSON
	chartJSON, _ := json.Marshal(re.ChartData)

	ctx := render.EntityPageContext{
		Site:           p.Site,
		Entity:         e,
		Slug:           re.Slug,
		URL:            re.URL,
		CanonicalURL:   re.CanonicalURL,
		Breadcrumbs:    re.Breadcrumbs,
		Pairings:       pairings,
		Related:        related,
		Enrichment:     re.Enrichment,
		AffiliateLinks: re.AffiliateLinks,
		CookModePrompt: re.CookModePrompt,
		JsonLD:         template.HTML(re.JsonLD),
		OG:             re.OG,
		ChartData:      template.JS(chartJSON),
		SourceCode:     re.SourceCode,
		SourceLang:     re.SourceLang,
		Taxonomies:     allTaxonomies,
		AllTaxonomies:  allTaxonomies,
		ValidSlugs:     validSlugs,
		Contributors:   p.Contributors,
		CTA:            p.Config.Extra.CTA,
		TOC:            re.TOC,
		ReadingTime:    re.ReadingTime,
		WordCount:      re.WordCount,
	}

	html, err := engine.RenderEntity(ctx)
	if err != nil {
		return err
	}

	outPath := filepath.Join(outputDir, e.Slug+".html")
	if err := os.WriteFile(outPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}

	// Generate arch map SVG
	if archMap := e.GetString("arch_map"); archMap != "" {
		imagesDir := filepath.Join(outputDir, "images")
		os.MkdirAll(imagesDir, 0755)
		if svg := generateArchMapSVG(archMap); svg != "" {
			svgPath := filepath.Join(imagesDir, e.Slug+"-arch.svg")
			os.WriteFile(svgPath, []byte(svg), 0644)
		}
	}

	addSitemapEntry("/"+e.Slug+".html",
		p.Config.Sitemap.Priorities["entity"],
		p.Config.Sitemap.ChangeFreqs["entity"])

	return nil
}

func renderTaxonomyPages(
	tg ir.TaxonomyGroup,
	p *ir.Program,
	engine *render.Engine,
	schemaGen *schema.Generator,
	allTaxonomies []taxonomy.Taxonomy,
	outputDir string,
	addSitemapEntry func(string, string, string),
	today string,
) error {
	tax := tg.Taxonomy
	taxDir := filepath.Join(outputDir, tax.Name)
	if err := os.MkdirAll(taxDir, 0755); err != nil {
		return fmt.Errorf("creating taxonomy dir: %w", err)
	}

	perPage := p.Config.Pagination.EntitiesPerPage

	// Render hub pages for each entry
	for _, entry := range tax.Entries {
		totalPages := (len(entry.Entities) + perPage - 1) / perPage
		if totalPages == 0 {
			totalPages = 1
		}

		hubTypeDist := countFieldDistribution(entry.Entities, "node_type", 8)
		hubLangDist := countFieldDistribution(entry.Entities, "language", 8)
		hubDomainDist := countFieldDistribution(entry.Entities, "domain", 8)
		hubExtDist := countFieldDistribution(entry.Entities, "extension", 8)

		type topEntity struct {
			Name  string `json:"name"`
			Lines int    `json:"lines"`
			Slug  string `json:"slug"`
			Type  string `json:"type"`
		}
		var topEnts []topEntity
		for _, ent := range entry.Entities {
			if lc := ent.GetInt("line_count"); lc > 0 {
				topEnts = append(topEnts, topEntity{
					Name:  ent.GetString("title"),
					Lines: lc,
					Slug:  ent.Slug,
					Type:  ent.GetString("node_type"),
				})
			}
		}
		sort.Slice(topEnts, func(i, j int) bool { return topEnts[i].Lines > topEnts[j].Lines })
		if len(topEnts) > 10 {
			topEnts = topEnts[:10]
		}

		type hubChart struct {
			EntryName     string                        `json:"entryName"`
			TotalEntities int                           `json:"totalEntities"`
			Distributions map[string][]render.NameCount `json:"distributions"`
			TopEntities   []topEntity                   `json:"topEntities"`
		}
		hubChartObj := hubChart{
			EntryName:     entry.Name,
			TotalEntities: len(entry.Entities),
			Distributions: map[string][]render.NameCount{
				"node_type": hubTypeDist,
				"language":  hubLangDist,
				"domain":    hubDomainDist,
				"extension": hubExtDist,
			},
			TopEntities: topEnts,
		}
		hubChartJSON, _ := json.Marshal(hubChartObj)

		hubSVG := render.GenerateHubShareSVG(p.Site.Name, entry.Name, tax.Label, len(entry.Entities), hubTypeDist)
		hubImageURL := writeShareSVG(outputDir, p.Site.BaseURL, fmt.Sprintf("%s-%s.svg", tax.Name, entry.Slug), hubSVG)

		for page := 1; page <= totalPages; page++ {
			pagination := taxonomy.ComputePagination(entry, page, perPage, tax.Name)

			pageEntities := entry.Entities
			if pagination.StartIndex < len(entry.Entities) {
				end := pagination.EndIndex
				if end > len(entry.Entities) {
					end = len(entry.Entities)
				}
				pageEntities = entry.Entities[pagination.StartIndex:end]
			}

			pageURL := fmt.Sprintf("%s%s", p.Site.BaseURL, taxonomy.HubPageURL(tax.Name, entry.Slug, page))
			var items []schema.ItemListEntry
			for _, e := range pageEntities {
				items = append(items, schema.ItemListEntry{
					Name: e.GetString("title"),
					URL:  fmt.Sprintf("%s/%s.html", p.Site.BaseURL, e.Slug),
				})
			}
			entityLabel := p.Config.Data.EntityType + "s"
			if entityLabel == "s" {
				entityLabel = "entries"
			}
			collectionSchema := schemaGen.GenerateCollectionPageSchema(
				entry.Name, fmt.Sprintf("%s %s %s", entry.Name, tax.LabelSingular, entityLabel),
				pageURL, items, hubImageURL,
			)

			breadcrumbs := []render.Breadcrumb{
				{Name: "Home", URL: p.Site.BaseURL + "/"},
				{Name: tax.Label, URL: fmt.Sprintf("%s/%s/", p.Site.BaseURL, tax.Name)},
				{Name: entry.Name, URL: ""},
			}
			breadcrumbSchema := schemaGen.GenerateBreadcrumbSchema(toBreadcrumbItems(breadcrumbs))
			jsonLD := schema.MarshalSchemas(collectionSchema, breadcrumbSchema)

			var contributorProfile map[string]interface{}
			if tax.Name == "author" && p.Contributors != nil {
				if profiles, ok := p.Contributors["profiles"].(map[string]interface{}); ok {
					contributorProfile, _ = profiles[entry.Slug].(map[string]interface{})
				}
			}

			og := makeOG(p.Site.Name,
				fmt.Sprintf("%s — %s | %s", entry.Name, tax.Label, p.Site.Name),
				fmt.Sprintf("Browse all %d %s %s on %s", len(entry.Entities), entry.Name, entityLabel, p.Site.Name),
				pageURL, hubImageURL, "article")

			ctx := render.HubPageContext{
				Site:               p.Site,
				Taxonomy:           tax,
				Entry:              entry,
				Entities:           pageEntities,
				Pagination:         pagination,
				JsonLD:             template.HTML(jsonLD),
				OG:                 og,
				ChartData:          template.JS(hubChartJSON),
				Breadcrumbs:        breadcrumbs,
				AllTaxonomies:      allTaxonomies,
				Contributors:       p.Contributors,
				ContributorProfile: contributorProfile,
				CTA:                p.Config.Extra.CTA,
			}

			html, err := engine.RenderHub(ctx)
			if err != nil {
				return fmt.Errorf("rendering hub %s/%s page %d: %w", tax.Name, entry.Slug, page, err)
			}

			var filename string
			if page == 1 {
				filename = entry.Slug + ".html"
			} else {
				filename = fmt.Sprintf("%s-page-%d.html", entry.Slug, page)
			}

			if err := os.WriteFile(filepath.Join(taxDir, filename), []byte(html), 0644); err != nil {
				return fmt.Errorf("writing hub page: %w", err)
			}

			priority := p.Config.Sitemap.Priorities["hub_page_1"]
			if page > 1 {
				priority = p.Config.Sitemap.Priorities["hub_page_n"]
			}
			addSitemapEntry(fmt.Sprintf("/%s/%s", tax.Name, filename), priority, p.Config.Sitemap.ChangeFreqs["hub"])
		}
	}

	// Render taxonomy index page
	hasLetters := len(tax.Entries) >= tax.Config.LetterPageThreshold
	letterGroups := taxonomy.GroupByLetter(tax.Entries)

	var letters []string
	for _, lg := range letterGroups {
		letters = append(letters, lg.Letter)
	}

	topEntries := taxonomy.TopEntries(tax.Entries, 12)

	var taxIndexStats []render.NameCount
	for _, entry := range tax.Entries {
		taxIndexStats = append(taxIndexStats, render.NameCount{Name: entry.Name, Count: len(entry.Entities)})
	}
	sort.Slice(taxIndexStats, func(i, j int) bool { return taxIndexStats[i].Count > taxIndexStats[j].Count })

	type taxIndexChart struct {
		TaxonomyName string             `json:"taxonomyName"`
		TaxonomyKey  string             `json:"taxonomyKey"`
		Entries      []render.NameCount `json:"entries"`
	}
	limit := len(taxIndexStats)
	if limit > 20 {
		limit = 20
	}
	taxChartObj := taxIndexChart{TaxonomyName: tax.Label, TaxonomyKey: tax.Name, Entries: taxIndexStats[:limit]}
	taxChartJSON, _ := json.Marshal(taxChartObj)

	taxIdxSVG := render.GenerateTaxIndexShareSVG(p.Site.Name, tax.Label, taxIndexStats)
	taxIdxImageURL := writeShareSVG(outputDir, p.Site.BaseURL, fmt.Sprintf("%s-index.svg", tax.Name), taxIdxSVG)

	var indexItems []schema.ItemListEntry
	for _, entry := range tax.Entries {
		indexItems = append(indexItems, schema.ItemListEntry{
			Name: entry.Name,
			URL:  fmt.Sprintf("%s/%s/%s.html", p.Site.BaseURL, tax.Name, entry.Slug),
		})
	}
	indexSchema := schemaGen.GenerateItemListSchema(tax.Label, fmt.Sprintf("Browse all %s", tax.Label), indexItems, taxIdxImageURL)
	breadcrumbs := []render.Breadcrumb{
		{Name: "Home", URL: p.Site.BaseURL + "/"},
		{Name: tax.Label, URL: ""},
	}
	breadcrumbSchema := schemaGen.GenerateBreadcrumbSchema(toBreadcrumbItems(breadcrumbs))
	jsonLD := schema.MarshalSchemas(indexSchema, breadcrumbSchema)

	taxIdxPageURL := fmt.Sprintf("%s/%s/", p.Site.BaseURL, tax.Name)
	taxIdxOG := makeOG(p.Site.Name,
		fmt.Sprintf("%s | %s", tax.Label, p.Site.Name),
		fmt.Sprintf("Browse %s by %s. %d categories available on %s.", p.Config.Data.EntityType+"s", strings.ToLower(tax.Label), len(tax.Entries), p.Site.Name),
		taxIdxPageURL, taxIdxImageURL, "article")

	ctx := render.TaxonomyIndexContext{
		Site:          p.Site,
		Taxonomy:      tax,
		Entries:       tax.Entries,
		TopEntries:    topEntries,
		LetterGroups:  letterGroups,
		HasLetters:    hasLetters,
		Letters:       letters,
		JsonLD:        template.HTML(jsonLD),
		OG:            taxIdxOG,
		ChartData:     template.JS(taxChartJSON),
		Breadcrumbs:   breadcrumbs,
		AllTaxonomies: allTaxonomies,
		CTA:           p.Config.Extra.CTA,
	}

	htmlOut, err := engine.RenderTaxonomyIndex(ctx)
	if err != nil {
		return fmt.Errorf("rendering taxonomy index %s: %w", tax.Name, err)
	}

	if err := os.WriteFile(filepath.Join(taxDir, "index.html"), []byte(htmlOut), 0644); err != nil {
		return fmt.Errorf("writing taxonomy index: %w", err)
	}
	addSitemapEntry(fmt.Sprintf("/%s/", tax.Name), p.Config.Sitemap.Priorities["taxonomy_index"], p.Config.Sitemap.ChangeFreqs["taxonomy_index"])

	// Render letter pages if threshold met
	if hasLetters {
		for _, lg := range letterGroups {
			if err := renderLetterPage(lg, tax, letters, p, engine, allTaxonomies, outputDir, taxDir, addSitemapEntry); err != nil {
				return err
			}
		}
	}

	return nil
}

func renderLetterPage(
	lg taxonomy.LetterGroup,
	tax taxonomy.Taxonomy,
	letters []string,
	p *ir.Program,
	engine *render.Engine,
	allTaxonomies []taxonomy.Taxonomy,
	outputDir, taxDir string,
	addSitemapEntry func(string, string, string),
) error {
	letterBreadcrumbs := []render.Breadcrumb{
		{Name: "Home", URL: p.Site.BaseURL + "/"},
		{Name: tax.Label, URL: fmt.Sprintf("%s/%s/", p.Site.BaseURL, tax.Name)},
		{Name: fmt.Sprintf("Letter %s", lg.Letter), URL: ""},
	}

	var letterStats []render.NameCount
	for _, entry := range lg.Entries {
		letterStats = append(letterStats, render.NameCount{Name: entry.Name, Count: len(entry.Entities)})
	}
	sort.Slice(letterStats, func(i, j int) bool { return letterStats[i].Count > letterStats[j].Count })

	type letterChart struct {
		Letter       string             `json:"letter"`
		TaxonomyName string             `json:"taxonomyName"`
		TaxonomyKey  string             `json:"taxonomyKey"`
		Entries      []render.NameCount `json:"entries"`
	}
	letterLimit := len(letterStats)
	if letterLimit > 15 {
		letterLimit = 15
	}
	letterChartObj := letterChart{Letter: lg.Letter, TaxonomyName: tax.Label, TaxonomyKey: tax.Name, Entries: letterStats[:letterLimit]}
	letterChartJSON, _ := json.Marshal(letterChartObj)

	letterSVG := render.GenerateLetterShareSVG(p.Site.Name, tax.Label, lg.Letter, len(lg.Entries))
	letterSafeFile := strings.ToLower(lg.Letter)
	if letterSafeFile == "#" {
		letterSafeFile = "num"
	}
	letterImageURL := writeShareSVG(outputDir, p.Site.BaseURL, fmt.Sprintf("%s-letter-%s.svg", tax.Name, letterSafeFile), letterSVG)

	letterFile := fmt.Sprintf("letter-%s.html", letterSafeFile)
	letterPageURL := fmt.Sprintf("%s/%s/%s", p.Site.BaseURL, tax.Name, letterFile)
	letterOG := makeOG(p.Site.Name,
		fmt.Sprintf("%s — %s | %s", tax.Label, lg.Letter, p.Site.Name),
		fmt.Sprintf("Browse %s entries starting with %s", strings.ToLower(tax.Label), lg.Letter),
		letterPageURL, letterImageURL, "article")

	letterCtx := render.LetterPageContext{
		Site:          p.Site,
		Taxonomy:      tax,
		Letter:        lg.Letter,
		Entries:       lg.Entries,
		Letters:       letters,
		JsonLD:        template.HTML(""),
		OG:            letterOG,
		ChartData:     template.JS(letterChartJSON),
		Breadcrumbs:   letterBreadcrumbs,
		AllTaxonomies: allTaxonomies,
		CTA:           p.Config.Extra.CTA,
	}

	letterHTML, err := engine.RenderLetter(letterCtx)
	if err != nil {
		return fmt.Errorf("rendering letter page %s/%s: %w", tax.Name, lg.Letter, err)
	}

	if err := os.WriteFile(filepath.Join(taxDir, letterFile), []byte(letterHTML), 0644); err != nil {
		return fmt.Errorf("writing letter page: %w", err)
	}
	addSitemapEntry(fmt.Sprintf("/%s/%s", tax.Name, letterFile),
		p.Config.Sitemap.Priorities["letter_page"],
		p.Config.Sitemap.ChangeFreqs["letter_page"])
	return nil
}

func renderHomepage(
	p *ir.Program,
	engine *render.Engine,
	schemaGen *schema.Generator,
	allTaxonomies []taxonomy.Taxonomy,
	favorites []*entity.Entity,
	outputDir string,
) error {
	rawEntities := p.RawEntities()

	var taxStats []render.NameCount
	for _, tax := range allTaxonomies {
		taxStats = append(taxStats, render.NameCount{Name: tax.Label, Count: len(tax.Entries)})
	}
	svg := render.GenerateHomepageShareSVG(p.Site.Name, p.Site.Description, taxStats, len(rawEntities))
	imageURL := writeShareSVG(outputDir, p.Site.BaseURL, "homepage.svg", svg)

	// Chart data with slugs
	type taxChartEntry struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
		Slug  string `json:"slug"`
	}
	type homepageChart struct {
		Taxonomies    []taxChartEntry `json:"taxonomies"`
		TotalEntities int             `json:"totalEntities"`
	}
	var taxChartEntries []taxChartEntry
	for _, tax := range allTaxonomies {
		taxChartEntries = append(taxChartEntries, taxChartEntry{
			Name:  tax.Label,
			Count: len(tax.Entries),
			Slug:  tax.Name,
		})
	}
	chartObj := homepageChart{Taxonomies: taxChartEntries, TotalEntities: len(rawEntities)}
	chartJSON, _ := json.Marshal(chartObj)

	// Architecture graph data
	type archNode struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Type  string `json:"type"`
		Count int    `json:"count"`
		Slug  string `json:"slug"`
	}
	type archLink struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	type archGraph struct {
		Nodes []archNode `json:"nodes"`
		Links []archLink `json:"links"`
	}

	var archNodes []archNode
	var archLinks []archLink

	archNodes = append(archNodes, archNode{
		ID: "root", Name: p.Site.Name, Type: "root", Count: len(rawEntities),
	})

	var domainTax, subdomainTax *taxonomy.Taxonomy
	for i := range allTaxonomies {
		if allTaxonomies[i].Name == "domain" {
			domainTax = &allTaxonomies[i]
		}
		if allTaxonomies[i].Name == "subdomain" {
			subdomainTax = &allTaxonomies[i]
		}
	}

	if domainTax != nil {
		for _, domEntry := range domainTax.Entries {
			domID := "domain-" + domEntry.Slug
			archNodes = append(archNodes, archNode{
				ID: domID, Name: domEntry.Name, Type: "domain", Count: len(domEntry.Entities), Slug: "domain/" + domEntry.Slug,
			})
			archLinks = append(archLinks, archLink{Source: "root", Target: domID})

			if subdomainTax != nil {
				domEntitySet := make(map[string]bool)
				for _, e := range domEntry.Entities {
					domEntitySet[e.Slug] = true
				}
				for _, sdEntry := range subdomainTax.Entries {
					overlap := 0
					for _, e := range sdEntry.Entities {
						if domEntitySet[e.Slug] {
							overlap++
						}
					}
					if overlap > 0 {
						sdID := "subdomain-" + sdEntry.Slug
						found := false
						for _, n := range archNodes {
							if n.ID == sdID {
								found = true
								break
							}
						}
						if !found {
							archNodes = append(archNodes, archNode{
								ID: sdID, Name: sdEntry.Name, Type: "subdomain", Count: len(sdEntry.Entities), Slug: "subdomain/" + sdEntry.Slug,
							})
						}
						archLinks = append(archLinks, archLink{Source: domID, Target: sdID})
					}
				}
			}
		}
	}

	archGraphObj := archGraph{Nodes: archNodes, Links: archLinks}
	archJSON, _ := json.Marshal(archGraphObj)

	// JSON-LD
	websiteSchema := schemaGen.GenerateWebSiteSchema(imageURL)

	var itemListSource []*entity.Entity
	if len(favorites) > 0 {
		itemListSource = favorites
	} else if len(rawEntities) > 10 {
		itemListSource = rawEntities[:10]
	} else {
		itemListSource = rawEntities
	}
	var items []schema.ItemListEntry
	for _, e := range itemListSource {
		items = append(items, schema.ItemListEntry{
			Name: e.GetString("title"),
			URL:  fmt.Sprintf("%s/%s.html", p.Site.BaseURL, e.Slug),
		})
	}
	itemListSchema := schemaGen.GenerateItemListSchema(p.Site.Name, p.Site.Description, items, imageURL)
	jsonLD := schema.MarshalSchemas(websiteSchema, itemListSchema)

	pageURL := p.Site.BaseURL + "/"
	ctx := render.HomepageContext{
		Site:         p.Site,
		Entities:     rawEntities,
		Taxonomies:   allTaxonomies,
		Favorites:    favorites,
		JsonLD:       template.HTML(jsonLD),
		OG:           makeOG(p.Site.Name, p.Site.Name, p.Site.Description, pageURL, imageURL, "website"),
		ChartData:    template.JS(chartJSON),
		ArchData:     template.JS(archJSON),
		EntityCount:  len(rawEntities),
		Contributors: p.Contributors,
		CTA:          p.Config.Extra.CTA,
	}

	htmlOut, err := engine.RenderHomepage(ctx)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(outputDir, "index.html"), []byte(htmlOut), 0644)
}

func renderAllEntitiesPages(
	p *ir.Program,
	engine *render.Engine,
	schemaGen *schema.Generator,
	allTaxonomies []taxonomy.Taxonomy,
	outputDir string,
	addSitemapEntry func(string, string, string),
) error {
	rawEntities := p.RawEntities()
	allDir := filepath.Join(outputDir, "all")
	if err := os.MkdirAll(allDir, 0755); err != nil {
		return fmt.Errorf("creating all-entities dir: %w", err)
	}

	perPage := p.Config.Pagination.EntitiesPerPage
	totalPages := (len(rawEntities) + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}

	allTypeDist := countFieldDistribution(rawEntities, "node_type", 8)
	type allEntitiesChart struct {
		TotalEntities    int                `json:"totalEntities"`
		TypeDistribution []render.NameCount `json:"typeDistribution"`
	}
	allChartObj := allEntitiesChart{TotalEntities: len(rawEntities), TypeDistribution: allTypeDist}
	allChartJSON, _ := json.Marshal(allChartObj)

	allSVG := render.GenerateAllEntitiesShareSVG(p.Site.Name, len(rawEntities), allTypeDist)
	allImageURL := writeShareSVG(outputDir, p.Site.BaseURL, "all-entities.svg", allSVG)

	for page := 1; page <= totalPages; page++ {
		pagination := taxonomy.ComputeAllEntitiesPagination(len(rawEntities), page, perPage)

		pageEntities := rawEntities
		if pagination.StartIndex < len(rawEntities) {
			end := pagination.EndIndex
			if end > len(rawEntities) {
				end = len(rawEntities)
			}
			pageEntities = rawEntities[pagination.StartIndex:end]
		}

		pageURL := fmt.Sprintf("%s%s", p.Site.BaseURL, taxonomy.AllEntitiesPageURL(page))
		var items []schema.ItemListEntry
		for _, e := range pageEntities {
			items = append(items, schema.ItemListEntry{
				Name: e.GetString("title"),
				URL:  fmt.Sprintf("%s/%s.html", p.Site.BaseURL, e.Slug),
			})
		}
		allLabel := "All " + strings.Title(p.Config.Data.EntityType) + "s"
		if p.Config.Data.EntityType == "" {
			allLabel = "All Entities"
		}
		collectionSchema := schemaGen.GenerateCollectionPageSchema(
			allLabel, fmt.Sprintf("Browse all %s on %s", p.Config.Data.EntityType+"s", p.Site.Name),
			pageURL, items, allImageURL,
		)
		breadcrumbs := []render.Breadcrumb{
			{Name: "Home", URL: p.Site.BaseURL + "/"},
			{Name: allLabel, URL: ""},
		}
		breadcrumbSchema := schemaGen.GenerateBreadcrumbSchema(toBreadcrumbItems(breadcrumbs))
		jsonLD := schema.MarshalSchemas(collectionSchema, breadcrumbSchema)

		allOG := makeOG(p.Site.Name,
			fmt.Sprintf("%s | %s", allLabel, p.Site.Name),
			fmt.Sprintf("Browse all %d %s on %s", len(rawEntities), p.Config.Data.EntityType+"s", p.Site.Name),
			pageURL, allImageURL, "article")

		ctx := render.AllEntitiesPageContext{
			Site:          p.Site,
			Entities:      pageEntities,
			Pagination:    pagination,
			JsonLD:        template.HTML(jsonLD),
			OG:            allOG,
			ChartData:     template.JS(allChartJSON),
			EntityCount:   len(rawEntities),
			Breadcrumbs:   breadcrumbs,
			AllTaxonomies: allTaxonomies,
			TotalEntities: len(rawEntities),
			CTA:           p.Config.Extra.CTA,
		}

		htmlOut, err := engine.RenderAllEntities(ctx)
		if err != nil {
			return fmt.Errorf("rendering all-entities page %d: %w", page, err)
		}

		var filename string
		if page == 1 {
			filename = "index.html"
		} else {
			filename = fmt.Sprintf("page-%d.html", page)
		}

		if err := os.WriteFile(filepath.Join(allDir, filename), []byte(htmlOut), 0644); err != nil {
			return fmt.Errorf("writing all-entities page: %w", err)
		}

		priority := p.Config.Sitemap.Priorities["hub_page_1"]
		if page > 1 {
			priority = p.Config.Sitemap.Priorities["hub_page_n"]
		}
		addSitemapEntry(fmt.Sprintf("/all/%s", filename), priority, p.Config.Sitemap.ChangeFreqs["hub"])
	}

	log.Printf("  Generated %d all-entities page(s)", totalPages)
	return nil
}

func renderStaticPage(
	pagePath, tmpl string,
	p *ir.Program,
	engine *render.Engine,
	allTaxonomies []taxonomy.Taxonomy,
	favorites []*entity.Entity,
	outputDir string,
) error {
	staticPageURL := fmt.Sprintf("%s/%s", p.Site.BaseURL, pagePath)
	staticOG := makeOG(p.Site.Name, p.Site.Name, p.Site.Description, staticPageURL, "", "website")

	staticWebPage := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "WebPage",
		"name":     p.Site.Name,
		"url":      staticPageURL,
		"isPartOf": map[string]interface{}{
			"@type": "WebSite",
			"name":  p.Site.Name,
			"url":   p.Site.BaseURL,
		},
	}
	staticJsonLD := schema.MarshalSchemas(staticWebPage)

	ctx := render.StaticPageContext{
		Site:          p.Site,
		JsonLD:        template.HTML(staticJsonLD),
		OG:            staticOG,
		AllTaxonomies: allTaxonomies,
		Taxonomies:    allTaxonomies,
		Favorites:     favorites,
		CTA:           p.Config.Extra.CTA,
	}

	htmlOut, err := engine.RenderStatic(tmpl, ctx)
	if err != nil {
		return err
	}

	outPath := filepath.Join(outputDir, pagePath)
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("creating dir for %s: %w", pagePath, err)
	}
	return os.WriteFile(outPath, []byte(htmlOut), 0644)
}

// --- Helpers ---

type searchEntry struct {
	T string `json:"t"`
	D string `json:"d,omitempty"`
	S string `json:"s"`
	N string `json:"n,omitempty"`
	L string `json:"l,omitempty"`
	M string `json:"m,omitempty"`
}

func generateSearchIndex(p *ir.Program, outputDir string) error {
	if !p.Config.Search.Enabled {
		return nil
	}

	entries := make([]searchEntry, 0, len(p.Entities))
	for _, re := range p.Entities {
		e := re.Raw
		desc := e.GetString("description")
		if len(desc) > 120 {
			desc = desc[:120]
		}
		entries = append(entries, searchEntry{
			T: e.GetString("title"),
			D: desc,
			S: e.Slug,
			N: e.GetString("node_type"),
			L: e.GetString("language"),
			M: e.GetString("domain"),
		})
	}

	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}

	outPath := filepath.Join(outputDir, "search-index.json")
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return err
	}
	log.Printf("  Generated search index (%d entries, %dKB)", len(entries), len(data)/1024)
	return nil
}

func toBreadcrumbItems(breadcrumbs []render.Breadcrumb) []schema.BreadcrumbItem {
	items := make([]schema.BreadcrumbItem, len(breadcrumbs))
	for i, bc := range breadcrumbs {
		items[i] = schema.BreadcrumbItem{Name: bc.Name, URL: bc.URL}
	}
	return items
}

func makeOG(siteName, title, description, pageURL, imageURL, ogType string) render.OGMeta {
	return render.OGMeta{
		Title:       title,
		Description: description,
		URL:         pageURL,
		ImageURL:    imageURL,
		Type:        ogType,
		SiteName:    siteName,
	}
}

func writeShareSVG(outDir, baseURL, filename, svg string) string {
	dir := filepath.Join(outDir, "images", "share")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, filename), []byte(svg), 0644)
	return fmt.Sprintf("%s/images/share/%s", baseURL, filename)
}

func countFieldDistribution(entities []*entity.Entity, field string, limit int) []render.NameCount {
	counts := make(map[string]int)
	for _, e := range entities {
		if v := e.GetString(field); v != "" {
			counts[v]++
		}
	}
	result := make([]render.NameCount, 0, len(counts))
	for name, count := range counts {
		result = append(result, render.NameCount{Name: name, Count: count})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Count > result[j].Count })
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return err
			}
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// archMapData is the structure of the arch_map frontmatter field.
type archMapData struct {
	Domain    *archMapNode `json:"domain,omitempty"`
	Subdomain *archMapNode `json:"subdomain,omitempty"`
	File      *archMapNode `json:"file,omitempty"`
	Entity    *archMapNode `json:"entity,omitempty"`
}
type archMapNode struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Type string `json:"type,omitempty"`
}

func generateArchMapSVG(archMapJSON string) string {
	var data archMapData
	if err := json.Unmarshal([]byte(archMapJSON), &data); err != nil {
		return ""
	}

	type item struct{ name string }
	var items []item
	if data.Domain != nil {
		items = append(items, item{name: data.Domain.Name})
	}
	if data.Subdomain != nil {
		items = append(items, item{name: data.Subdomain.Name})
	}
	if data.File != nil {
		items = append(items, item{name: data.File.Name})
	}
	if data.Entity != nil {
		items = append(items, item{name: data.Entity.Name})
	}
	if len(items) < 2 {
		return ""
	}

	boxW, boxH, arrowW, pad := 140, 36, 28, 12
	totalW := len(items)*boxW + (len(items)-1)*arrowW + pad*2
	totalH := boxH + pad*2

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" style="background:#0f1117">`, totalW, totalH, totalW, totalH))

	for i, it := range items {
		x := pad + i*(boxW+arrowW)
		y := pad
		isLast := i == len(items)-1
		fill := "#1a1d27"
		stroke := "#2a2e3e"
		textColor := "#e4e4e7"
		if isLast {
			fill = "#6366f1"
			stroke = "#818cf8"
			textColor = "#fff"
		}
		label := it.name
		if len(label) > 16 {
			label = label[:14] + ".."
		}
		label = strings.ReplaceAll(label, "&", "&amp;")
		label = strings.ReplaceAll(label, "<", "&lt;")
		label = strings.ReplaceAll(label, ">", "&gt;")

		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="6" fill="%s" stroke="%s" stroke-width="1"/>`,
			x, y, boxW, boxH, fill, stroke))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" fill="%s" font-size="12" font-family="Inter,system-ui,sans-serif">%s</text>`,
			x+boxW/2, y+boxH/2+5, textColor, label))

		if i < len(items)-1 {
			ax := x + boxW + 4
			ay := y + boxH/2
			sb.WriteString(fmt.Sprintf(`<path d="M%d %d L%d %d" stroke="#2a2e3e" stroke-width="1.5" fill="none"/>`,
				ax, ay, ax+arrowW-8, ay))
			sb.WriteString(fmt.Sprintf(`<polygon points="%d,%d %d,%d %d,%d" fill="#2a2e3e"/>`,
				ax+arrowW-8, ay-4, ax+arrowW-2, ay, ax+arrowW-8, ay+4))
		}
	}

	sb.WriteString(`</svg>`)
	return sb.String()
}
