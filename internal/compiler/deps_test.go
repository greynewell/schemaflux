package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestNonMISTDependencies enforces that go.mod contains no require directives
// beyond the MIST stack shared library. The compiler itself uses only the Go
// standard library; only the MIST protocol integration layer depends on mist-go.
func TestNonMISTDependencies(t *testing.T) {
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

	allowed := map[string]bool{
		"github.com/greynewell/mist-go": true,
	}

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "require (" || trimmed == ")" || trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "require ") {
			dep := strings.TrimPrefix(trimmed, "require ")
			dep = strings.Fields(dep)[0]
			if !allowed[dep] {
				t.Fatalf("go.mod contains a disallowed dependency: %s\nOnly mist-go is permitted.", dep)
			}
		}
	}
}
