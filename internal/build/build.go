package build

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

	"github.com/greynewell/pssg/internal/affiliate"
	"github.com/greynewell/pssg/internal/config"
	"github.com/greynewell/pssg/internal/enrichment"
	"github.com/greynewell/pssg/internal/entity"
	"github.com/greynewell/pssg/internal/loader"
	"github.com/greynewell/pssg/internal/output"
	"github.com/greynewell/pssg/internal/render"
	"github.com/greynewell/pssg/internal/schema"
	"github.com/greynewell/pssg/internal/taxonomy"
)

// Builder orchestrates the entire static site generation pipeline.
type Builder struct {
	cfg   *config.Config
	force bool
}

// NewBuilder creates a new builder.
func NewBuilder(cfg *config.Config, force bool) *Builder {
	return &Builder{cfg: cfg, force: force}
}

// Build runs the complete build pipeline.
func (b *Builder) Build() error {
	start := time.Now()
	log.Printf("Building site: %s", b.cfg.Site.Name)

	// 1. Load entities
	log.Printf("Loading entities from %s...", b.cfg.Paths.Data)
	ldr := loader.New(b.cfg)
	entities, err := ldr.Load()
	if err != nil {
		return fmt.Errorf("loading entities: %w", err)
	}
	log.Printf("Loaded %d entities", len(entities))

	// 2. Build slug lookup
	slugMap := make(map[string]*entity.Entity)
	for _, e := range entities {
		slugMap[e.Slug] = e
	}

	// 3. Load enrichment cache
	enrichmentData := make(map[string]map[string]interface{})
	if b.cfg.Enrichment.CacheDir != "" {
		log.Printf("Loading enrichment cache from %s...", b.cfg.Enrichment.CacheDir)
		var err error
		enrichmentData, err = enrichment.ReadAllCaches(b.cfg.Enrichment.CacheDir)
		if err != nil {
			log.Printf("Warning: failed to load enrichment cache: %v", err)
		} else {
			log.Printf("Loaded enrichment data for %d entities", len(enrichmentData))
		}
	}

	// 4. Load extra data
	favorites := b.loadFavorites(slugMap)
	contributors := b.loadContributors()

	// 5. Set up affiliate registry
	affiliateRegistry := affiliate.NewRegistry(b.cfg.Affiliates)

	// 6. Build taxonomies
	log.Printf("Building taxonomies...")
	taxonomies := taxonomy.BuildAll(entities, b.cfg.Taxonomies, enrichmentData)
	for _, tax := range taxonomies {
		log.Printf("  %s: %d entries", tax.Label, len(tax.Entries))
	}

	// 7. Build valid taxonomy slug lookup
	validSlugs := make(map[string]map[string]bool)
	for _, tax := range taxonomies {
		slugSet := make(map[string]bool)
		for _, entry := range tax.Entries {
			slugSet[entry.Slug] = true
		}
		validSlugs[tax.Name] = slugSet
	}

	// 8. Ensure output directory exists
	outDir := b.cfg.Paths.Output
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	// 9. Initialize render engine
	log.Printf("Loading templates from %s...", b.cfg.Paths.Templates)
	engine, err := render.NewEngine(b.cfg)
	if err != nil {
		return fmt.Errorf("initializing render engine: %w", err)
	}

	// 10. Extract CSS/JS
	if b.cfg.Output.ExtractCSS != "" {
		cssContent, err := engine.RenderCSS()
		if err != nil {
			log.Printf("Warning: failed to render CSS: %v", err)
		} else if cssContent != "" {
			cssPath := filepath.Join(outDir, b.cfg.Output.ExtractCSS)
			if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
				return fmt.Errorf("writing CSS: %w", err)
			}
		}
	}
	if b.cfg.Output.ExtractJS != "" {
		jsContent, err := engine.RenderJS()
		if err != nil {
			log.Printf("Warning: failed to render JS: %v", err)
		} else if jsContent != "" {
			jsPath := filepath.Join(outDir, b.cfg.Output.ExtractJS)
			if err := os.WriteFile(jsPath, []byte(jsContent), 0644); err != nil {
				return fmt.Errorf("writing JS: %w", err)
			}
		}
	}

	// JSON-LD generator
	schemaGen := schema.NewGenerator(b.cfg.Site, b.cfg.Schema)

	// Track sitemap entries
	var sitemapEntries []output.SitemapEntry
	var sitemapMu sync.Mutex
	today := time.Now().Format("2006-01-02")

	addSitemapEntry := func(path, priority, changefreq string) {
		sitemapMu.Lock()
		defer sitemapMu.Unlock()
		sitemapEntries = append(sitemapEntries, output.NewSitemapEntry(
			b.cfg.Site.BaseURL, path, today, priority, changefreq,
		))
	}

	// Track category taxonomy entries for RSS
	categoryEntries := make(map[string][]*entity.Entity)

	// 11. Render entity pages (concurrent)
	log.Printf("Rendering %d entity pages...", len(entities))
	var entityErrors int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, 32) // 32-goroutine pool

	for _, e := range entities {
		wg.Add(1)
		sem <- struct{}{} // acquire
		go func(e *entity.Entity) {
			defer wg.Done()
			defer func() { <-sem }() // release

			err := b.renderEntityPage(e, engine, schemaGen, slugMap, enrichmentData,
				affiliateRegistry, taxonomies, validSlugs, contributors, outDir, addSitemapEntry)
			if err != nil {
				atomic.AddInt64(&entityErrors, 1)
				fmt.Fprintf(os.Stderr, "Warning: failed to render %s: %v\n", e.Slug, err)
			}
		}(e)
	}
	wg.Wait()
	if entityErrors > 0 {
		log.Printf("  %d entity pages had errors", entityErrors)
	}

	// 11b. Generate search index
	if len(entities) > 0 {
		if err := b.generateSearchIndex(entities, outDir); err != nil {
			log.Printf("Warning: failed to generate search index: %v", err)
		}
	}

	// Build category entries for RSS
	for _, tax := range taxonomies {
		if tax.Name == b.cfg.RSS.CategoryTaxonomy {
			for _, entry := range tax.Entries {
				categoryEntries[entry.Slug] = entry.Entities
			}
		}
	}

	// 12. Render taxonomy pages
	log.Printf("Rendering taxonomy pages...")
	for _, tax := range taxonomies {
		if err := b.renderTaxonomyPages(tax, engine, schemaGen, taxonomies, contributors, outDir, addSitemapEntry, today); err != nil {
			return fmt.Errorf("rendering taxonomy %s: %w", tax.Name, err)
		}
	}

	// 12b. Render all-entities pages
	if b.cfg.Templates.AllEntities != "" {
		log.Printf("Rendering all-entities pages...")
		if err := b.renderAllEntitiesPages(engine, schemaGen, entities, taxonomies, outDir, addSitemapEntry); err != nil {
			return fmt.Errorf("rendering all-entities pages: %w", err)
		}
	}

	// 13. Render homepage
	log.Printf("Rendering homepage...")
	if err := b.renderHomepage(engine, schemaGen, entities, taxonomies, favorites, contributors, outDir); err != nil {
		return fmt.Errorf("rendering homepage: %w", err)
	}
	addSitemapEntry("/index.html", b.cfg.Sitemap.Priorities["homepage"], b.cfg.Sitemap.ChangeFreqs["homepage"])

	// 14. Render static pages
	for path, tmpl := range b.cfg.Templates.StaticPages {
		ctx := render.StaticPageContext{
			Site:          b.cfg.Site,
			AllTaxonomies: taxonomies,
		}
		html, err := engine.RenderStatic(tmpl, ctx)
		if err != nil {
			log.Printf("Warning: failed to render static page %s: %v", path, err)
			continue
		}
		outPath := filepath.Join(outDir, path)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return fmt.Errorf("creating dir for %s: %w", path, err)
		}
		if err := os.WriteFile(outPath, []byte(html), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	// 15. Generate sitemap
	log.Printf("Generating sitemap (%d entries)...", len(sitemapEntries))
	sitemapFiles := output.GenerateSitemapFiles(sitemapEntries, b.cfg.Site.BaseURL, b.cfg.Sitemap.MaxURLsPerFile)
	for _, sf := range sitemapFiles {
		if err := os.WriteFile(filepath.Join(outDir, sf.Filename), []byte(sf.Content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", sf.Filename, err)
		}
	}
	log.Printf("  Generated %d sitemap file(s)", len(sitemapFiles))

	// 16. Generate RSS
	rssFeeds := output.GenerateRSSFeeds(entities, b.cfg, categoryEntries)
	for _, feed := range rssFeeds {
		feedPath := filepath.Join(outDir, feed.RelativePath)
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

	// 17. Generate robots.txt
	robotsContent := output.GenerateRobotsTxt(b.cfg)
	if err := os.WriteFile(filepath.Join(outDir, "robots.txt"), []byte(robotsContent), 0644); err != nil {
		return fmt.Errorf("writing robots.txt: %w", err)
	}

	// 18. Generate llms.txt
	if b.cfg.LlmsTxt.Enabled {
		llmsContent := output.GenerateLlmsTxt(b.cfg, entities, taxonomies)
		if err := os.WriteFile(filepath.Join(outDir, "llms.txt"), []byte(llmsContent), 0644); err != nil {
			return fmt.Errorf("writing llms.txt: %w", err)
		}
	}

	// 19. Generate manifest.json
	manifestContent := output.GenerateManifest(b.cfg)
	if err := os.WriteFile(filepath.Join(outDir, "manifest.json"), []byte(manifestContent), 0644); err != nil {
		return fmt.Errorf("writing manifest.json: %w", err)
	}

	// 20. Write CNAME if configured
	if b.cfg.Site.CNAME != "" {
		if err := os.WriteFile(filepath.Join(outDir, "CNAME"), []byte(b.cfg.Site.CNAME+"\n"), 0644); err != nil {
			return fmt.Errorf("writing CNAME: %w", err)
		}
	}

	// 21. Copy static assets
	if b.cfg.Paths.Static != "" {
		if err := copyDir(b.cfg.Paths.Static, outDir); err != nil {
			log.Printf("Warning: failed to copy static assets: %v", err)
		}
	}

	elapsed := time.Since(start)
	log.Printf("\nBuild complete!")
	log.Printf("  Entities:  %d", len(entities))
	log.Printf("  Taxonomies: %d (%d total entries)", len(taxonomies), countTaxEntries(taxonomies))
	log.Printf("  Sitemap:   %d URLs in %d file(s)", len(sitemapEntries), len(sitemapFiles))
	log.Printf("  Output:    %s", outDir)
	log.Printf("  Duration:  %s", elapsed.Round(time.Millisecond))

	return nil
}

func (b *Builder) renderEntityPage(
	e *entity.Entity,
	engine *render.Engine,
	schemaGen *schema.Generator,
	slugMap map[string]*entity.Entity,
	enrichmentData map[string]map[string]interface{},
	affiliateReg *affiliate.Registry,
	taxonomies []taxonomy.Taxonomy,
	validSlugs map[string]map[string]bool,
	contributors map[string]interface{},
	outDir string,
	addSitemapEntry func(string, string, string),
) error {
	entityURL := fmt.Sprintf("%s/%s.html", b.cfg.Site.BaseURL, e.Slug)

	// Resolve pairings
	var pairings []*entity.Entity
	if pairingsSlugs := e.GetStringSlice("pairings"); len(pairingsSlugs) > 0 {
		for _, ps := range pairingsSlugs {
			if paired, ok := slugMap[ps]; ok {
				pairings = append(pairings, paired)
			}
		}
	}

	// Enrichment data for this entity
	eData := enrichmentData[e.Slug]

	// Generate affiliate links
	var affLinks []affiliate.Link
	if eData != nil {
		affLinks = affiliateReg.GenerateLinks(eData, b.cfg.Affiliates.SearchTermPaths)
	}

	// Cook mode prompt
	cookPrompt := render.GenerateCookModePrompt(e, eData, affLinks)

	// JSON-LD
	entitySchema := schemaGen.GenerateEntitySchema(e, entityURL)

	// Breadcrumbs
	var breadcrumbs []render.Breadcrumb
	breadcrumbs = append(breadcrumbs, render.Breadcrumb{Name: "Home", URL: b.cfg.Site.BaseURL + "/"})
	if cat := e.GetString("recipe_category"); cat != "" {
		catSlug := entity.ToSlug(cat)
		breadcrumbs = append(breadcrumbs, render.Breadcrumb{
			Name: cat,
			URL:  fmt.Sprintf("%s/category/%s.html", b.cfg.Site.BaseURL, catSlug),
		})
	}
	breadcrumbs = append(breadcrumbs, render.Breadcrumb{Name: e.GetString("title"), URL: ""})

	breadcrumbSchema := schemaGen.GenerateBreadcrumbSchema(toBreadcrumbItems(breadcrumbs))

	// FAQ schema
	var faqSchema map[string]interface{}
	if faqs := e.GetFAQs(); len(faqs) > 0 {
		faqSchema = schemaGen.GenerateFAQSchema(faqs)
	}

	jsonLD := schema.MarshalSchemas(entitySchema, breadcrumbSchema, faqSchema)

	// Share image + OG
	entityShareSVG := render.GenerateEntityShareSVG(
		b.cfg.Site.Name,
		e.GetString("title"),
		e.GetString("node_type"),
		e.GetString("language"),
		e.GetString("domain"),
	)
	entityImageURL := writeShareSVG(outDir, b.cfg.Site.BaseURL, e.Slug+".svg", entityShareSVG)

	og := makeOG(b.cfg.Site.Name, e.GetString("title")+" | "+b.cfg.Site.Name,
		e.GetString("description"), entityURL, entityImageURL, "article")

	// Compact entity chart data — only non-zero values, short keys
	chartMap := make(map[string]interface{})
	if v := e.GetInt("line_count"); v > 0 { chartMap["lc"] = v }
	if v := e.GetInt("start_line"); v > 0 { chartMap["sl"] = v }
	if v := e.GetInt("end_line"); v > 0 { chartMap["el"] = v }
	if v := e.GetInt("call_count"); v > 0 { chartMap["co"] = v }
	if v := e.GetInt("called_by_count"); v > 0 { chartMap["cb"] = v }
	if v := e.GetInt("import_count"); v > 0 { chartMap["ic"] = v }
	if v := e.GetInt("imported_by_count"); v > 0 { chartMap["ib"] = v }
	if v := e.GetInt("function_count"); v > 0 { chartMap["fn"] = v }
	if v := e.GetInt("class_count"); v > 0 { chartMap["cl"] = v }
	if v := e.GetInt("type_count"); v > 0 { chartMap["tc"] = v }
	if v := e.GetInt("file_count"); v > 0 { chartMap["fc"] = v }

	// Enrich graph_data: add lineCount/language to nodes, count edge types
	if graphJSON := e.GetString("graph_data"); graphJSON != "" {
		var graphObj struct {
			Nodes []map[string]interface{} `json:"nodes"`
			Edges []struct {
				Source string `json:"source"`
				Target string `json:"target"`
				Type   string `json:"type"`
			} `json:"edges"`
		}
		if json.Unmarshal([]byte(graphJSON), &graphObj) == nil {
			// Enrich nodes with metadata from slugMap
			for i, node := range graphObj.Nodes {
				if slugVal, ok := node["slug"].(string); ok && slugVal != "" {
					if ref, found := slugMap[slugVal]; found {
						if lc := ref.GetInt("line_count"); lc > 0 {
							graphObj.Nodes[i]["lc"] = lc
						}
						if lang := ref.GetString("language"); lang != "" {
							graphObj.Nodes[i]["lang"] = lang
						}
						if cc := ref.GetInt("call_count"); cc > 0 {
							graphObj.Nodes[i]["cc"] = cc
						}
						if cbc := ref.GetInt("called_by_count"); cbc > 0 {
							graphObj.Nodes[i]["cbc"] = cbc
						}
					}
				}
			}
			// Write enriched graph back
			enrichedGraph, _ := json.Marshal(graphObj)
			e.Fields["graph_data"] = string(enrichedGraph)

			// Edge type counts for chart
			etCounts := make(map[string]int)
			for _, edge := range graphObj.Edges {
				etCounts[edge.Type]++
			}
			chartMap["et"] = etCounts
			chartMap["nc"] = len(graphObj.Nodes)
			chartMap["ec"] = len(graphObj.Edges)
		}
	}

	entityChartJSON, _ := json.Marshal(chartMap)

	// Extract source code snippet
	var sourceCode, sourceLang string
	if srcDir := b.cfg.Paths.SourceDir; srcDir != "" {
		if filePath := e.GetString("file_path"); filePath != "" {
			startLine := e.GetInt("start_line")
			endLine := e.GetInt("end_line")
			sourceLang = e.GetString("language")
			absPath := filepath.Join(srcDir, filePath)
			if data, err := os.ReadFile(absPath); err == nil {
				lines := strings.Split(string(data), "\n")
				if startLine > 0 && endLine > 0 && startLine <= len(lines) {
					if endLine > len(lines) {
						endLine = len(lines)
					}
					// Cap snippet at 80 lines for sanity
					if endLine-startLine > 80 {
						endLine = startLine + 80
					}
					sourceCode = strings.Join(lines[startLine-1:endLine], "\n")
				} else if e.GetString("node_type") == "File" && len(lines) <= 120 {
					// For small files, show entire content
					sourceCode = string(data)
				} else if e.GetString("node_type") == "File" && len(lines) > 120 {
					// For large files, show first 60 lines
					sourceCode = strings.Join(lines[:60], "\n") + "\n// ... (" + fmt.Sprintf("%d", len(lines)-60) + " more lines)"
				}
			}
		}
	}

	ctx := render.EntityPageContext{
		Site:           b.cfg.Site,
		Entity:         e,
		Slug:           e.Slug,
		URL:            entityURL,
		CanonicalURL:   entityURL,
		Breadcrumbs:    breadcrumbs,
		Pairings:       pairings,
		Enrichment:     eData,
		AffiliateLinks: affLinks,
		CookModePrompt: cookPrompt,
		JsonLD:         toTemplateHTML(jsonLD),
		OG:             og,
		ChartData:      template.JS(entityChartJSON),
		SourceCode:     sourceCode,
		SourceLang:     sourceLang,
		Taxonomies:     taxonomies,
		AllTaxonomies:  taxonomies,
		ValidSlugs:     validSlugs,
		Contributors:   contributors,
		CTA:            b.cfg.Extra.CTA,
	}

	html, err := engine.RenderEntity(ctx)
	if err != nil {
		return err
	}

	outPath := filepath.Join(outDir, e.Slug+".html")
	if err := os.WriteFile(outPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}

	// Generate static arch map SVG for structured data image
	if archMap := e.GetString("arch_map"); archMap != "" {
		imagesDir := filepath.Join(outDir, "images")
		os.MkdirAll(imagesDir, 0755)
		if svg := generateArchMapSVG(archMap); svg != "" {
			svgPath := filepath.Join(imagesDir, e.Slug+"-arch.svg")
			os.WriteFile(svgPath, []byte(svg), 0644)
		}
	}

	addSitemapEntry("/"+e.Slug+".html",
		b.cfg.Sitemap.Priorities["entity"],
		b.cfg.Sitemap.ChangeFreqs["entity"])

	return nil
}

func (b *Builder) renderTaxonomyPages(
	tax taxonomy.Taxonomy,
	engine *render.Engine,
	schemaGen *schema.Generator,
	allTaxonomies []taxonomy.Taxonomy,
	contributors map[string]interface{},
	outDir string,
	addSitemapEntry func(string, string, string),
	today string,
) error {
	// Ensure taxonomy type directory exists
	taxDir := filepath.Join(outDir, tax.Name)
	if err := os.MkdirAll(taxDir, 0755); err != nil {
		return fmt.Errorf("creating taxonomy dir: %w", err)
	}

	perPage := b.cfg.Pagination.EntitiesPerPage

	// Render hub pages for each entry
	for _, entry := range tax.Entries {
		totalPages := (len(entry.Entities) + perPage - 1) / perPage
		if totalPages == 0 {
			totalPages = 1
		}

		// Hub share image + chart data (compute once per entry, reuse for all pages)
		hubTypeDist := countFieldDistribution(entry.Entities, "node_type", 8)
		hubLangDist := countFieldDistribution(entry.Entities, "language", 8)
		hubDomainDist := countFieldDistribution(entry.Entities, "domain", 8)
		hubExtDist := countFieldDistribution(entry.Entities, "extension", 8)
		// Top entities by line count
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

		hubSVG := render.GenerateHubShareSVG(b.cfg.Site.Name, entry.Name, tax.Label, len(entry.Entities), hubTypeDist)
		hubImageURL := writeShareSVG(outDir, b.cfg.Site.BaseURL, fmt.Sprintf("%s-%s.svg", tax.Name, entry.Slug), hubSVG)

		for page := 1; page <= totalPages; page++ {
			pagination := taxonomy.ComputePagination(entry, page, perPage, tax.Name)

			// Get entities for this page
			pageEntities := entry.Entities
			if pagination.StartIndex < len(entry.Entities) {
				end := pagination.EndIndex
				if end > len(entry.Entities) {
					end = len(entry.Entities)
				}
				pageEntities = entry.Entities[pagination.StartIndex:end]
			}

			// JSON-LD
			pageURL := fmt.Sprintf("%s%s", b.cfg.Site.BaseURL, taxonomy.HubPageURL(tax.Name, entry.Slug, page))
			var items []schema.ItemListEntry
			for _, e := range pageEntities {
				items = append(items, schema.ItemListEntry{
					Name: e.GetString("title"),
					URL:  fmt.Sprintf("%s/%s.html", b.cfg.Site.BaseURL, e.Slug),
				})
			}
			collectionSchema := schemaGen.GenerateCollectionPageSchema(
				entry.Name, fmt.Sprintf("%s %s entities", entry.Name, tax.LabelSingular),
				pageURL, items, hubImageURL,
			)

			// Breadcrumbs
			breadcrumbs := []render.Breadcrumb{
				{Name: "Home", URL: b.cfg.Site.BaseURL + "/"},
				{Name: tax.Label, URL: fmt.Sprintf("%s/%s/", b.cfg.Site.BaseURL, tax.Name)},
				{Name: entry.Name, URL: ""},
			}
			breadcrumbSchema := schemaGen.GenerateBreadcrumbSchema(toBreadcrumbItems(breadcrumbs))
			jsonLD := schema.MarshalSchemas(collectionSchema, breadcrumbSchema)

			// Contributor profile for author taxonomy
			var contributorProfile map[string]interface{}
			if tax.Name == "author" && contributors != nil {
				if profiles, ok := contributors["profiles"].(map[string]interface{}); ok {
					contributorProfile, _ = profiles[entry.Slug].(map[string]interface{})
				}
			}

			og := makeOG(b.cfg.Site.Name,
				fmt.Sprintf("%s — %s | %s", entry.Name, tax.Label, b.cfg.Site.Name),
				fmt.Sprintf("Browse all %d %s entities in %s", len(entry.Entities), entry.Name, b.cfg.Site.Name),
				pageURL, hubImageURL, "article")

			ctx := render.HubPageContext{
				Site:               b.cfg.Site,
				Taxonomy:           tax,
				Entry:              entry,
				Entities:           pageEntities,
				Pagination:         pagination,
				JsonLD:             toTemplateHTML(jsonLD),
				OG:                 og,
				ChartData:          template.JS(hubChartJSON),
				Breadcrumbs:        breadcrumbs,
				AllTaxonomies:      allTaxonomies,
				Contributors:       contributors,
				ContributorProfile: contributorProfile,
				CTA:                b.cfg.Extra.CTA,
			}

			html, err := engine.RenderHub(ctx)
			if err != nil {
				return fmt.Errorf("rendering hub %s/%s page %d: %w", tax.Name, entry.Slug, page, err)
			}

			// Determine filename
			var filename string
			if page == 1 {
				filename = entry.Slug + ".html"
			} else {
				filename = fmt.Sprintf("%s-page-%d.html", entry.Slug, page)
			}

			if err := os.WriteFile(filepath.Join(taxDir, filename), []byte(html), 0644); err != nil {
				return fmt.Errorf("writing hub page: %w", err)
			}

			// Sitemap
			priority := b.cfg.Sitemap.Priorities["hub_page_1"]
			if page > 1 {
				priority = b.cfg.Sitemap.Priorities["hub_page_n"]
			}
			addSitemapEntry(fmt.Sprintf("/%s/%s", tax.Name, filename), priority, b.cfg.Sitemap.ChangeFreqs["hub"])
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

	// Tax index chart data + share image
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

	taxIdxSVG := render.GenerateTaxIndexShareSVG(b.cfg.Site.Name, tax.Label, taxIndexStats)
	taxIdxImageURL := writeShareSVG(outDir, b.cfg.Site.BaseURL, fmt.Sprintf("%s-index.svg", tax.Name), taxIdxSVG)

	// Index page JSON-LD
	var indexItems []schema.ItemListEntry
	for _, entry := range tax.Entries {
		indexItems = append(indexItems, schema.ItemListEntry{
			Name: entry.Name,
			URL:  fmt.Sprintf("%s/%s/%s.html", b.cfg.Site.BaseURL, tax.Name, entry.Slug),
		})
	}
	indexSchema := schemaGen.GenerateItemListSchema(tax.Label, fmt.Sprintf("Browse all %s", tax.Label), indexItems, taxIdxImageURL)
	breadcrumbs := []render.Breadcrumb{
		{Name: "Home", URL: b.cfg.Site.BaseURL + "/"},
		{Name: tax.Label, URL: ""},
	}
	breadcrumbSchema := schemaGen.GenerateBreadcrumbSchema(toBreadcrumbItems(breadcrumbs))
	jsonLD := schema.MarshalSchemas(indexSchema, breadcrumbSchema)

	taxIdxPageURL := fmt.Sprintf("%s/%s/", b.cfg.Site.BaseURL, tax.Name)
	taxIdxOG := makeOG(b.cfg.Site.Name,
		fmt.Sprintf("%s | %s", tax.Label, b.cfg.Site.Name),
		fmt.Sprintf("Browse architecture documentation by %s. %d categories available.", strings.ToLower(tax.Label), len(tax.Entries)),
		taxIdxPageURL, taxIdxImageURL, "article")

	ctx := render.TaxonomyIndexContext{
		Site:          b.cfg.Site,
		Taxonomy:      tax,
		Entries:       tax.Entries,
		TopEntries:    topEntries,
		LetterGroups:  letterGroups,
		HasLetters:    hasLetters,
		Letters:       letters,
		JsonLD:        toTemplateHTML(jsonLD),
		OG:            taxIdxOG,
		ChartData:     template.JS(taxChartJSON),
		Breadcrumbs:   breadcrumbs,
		AllTaxonomies: allTaxonomies,
		CTA:           b.cfg.Extra.CTA,
	}

	html, err := engine.RenderTaxonomyIndex(ctx)
	if err != nil {
		return fmt.Errorf("rendering taxonomy index %s: %w", tax.Name, err)
	}

	if err := os.WriteFile(filepath.Join(taxDir, "index.html"), []byte(html), 0644); err != nil {
		return fmt.Errorf("writing taxonomy index: %w", err)
	}
	addSitemapEntry(fmt.Sprintf("/%s/", tax.Name), b.cfg.Sitemap.Priorities["taxonomy_index"], b.cfg.Sitemap.ChangeFreqs["taxonomy_index"])

	// Render letter pages if threshold met
	if hasLetters {
		for _, lg := range letterGroups {
			letterBreadcrumbs := []render.Breadcrumb{
				{Name: "Home", URL: b.cfg.Site.BaseURL + "/"},
				{Name: tax.Label, URL: fmt.Sprintf("%s/%s/", b.cfg.Site.BaseURL, tax.Name)},
				{Name: fmt.Sprintf("Letter %s", lg.Letter), URL: ""},
			}

			// Letter chart data + share image
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

			letterSVG := render.GenerateLetterShareSVG(b.cfg.Site.Name, tax.Label, lg.Letter, len(lg.Entries))
			letterSafeFile := strings.ToLower(lg.Letter)
			if letterSafeFile == "#" {
				letterSafeFile = "num"
			}
			letterImageURL := writeShareSVG(outDir, b.cfg.Site.BaseURL, fmt.Sprintf("%s-letter-%s.svg", tax.Name, letterSafeFile), letterSVG)

			letterFile := fmt.Sprintf("letter-%s.html", letterSafeFile)
			letterPageURL := fmt.Sprintf("%s/%s/%s", b.cfg.Site.BaseURL, tax.Name, letterFile)
			letterOG := makeOG(b.cfg.Site.Name,
				fmt.Sprintf("%s — %s | %s", tax.Label, lg.Letter, b.cfg.Site.Name),
				fmt.Sprintf("Browse %s entries starting with %s", strings.ToLower(tax.Label), lg.Letter),
				letterPageURL, letterImageURL, "article")

			letterCtx := render.LetterPageContext{
				Site:          b.cfg.Site,
				Taxonomy:      tax,
				Letter:        lg.Letter,
				Entries:       lg.Entries,
				Letters:       letters,
				JsonLD:        toTemplateHTML(""),
				OG:            letterOG,
				ChartData:     template.JS(letterChartJSON),
				Breadcrumbs:   letterBreadcrumbs,
				AllTaxonomies: allTaxonomies,
				CTA:           b.cfg.Extra.CTA,
			}

			letterHTML, err := engine.RenderLetter(letterCtx)
			if err != nil {
				return fmt.Errorf("rendering letter page %s/%s: %w", tax.Name, lg.Letter, err)
			}

			if err := os.WriteFile(filepath.Join(taxDir, letterFile), []byte(letterHTML), 0644); err != nil {
				return fmt.Errorf("writing letter page: %w", err)
			}
			addSitemapEntry(fmt.Sprintf("/%s/%s", tax.Name, letterFile),
				b.cfg.Sitemap.Priorities["letter_page"],
				b.cfg.Sitemap.ChangeFreqs["letter_page"])
		}
	}

	return nil
}

func (b *Builder) renderHomepage(
	engine *render.Engine,
	schemaGen *schema.Generator,
	entities []*entity.Entity,
	taxonomies []taxonomy.Taxonomy,
	favorites []*entity.Entity,
	contributors map[string]interface{},
	outDir string,
) error {
	// Share image
	var taxStats []render.NameCount
	for _, tax := range taxonomies {
		taxStats = append(taxStats, render.NameCount{Name: tax.Label, Count: len(tax.Entries)})
	}
	svg := render.GenerateHomepageShareSVG(b.cfg.Site.Name, b.cfg.Site.Description, taxStats, len(entities))
	imageURL := writeShareSVG(outDir, b.cfg.Site.BaseURL, "homepage.svg", svg)

	// Chart data (with slugs for clickable treemap)
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
	for _, tax := range taxonomies {
		taxChartEntries = append(taxChartEntries, taxChartEntry{
			Name:  tax.Label,
			Count: len(tax.Entries),
			Slug:  tax.Name,
		})
	}
	chartObj := homepageChart{Taxonomies: taxChartEntries, TotalEntities: len(entities)}
	chartJSON, _ := json.Marshal(chartObj)

	// Architecture graph data: domains → subdomains
	type archNode struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Type  string `json:"type"` // "root", "domain", "subdomain"
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

	// Root node
	archNodes = append(archNodes, archNode{
		ID: "root", Name: b.cfg.Site.Name, Type: "root", Count: len(entities),
	})

	// Find domain and subdomain taxonomies
	var domainTax, subdomainTax *taxonomy.Taxonomy
	for i := range taxonomies {
		if taxonomies[i].Name == "domain" {
			domainTax = &taxonomies[i]
		}
		if taxonomies[i].Name == "subdomain" {
			subdomainTax = &taxonomies[i]
		}
	}

	// Build domain→subdomain mapping via entity field overlap
	if domainTax != nil {
		for _, domEntry := range domainTax.Entries {
			domID := "domain-" + domEntry.Slug
			archNodes = append(archNodes, archNode{
				ID: domID, Name: domEntry.Name, Type: "domain", Count: len(domEntry.Entities), Slug: "domain/" + domEntry.Slug,
			})
			archLinks = append(archLinks, archLink{Source: "root", Target: domID})

			if subdomainTax != nil {
				// For each subdomain, check if its entities overlap with this domain
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
						// Only add subdomain node once
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

	var items []schema.ItemListEntry
	for _, e := range entities {
		items = append(items, schema.ItemListEntry{
			Name: e.GetString("title"),
			URL:  fmt.Sprintf("%s/%s.html", b.cfg.Site.BaseURL, e.Slug),
		})
	}
	itemListSchema := schemaGen.GenerateItemListSchema(
		b.cfg.Site.Name,
		b.cfg.Site.Description,
		items,
		imageURL,
	)

	jsonLD := schema.MarshalSchemas(websiteSchema, itemListSchema)

	pageURL := b.cfg.Site.BaseURL + "/"
	ctx := render.HomepageContext{
		Site:         b.cfg.Site,
		Entities:     entities,
		Taxonomies:   taxonomies,
		Favorites:    favorites,
		JsonLD:       toTemplateHTML(jsonLD),
		OG:           makeOG(b.cfg.Site.Name, b.cfg.Site.Name+" — Architecture Documentation", b.cfg.Site.Description, pageURL, imageURL, "website"),
		ChartData:    template.JS(chartJSON),
		ArchData:     template.JS(archJSON),
		EntityCount:  len(entities),
		Contributors: contributors,
		CTA:          b.cfg.Extra.CTA,
	}

	html, err := engine.RenderHomepage(ctx)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(outDir, "index.html"), []byte(html), 0644)
}

func (b *Builder) renderAllEntitiesPages(
	engine *render.Engine,
	schemaGen *schema.Generator,
	entities []*entity.Entity,
	allTaxonomies []taxonomy.Taxonomy,
	outDir string,
	addSitemapEntry func(string, string, string),
) error {
	allDir := filepath.Join(outDir, "all")
	if err := os.MkdirAll(allDir, 0755); err != nil {
		return fmt.Errorf("creating all-entities dir: %w", err)
	}

	perPage := b.cfg.Pagination.EntitiesPerPage
	totalPages := (len(entities) + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}

	// Compute type distribution + share image (once for all pages)
	allTypeDist := countFieldDistribution(entities, "node_type", 8)
	type allEntitiesChart struct {
		TotalEntities    int                `json:"totalEntities"`
		TypeDistribution []render.NameCount `json:"typeDistribution"`
	}
	allChartObj := allEntitiesChart{TotalEntities: len(entities), TypeDistribution: allTypeDist}
	allChartJSON, _ := json.Marshal(allChartObj)

	allSVG := render.GenerateAllEntitiesShareSVG(b.cfg.Site.Name, len(entities), allTypeDist)
	allImageURL := writeShareSVG(outDir, b.cfg.Site.BaseURL, "all-entities.svg", allSVG)

	for page := 1; page <= totalPages; page++ {
		pagination := taxonomy.ComputeAllEntitiesPagination(len(entities), page, perPage)

		pageEntities := entities
		if pagination.StartIndex < len(entities) {
			end := pagination.EndIndex
			if end > len(entities) {
				end = len(entities)
			}
			pageEntities = entities[pagination.StartIndex:end]
		}

		// JSON-LD
		pageURL := fmt.Sprintf("%s%s", b.cfg.Site.BaseURL, taxonomy.AllEntitiesPageURL(page))
		var items []schema.ItemListEntry
		for _, e := range pageEntities {
			items = append(items, schema.ItemListEntry{
				Name: e.GetString("title"),
				URL:  fmt.Sprintf("%s/%s.html", b.cfg.Site.BaseURL, e.Slug),
			})
		}
		collectionSchema := schemaGen.GenerateCollectionPageSchema(
			"All Entities", "Browse all entities in the architecture documentation",
			pageURL, items, allImageURL,
		)
		breadcrumbs := []render.Breadcrumb{
			{Name: "Home", URL: b.cfg.Site.BaseURL + "/"},
			{Name: "All Entities", URL: ""},
		}
		breadcrumbSchema := schemaGen.GenerateBreadcrumbSchema(toBreadcrumbItems(breadcrumbs))
		jsonLD := schema.MarshalSchemas(collectionSchema, breadcrumbSchema)

		allOG := makeOG(b.cfg.Site.Name,
			fmt.Sprintf("All Entities | %s", b.cfg.Site.Name),
			fmt.Sprintf("Browse all %d entities in the architecture documentation", len(entities)),
			pageURL, allImageURL, "article")

		ctx := render.AllEntitiesPageContext{
			Site:          b.cfg.Site,
			Entities:      pageEntities,
			Pagination:    pagination,
			JsonLD:        toTemplateHTML(jsonLD),
			OG:            allOG,
			ChartData:     template.JS(allChartJSON),
			EntityCount:   len(entities),
			Breadcrumbs:   breadcrumbs,
			AllTaxonomies: allTaxonomies,
			TotalEntities: len(entities),
			CTA:           b.cfg.Extra.CTA,
		}

		html, err := engine.RenderAllEntities(ctx)
		if err != nil {
			return fmt.Errorf("rendering all-entities page %d: %w", page, err)
		}

		var filename string
		if page == 1 {
			filename = "index.html"
		} else {
			filename = fmt.Sprintf("page-%d.html", page)
		}

		if err := os.WriteFile(filepath.Join(allDir, filename), []byte(html), 0644); err != nil {
			return fmt.Errorf("writing all-entities page: %w", err)
		}

		priority := b.cfg.Sitemap.Priorities["hub_page_1"]
		if page > 1 {
			priority = b.cfg.Sitemap.Priorities["hub_page_n"]
		}
		addSitemapEntry(fmt.Sprintf("/all/%s", filename), priority, b.cfg.Sitemap.ChangeFreqs["hub"])
	}

	log.Printf("  Generated %d all-entities page(s)", totalPages)
	return nil
}

func (b *Builder) loadFavorites(slugMap map[string]*entity.Entity) []*entity.Entity {
	if b.cfg.Extra.Favorites == "" {
		return nil
	}

	data, err := os.ReadFile(b.cfg.Extra.Favorites)
	if err != nil {
		log.Printf("Warning: failed to load favorites: %v", err)
		return nil
	}

	var slugs []string
	if err := json.Unmarshal(data, &slugs); err != nil {
		log.Printf("Warning: failed to parse favorites: %v", err)
		return nil
	}

	var result []*entity.Entity
	for _, slug := range slugs {
		if e, ok := slugMap[slug]; ok {
			result = append(result, e)
		}
	}
	return result
}

func (b *Builder) loadContributors() map[string]interface{} {
	if b.cfg.Extra.Contributors == "" {
		return nil
	}

	data, err := os.ReadFile(b.cfg.Extra.Contributors)
	if err != nil {
		log.Printf("Warning: failed to load contributors: %v", err)
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Printf("Warning: failed to parse contributors: %v", err)
		return nil
	}
	return result
}

func toBreadcrumbItems(breadcrumbs []render.Breadcrumb) []schema.BreadcrumbItem {
	items := make([]schema.BreadcrumbItem, len(breadcrumbs))
	for i, bc := range breadcrumbs {
		items[i] = schema.BreadcrumbItem{Name: bc.Name, URL: bc.URL}
	}
	return items
}

// toTemplateHTML converts a string to template.HTML (trusted HTML).
func toTemplateHTML(s string) template.HTML {
	return template.HTML(s)
}

func countTaxEntries(taxonomies []taxonomy.Taxonomy) int {
	total := 0
	for _, tax := range taxonomies {
		total += len(tax.Entries)
	}
	return total
}

type searchEntry struct {
	T string `json:"t"`           // title
	D string `json:"d,omitempty"` // description (truncated)
	S string `json:"s"`           // slug
	N string `json:"n,omitempty"` // node_type
	L string `json:"l,omitempty"` // language
	M string `json:"m,omitempty"` // domain
}

func (b *Builder) generateSearchIndex(entities []*entity.Entity, outDir string) error {
	if !b.cfg.Search.Enabled {
		return nil
	}

	entries := make([]searchEntry, 0, len(entities))
	for _, e := range entities {
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

	outPath := filepath.Join(outDir, "search-index.json")
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return err
	}
	log.Printf("  Generated search index (%d entries, %dKB)", len(entries), len(data)/1024)
	return nil
}

// copyDir copies files from src to dst directory.
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

// countFieldDistribution counts occurrences of a string field across entities, returns sorted desc, capped at limit.
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

// makeOG creates an OGMeta for a page.
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

// writeShareSVG writes an SVG file to the share images directory and returns its public URL.
func writeShareSVG(outDir, baseURL, filename, svg string) string {
	dir := filepath.Join(outDir, "images", "share")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, filename), []byte(svg), 0644)
	return fmt.Sprintf("%s/images/share/%s", baseURL, filename)
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

// generateArchMapSVG renders a static SVG for the architecture map breadcrumb.
func generateArchMapSVG(archMapJSON string) string {
	var data archMapData
	if err := json.Unmarshal([]byte(archMapJSON), &data); err != nil {
		return ""
	}

	type item struct {
		name string
	}
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
		// Escape XML special chars in label
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
