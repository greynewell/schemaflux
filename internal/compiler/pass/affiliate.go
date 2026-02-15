package pass

import (
	"github.com/greynewell/schemaflux/internal/affiliate"
	"github.com/greynewell/schemaflux/internal/compiler/ir"
)

// AffiliatePass generates affiliate links from enrichment data for each entity.
type AffiliatePass struct{}

func (a *AffiliatePass) Name() string        { return "Affiliate" }
func (a *AffiliatePass) Requires() []string { return []string{"Enrichment"} }

func (a *AffiliatePass) Run(p *ir.Program) error {
	registry := affiliate.NewRegistry(p.Config.Affiliates)

	for _, re := range p.Entities {
		if re.Enrichment != nil {
			re.AffiliateLinks = registry.GenerateLinks(re.Enrichment, p.Config.Affiliates.SearchTermPaths)
		}
	}

	return nil
}
