package build

import (
	"sort"

	"github.com/greynewell/pssg/internal/config"
	"github.com/greynewell/pssg/internal/entity"
	"github.com/greynewell/pssg/internal/taxonomy"
)

// ComputeRelated computes related entities for each entity based on shared taxonomy values.
// Returns a map from entity slug to a slice of related entities.
func ComputeRelated(
	entities []*entity.Entity,
	taxonomies []taxonomy.Taxonomy,
	cfg config.RelatedConfig,
) map[string][]*entity.Entity {
	result := make(map[string][]*entity.Entity)
	if !cfg.Enabled || len(entities) < 2 {
		return result
	}

	maxRelated := cfg.Max
	if maxRelated <= 0 {
		maxRelated = 3
	}

	// Build a map: entity slug -> set of taxonomy values
	// We use "taxname:value" as composite keys
	entityTaxValues := make(map[string]map[string]bool)
	for _, e := range entities {
		vals := make(map[string]bool)
		for _, tax := range taxonomies {
			if tax.Config.MultiValue {
				for _, v := range e.GetStringSlice(tax.Config.Field) {
					vals[tax.Name+":"+v] = true
				}
			} else {
				if v := e.GetString(tax.Config.Field); v != "" {
					vals[tax.Name+":"+v] = true
				}
			}
		}
		entityTaxValues[e.Slug] = vals
	}

	// Build slug -> entity lookup
	slugMap := make(map[string]*entity.Entity)
	for _, e := range entities {
		slugMap[e.Slug] = e
	}

	// For each entity, score all others by shared values
	for _, e := range entities {
		myVals := entityTaxValues[e.Slug]
		if len(myVals) == 0 {
			continue
		}

		type scored struct {
			entity *entity.Entity
			score  int
		}
		var candidates []scored

		for _, other := range entities {
			if other.Slug == e.Slug {
				continue
			}
			otherVals := entityTaxValues[other.Slug]
			score := 0
			for val := range myVals {
				if otherVals[val] {
					score++
				}
			}
			if score > 0 {
				candidates = append(candidates, scored{entity: other, score: score})
			}
		}

		// Sort by score descending
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].score > candidates[j].score
		})

		// Take top N
		n := maxRelated
		if n > len(candidates) {
			n = len(candidates)
		}
		related := make([]*entity.Entity, n)
		for i := 0; i < n; i++ {
			related[i] = candidates[i].entity
		}
		result[e.Slug] = related
	}

	return result
}
