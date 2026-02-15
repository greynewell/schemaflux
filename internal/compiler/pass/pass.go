package pass

import (
	"github.com/greynewell/schemaflux/internal/compiler/ir"
)

// Pass is a single transformation step in the compiler pipeline.
// A pass may ADD data to the IR but must not REMOVE data a previous pass set.
type Pass interface {
	Name() string
	Run(p *ir.Program) error
}

// DependencyAware is an optional interface a Pass may implement to declare
// which other passes must run before it. The registry validates ordering
// before execution begins.
type DependencyAware interface {
	Requires() []string
}
