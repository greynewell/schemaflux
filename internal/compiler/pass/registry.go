package pass

import (
	"fmt"
	"log"
	"time"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
)

// Registry holds an ordered list of passes and executes them sequentially.
type Registry struct {
	passes []Pass
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Add appends a pass to the registry.
func (r *Registry) Add(p Pass) {
	r.passes = append(r.passes, p)
}

// Passes returns the registered passes in order.
func (r *Registry) Passes() []Pass {
	result := make([]Pass, len(r.passes))
	copy(result, r.passes)
	return result
}

// Len returns the number of registered passes.
func (r *Registry) Len() int {
	return len(r.passes)
}

// PassTiming records how long a pass took to execute.
type PassTiming struct {
	Name     string
	Duration time.Duration
}

// Validate checks that all declared dependencies are satisfied by earlier passes.
// Returns an error if a pass requires another that is missing or registered later.
func (r *Registry) Validate() error {
	seen := make(map[string]int) // pass name -> index
	for i, p := range r.passes {
		if da, ok := p.(DependencyAware); ok {
			for _, req := range da.Requires() {
				idx, found := seen[req]
				if !found {
					return fmt.Errorf("pass %q requires %q, which is not registered", p.Name(), req)
				}
				if idx >= i {
					return fmt.Errorf("pass %q requires %q, but it is registered later", p.Name(), req)
				}
			}
		}
		seen[p.Name()] = i
	}
	return nil
}

// RunAll validates pass ordering, then executes all passes in order, logging timing.
// Returns timings and the first error encountered (stops on error).
func (r *Registry) RunAll(p *ir.Program) ([]PassTiming, error) {
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("pass ordering: %w", err)
	}

	timings := make([]PassTiming, 0, len(r.passes))

	for i, pass := range r.passes {
		start := time.Now()
		log.Printf("  [%d/%d] %s...", i+1, len(r.passes), pass.Name())

		if err := pass.Run(p); err != nil {
			elapsed := time.Since(start)
			timings = append(timings, PassTiming{Name: pass.Name(), Duration: elapsed})
			return timings, fmt.Errorf("pass %q failed: %w", pass.Name(), err)
		}

		elapsed := time.Since(start)
		timings = append(timings, PassTiming{Name: pass.Name(), Duration: elapsed})
		log.Printf("  [%d/%d] %s done (%s)", i+1, len(r.passes), pass.Name(), elapsed.Round(time.Millisecond))
	}

	return timings, nil
}
