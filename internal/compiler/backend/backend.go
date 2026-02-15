package backend

import (
	"github.com/greynewell/schemaflux/internal/compiler/ir"
)

// Backend emits output from a compiled IR Program.
type Backend interface {
	Name() string
	Emit(p *ir.Program, outputDir string) error
}
