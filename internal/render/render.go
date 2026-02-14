package render

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/greynewell/pssg/internal/affiliate"
	"github.com/greynewell/pssg/internal/config"
	"github.com/greynewell/pssg/internal/entity"
	"github.com/greynewell/pssg/internal/taxonomy"
)

// Engine is the template rendering engine.
type Engine struct {
	tmpl    *template.Template
	cfg     *config.Config
}

// TOCEntry represents a heading in the table of contents.
type TOCEntry struct {
	Level int    // 2-6 (h2-h6)
	Text  string // heading text
	ID    string // slug of heading text, for anchor links
}

// EntityPageContext is the template context for entity (recipe) pages.
type EntityPageContext struct {
	Site            config.SiteConfig
	Entity          *entity.Entity
	Slug            string
	URL             string
	CanonicalURL    string
	Breadcrumbs     []Breadcrumb
	Pairings        []*entity.Entity
	Related         []*entity.Entity
	Enrichment      map[string]interface{}
	AffiliateLinks  []affiliate.Link
	CookModePrompt  string
	JsonLD          template.HTML
	OG              OGMeta
	ChartData       template.JS
	SourceCode      string
	SourceLang      string
	Taxonomies      []taxonomy.Taxonomy
	AllTaxonomies   []taxonomy.Taxonomy
	ValidSlugs      map[string]map[string]bool
	Contributors    map[string]interface{}
	CTA             config.CTAConfig
	TOC             []TOCEntry
	ReadingTime     int
	WordCount       int
}

// HomepageContext is the template context for the homepage.
type HomepageContext struct {
	Site          config.SiteConfig
	Entities      []*entity.Entity
	Taxonomies    []taxonomy.Taxonomy
	Favorites     []*entity.Entity
	JsonLD        template.HTML
	OG            OGMeta
	ChartData     template.JS
	ArchData      template.JS
	EntityCount   int
	Contributors  map[string]interface{}
	CTA           config.CTAConfig
}

// HubPageContext is the template context for taxonomy hub (category) pages.
type HubPageContext struct {
	Site           config.SiteConfig
	Taxonomy       taxonomy.Taxonomy
	Entry          taxonomy.Entry
	Entities       []*entity.Entity
	Pagination     taxonomy.PaginationInfo
	JsonLD         template.HTML
	OG             OGMeta
	ChartData      template.JS
	Breadcrumbs    []Breadcrumb
	AllTaxonomies  []taxonomy.Taxonomy
	Contributors   map[string]interface{}
	ContributorProfile map[string]interface{}
	CTA            config.CTAConfig
}

// TaxonomyIndexContext is the template context for taxonomy index pages.
type TaxonomyIndexContext struct {
	Site          config.SiteConfig
	Taxonomy      taxonomy.Taxonomy
	Entries       []taxonomy.Entry
	TopEntries    []taxonomy.Entry
	LetterGroups  []taxonomy.LetterGroup
	HasLetters    bool
	Letters       []string
	JsonLD        template.HTML
	OG            OGMeta
	ChartData     template.JS
	Breadcrumbs   []Breadcrumb
	AllTaxonomies []taxonomy.Taxonomy
	CTA           config.CTAConfig
}

// LetterPageContext is the template context for A-Z letter pages.
type LetterPageContext struct {
	Site          config.SiteConfig
	Taxonomy      taxonomy.Taxonomy
	Letter        string
	Entries       []taxonomy.Entry
	Letters       []string
	JsonLD        template.HTML
	OG            OGMeta
	ChartData     template.JS
	Breadcrumbs   []Breadcrumb
	AllTaxonomies []taxonomy.Taxonomy
	CTA           config.CTAConfig
}

// AllEntitiesPageContext is the template context for the "all entities" paginated listing.
type AllEntitiesPageContext struct {
	Site          config.SiteConfig
	Entities      []*entity.Entity
	Pagination    taxonomy.PaginationInfo
	JsonLD        template.HTML
	OG            OGMeta
	ChartData     template.JS
	EntityCount   int
	Breadcrumbs   []Breadcrumb
	AllTaxonomies []taxonomy.Taxonomy
	TotalEntities int
	CTA           config.CTAConfig
}

// StaticPageContext is the template context for static pages.
type StaticPageContext struct {
	Site          config.SiteConfig
	Title         string
	Content       template.HTML
	JsonLD        template.HTML
	OG            OGMeta
	Breadcrumbs   []Breadcrumb
	AllTaxonomies []taxonomy.Taxonomy
	Taxonomies    []taxonomy.Taxonomy
	Favorites     []*entity.Entity
	CTA           config.CTAConfig
}

// OGMeta holds Open Graph metadata for social sharing.
type OGMeta struct {
	Title       string
	Description string
	URL         string
	ImageURL    string
	Type        string // "website" for homepage, "article" for all others
	SiteName    string
}

// NameCount is a label + count pair used for chart data.
type NameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Breadcrumb is a single breadcrumb entry.
type Breadcrumb struct {
	Name string
	URL  string
}

// NewEngine creates a render engine loading templates from the given directory.
func NewEngine(cfg *config.Config) (*Engine, error) {
	funcMap := BuildFuncMap()

	tmplDir := cfg.Paths.Templates
	entries, err := os.ReadDir(tmplDir)
	if err != nil {
		return nil, fmt.Errorf("reading template dir %s: %w", tmplDir, err)
	}

	tmpl := template.New("").Funcs(funcMap)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".html" && ext != ".css" && ext != ".js" {
			continue
		}

		path := filepath.Join(tmplDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading template %s: %w", name, err)
		}

		_, err = tmpl.New(name).Parse(string(data))
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", name, err)
		}
	}

	return &Engine{tmpl: tmpl, cfg: cfg}, nil
}

// RenderEntity renders an entity page. If the entity has a "template" field,
// it overrides the default entity template from config.
func (e *Engine) RenderEntity(ctx EntityPageContext) (string, error) {
	templateName := e.cfg.Templates.Entity
	if override := ctx.Entity.GetString("template"); override != "" {
		templateName = override
	}
	return e.render(templateName, ctx)
}

// RenderHomepage renders the homepage.
func (e *Engine) RenderHomepage(ctx HomepageContext) (string, error) {
	return e.render(e.cfg.Templates.Homepage, ctx)
}

// RenderHub renders a taxonomy hub page.
func (e *Engine) RenderHub(ctx HubPageContext) (string, error) {
	templateName := ctx.Taxonomy.Config.Template
	return e.render(templateName, ctx)
}

// RenderTaxonomyIndex renders a taxonomy index page.
func (e *Engine) RenderTaxonomyIndex(ctx TaxonomyIndexContext) (string, error) {
	templateName := ctx.Taxonomy.Config.IndexTemplate
	return e.render(templateName, ctx)
}

// RenderLetter renders a letter page.
func (e *Engine) RenderLetter(ctx LetterPageContext) (string, error) {
	templateName := ctx.Taxonomy.Config.LetterTemplate
	return e.render(templateName, ctx)
}

// RenderAllEntities renders an all-entities listing page.
func (e *Engine) RenderAllEntities(ctx AllEntitiesPageContext) (string, error) {
	return e.render(e.cfg.Templates.AllEntities, ctx)
}

// RenderStatic renders a static page.
func (e *Engine) RenderStatic(templateName string, ctx StaticPageContext) (string, error) {
	return e.render(templateName, ctx)
}

// pssgFooter is injected into every rendered HTML page and cannot be disabled.
const pssgFooter = `<div style="text-align:center;padding:12px 8px;font-size:11px;opacity:0.6;"><a href="https://github.com/greynewell/pssg" style="color:inherit;">Generated with pssg</a></div>`

func (e *Engine) render(name string, data interface{}) (string, error) {
	t := e.tmpl.Lookup(name)
	if t == nil {
		return "", fmt.Errorf("template %q not found", name)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template %q: %w", name, err)
	}

	html := buf.String()

	// Inject mandatory pssg attribution before </body>
	if idx := strings.LastIndex(html, "</body>"); idx >= 0 {
		html = html[:idx] + pssgFooter + html[idx:]
	}

	return html, nil
}

// RenderCSS reads and returns the CSS template content.
func (e *Engine) RenderCSS() (string, error) {
	t := e.tmpl.Lookup("_styles.css")
	if t == nil {
		return "", nil
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, nil); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderJS reads and returns the JS file content.
// Reads raw file to avoid html/template escaping JS operators like '<'.
func (e *Engine) RenderJS() (string, error) {
	jsPath := filepath.Join(e.cfg.Paths.Templates, "_main.js")
	data, err := os.ReadFile(jsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// GenerateCookModePrompt builds a cook-with-AI prompt for a recipe.
func GenerateCookModePrompt(e *entity.Entity, enrichment map[string]interface{}, affiliateLinks []affiliate.Link) string {
	if enrichment == nil {
		return ""
	}

	var parts []string

	title := e.GetString("title")
	parts = append(parts, fmt.Sprintf("I want to cook: %s", title))

	// Coaching prompt
	if cp, ok := enrichment["coachingPrompt"].(string); ok && cp != "" {
		parts = append(parts, cp)
	}

	// Ingredients
	if ingredients := e.GetIngredients(); len(ingredients) > 0 {
		items := make([]string, len(ingredients))
		for i, ing := range ingredients {
			items[i] = "- " + ing
		}
		parts = append(parts, "Ingredients:\n"+strings.Join(items, "\n"))
	}

	// Instructions
	if instructions := e.GetInstructions(); len(instructions) > 0 {
		items := make([]string, len(instructions))
		for i, inst := range instructions {
			items[i] = fmt.Sprintf("%d. %s", i+1, inst)
		}
		parts = append(parts, "Instructions:\n"+strings.Join(items, "\n"))
	}

	// Cooking tips
	if tips, ok := enrichment["cookingTips"].([]interface{}); ok && len(tips) > 0 {
		items := make([]string, 0, len(tips))
		for _, tip := range tips {
			if s, ok := tip.(string); ok {
				items = append(items, "- "+s)
			}
		}
		if len(items) > 0 {
			parts = append(parts, "Key Tips:\n"+strings.Join(items, "\n"))
		}
	}

	// Shopping links
	if len(affiliateLinks) > 0 {
		items := make([]string, 0, len(affiliateLinks))
		for _, link := range affiliateLinks {
			items = append(items, fmt.Sprintf("- %s: %s (%s)", link.Term, link.URL, link.Provider))
		}
		parts = append(parts, "Shopping Links:\n"+strings.Join(items, "\n"))
	}

	parts = append(parts, "Please guide me through this recipe step by step, including timing, technique details, and what to watch for at each stage.")

	return strings.Join(parts, "\n\n")
}
