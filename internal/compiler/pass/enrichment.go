package pass

import (
	"log"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
	"github.com/greynewell/schemaflux/internal/enrichment"
)

// EnrichmentPass reads JSON sidecar cache files and merges data into ResolvedEntity.Enrichment.
type EnrichmentPass struct{}

func (e *EnrichmentPass) Name() string        { return "Enrichment" }
func (e *EnrichmentPass) Requires() []string { return []string{"SlugResolution"} }

func (e *EnrichmentPass) Run(p *ir.Program) error {
	if p.Config.Enrichment.CacheDir == "" {
		return nil
	}

	log.Printf("Loading enrichment cache from %s...", p.Config.Enrichment.CacheDir)
	enrichmentData, err := enrichment.ReadAllCaches(p.Config.Enrichment.CacheDir)
	if err != nil {
		log.Printf("Warning: failed to load enrichment cache: %v", err)
		return nil
	}
	log.Printf("Loaded enrichment data for %d entities", len(enrichmentData))

	for _, re := range p.Entities {
		if data, ok := enrichmentData[re.Slug]; ok {
			re.Enrichment = data
		}
	}

	return nil
}
