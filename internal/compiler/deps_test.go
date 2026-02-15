package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestZeroDependencies enforces that go.mod contains no require directives.
// schemaflux uses only the Go standard library. See AGENTS.md for rationale.
func TestZeroDependencies(t *testing.T) {
	// Walk up from the test file to find the module root (directory containing go.mod).
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod")
		}
		dir = parent
	}

	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatalf("reading go.mod: %v", err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "require ") || trimmed == "require (" {
			t.Fatalf("go.mod contains a require directive — schemaflux must have zero external dependencies.\nLine: %s", trimmed)
		}
	}
}
