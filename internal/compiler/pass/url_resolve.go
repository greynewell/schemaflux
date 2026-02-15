package pass

import (
	"fmt"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
)

// URLResolutionPass computes all entity, taxonomy, and pagination URLs.
type URLResolutionPass struct{}

func (u *URLResolutionPass) Name() string        { return "URLResolution" }
func (u *URLResolutionPass) Requires() []string { return []string{"Taxonomy"} }

func (u *URLResolutionPass) Run(p *ir.Program) error {
	baseURL := p.Site.BaseURL

	for _, re := range p.Entities {
		re.URL = fmt.Sprintf("%s/%s.html", baseURL, re.Slug)
		re.CanonicalURL = re.URL
	}

	// Taxonomy index URLs
	for i := range p.Taxonomies {
		tg := &p.Taxonomies[i]
		tg.IndexURL = fmt.Sprintf("%s/%s/", baseURL, tg.Taxonomy.Name)
	}

	return nil
}
