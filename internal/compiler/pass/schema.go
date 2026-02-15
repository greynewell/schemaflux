package pass

import (
	"encoding/json"
	"fmt"

	"github.com/greynewell/schemaflux/internal/affiliate"
	"github.com/greynewell/schemaflux/internal/compiler/ir"
	"github.com/greynewell/schemaflux/internal/entity"
	"github.com/greynewell/schemaflux/internal/render"
	"github.com/greynewell/schemaflux/internal/schema"
)

// SchemaPass generates JSON-LD, OG metadata, breadcrumbs, chart data, and share SVGs.
type SchemaPass struct{}

func (s *SchemaPass) Name() string        { return "Schema" }
func (s *SchemaPass) Requires() []string { return []string{"URLResolution", "Enrichment", "GraphEnrichment"} }

func (s *SchemaPass) Run(p *ir.Program) error {
	schemaGen := schema.NewGenerator(p.Site, p.Config.Schema)

	for _, re := range p.Entities {
		s.resolveEntity(re, p, schemaGen)
	}

	return nil
}

func (s *SchemaPass) resolveEntity(re *ir.ResolvedEntity, p *ir.Program, schemaGen *schema.Generator) {
	e := re.Raw
	entityURL := re.URL

	// Cook mode prompt
	re.CookModePrompt = render.GenerateCookModePrompt(e, re.Enrichment, re.AffiliateLinks)

	// Share image — compute URL deterministically so the backend only writes the file.
	re.ShareImageSVG = render.GenerateEntityShareSVG(
		p.Site.Name,
		e.GetString("title"),
		e.GetString("node_type"),
		e.GetString("language"),
		e.GetString("domain"),
	)
	if re.ShareImageSVG != "" {
		re.ShareImageURL = fmt.Sprintf("%s/images/share/%s.svg", p.Site.BaseURL, re.Slug)
	}

	// JSON-LD
	var entitySchema map[string]interface{}
	if p.Config.Schema.EntityType == "Recipe" {
		entitySchema = schemaGen.GenerateRecipeSchema(e, entityURL)
		if _, hasImage := entitySchema["image"]; !hasImage && re.ShareImageURL != "" {
			entitySchema["image"] = []string{re.ShareImageURL}
		}
		if steps, ok := entitySchema["recipeInstructions"].([]map[string]interface{}); ok {
			for i := range steps {
				steps[i]["url"] = fmt.Sprintf("%s#step-%d", entityURL, i+1)
			}
		}
	} else {
		entitySchema = schemaGen.GenerateEntitySchema(e, entityURL)
	}

	// Breadcrumbs
	re.Breadcrumbs = []render.Breadcrumb{
		{Name: "Home", URL: p.Site.BaseURL + "/"},
	}
	if cat := e.GetString("recipe_category"); cat != "" {
		catSlug := entity.ToSlug(cat)
		re.Breadcrumbs = append(re.Breadcrumbs, render.Breadcrumb{
			Name: cat,
			URL:  fmt.Sprintf("%s/category/%s.html", p.Site.BaseURL, catSlug),
		})
	}
	re.Breadcrumbs = append(re.Breadcrumbs, render.Breadcrumb{Name: e.GetString("title"), URL: ""})

	breadcrumbItems := make([]schema.BreadcrumbItem, len(re.Breadcrumbs))
	for i, bc := range re.Breadcrumbs {
		breadcrumbItems[i] = schema.BreadcrumbItem{Name: bc.Name, URL: bc.URL}
	}
	breadcrumbSchema := schemaGen.GenerateBreadcrumbSchema(breadcrumbItems)

	var faqSchema map[string]interface{}
	if faqs := e.GetFAQs(); len(faqs) > 0 {
		faqSchema = schemaGen.GenerateFAQSchema(faqs)
	}

	re.JsonLD = schema.MarshalSchemas(entitySchema, breadcrumbSchema, faqSchema)

	// OG
	re.OG = render.OGMeta{
		Title:       e.GetString("title") + " | " + p.Site.Name,
		Description: e.GetString("description"),
		URL:         entityURL,
		ImageURL:    re.ShareImageURL,
		Type:        "article",
		SiteName:    p.Site.Name,
	}

	// Chart data
	s.buildChartData(re)
}

func (s *SchemaPass) buildChartData(re *ir.ResolvedEntity) {
	e := re.Raw
	chartMap := re.ChartData
	if v := e.GetInt("line_count"); v > 0 {
		chartMap["lc"] = v
	}
	if v := e.GetInt("start_line"); v > 0 {
		chartMap["sl"] = v
	}
	if v := e.GetInt("end_line"); v > 0 {
		chartMap["el"] = v
	}
	if v := e.GetInt("call_count"); v > 0 {
		chartMap["co"] = v
	}
	if v := e.GetInt("called_by_count"); v > 0 {
		chartMap["cb"] = v
	}
	if v := e.GetInt("import_count"); v > 0 {
		chartMap["ic"] = v
	}
	if v := e.GetInt("imported_by_count"); v > 0 {
		chartMap["ib"] = v
	}
	if v := e.GetInt("function_count"); v > 0 {
		chartMap["fn"] = v
	}
	if v := e.GetInt("class_count"); v > 0 {
		chartMap["cl"] = v
	}
	if v := e.GetInt("type_count"); v > 0 {
		chartMap["tc"] = v
	}
	if v := e.GetInt("file_count"); v > 0 {
		chartMap["fc"] = v
	}

	// Enrich graph_data edge type counts for chart.
	// Prefer enriched graph data from GraphEnrichmentPass; fall back to raw field.
	graphJSON := re.EnrichedGraphData
	if graphJSON == "" {
		graphJSON = e.GetString("graph_data")
	}
	if graphJSON != "" {
		var graphObj struct {
			Nodes []map[string]interface{} `json:"nodes"`
			Edges []struct {
				Source string `json:"source"`
				Target string `json:"target"`
				Type   string `json:"type"`
			} `json:"edges"`
		}
		if json.Unmarshal([]byte(graphJSON), &graphObj) == nil {
			etCounts := make(map[string]int)
			for _, edge := range graphObj.Edges {
				etCounts[edge.Type]++
			}
			chartMap["et"] = etCounts
			chartMap["nc"] = len(graphObj.Nodes)
			chartMap["ec"] = len(graphObj.Edges)
		}
	}
}

// CookModePromptForEntity generates cook mode prompt — exported for backend use.
func CookModePromptForEntity(e *entity.Entity, enrichmentData map[string]interface{}, affLinks []affiliate.Link) string {
	return render.GenerateCookModePrompt(e, enrichmentData, affLinks)
}
