package pass

import (
	"log"
	"sort"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
)

// RelatedEntitiesPass computes related entities via taxonomy overlap scoring.
type RelatedEntitiesPass struct{}

func (r *RelatedEntitiesPass) Name() string        { return "RelatedEntities" }
func (r *RelatedEntitiesPass) Requires() []string { return []string{"Taxonomy"} }

func (r *RelatedEntitiesPass) Run(p *ir.Program) error {
	cfg := p.Config.RelatedEntities
	if !cfg.Enabled || len(p.Entities) < 2 {
		return nil
	}

	log.Printf("Computing related entities...")

	maxRelated := cfg.Max
	if maxRelated <= 0 {
		maxRelated = 3
	}

	// Build taxonomy value sets per entity using composite keys "taxname:value"
	entityTaxValues := make(map[string]map[string]bool)
	for _, re := range p.Entities {
		vals := make(map[string]bool)
		for _, tg := range p.Taxonomies {
			tc := tg.Taxonomy.Config
			if tc.MultiValue {
				for _, v := range re.Raw.GetStringSlice(tc.Field) {
					vals[tg.Taxonomy.Name+":"+v] = true
				}
			} else {
				if v := re.Raw.GetString(tc.Field); v != "" {
					vals[tg.Taxonomy.Name+":"+v] = true
				}
			}
		}
		entityTaxValues[re.Slug] = vals
	}

	// Score and rank
	for _, re := range p.Entities {
		myVals := entityTaxValues[re.Slug]
		if len(myVals) == 0 {
			continue
		}

		type scored struct {
			entity *ir.ResolvedEntity
			score  int
		}
		var candidates []scored

		for _, other := range p.Entities {
			if other.Slug == re.Slug {
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

		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].score > candidates[j].score
		})

		n := maxRelated
		if n > len(candidates) {
			n = len(candidates)
		}
		re.Related = make([]*ir.ResolvedEntity, n)
		for i := 0; i < n; i++ {
			re.Related[i] = candidates[i].entity
		}
	}

	return nil
}
