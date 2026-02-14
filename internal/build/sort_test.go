package build

import (
	"testing"

	"github.com/greynewell/pssg/internal/config"
	"github.com/greynewell/pssg/internal/entity"
)

func makeEntity(slug string, fields map[string]interface{}) *entity.Entity {
	return &entity.Entity{Slug: slug, Fields: fields}
}

func TestSortEntitiesNoConfig(t *testing.T) {
	entities := []*entity.Entity{
		makeEntity("c", map[string]interface{}{"title": "C"}),
		makeEntity("a", map[string]interface{}{"title": "A"}),
		makeEntity("b", map[string]interface{}{"title": "B"}),
	}

	SortEntities(entities, config.SortConfig{})

	// Order should be unchanged
	if entities[0].Slug != "c" || entities[1].Slug != "a" || entities[2].Slug != "b" {
		t.Errorf("Expected original order, got: %s, %s, %s", entities[0].Slug, entities[1].Slug, entities[2].Slug)
	}
}

func TestSortEntitiesByTitleAsc(t *testing.T) {
	entities := []*entity.Entity{
		makeEntity("c", map[string]interface{}{"title": "Cherry"}),
		makeEntity("a", map[string]interface{}{"title": "Apple"}),
		makeEntity("b", map[string]interface{}{"title": "Banana"}),
	}

	SortEntities(entities, config.SortConfig{Field: "title", Order: "asc"})

	want := []string{"a", "b", "c"}
	for i, slug := range want {
		if entities[i].Slug != slug {
			t.Errorf("entities[%d].Slug = %q, want %q", i, entities[i].Slug, slug)
		}
	}
}

func TestSortEntitiesByTitleDesc(t *testing.T) {
	entities := []*entity.Entity{
		makeEntity("a", map[string]interface{}{"title": "Apple"}),
		makeEntity("b", map[string]interface{}{"title": "Banana"}),
		makeEntity("c", map[string]interface{}{"title": "Cherry"}),
	}

	SortEntities(entities, config.SortConfig{Field: "title", Order: "desc"})

	want := []string{"c", "b", "a"}
	for i, slug := range want {
		if entities[i].Slug != slug {
			t.Errorf("entities[%d].Slug = %q, want %q", i, entities[i].Slug, slug)
		}
	}
}

func TestSortEntitiesByDateDesc(t *testing.T) {
	entities := []*entity.Entity{
		makeEntity("old", map[string]interface{}{"date": "2024-01-01"}),
		makeEntity("new", map[string]interface{}{"date": "2025-06-15"}),
		makeEntity("mid", map[string]interface{}{"date": "2024-06-15"}),
	}

	SortEntities(entities, config.SortConfig{Field: "date", Order: "desc"})

	want := []string{"new", "mid", "old"}
	for i, slug := range want {
		if entities[i].Slug != slug {
			t.Errorf("entities[%d].Slug = %q, want %q", i, entities[i].Slug, slug)
		}
	}
}

func TestSortEntitiesByDateAsc(t *testing.T) {
	entities := []*entity.Entity{
		makeEntity("new", map[string]interface{}{"date": "2025-06-15"}),
		makeEntity("old", map[string]interface{}{"date": "2024-01-01"}),
		makeEntity("mid", map[string]interface{}{"date": "2024-06-15"}),
	}

	SortEntities(entities, config.SortConfig{Field: "date", Order: "asc"})

	want := []string{"old", "mid", "new"}
	for i, slug := range want {
		if entities[i].Slug != slug {
			t.Errorf("entities[%d].Slug = %q, want %q", i, entities[i].Slug, slug)
		}
	}
}

func TestSortEntitiesMissingFieldSortLast(t *testing.T) {
	entities := []*entity.Entity{
		makeEntity("no-date", map[string]interface{}{"title": "No Date"}),
		makeEntity("has-date", map[string]interface{}{"title": "Has Date", "date": "2024-01-01"}),
		makeEntity("also-no-date", map[string]interface{}{"title": "Also No Date"}),
	}

	SortEntities(entities, config.SortConfig{Field: "date", Order: "desc"})

	// has-date should be first, the two without date should be last
	if entities[0].Slug != "has-date" {
		t.Errorf("entities[0].Slug = %q, want %q", entities[0].Slug, "has-date")
	}
}

func TestSortEntitiesDefaultOrderIsAsc(t *testing.T) {
	entities := []*entity.Entity{
		makeEntity("b", map[string]interface{}{"title": "Banana"}),
		makeEntity("a", map[string]interface{}{"title": "Apple"}),
	}

	// No order specified, should default to asc
	SortEntities(entities, config.SortConfig{Field: "title"})

	if entities[0].Slug != "a" {
		t.Errorf("Expected ascending default order, got %q first", entities[0].Slug)
	}
}
