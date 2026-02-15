package ir

import (
	"testing"

	"github.com/greynewell/schemaflux/internal/entity"
)

func TestValidateDuplicateSlugs(t *testing.T) {
	p := NewProgram(testConfig())

	e1 := NewResolvedEntity(testEntity("dup", map[string]interface{}{"title": "First"}))
	e1.URL = "https://example.com/dup.html"
	e2 := NewResolvedEntity(testEntity("dup", map[string]interface{}{"title": "Second"}))
	e2.URL = "https://example.com/dup.html"

	p.Entities = []*ResolvedEntity{e1, e2}
	p.EntityBySlug["dup"] = e1

	Validate(p)

	if !p.HasErrors() {
		t.Error("Expected error for duplicate slugs")
	}

	found := false
	for _, d := range p.Diagnostics {
		if d.Level == DiagError && d.Entity == "dup" {
			found = true
		}
	}
	if !found {
		t.Error("Expected duplicate slug error diagnostic")
	}
}

func TestValidateMissingTitle(t *testing.T) {
	p := NewProgram(testConfig())

	e := NewResolvedEntity(testEntity("no-title", map[string]interface{}{}))
	e.URL = "https://example.com/no-title.html"
	p.Entities = []*ResolvedEntity{e}
	p.EntityBySlug["no-title"] = e

	Validate(p)

	found := false
	for _, d := range p.Diagnostics {
		if d.Level == DiagWarning && d.Entity == "no-title" {
			found = true
		}
	}
	if !found {
		t.Error("Expected warning for missing title")
	}
}

func TestValidateDanglingPairing(t *testing.T) {
	p := NewProgram(testConfig())

	e := NewResolvedEntity(&entity.Entity{
		Slug:   "recipe-a",
		Fields: map[string]interface{}{"title": "Recipe A", "pairings": []interface{}{"nonexistent"}},
	})
	e.URL = "https://example.com/recipe-a.html"
	p.Entities = []*ResolvedEntity{e}
	p.EntityBySlug["recipe-a"] = e

	Validate(p)

	found := false
	for _, d := range p.Diagnostics {
		if d.Entity == "recipe-a" && d.Level == DiagWarning {
			found = true
		}
	}
	if !found {
		t.Error("Expected warning for dangling pairing reference")
	}
}

func TestValidateMissingURL(t *testing.T) {
	p := NewProgram(testConfig())

	e := NewResolvedEntity(testEntity("no-url", map[string]interface{}{"title": "No URL"}))
	// URL intentionally left empty
	p.Entities = []*ResolvedEntity{e}
	p.EntityBySlug["no-url"] = e

	Validate(p)

	found := false
	for _, d := range p.Diagnostics {
		if d.Entity == "no-url" && d.Level == DiagWarning {
			found = true
		}
	}
	if !found {
		t.Error("Expected warning for missing URL")
	}
}

func TestValidateCleanProgram(t *testing.T) {
	p := NewProgram(testConfig())

	e := NewResolvedEntity(testEntity("good", map[string]interface{}{"title": "Good Entity"}))
	e.URL = "https://example.com/good.html"
	p.Entities = []*ResolvedEntity{e}
	p.EntityBySlug["good"] = e

	Validate(p)

	if p.HasErrors() {
		t.Error("Clean program should have no errors")
	}
	// Should only have no diagnostics for a fully resolved entity
	for _, d := range p.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("Unexpected error: %s", d.Message)
		}
	}
}
