package pass

import (
	"os"
	"testing"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
	"github.com/greynewell/schemaflux/internal/config"
	"github.com/greynewell/schemaflux/internal/entity"
	"github.com/greynewell/schemaflux/internal/taxonomy"
)

func makeTestConfig() *config.Config {
	return &config.Config{
		Site: config.SiteConfig{
			Name:    "Test Site",
			BaseURL: "https://example.com",
		},
	}
}

func makeTestEntity(slug string, fields map[string]interface{}) *entity.Entity {
	return &entity.Entity{Slug: slug, Fields: fields}
}

func makeProgram(entities ...*entity.Entity) *ir.Program {
	p := ir.NewProgram(makeTestConfig())
	for _, e := range entities {
		p.Entities = append(p.Entities, ir.NewResolvedEntity(e))
	}
	return p
}

// --- SlugResolutionPass ---

func TestSlugResolutionBuildMap(t *testing.T) {
	p := makeProgram(
		makeTestEntity("a", map[string]interface{}{"title": "A"}),
		makeTestEntity("b", map[string]interface{}{"title": "B"}),
	)

	pass := &SlugResolutionPass{}
	if err := pass.Run(p); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if len(p.EntityBySlug) != 2 {
		t.Errorf("EntityBySlug len = %d, want 2", len(p.EntityBySlug))
	}
	if p.EntityBySlug["a"] == nil || p.EntityBySlug["b"] == nil {
		t.Error("Expected entities in slug map")
	}
	if p.TotalEntityCount != 2 {
		t.Errorf("TotalEntityCount = %d, want 2", p.TotalEntityCount)
	}
}

func TestSlugResolutionPairings(t *testing.T) {
	p := makeProgram(
		makeTestEntity("a", map[string]interface{}{"title": "A", "pairings": []interface{}{"b"}}),
		makeTestEntity("b", map[string]interface{}{"title": "B"}),
		makeTestEntity("c", map[string]interface{}{"title": "C", "pairings": []interface{}{"nonexistent"}}),
	)

	pass := &SlugResolutionPass{}
	if err := pass.Run(p); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if len(p.Entities[0].Pairings) != 1 {
		t.Fatalf("Entity 'a' pairings = %d, want 1", len(p.Entities[0].Pairings))
	}
	if p.Entities[0].Pairings[0].Slug != "b" {
		t.Errorf("Pairing slug = %q, want %q", p.Entities[0].Pairings[0].Slug, "b")
	}
	// c has a nonexistent pairing, should be empty
	if len(p.Entities[2].Pairings) != 0 {
		t.Errorf("Entity 'c' pairings = %d, want 0", len(p.Entities[2].Pairings))
	}
}

// --- SortPass ---

func TestSortPassNoConfig(t *testing.T) {
	p := makeProgram(
		makeTestEntity("c", map[string]interface{}{"title": "C"}),
		makeTestEntity("a", map[string]interface{}{"title": "A"}),
		makeTestEntity("b", map[string]interface{}{"title": "B"}),
	)

	pass := &SortPass{}
	if err := pass.Run(p); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	// Order unchanged
	if p.Entities[0].Slug != "c" {
		t.Errorf("Expected 'c' first, got %q", p.Entities[0].Slug)
	}
}

func TestSortPassByTitleAsc(t *testing.T) {
	p := makeProgram(
		makeTestEntity("c", map[string]interface{}{"title": "Cherry"}),
		makeTestEntity("a", map[string]interface{}{"title": "Apple"}),
		makeTestEntity("b", map[string]interface{}{"title": "Banana"}),
	)
	p.Config.Sort = config.SortConfig{Field: "title", Order: "asc"}

	pass := &SortPass{}
	if err := pass.Run(p); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	want := []string{"a", "b", "c"}
	for i, slug := range want {
		if p.Entities[i].Slug != slug {
			t.Errorf("Entities[%d].Slug = %q, want %q", i, p.Entities[i].Slug, slug)
		}
	}
}

func TestSortPassByDateDesc(t *testing.T) {
	p := makeProgram(
		makeTestEntity("old", map[string]interface{}{"date": "2024-01-01"}),
		makeTestEntity("new", map[string]interface{}{"date": "2025-06-15"}),
		makeTestEntity("mid", map[string]interface{}{"date": "2024-06-15"}),
	)
	p.Config.Sort = config.SortConfig{Field: "date", Order: "desc"}

	pass := &SortPass{}
	if err := pass.Run(p); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	want := []string{"new", "mid", "old"}
	for i, slug := range want {
		if p.Entities[i].Slug != slug {
			t.Errorf("Entities[%d].Slug = %q, want %q", i, p.Entities[i].Slug, slug)
		}
	}
}

// --- URLResolutionPass ---

func TestURLResolution(t *testing.T) {
	p := makeProgram(
		makeTestEntity("hello", map[string]interface{}{"title": "Hello"}),
	)
	p.Taxonomies = append(p.Taxonomies, ir.TaxonomyGroup{
		Taxonomy: makeTestTaxonomy("category"),
	})

	pass := &URLResolutionPass{}
	if err := pass.Run(p); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if p.Entities[0].URL != "https://example.com/hello.html" {
		t.Errorf("URL = %q, want %q", p.Entities[0].URL, "https://example.com/hello.html")
	}
	if p.Entities[0].CanonicalURL != p.Entities[0].URL {
		t.Error("CanonicalURL should match URL")
	}
	if p.Taxonomies[0].IndexURL != "https://example.com/category/" {
		t.Errorf("IndexURL = %q", p.Taxonomies[0].IndexURL)
	}
}

// --- ValidationPass ---

func TestValidationPassRuns(t *testing.T) {
	p := makeProgram(
		makeTestEntity("test", map[string]interface{}{"title": "Test"}),
	)
	// Set slug map
	p.EntityBySlug["test"] = p.Entities[0]
	p.Entities[0].URL = "https://example.com/test.html"

	pass := &ValidationPass{}
	if err := pass.Run(p); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if p.HasErrors() {
		t.Error("Clean entity should not produce errors")
	}
}

// --- AffiliatePass ---

func TestAffiliatePassNoEnrichment(t *testing.T) {
	p := makeProgram(
		makeTestEntity("test", map[string]interface{}{"title": "Test"}),
	)

	pass := &AffiliatePass{}
	if err := pass.Run(p); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if len(p.Entities[0].AffiliateLinks) != 0 {
		t.Error("Expected no affiliate links without enrichment")
	}
}

// --- ContentAnalysisPass ---

func TestContentAnalysis(t *testing.T) {
	e := &entity.Entity{
		Slug:   "test",
		Fields: map[string]interface{}{"title": "Test"},
		Body:   "## Introduction\n\nThis is a test body with some words for counting.\n\n## Details\n\nMore content here.",
	}
	p := makeProgram(e)

	pass := &ContentAnalysisPass{}
	if err := pass.Run(p); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	re := p.Entities[0]
	if len(re.TOC) != 2 {
		t.Errorf("TOC entries = %d, want 2", len(re.TOC))
	}
	if re.WordCount == 0 {
		t.Error("WordCount should be > 0")
	}
	if re.ReadingTime == 0 {
		t.Error("ReadingTime should be > 0")
	}
}

// --- SchemaPass ---

func TestSchemaPass(t *testing.T) {
	p := makeProgram(
		makeTestEntity("test", map[string]interface{}{
			"title":       "Test Entity",
			"description": "A test entity",
		}),
	)
	p.Entities[0].URL = "https://example.com/test.html"
	p.Entities[0].CanonicalURL = "https://example.com/test.html"

	pass := &SchemaPass{}
	if err := pass.Run(p); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	re := p.Entities[0]
	if re.JsonLD == "" {
		t.Error("JsonLD should not be empty")
	}
	if re.OG.Title == "" {
		t.Error("OG.Title should not be empty")
	}
	if len(re.Breadcrumbs) < 2 {
		t.Errorf("Breadcrumbs len = %d, want >= 2", len(re.Breadcrumbs))
	}
}

// --- FavoritesPass ---

func TestFavoritesPassNoConfig(t *testing.T) {
	p := makeProgram(
		makeTestEntity("a", map[string]interface{}{"title": "A"}),
	)
	p.EntityBySlug["a"] = p.Entities[0]

	pass := &FavoritesPass{}
	if err := pass.Run(p); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if len(p.Favorites) != 0 {
		t.Errorf("Favorites = %d, want 0", len(p.Favorites))
	}
}

func TestFavoritesPassResolvesFromFile(t *testing.T) {
	// Create a temp favorites file
	dir := t.TempDir()
	favPath := dir + "/favorites.json"
	os.WriteFile(favPath, []byte(`["b","a","nonexistent"]`), 0644)

	cfg := makeTestConfig()
	cfg.Extra.Favorites = favPath
	p := ir.NewProgram(cfg)

	eA := makeTestEntity("a", map[string]interface{}{"title": "A"})
	eB := makeTestEntity("b", map[string]interface{}{"title": "B"})
	p.Entities = append(p.Entities, ir.NewResolvedEntity(eA), ir.NewResolvedEntity(eB))
	p.EntityBySlug["a"] = p.Entities[0]
	p.EntityBySlug["b"] = p.Entities[1]

	pass := &FavoritesPass{}
	if err := pass.Run(p); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if len(p.Favorites) != 2 {
		t.Fatalf("Favorites = %d, want 2", len(p.Favorites))
	}
	if p.Favorites[0].Slug != "b" {
		t.Errorf("Favorites[0].Slug = %q, want %q", p.Favorites[0].Slug, "b")
	}
	if p.Favorites[1].Slug != "a" {
		t.Errorf("Favorites[1].Slug = %q, want %q", p.Favorites[1].Slug, "a")
	}
}

// helpers

func makeTestTaxonomy(name string) taxonomy.Taxonomy {
	return taxonomy.Taxonomy{
		Name:  name,
		Label: name,
		Config: config.TaxonomyConfig{
			Name: name,
		},
	}
}
