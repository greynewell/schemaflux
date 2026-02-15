package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "schemaflux.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	// Create the data directory so validation doesn't fail on missing paths
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadWithFields(t *testing.T) {
	path := writeTestConfig(t, `
site:
  name: Test
  base_url: https://example.com
paths:
  data: data
data:
  fields:
    - name: title
      type: string
      required: true
    - name: servings
      type: int
    - name: language
      type: enum
      allowed:
        - go
        - python
        - rust
    - name: date
      type: date
    - name: tags
      type: list
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.FieldIndex == nil {
		t.Fatal("FieldIndex is nil")
	}
	if !cfg.FieldIndex.HasSchema() {
		t.Error("HasSchema() should be true")
	}

	all := cfg.FieldIndex.All()
	if len(all) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(all))
	}

	title, ok := cfg.FieldIndex.Get("title")
	if !ok {
		t.Fatal("title not found in index")
	}
	if title.Type != "string" {
		t.Errorf("title.Type = %q, want string", title.Type)
	}
	if !title.Required {
		t.Error("title should be required")
	}

	lang, ok := cfg.FieldIndex.Get("language")
	if !ok {
		t.Fatal("language not found in index")
	}
	if lang.Type != "enum" {
		t.Errorf("language.Type = %q", lang.Type)
	}
	if len(lang.Allowed) != 3 {
		t.Errorf("language.Allowed len = %d, want 3", len(lang.Allowed))
	}
}

func TestLoadWithoutFields(t *testing.T) {
	path := writeTestConfig(t, `
site:
  name: Test
  base_url: https://example.com
paths:
  data: data
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.FieldIndex == nil {
		t.Fatal("FieldIndex should not be nil even without fields")
	}
	if cfg.FieldIndex.HasSchema() {
		t.Error("HasSchema() should be false when no fields declared")
	}
}

func TestLoadRejectsUnknownFieldType(t *testing.T) {
	path := writeTestConfig(t, `
site:
  name: Test
  base_url: https://example.com
paths:
  data: data
data:
  fields:
    - name: title
      type: invalid_type
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unknown field type")
	}
}

func TestFieldSchemaDefault(t *testing.T) {
	path := writeTestConfig(t, `
site:
  name: Test
  base_url: https://example.com
paths:
  data: data
data:
  fields:
    - name: language
      type: string
      default: en
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	lang, ok := cfg.FieldIndex.Get("language")
	if !ok {
		t.Fatal("language not found")
	}
	if lang.Default != "en" {
		t.Errorf("language.Default = %q, want en", lang.Default)
	}
}
