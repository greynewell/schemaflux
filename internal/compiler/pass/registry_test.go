package pass

import (
	"fmt"
	"strings"
	"testing"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
	"github.com/greynewell/schemaflux/internal/config"
)

// testPass is a mock pass for testing.
type testPass struct {
	name    string
	runFunc func(*ir.Program) error
}

func (t *testPass) Name() string            { return t.name }
func (t *testPass) Run(p *ir.Program) error  { return t.runFunc(p) }

func testProgram() *ir.Program {
	return ir.NewProgram(&config.Config{
		Site: config.SiteConfig{Name: "Test"},
	})
}

func TestRegistryAddAndLen(t *testing.T) {
	r := NewRegistry()
	if r.Len() != 0 {
		t.Errorf("Len = %d, want 0", r.Len())
	}

	r.Add(&testPass{name: "A", runFunc: func(*ir.Program) error { return nil }})
	r.Add(&testPass{name: "B", runFunc: func(*ir.Program) error { return nil }})

	if r.Len() != 2 {
		t.Errorf("Len = %d, want 2", r.Len())
	}
}

func TestRegistryPassesOrder(t *testing.T) {
	r := NewRegistry()
	r.Add(&testPass{name: "First", runFunc: func(*ir.Program) error { return nil }})
	r.Add(&testPass{name: "Second", runFunc: func(*ir.Program) error { return nil }})
	r.Add(&testPass{name: "Third", runFunc: func(*ir.Program) error { return nil }})

	passes := r.Passes()
	if len(passes) != 3 {
		t.Fatalf("len(Passes) = %d, want 3", len(passes))
	}

	want := []string{"First", "Second", "Third"}
	for i, p := range passes {
		if p.Name() != want[i] {
			t.Errorf("passes[%d].Name() = %q, want %q", i, p.Name(), want[i])
		}
	}
}

func TestRegistryRunAllSuccess(t *testing.T) {
	r := NewRegistry()

	var order []string
	r.Add(&testPass{name: "A", runFunc: func(*ir.Program) error {
		order = append(order, "A")
		return nil
	}})
	r.Add(&testPass{name: "B", runFunc: func(*ir.Program) error {
		order = append(order, "B")
		return nil
	}})

	p := testProgram()
	timings, err := r.RunAll(p)
	if err != nil {
		t.Fatalf("RunAll error: %v", err)
	}

	if len(order) != 2 || order[0] != "A" || order[1] != "B" {
		t.Errorf("Execution order = %v, want [A B]", order)
	}

	if len(timings) != 2 {
		t.Fatalf("len(timings) = %d, want 2", len(timings))
	}
	if timings[0].Name != "A" || timings[1].Name != "B" {
		t.Errorf("Timing names = [%s, %s], want [A, B]", timings[0].Name, timings[1].Name)
	}
}

func TestRegistryRunAllStopsOnError(t *testing.T) {
	r := NewRegistry()

	var order []string
	r.Add(&testPass{name: "A", runFunc: func(*ir.Program) error {
		order = append(order, "A")
		return nil
	}})
	r.Add(&testPass{name: "B", runFunc: func(*ir.Program) error {
		order = append(order, "B")
		return fmt.Errorf("B failed")
	}})
	r.Add(&testPass{name: "C", runFunc: func(*ir.Program) error {
		order = append(order, "C")
		return nil
	}})

	p := testProgram()
	timings, err := r.RunAll(p)
	if err == nil {
		t.Fatal("Expected error from RunAll")
	}

	// C should not have run
	if len(order) != 2 || order[0] != "A" || order[1] != "B" {
		t.Errorf("Execution order = %v, want [A B]", order)
	}

	// Timings should include A and B (B's timing recorded before error returned)
	if len(timings) != 2 {
		t.Fatalf("len(timings) = %d, want 2", len(timings))
	}
}

func TestRegistryRunAllEmpty(t *testing.T) {
	r := NewRegistry()
	p := testProgram()

	timings, err := r.RunAll(p)
	if err != nil {
		t.Fatalf("RunAll error: %v", err)
	}
	if len(timings) != 0 {
		t.Errorf("len(timings) = %d, want 0", len(timings))
	}
}

// depPass is a mock pass that declares dependencies.
type depPass struct {
	name     string
	requires []string
	runFunc  func(*ir.Program) error
}

func (d *depPass) Name() string            { return d.name }
func (d *depPass) Run(p *ir.Program) error { return d.runFunc(p) }
func (d *depPass) Requires() []string      { return d.requires }

func TestRegistryValidateSuccess(t *testing.T) {
	r := NewRegistry()
	noop := func(*ir.Program) error { return nil }
	r.Add(&depPass{name: "A", runFunc: noop})
	r.Add(&depPass{name: "B", requires: []string{"A"}, runFunc: noop})
	r.Add(&depPass{name: "C", requires: []string{"A", "B"}, runFunc: noop})

	if err := r.Validate(); err != nil {
		t.Fatalf("Validate error: %v", err)
	}
}

func TestRegistryValidateMissingDep(t *testing.T) {
	r := NewRegistry()
	noop := func(*ir.Program) error { return nil }
	r.Add(&depPass{name: "B", requires: []string{"A"}, runFunc: noop})

	err := r.Validate()
	if err == nil {
		t.Fatal("Expected error for missing dependency")
	}
	if !strings.Contains(err.Error(), "requires \"A\"") {
		t.Errorf("Error = %q, expected mention of missing dep A", err.Error())
	}
}

func TestRegistryValidateWrongOrder(t *testing.T) {
	r := NewRegistry()
	noop := func(*ir.Program) error { return nil }
	// B requires A, but A is registered after B
	r.Add(&depPass{name: "B", requires: []string{"A"}, runFunc: noop})
	r.Add(&depPass{name: "A", runFunc: noop})

	err := r.Validate()
	if err == nil {
		t.Fatal("Expected error for wrong order")
	}
}

func TestRegistryRunAllValidatesFirst(t *testing.T) {
	r := NewRegistry()
	noop := func(*ir.Program) error { return nil }
	r.Add(&depPass{name: "B", requires: []string{"A"}, runFunc: noop})

	p := testProgram()
	_, err := r.RunAll(p)
	if err == nil {
		t.Fatal("Expected validation error from RunAll")
	}
}

func TestRegistryPassesCopied(t *testing.T) {
	r := NewRegistry()
	r.Add(&testPass{name: "A", runFunc: func(*ir.Program) error { return nil }})

	passes := r.Passes()
	passes[0] = nil // mutate returned slice

	// Original should be unchanged
	original := r.Passes()
	if original[0] == nil {
		t.Error("Passes() should return a copy")
	}
}
