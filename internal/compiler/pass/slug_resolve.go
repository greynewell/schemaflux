package pass

import (
	"github.com/greynewell/schemaflux/internal/compiler/ir"
)

// SlugResolutionPass builds EntityBySlug map and resolves pairing references.
type SlugResolutionPass struct{}

func (s *SlugResolutionPass) Name() string { return "SlugResolution" }

func (s *SlugResolutionPass) Run(p *ir.Program) error {
	// Build EntityBySlug map
	for _, re := range p.Entities {
		p.EntityBySlug[re.Slug] = re
	}

	// Resolve pairing references
	for _, re := range p.Entities {
		pairingSlugs := re.Raw.GetStringSlice("pairings")
		if len(pairingSlugs) == 0 {
			continue
		}
		for _, ps := range pairingSlugs {
			if paired, ok := p.EntityBySlug[ps]; ok {
				re.Pairings = append(re.Pairings, paired)
			}
		}
	}

	p.TotalEntityCount = len(p.Entities)
	return nil
}
