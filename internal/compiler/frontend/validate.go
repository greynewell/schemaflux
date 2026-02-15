package frontend

import (
	"fmt"
	"strings"
	"time"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
	"github.com/greynewell/schemaflux/internal/config"
	"github.com/greynewell/schemaflux/internal/entity"
)

// validateEntitySchema checks an entity's fields against the declared schema.
// It returns diagnostics for required-missing, type-mismatch, and enum violations.
// It also applies defaults for missing optional fields with a declared default.
func validateEntitySchema(e *entity.Entity, schema *config.FieldSchemaIndex) []ir.Diagnostic {
	var diags []ir.Diagnostic

	for _, fs := range schema.All() {
		pos := ir.SourcePos{File: e.SourceFile}
		if e.FieldPositions != nil {
			if line, ok := e.FieldPositions[fs.Name]; ok {
				pos.Line = line
			}
		}

		val, exists := e.Fields[fs.Name]
		isEmpty := !exists || val == nil || val == ""

		// Required check
		if fs.Required && isEmpty {
			diags = append(diags, ir.Diagnostic{
				Level:   ir.DiagError,
				Message: fmt.Sprintf("required field %q is missing", fs.Name),
				Entity:  e.Slug,
				Pos:     pos,
			})
			continue
		}

		// Apply default for missing optional fields
		if isEmpty && fs.Default != "" {
			e.Fields[fs.Name] = fs.Default
			continue
		}

		// Skip type checking if field is absent and not required
		if isEmpty {
			continue
		}

		// Type check
		if err := checkFieldType(val, fs.Type); err != nil {
			diags = append(diags, ir.Diagnostic{
				Level:   ir.DiagError,
				Message: fmt.Sprintf("field %q: %s", fs.Name, err.Error()),
				Entity:  e.Slug,
				Pos:     pos,
			})
			continue
		}

		// Enum check
		if fs.Type == "enum" && len(fs.Allowed) > 0 {
			s, ok := val.(string)
			if ok && !contains(fs.Allowed, s) {
				diags = append(diags, ir.Diagnostic{
					Level:   ir.DiagError,
					Message: fmt.Sprintf("field %q: value %q not in allowed list %v", fs.Name, s, fs.Allowed),
					Entity:  e.Slug,
					Pos:     pos,
				})
			}
		}
	}

	return diags
}

// checkFieldType validates that val matches the declared type.
func checkFieldType(val interface{}, typ string) error {
	switch typ {
	case "string":
		if _, ok := val.(string); !ok {
			return fmt.Errorf("expected string, got %T", val)
		}
	case "int":
		switch val.(type) {
		case int, int64:
			// ok
		default:
			return fmt.Errorf("expected int, got %T", val)
		}
	case "float":
		switch val.(type) {
		case float64, int, int64:
			// ok — ints are valid floats
		default:
			return fmt.Errorf("expected float, got %T", val)
		}
	case "bool":
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("expected bool, got %T", val)
		}
	case "date":
		s, ok := val.(string)
		if !ok {
			return fmt.Errorf("expected date string, got %T", val)
		}
		if _, err := time.Parse("2006-01-02", s); err != nil {
			return fmt.Errorf("expected date (YYYY-MM-DD), got %q", s)
		}
	case "list":
		switch val.(type) {
		case []interface{}, []string:
			// ok
		default:
			return fmt.Errorf("expected list, got %T", val)
		}
	case "enum":
		if _, ok := val.(string); !ok {
			return fmt.Errorf("expected string (enum), got %T", val)
		}
	case "":
		// no type declared — skip
	default:
		return fmt.Errorf("unknown type %q", typ)
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if strings.EqualFold(item, s) {
			return true
		}
	}
	return false
}
