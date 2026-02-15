package frontend

import (
	"testing"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
	"github.com/greynewell/schemaflux/internal/config"
	"github.com/greynewell/schemaflux/internal/entity"
)

// buildIndex is a test helper that builds a FieldSchemaIndex.
func buildIndex(fields []config.FieldSchema) *config.FieldSchemaIndex {
	// We need to go through Load or build manually. Since the fields are
	// unexported, let's use a minimal config load approach.
	// Actually, let's just directly construct what we need using the public API.
	cfg := &config.Config{
		Site:  config.SiteConfig{Name: "Test", BaseURL: "https://example.com"},
		Paths: config.PathsConfig{Data: "/tmp"},
		Data:  config.DataConfig{Fields: fields},
	}
	cfg.FieldIndex = config.BuildFieldIndex(cfg.Data.Fields)
	return cfg.FieldIndex
}

func TestValidateRequiredMissing(t *testing.T) {
	schema := buildIndex([]config.FieldSchema{
		{Name: "title", Type: "string", Required: true},
	})

	e := &entity.Entity{
		Slug:       "test-entity",
		SourceFile: "content/test.md",
		Fields:     map[string]interface{}{},
		FieldPositions: map[string]int{},
	}

	diags := validateEntitySchema(e, schema)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Level != ir.DiagError {
		t.Errorf("level = %v, want DiagError", diags[0].Level)
	}
	if diags[0].Pos.File != "content/test.md" {
		t.Errorf("pos.File = %q", diags[0].Pos.File)
	}
}

func TestValidateRequiredMissingWithSourcePos(t *testing.T) {
	schema := buildIndex([]config.FieldSchema{
		{Name: "title", Type: "string", Required: true},
	})

	e := &entity.Entity{
		Slug:       "test-entity",
		SourceFile: "content/test.md",
		Fields:     map[string]interface{}{"title": ""},
		FieldPositions: map[string]int{"title": 3},
	}

	diags := validateEntitySchema(e, schema)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Pos.Line != 3 {
		t.Errorf("pos.Line = %d, want 3", diags[0].Pos.Line)
	}
}

func TestValidateTypeMismatchIntField(t *testing.T) {
	schema := buildIndex([]config.FieldSchema{
		{Name: "servings", Type: "int"},
	})

	e := &entity.Entity{
		Slug:       "test-entity",
		SourceFile: "content/test.md",
		Fields:     map[string]interface{}{"servings": "four"},
		FieldPositions: map[string]int{"servings": 7},
	}

	diags := validateEntitySchema(e, schema)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Level != ir.DiagError {
		t.Errorf("level = %v, want DiagError", diags[0].Level)
	}
	if diags[0].Pos.Line != 7 {
		t.Errorf("pos.Line = %d, want 7", diags[0].Pos.Line)
	}
}

func TestValidateDateValid(t *testing.T) {
	schema := buildIndex([]config.FieldSchema{
		{Name: "date", Type: "date"},
	})

	e := &entity.Entity{
		Slug:       "test-entity",
		SourceFile: "content/test.md",
		Fields:     map[string]interface{}{"date": "2024-06-15"},
		FieldPositions: map[string]int{"date": 5},
	}

	diags := validateEntitySchema(e, schema)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for valid date, got %d: %v", len(diags), diags)
	}
}

func TestValidateDateInvalid(t *testing.T) {
	schema := buildIndex([]config.FieldSchema{
		{Name: "date", Type: "date"},
	})

	e := &entity.Entity{
		Slug:       "test-entity",
		SourceFile: "content/test.md",
		Fields:     map[string]interface{}{"date": "not-a-date"},
		FieldPositions: map[string]int{"date": 5},
	}

	diags := validateEntitySchema(e, schema)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic for invalid date, got %d", len(diags))
	}
	if diags[0].Level != ir.DiagError {
		t.Errorf("level = %v, want DiagError", diags[0].Level)
	}
}

func TestValidateEnumValid(t *testing.T) {
	schema := buildIndex([]config.FieldSchema{
		{Name: "language", Type: "enum", Allowed: []string{"go", "python", "rust"}},
	})

	e := &entity.Entity{
		Slug:       "test-entity",
		SourceFile: "content/test.md",
		Fields:     map[string]interface{}{"language": "go"},
		FieldPositions: map[string]int{"language": 4},
	}

	diags := validateEntitySchema(e, schema)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for valid enum, got %d", len(diags))
	}
}

func TestValidateEnumInvalid(t *testing.T) {
	schema := buildIndex([]config.FieldSchema{
		{Name: "cuisine", Type: "enum", Allowed: []string{"Italian", "Indian", "Mexican"}},
	})

	e := &entity.Entity{
		Slug:       "pad-thai",
		SourceFile: "content/pad-thai.md",
		Fields:     map[string]interface{}{"cuisine": "Thai"},
		FieldPositions: map[string]int{"cuisine": 5},
	}

	diags := validateEntitySchema(e, schema)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic for invalid enum, got %d", len(diags))
	}
	if diags[0].Level != ir.DiagError {
		t.Errorf("level = %v, want DiagError", diags[0].Level)
	}
	if diags[0].Pos.Line != 5 {
		t.Errorf("pos.Line = %d, want 5", diags[0].Pos.Line)
	}
}

func TestValidateDefaultApplied(t *testing.T) {
	schema := buildIndex([]config.FieldSchema{
		{Name: "language", Type: "string", Default: "en"},
	})

	e := &entity.Entity{
		Slug:       "test-entity",
		SourceFile: "content/test.md",
		Fields:     map[string]interface{}{},
		FieldPositions: map[string]int{},
	}

	diags := validateEntitySchema(e, schema)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics when default is applied, got %d", len(diags))
	}
	if e.Fields["language"] != "en" {
		t.Errorf("default not applied: language = %v", e.Fields["language"])
	}
}

func TestValidateNoSchema(t *testing.T) {
	schema := buildIndex(nil)

	e := &entity.Entity{
		Slug:       "test-entity",
		SourceFile: "content/test.md",
		Fields:     map[string]interface{}{"anything": "goes"},
	}

	diags := validateEntitySchema(e, schema)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics with no schema, got %d", len(diags))
	}
}

func TestValidateMultipleErrors(t *testing.T) {
	schema := buildIndex([]config.FieldSchema{
		{Name: "title", Type: "string", Required: true},
		{Name: "servings", Type: "int"},
		{Name: "cuisine", Type: "enum", Allowed: []string{"Italian", "Indian", "Mexican"}},
	})

	e := &entity.Entity{
		Slug:       "test-entity",
		SourceFile: "content/test.md",
		Fields: map[string]interface{}{
			"servings": "four",
			"cuisine":  "Thai",
		},
		FieldPositions: map[string]int{
			"servings": 7,
			"cuisine":  5,
		},
	}

	diags := validateEntitySchema(e, schema)
	if len(diags) != 3 {
		t.Fatalf("expected 3 diagnostics (required + type + enum), got %d", len(diags))
	}

	errorCount := 0
	for _, d := range diags {
		if d.Level == ir.DiagError {
			errorCount++
		}
	}
	if errorCount != 3 {
		t.Errorf("expected 3 errors, got %d", errorCount)
	}
}

func TestValidateBoolField(t *testing.T) {
	schema := buildIndex([]config.FieldSchema{
		{Name: "featured", Type: "bool"},
	})

	// Valid bool
	e := &entity.Entity{
		Slug:       "test",
		SourceFile: "test.md",
		Fields:     map[string]interface{}{"featured": true},
	}
	diags := validateEntitySchema(e, schema)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for valid bool, got %d", len(diags))
	}

	// Invalid bool
	e2 := &entity.Entity{
		Slug:       "test",
		SourceFile: "test.md",
		Fields:     map[string]interface{}{"featured": "yes"},
	}
	diags2 := validateEntitySchema(e2, schema)
	if len(diags2) != 1 {
		t.Fatalf("expected 1 diagnostic for invalid bool, got %d", len(diags2))
	}
}

func TestValidateListField(t *testing.T) {
	schema := buildIndex([]config.FieldSchema{
		{Name: "tags", Type: "list"},
	})

	e := &entity.Entity{
		Slug:       "test",
		SourceFile: "test.md",
		Fields:     map[string]interface{}{"tags": []interface{}{"go", "yaml"}},
	}
	diags := validateEntitySchema(e, schema)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for valid list, got %d", len(diags))
	}
}

func TestValidateFloatField(t *testing.T) {
	schema := buildIndex([]config.FieldSchema{
		{Name: "rating", Type: "float"},
	})

	// float64 is valid
	e := &entity.Entity{
		Slug:       "test",
		SourceFile: "test.md",
		Fields:     map[string]interface{}{"rating": 4.5},
	}
	diags := validateEntitySchema(e, schema)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for float64, got %d", len(diags))
	}

	// int is also valid for float
	e2 := &entity.Entity{
		Slug:       "test",
		SourceFile: "test.md",
		Fields:     map[string]interface{}{"rating": 4},
	}
	diags2 := validateEntitySchema(e2, schema)
	if len(diags2) != 0 {
		t.Errorf("expected 0 diagnostics for int-as-float, got %d", len(diags2))
	}
}
