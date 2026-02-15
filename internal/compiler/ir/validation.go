package ir

import (
	"fmt"
)

// Validate enforces IR contracts:
// - Required fields (title) on every entity
// - Slug uniqueness
// - Pairing reference integrity (no dangling refs)
// - URL presence after URLResolutionPass
func Validate(p *Program) {
	seen := make(map[string]bool)

	for _, re := range p.Entities {
		slug := re.Slug

		// Slug uniqueness
		if seen[slug] {
			p.AddDiagnostic(DiagError, fmt.Sprintf("duplicate slug: %q", slug), slug)
		}
		seen[slug] = true

		// Required field: title
		if re.Raw.GetString("title") == "" {
			p.AddDiagnostic(DiagWarning, "missing required field: title", slug)
		}

		// Pairing reference integrity
		for _, ps := range re.Raw.GetStringSlice("pairings") {
			if _, ok := p.EntityBySlug[ps]; !ok {
				p.AddDiagnostic(DiagWarning,
					fmt.Sprintf("pairing reference %q not found", ps), slug)
			}
		}

		// URL should be set after URLResolutionPass
		if re.URL == "" {
			p.AddDiagnostic(DiagWarning, "entity has no URL (URLResolutionPass may not have run)", slug)
		}
	}
}
