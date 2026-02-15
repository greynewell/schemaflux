package ir

import (
	"testing"

	"github.com/greynewell/schemaflux/internal/config"
	"github.com/greynewell/schemaflux/internal/entity"
)

func testConfig() *config.Config {
	return &config.Config{
		Site: config.SiteConfig{
			Name:    "Test Site",
			BaseURL: "https://example.com",
		},
	}
}

func testEntity(slug string, fields map[string]interface{}) *entity.Entity {
	return &entity.Entity{
		Slug:   slug,
		Fields: fields,
	}
}

func TestNewProgram(t *testing.T) {
	cfg := testConfig()
	p := NewProgram(cfg)

	if p.Config != cfg {
		t.Error("Config not set")
	}
	if p.Site.Name != "Test Site" {
		t.Errorf("Site.Name = %q, want %q", p.Site.Name, "Test Site")
	}
	if p.EntityBySlug == nil {
		t.Error("EntityBySlug should be initialized")
	}
}

func TestNewResolvedEntity(t *testing.T) {
	e := testEntity("test-slug", map[string]interface{}{"title": "Test"})
	re := NewResolvedEntity(e)

	if re.Raw != e {
		t.Error("Raw entity not set")
	}
	if re.Slug != "test-slug" {
		t.Errorf("Slug = %q, want %q", re.Slug, "test-slug")
	}
	if re.TaxonomyMemberships == nil {
		t.Error("TaxonomyMemberships should be initialized")
	}
	if re.ValidSlugs == nil {
		t.Error("ValidSlugs should be initialized")
	}
	if re.ChartData == nil {
		t.Error("ChartData should be initialized")
	}
}

func TestProgramRawEntities(t *testing.T) {
	cfg := testConfig()
	p := NewProgram(cfg)

	e1 := testEntity("a", map[string]interface{}{"title": "A"})
	e2 := testEntity("b", map[string]interface{}{"title": "B"})

	p.Entities = []*ResolvedEntity{
		NewResolvedEntity(e1),
		NewResolvedEntity(e2),
	}

	raw := p.RawEntities()
	if len(raw) != 2 {
		t.Fatalf("len(RawEntities) = %d, want 2", len(raw))
	}
	if raw[0] != e1 {
		t.Error("raw[0] != e1")
	}
	if raw[1] != e2 {
		t.Error("raw[1] != e2")
	}
}

func TestProgramEntityBySlugRaw(t *testing.T) {
	cfg := testConfig()
	p := NewProgram(cfg)

	e := testEntity("test", map[string]interface{}{"title": "Test"})
	re := NewResolvedEntity(e)
	p.EntityBySlug["test"] = re

	got := p.EntityBySlugRaw("test")
	if got != e {
		t.Error("EntityBySlugRaw returned wrong entity")
	}

	if p.EntityBySlugRaw("nonexistent") != nil {
		t.Error("EntityBySlugRaw should return nil for nonexistent slug")
	}
}

func TestDiagnostics(t *testing.T) {
	cfg := testConfig()
	p := NewProgram(cfg)

	if p.HasErrors() {
		t.Error("Empty program should have no errors")
	}

	p.AddDiagnostic(DiagWarning, "test warning", "slug-1")
	if p.HasErrors() {
		t.Error("Warning should not count as error")
	}
	if len(p.Diagnostics) != 1 {
		t.Fatalf("Expected 1 diagnostic, got %d", len(p.Diagnostics))
	}

	p.AddDiagnostic(DiagError, "test error", "slug-2")
	if !p.HasErrors() {
		t.Error("Should have errors after adding error diagnostic")
	}
	if len(p.Diagnostics) != 2 {
		t.Fatalf("Expected 2 diagnostics, got %d", len(p.Diagnostics))
	}
}

func TestDiagLevelString(t *testing.T) {
	if DiagWarning.String() != "warning" {
		t.Errorf("DiagWarning.String() = %q", DiagWarning.String())
	}
	if DiagError.String() != "error" {
		t.Errorf("DiagError.String() = %q", DiagError.String())
	}
}

func TestRawEntitySlice(t *testing.T) {
	e1 := testEntity("a", nil)
	e2 := testEntity("b", nil)
	resolved := []*ResolvedEntity{NewResolvedEntity(e1), NewResolvedEntity(e2)}

	raw := RawEntitySlice(resolved)
	if len(raw) != 2 {
		t.Fatalf("len = %d, want 2", len(raw))
	}
	if raw[0] != e1 || raw[1] != e2 {
		t.Error("RawEntitySlice returned wrong entities")
	}
}
