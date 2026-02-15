package compiler

import (
	"fmt"
	"log"
	"time"

	"github.com/greynewell/schemaflux/internal/compiler/backend"
	"github.com/greynewell/schemaflux/internal/compiler/backend/html"
	"github.com/greynewell/schemaflux/internal/compiler/frontend"
	"github.com/greynewell/schemaflux/internal/compiler/pass"
	"github.com/greynewell/schemaflux/internal/config"
)

// Compile runs the full compiler pipeline: frontend -> passes -> backend.
func Compile(cfg *config.Config) error {
	start := time.Now()
	log.Printf("Compiling site: %s", cfg.Site.Name)

	// Frontend: parse .md files into IR
	p, err := frontend.Parse(cfg)
	if err != nil {
		return fmt.Errorf("frontend: %w", err)
	}

	// Register passes in order
	registry := pass.NewRegistry()
	registry.Add(&pass.SlugResolutionPass{})
	registry.Add(&pass.FavoritesPass{})
	registry.Add(&pass.SortPass{})
	registry.Add(&pass.EnrichmentPass{})
	registry.Add(&pass.AffiliatePass{})
	registry.Add(&pass.TaxonomyPass{})
	registry.Add(&pass.RelatedEntitiesPass{})
	registry.Add(&pass.GraphEnrichmentPass{})
	registry.Add(&pass.ContentAnalysisPass{})
	registry.Add(&pass.URLResolutionPass{})
	registry.Add(&pass.SchemaPass{})
	registry.Add(&pass.ValidationPass{})

	// Run all passes
	log.Printf("Running %d passes...", registry.Len())
	timings, err := registry.RunAll(p)
	if err != nil {
		return fmt.Errorf("pass pipeline: %w", err)
	}

	// Log diagnostics
	for _, d := range p.Diagnostics {
		if d.Entity != "" {
			log.Printf("  [%s] %s: %s", d.Level, d.Entity, d.Message)
		} else {
			log.Printf("  [%s] %s", d.Level, d.Message)
		}
	}

	// Emit via backend
	var be backend.Backend = &html.Backend{}
	log.Printf("Emitting via %s backend...", be.Name())
	if err := be.Emit(p, cfg.Paths.Output); err != nil {
		return fmt.Errorf("backend %s: %w", be.Name(), err)
	}

	elapsed := time.Since(start)
	log.Printf("\nCompile complete!")
	log.Printf("  Entities:   %d", p.TotalEntityCount)
	log.Printf("  Taxonomies: %d", len(p.Taxonomies))
	log.Printf("  Passes:     %d", len(timings))
	for _, t := range timings {
		log.Printf("    %-20s %s", t.Name, t.Duration.Round(time.Millisecond))
	}
	log.Printf("  Output:     %s", cfg.Paths.Output)
	log.Printf("  Duration:   %s", elapsed.Round(time.Millisecond))

	return nil
}
