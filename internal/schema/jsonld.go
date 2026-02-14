package schema

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/greynewell/schemaflux/internal/config"
	"github.com/greynewell/schemaflux/internal/entity"
)

// Generator creates JSON-LD structured data.
type Generator struct {
	SiteConfig config.SiteConfig
	Schema     config.SchemaConfig
}

// NewGenerator creates a new JSON-LD generator.
func NewGenerator(siteCfg config.SiteConfig, schemaCfg config.SchemaConfig) *Generator {
	return &Generator{
		SiteConfig: siteCfg,
		Schema:     schemaCfg,
	}
}

// GenerateRecipeSchema generates entity JSON-LD using the configured schema type.
func (g *Generator) GenerateRecipeSchema(e *entity.Entity, entityURL string) map[string]interface{} {
	schemaType := g.Schema.EntityType
	if schemaType == "" {
		schemaType = "Recipe"
	}
	schema := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       schemaType,
		"name":        e.GetString("title"),
		"description": e.GetString("description"),
		"url":         entityURL,
	}

	// Author
	authorName := e.GetString("author")
	if authorName != "" {
		authorSlug := entity.ToSlug(authorName)
		schema["author"] = map[string]interface{}{
			"@type": "Person",
			"name":  authorName,
			"url":   fmt.Sprintf("%s/author/%s.html", g.SiteConfig.BaseURL, authorSlug),
		}
	}

	// Date published
	schema["datePublished"] = g.Schema.DatePublished

	// Times
	prepTime := e.GetString("prep_time")
	cookTime := e.GetString("cook_time")
	if prepTime != "" {
		schema["prepTime"] = prepTime
	}
	if cookTime != "" {
		schema["cookTime"] = cookTime
	}
	if prepTime != "" && cookTime != "" {
		schema["totalTime"] = computeTotalTime(prepTime, cookTime)
	}

	// Servings
	if servings := e.GetInt("servings"); servings > 0 {
		schema["recipeYield"] = fmt.Sprintf("%d servings", servings)
	}

	// Category & cuisine
	if cat := e.GetString("recipe_category"); cat != "" {
		schema["recipeCategory"] = cat
	}
	if cuisine := e.GetString("cuisine"); cuisine != "" {
		schema["recipeCuisine"] = cuisine
	}

	// Image
	if img := e.GetString("image"); img != "" {
		schema["image"] = []string{img}
	}

	// Nutrition
	if cal := e.GetInt("calories"); cal > 0 {
		schema["nutrition"] = map[string]interface{}{
			"@type":    "NutritionInformation",
			"calories": fmt.Sprintf("%d calories", cal),
		}
	}

	// Ingredients
	if ingredients := e.GetIngredients(); len(ingredients) > 0 {
		schema["recipeIngredient"] = ingredients
	}

	// Instructions as HowToSteps
	if instructions := e.GetInstructions(); len(instructions) > 0 {
		var steps []map[string]interface{}
		for i, inst := range instructions {
			steps = append(steps, map[string]interface{}{
				"@type":    "HowToStep",
				"text":     inst,
				"name":     stepName(inst),
				"position": i + 1,
			})
		}
		schema["recipeInstructions"] = steps
	}

	// Keywords
	keywords := e.GetStringSlice("keywords")
	extra := g.Schema.ExtraKeywords
	allKeywords := append(keywords, extra...)
	if len(allKeywords) > 0 {
		schema["keywords"] = strings.Join(allKeywords, ", ")
	}

	// Pairings as isRelatedTo
	if pairings := e.GetStringSlice("pairings"); len(pairings) > 0 {
		var related []map[string]interface{}
		for _, slug := range pairings {
			related = append(related, map[string]interface{}{
				"@type": "Recipe",
				"name":  slug, // Will be resolved to title by the builder
				"url":   fmt.Sprintf("%s/%s.html", g.SiteConfig.BaseURL, slug),
			})
		}
		schema["isRelatedTo"] = related
	}

	return schema
}

// GenerateEntitySchema generates entity JSON-LD using SoftwareSourceCode.
func (g *Generator) GenerateEntitySchema(e *entity.Entity, entityURL string) map[string]interface{} {
	schemaType := g.Schema.EntityType
	if schemaType == "" {
		schemaType = "SoftwareSourceCode"
	}
	schema := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       schemaType,
		"name":        e.GetString("title"),
		"description": e.GetString("description"),
		"url":         entityURL,
	}

	// Programming language
	if lang := e.GetString("language"); lang != "" {
		schema["programmingLanguage"] = lang
	}

	// Code repository
	if repoURL := e.GetString("repo_url"); repoURL != "" {
		schema["codeRepository"] = repoURL
	}

	// Date published
	if g.Schema.DatePublished != "" {
		schema["datePublished"] = g.Schema.DatePublished
	}

	// Keywords
	keywords := e.GetStringSlice("tags")
	extra := g.Schema.ExtraKeywords
	allKeywords := append(keywords, extra...)
	if len(allKeywords) > 0 {
		schema["keywords"] = strings.Join(allKeywords, ", ")
	}

	// Architecture map image
	if archMap := e.GetString("arch_map"); archMap != "" {
		imgURL := fmt.Sprintf("%s/images/%s-arch.svg", g.SiteConfig.BaseURL, e.Slug)
		schema["image"] = imgURL
		schema["thumbnailUrl"] = imgURL
	}

	// Relationships from graph_data
	if graphRaw := e.GetString("graph_data"); graphRaw != "" {
		g.addGraphRelationships(schema, e, graphRaw)
	}

	return schema
}

// graphDataJSON is the structure of the graph_data frontmatter field.
type graphDataJSON struct {
	Nodes []graphNodeJSON `json:"nodes"`
	Edges []graphEdgeJSON `json:"edges"`
}
type graphNodeJSON struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
	Slug  string `json:"slug"`
}
type graphEdgeJSON struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

// addGraphRelationships parses graph_data and adds isPartOf, hasPart, and dependency to the schema.
func (g *Generator) addGraphRelationships(s map[string]interface{}, e *entity.Entity, graphRaw string) {
	var gd graphDataJSON
	if err := json.Unmarshal([]byte(graphRaw), &gd); err != nil {
		return
	}

	nodeByID := make(map[string]graphNodeJSON, len(gd.Nodes))
	for _, n := range gd.Nodes {
		nodeByID[n.ID] = n
	}

	var isPartOf []map[string]interface{}
	var hasPart []map[string]interface{}
	var dependencies []map[string]interface{}

	for _, edge := range gd.Edges {
		source := nodeByID[edge.Source]
		target := nodeByID[edge.Target]

		switch edge.Type {
		case "belongsTo", "partOf":
			// current entity belongs to target
			if target.Slug != "" && target.Slug != e.Slug {
				isPartOf = append(isPartOf, g.nodeToSchemaRef(target))
			}
		case "contains", "defines":
			// current entity contains/defines target
			if source.Slug == e.Slug && target.Slug != "" {
				hasPart = append(hasPart, g.nodeToSchemaRef(target))
			}
			// or source defines current entity → current isPartOf source
			if target.Slug == e.Slug && source.Slug != "" {
				isPartOf = append(isPartOf, g.nodeToSchemaRef(source))
			}
		case "imports":
			if source.Slug == e.Slug && target.Slug != "" {
				dependencies = append(dependencies, g.nodeToSchemaRef(target))
			}
		case "calls":
			if source.Slug == e.Slug && target.Slug != "" {
				ref := g.nodeToSchemaRef(target)
				ref["description"] = "Called by " + e.GetString("title")
				hasPart = append(hasPart, ref)
			}
		case "extends":
			if source.Slug == e.Slug && target.Slug != "" {
				isPartOf = append(isPartOf, g.nodeToSchemaRef(target))
			}
		}
	}

	if len(isPartOf) == 1 {
		s["isPartOf"] = isPartOf[0]
	} else if len(isPartOf) > 1 {
		s["isPartOf"] = isPartOf
	}
	if len(hasPart) > 0 {
		s["hasPart"] = hasPart
	}
	if len(dependencies) > 0 {
		s["dependency"] = dependencies
	}
}

func (g *Generator) nodeToSchemaRef(n graphNodeJSON) map[string]interface{} {
	ref := map[string]interface{}{
		"@type": "SoftwareSourceCode",
		"name":  n.Label,
		"url":   fmt.Sprintf("%s/%s.html", g.SiteConfig.BaseURL, n.Slug),
	}
	return ref
}

// GenerateBreadcrumbSchema generates BreadcrumbList JSON-LD.
func (g *Generator) GenerateBreadcrumbSchema(items []BreadcrumbItem) map[string]interface{} {
	var listItems []map[string]interface{}
	for i, item := range items {
		li := map[string]interface{}{
			"@type":    "ListItem",
			"position": i + 1,
			"name":     item.Name,
		}
		if item.URL != "" {
			li["item"] = item.URL
		}
		listItems = append(listItems, li)
	}

	return map[string]interface{}{
		"@context":        "https://schema.org",
		"@type":           "BreadcrumbList",
		"itemListElement": listItems,
	}
}

// BreadcrumbItem is a single breadcrumb entry.
type BreadcrumbItem struct {
	Name string
	URL  string
}

// GenerateFAQSchema generates FAQPage JSON-LD from FAQs.
func (g *Generator) GenerateFAQSchema(faqs []entity.FAQ) map[string]interface{} {
	if len(faqs) == 0 {
		return nil
	}

	var mainEntity []map[string]interface{}
	for _, faq := range faqs {
		mainEntity = append(mainEntity, map[string]interface{}{
			"@type": "Question",
			"name":  faq.Question,
			"acceptedAnswer": map[string]interface{}{
				"@type": "Answer",
				"text":  faq.Answer,
			},
		})
	}

	return map[string]interface{}{
		"@context":   "https://schema.org",
		"@type":      "FAQPage",
		"mainEntity": mainEntity,
	}
}

// GenerateWebSiteSchema generates WebSite JSON-LD.
func (g *Generator) GenerateWebSiteSchema(imageURL string) map[string]interface{} {
	s := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "WebSite",
		"name":        g.SiteConfig.Name,
		"url":         g.SiteConfig.BaseURL,
		"description": g.SiteConfig.Description,
		"publisher": map[string]interface{}{
			"@type": "Organization",
			"name":  g.SiteConfig.Name,
			"url":   g.SiteConfig.BaseURL,
		},
	}
	if imageURL != "" {
		s["image"] = imageURL
	}
	return s
}

// GenerateItemListSchema generates ItemList JSON-LD.
func (g *Generator) GenerateItemListSchema(name, description string, items []ItemListEntry, imageURL string) map[string]interface{} {
	var listItems []map[string]interface{}
	for i, item := range items {
		listItems = append(listItems, map[string]interface{}{
			"@type":    "ListItem",
			"position": i + 1,
			"url":      item.URL,
			"name":     item.Name,
		})
	}

	s := map[string]interface{}{
		"@context":        "https://schema.org",
		"@type":           "ItemList",
		"name":            name,
		"description":     description,
		"numberOfItems":   len(items),
		"itemListElement": listItems,
	}
	if imageURL != "" {
		s["image"] = imageURL
	}
	return s
}

// ItemListEntry is a single item in an ItemList.
type ItemListEntry struct {
	Name string
	URL  string
}

// GenerateCollectionPageSchema generates CollectionPage JSON-LD.
func (g *Generator) GenerateCollectionPageSchema(name, description, pageURL string, items []ItemListEntry, imageURL string) map[string]interface{} {
	var listItems []map[string]interface{}
	for i, item := range items {
		listItems = append(listItems, map[string]interface{}{
			"@type":    "ListItem",
			"position": i + 1,
			"url":      item.URL,
			"name":     item.Name,
		})
	}

	s := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "CollectionPage",
		"name":        name,
		"url":         pageURL,
		"description": description,
		"mainEntity": map[string]interface{}{
			"@type":           "ItemList",
			"numberOfItems":   len(items),
			"itemListElement": listItems,
		},
	}
	if imageURL != "" {
		s["image"] = imageURL
	}
	return s
}

// MarshalSchemas encodes one or more schemas as a JSON-LD script block.
func MarshalSchemas(schemas ...map[string]interface{}) string {
	var parts []string
	for _, s := range schemas {
		if s == nil {
			continue
		}
		data, err := json.Marshal(s)
		if err != nil {
			continue
		}
		parts = append(parts, fmt.Sprintf(`<script type="application/ld+json">%s</script>`, string(data)))
	}
	return strings.Join(parts, "\n")
}

// stepName extracts a short name from an instruction step.
func stepName(step string) string {
	// Take first sentence
	for _, sep := range []string{". ", ".\n"} {
		if idx := strings.Index(step, sep); idx > 0 && idx < 80 {
			return step[:idx+1]
		}
	}
	// Truncate if too long
	if len(step) > 80 {
		return step[:77] + "..."
	}
	return step
}

var durationRegex = regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?`)

// parseDurationMinutes parses an ISO 8601 duration to minutes.
func parseDurationMinutes(d string) int {
	matches := durationRegex.FindStringSubmatch(d)
	if matches == nil {
		return 0
	}
	hours, _ := strconv.Atoi(matches[1])
	minutes, _ := strconv.Atoi(matches[2])
	return hours*60 + minutes
}

// computeTotalTime adds two ISO 8601 durations and returns the result.
func computeTotalTime(d1, d2 string) string {
	total := parseDurationMinutes(d1) + parseDurationMinutes(d2)
	hours := total / 60
	minutes := total % 60
	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("PT%dH%dM", hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("PT%dH", hours)
	}
	return fmt.Sprintf("PT%dM", minutes)
}
