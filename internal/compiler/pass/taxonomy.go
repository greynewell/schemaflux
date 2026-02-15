package pass

import (
	"log"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
	"github.com/greynewell/schemaflux/internal/taxonomy"
)

// TaxonomyPass groups entities into TaxonomyGroups.
type TaxonomyPass struct{}

func (t *TaxonomyPass) Name() string        { return "Taxonomy" }
func (t *TaxonomyPass) Requires() []string { return []string{"Sort", "Enrichment"} }

func (t *TaxonomyPass) Run(p *ir.Program) error {
	// Build enrichment data map from resolved entities for taxonomy overrides
	enrichmentData := make(map[string]map[string]interface{})
	for _, re := range p.Entities {
		if re.Enrichment != nil {
			enrichmentData[re.Slug] = re.Enrichment
		}
	}

	rawEntities := p.RawEntities()
	taxonomies := taxonomy.BuildAll(rawEntities, p.Config.Taxonomies, enrichmentData)

	for _, tax := range taxonomies {
		log.Printf("  %s: %d entries", tax.Label, len(tax.Entries))

		slugSet := make(map[string]bool)
		for _, entry := range tax.Entries {
			slugSet[entry.Slug] = true
		}

		p.Taxonomies = append(p.Taxonomies, ir.TaxonomyGroup{
			Taxonomy:   tax,
			ValidSlugs: slugSet,
		})
	}

	// Build inverse index: for each taxonomy, map entity slug -> list of entry slugs.
	// This replaces the previous O(n*m*k) nested scan with O(m*k) build + O(n*m) lookup.
	type taxMembership struct {
		taxName    string
		validSlugs map[string]bool
		entityToEntries map[string][]string // entity slug -> entry slugs
	}
	memberships := make([]taxMembership, len(p.Taxonomies))
	for i, tg := range p.Taxonomies {
		inv := make(map[string][]string)
		for _, entry := range tg.Taxonomy.Entries {
			for _, e := range entry.Entities {
				inv[e.Slug] = append(inv[e.Slug], entry.Slug)
			}
		}
		memberships[i] = taxMembership{
			taxName:         tg.Taxonomy.Name,
			validSlugs:      tg.ValidSlugs,
			entityToEntries: inv,
		}
	}

	// Populate ValidSlugs and TaxonomyMemberships on each entity via O(1) lookup.
	for _, re := range p.Entities {
		for _, tm := range memberships {
			re.ValidSlugs[tm.taxName] = tm.validSlugs
			if entries, ok := tm.entityToEntries[re.Slug]; ok {
				re.TaxonomyMemberships[tm.taxName] = entries
			}
		}
	}

	return nil
}
