package build

import (
	"testing"

	"github.com/greynewell/pssg/internal/config"
	"github.com/greynewell/pssg/internal/entity"
	"github.com/greynewell/pssg/internal/taxonomy"
)

func TestComputeRelatedDisabled(t *testing.T) {
	entities := []*entity.Entity{
		makeEntity("a", map[string]interface{}{"tags": []interface{}{"go", "web"}}),
		makeEntity("b", map[string]interface{}{"tags": []interface{}{"go"}}),
	}

	result := ComputeRelated(entities, nil, config.RelatedConfig{Enabled: false})
	if len(result) != 0 {
		t.Errorf("Expected empty result when disabled, got %d entries", len(result))
	}
}

func TestComputeRelatedBasic(t *testing.T) {
	entities := []*entity.Entity{
		makeEntity("a", map[string]interface{}{"tags": []interface{}{"go", "web", "api"}}),
		makeEntity("b", map[string]interface{}{"tags": []interface{}{"go", "web"}}),
		makeEntity("c", map[string]interface{}{"tags": []interface{}{"go"}}),
		makeEntity("d", map[string]interface{}{"tags": []interface{}{"python"}}),
	}

	taxCfg := []config.TaxonomyConfig{{Name: "tags", Field: "tags", MultiValue: true}}
	taxonomies := taxonomy.BuildAll(entities, taxCfg, nil)

	result := ComputeRelated(entities, taxonomies, config.RelatedConfig{Enabled: true, Max: 2})

	// Entity "a" should be most related to "b" (2 shared: go, web), then "c" (1 shared: go)
	relA := result["a"]
	if len(relA) != 2 {
		t.Fatalf("Expected 2 related for 'a', got %d", len(relA))
	}
	if relA[0].Slug != "b" {
		t.Errorf("Expected 'b' as most related to 'a', got %q", relA[0].Slug)
	}
	if relA[1].Slug != "c" {
		t.Errorf("Expected 'c' as second related to 'a', got %q", relA[1].Slug)
	}

	// Entity "d" has no shared tags with others
	relD := result["d"]
	if len(relD) != 0 {
		t.Errorf("Expected 0 related for 'd', got %d", len(relD))
	}
}

func TestComputeRelatedExcludesSelf(t *testing.T) {
	entities := []*entity.Entity{
		makeEntity("a", map[string]interface{}{"tags": []interface{}{"go"}}),
		makeEntity("b", map[string]interface{}{"tags": []interface{}{"go"}}),
	}

	taxCfg := []config.TaxonomyConfig{{Name: "tags", Field: "tags", MultiValue: true}}
	taxonomies := taxonomy.BuildAll(entities, taxCfg, nil)

	result := ComputeRelated(entities, taxonomies, config.RelatedConfig{Enabled: true, Max: 3})

	relA := result["a"]
	for _, r := range relA {
		if r.Slug == "a" {
			t.Error("Entity 'a' should not appear in its own related list")
		}
	}
}

func TestComputeRelatedDefaultMax(t *testing.T) {
	entities := []*entity.Entity{
		makeEntity("a", map[string]interface{}{"tags": []interface{}{"shared"}}),
		makeEntity("b", map[string]interface{}{"tags": []interface{}{"shared"}}),
		makeEntity("c", map[string]interface{}{"tags": []interface{}{"shared"}}),
		makeEntity("d", map[string]interface{}{"tags": []interface{}{"shared"}}),
		makeEntity("e", map[string]interface{}{"tags": []interface{}{"shared"}}),
	}

	taxCfg := []config.TaxonomyConfig{{Name: "tags", Field: "tags", MultiValue: true}}
	taxonomies := taxonomy.BuildAll(entities, taxCfg, nil)

	// Max=0 should default to 3
	result := ComputeRelated(entities, taxonomies, config.RelatedConfig{Enabled: true, Max: 0})

	relA := result["a"]
	if len(relA) != 3 {
		t.Errorf("Expected default max 3 related, got %d", len(relA))
	}
}

func TestComputeRelatedSingleValueTaxonomy(t *testing.T) {
	entities := []*entity.Entity{
		makeEntity("a", map[string]interface{}{"category": "guide"}),
		makeEntity("b", map[string]interface{}{"category": "guide"}),
		makeEntity("c", map[string]interface{}{"category": "reference"}),
	}

	taxCfg := []config.TaxonomyConfig{{Name: "category", Field: "category", MultiValue: false}}
	taxonomies := taxonomy.BuildAll(entities, taxCfg, nil)

	result := ComputeRelated(entities, taxonomies, config.RelatedConfig{Enabled: true, Max: 3})

	relA := result["a"]
	if len(relA) != 1 {
		t.Fatalf("Expected 1 related for 'a', got %d", len(relA))
	}
	if relA[0].Slug != "b" {
		t.Errorf("Expected 'b' related to 'a', got %q", relA[0].Slug)
	}
}

func TestComputeRelatedTooFewEntities(t *testing.T) {
	entities := []*entity.Entity{
		makeEntity("a", map[string]interface{}{"tags": []interface{}{"go"}}),
	}

	result := ComputeRelated(entities, nil, config.RelatedConfig{Enabled: true, Max: 3})
	if len(result) != 0 {
		t.Errorf("Expected empty result for single entity, got %d entries", len(result))
	}
}
