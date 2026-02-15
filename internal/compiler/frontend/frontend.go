package frontend

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
	"github.com/greynewell/schemaflux/internal/config"
)

// Parse reads .md files from the data directory and produces an IR Program
// with unresolved ResolvedEntity wrappers.
func Parse(cfg *config.Config) (*ir.Program, error) {
	log.Printf("Loading entities from %s...", cfg.Paths.Data)
	ldr := &markdownLoader{cfg: cfg}
	entities, err := ldr.load()
	if err != nil {
		return nil, fmt.Errorf("loading entities: %w", err)
	}
	log.Printf("Loaded %d entities", len(entities))

	p := ir.NewProgram(cfg)
	for _, e := range entities {
		p.Entities = append(p.Entities, ir.NewResolvedEntity(e))
	}

	// Schema validation (if configured)
	if cfg.FieldIndex != nil && cfg.FieldIndex.HasSchema() {
		errorCount := 0
		for _, e := range entities {
			diags := validateEntitySchema(e, cfg.FieldIndex)
			for _, d := range diags {
				p.Diagnostics = append(p.Diagnostics, d)
				if d.Level == ir.DiagError {
					errorCount++
				}
			}
		}
		if p.HasErrors() {
			return p, fmt.Errorf("schema validation failed: %d error(s)", errorCount)
		}
	}

	// Load contributors
	loadContributors(p)

	return p, nil
}

func loadContributors(p *ir.Program) {
	path := p.Config.Extra.Contributors
	if path == "" {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Warning: failed to load contributors: %v", err)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Printf("Warning: failed to parse contributors: %v", err)
		return
	}
	p.Contributors = result
}
