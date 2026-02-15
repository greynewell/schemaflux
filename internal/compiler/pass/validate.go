package pass

import (
	"github.com/greynewell/schemaflux/internal/compiler/ir"
)

// ValidationPass runs ir.Validate() and appends diagnostics.
type ValidationPass struct{}

func (v *ValidationPass) Name() string        { return "Validation" }
func (v *ValidationPass) Requires() []string { return []string{"URLResolution", "SlugResolution"} }

func (v *ValidationPass) Run(p *ir.Program) error {
	ir.Validate(p)
	return nil
}
