package build

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
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
	recipeSchema := schemaGen.GenerateRecipeSchema(e, entityURL)

	// Fix pairing names from slugMap
	if related, ok := recipeSchema["isRelatedTo"].([]map[string]interface{}); ok {
		for i, r := range related {
			if slug, ok := r["name"].(string); ok {
				if paired, ok := slugMap[slug]; ok {
					related[i]["name"] = paired.GetString("title")
				}
			}
		}
	}

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

	jsonLD := schema.MarshalSchemas(recipeSchema, breadcrumbSchema, faqSchema)

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
		Taxonomies:     taxonomies,
		AllTaxonomies:  taxonomies,
		ValidSlugs:     validSlugs,
		Contributors:   contributors,
	}

	html, err := engine.RenderEntity(ctx)
	if err != nil {
		return err
	}

	outPath := filepath.Join(outDir, e.Slug+".html")
	if err := os.WriteFile(outPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
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
				entry.Name, fmt.Sprintf("%s %s recipes", entry.Name, tax.LabelSingular),
				pageURL, items,
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

			ctx := render.HubPageContext{
				Site:               b.cfg.Site,
				Taxonomy:           tax,
				Entry:              entry,
				Entities:           pageEntities,
				Pagination:         pagination,
				JsonLD:             toTemplateHTML(jsonLD),
				Breadcrumbs:        breadcrumbs,
				AllTaxonomies:      allTaxonomies,
				Contributors:       contributors,
				ContributorProfile: contributorProfile,
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

	// Index page JSON-LD
	var indexItems []schema.ItemListEntry
	for _, entry := range tax.Entries {
		indexItems = append(indexItems, schema.ItemListEntry{
			Name: entry.Name,
			URL:  fmt.Sprintf("%s/%s/%s.html", b.cfg.Site.BaseURL, tax.Name, entry.Slug),
		})
	}
	indexSchema := schemaGen.GenerateItemListSchema(tax.Label, fmt.Sprintf("Browse all %s", tax.Label), indexItems)
	breadcrumbs := []render.Breadcrumb{
		{Name: "Home", URL: b.cfg.Site.BaseURL + "/"},
		{Name: tax.Label, URL: ""},
	}
	breadcrumbSchema := schemaGen.GenerateBreadcrumbSchema(toBreadcrumbItems(breadcrumbs))
	jsonLD := schema.MarshalSchemas(indexSchema, breadcrumbSchema)

	ctx := render.TaxonomyIndexContext{
		Site:          b.cfg.Site,
		Taxonomy:      tax,
		Entries:       tax.Entries,
		TopEntries:    topEntries,
		LetterGroups:  letterGroups,
		HasLetters:    hasLetters,
		Letters:       letters,
		JsonLD:        toTemplateHTML(jsonLD),
		Breadcrumbs:   breadcrumbs,
		AllTaxonomies: allTaxonomies,
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

			letterCtx := render.LetterPageContext{
				Site:          b.cfg.Site,
				Taxonomy:      tax,
				Letter:        lg.Letter,
				Entries:       lg.Entries,
				Letters:       letters,
				Breadcrumbs:   letterBreadcrumbs,
				AllTaxonomies: allTaxonomies,
			}

			letterHTML, err := engine.RenderLetter(letterCtx)
			if err != nil {
				return fmt.Errorf("rendering letter page %s/%s: %w", tax.Name, lg.Letter, err)
			}

			letterFile := fmt.Sprintf("letter-%s.html", strings.ToLower(lg.Letter))
			if lg.Letter == "#" {
				letterFile = "letter-num.html"
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
	// JSON-LD
	websiteSchema := schemaGen.GenerateWebSiteSchema()

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
	)

	jsonLD := schema.MarshalSchemas(websiteSchema, itemListSchema)

	ctx := render.HomepageContext{
		Site:         b.cfg.Site,
		Entities:     entities,
		Taxonomies:   taxonomies,
		Favorites:    favorites,
		JsonLD:       toTemplateHTML(jsonLD),
		EntityCount:  len(entities),
		Contributors: contributors,
	}

	html, err := engine.RenderHomepage(ctx)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(outDir, "index.html"), []byte(html), 0644)
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


