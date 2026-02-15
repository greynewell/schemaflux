// Package schemaflux implements the SchemaFlux data pipeline tool for the
// MIST stack. It manages schemas, validates data against them, and reports
// trace spans to TokenTrace.
package schemaflux

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// Schema defines the structure of a data entity.
type Schema struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

// Field is a single field in a schema.
type Field struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // "string", "int", "float", "bool", "object", "array"
	Required bool   `json:"required"`
}

// Validate checks that the schema definition is well-formed.
func (s *Schema) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("schemaflux: schema name is required")
	}
	if len(s.Fields) == 0 {
		return fmt.Errorf("schemaflux: schema %q has no fields", s.Name)
	}
	validTypes := map[string]bool{
		"string": true, "int": true, "float": true,
		"bool": true, "object": true, "array": true,
	}
	for i, f := range s.Fields {
		if f.Name == "" {
			return fmt.Errorf("schemaflux: schema %q field[%d] has no name", s.Name, i)
		}
		if !validTypes[f.Type] {
			return fmt.Errorf("schemaflux: schema %q field %q has invalid type %q", s.Name, f.Name, f.Type)
		}
	}
	return nil
}

// Registry holds named schemas, thread-safe.
type Registry struct {
	mu      sync.RWMutex
	schemas map[string]*Schema
}

// NewRegistry creates an empty schema registry.
func NewRegistry() *Registry {
	return &Registry{schemas: make(map[string]*Schema)}
}

// Register adds a schema to the registry.
func (r *Registry) Register(s *Schema) error {
	if err := s.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.schemas[s.Name] = s
	return nil
}

// Get returns a schema by name.
func (r *Registry) Get(name string) (*Schema, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.schemas[name]
	return s, ok
}

// Names returns all registered schema names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.schemas))
	for name := range r.schemas {
		names = append(names, name)
	}
	return names
}

// All returns a copy of all registered schemas.
func (r *Registry) All() []*Schema {
	r.mu.RLock()
	defer r.mu.RUnlock()
	schemas := make([]*Schema, 0, len(r.schemas))
	for _, s := range r.schemas {
		schemas = append(schemas, s)
	}
	return schemas
}

// ValidateData checks a JSON object against a named schema.
// Returns a list of validation errors (empty means valid).
func (r *Registry) ValidateData(schemaName string, data json.RawMessage) []string {
	r.mu.RLock()
	s, ok := r.schemas[schemaName]
	r.mu.RUnlock()

	if !ok {
		return []string{fmt.Sprintf("unknown schema: %s", schemaName)}
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return []string{fmt.Sprintf("invalid JSON object: %s", err)}
	}

	var errs []string
	for _, field := range s.Fields {
		raw, exists := obj[field.Name]
		if !exists || string(raw) == "null" {
			if field.Required {
				errs = append(errs, fmt.Sprintf("missing required field: %s", field.Name))
			}
			continue
		}
		if err := checkType(field.Name, field.Type, raw); err != "" {
			errs = append(errs, err)
		}
	}
	return errs
}

func checkType(name, typ string, raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if len(s) == 0 {
		return fmt.Sprintf("field %s: empty value", name)
	}

	switch typ {
	case "string":
		if s[0] != '"' {
			return fmt.Sprintf("field %s: expected string, got %s", name, truncate(s))
		}
	case "int":
		// Must be a number without a decimal point.
		var v json.Number
		if err := json.Unmarshal(raw, &v); err != nil {
			return fmt.Sprintf("field %s: expected int, got %s", name, truncate(s))
		}
		if strings.Contains(v.String(), ".") {
			return fmt.Sprintf("field %s: expected int, got float", name)
		}
	case "float":
		var v float64
		if err := json.Unmarshal(raw, &v); err != nil {
			return fmt.Sprintf("field %s: expected float, got %s", name, truncate(s))
		}
	case "bool":
		if s != "true" && s != "false" {
			return fmt.Sprintf("field %s: expected bool, got %s", name, truncate(s))
		}
	case "object":
		if s[0] != '{' {
			return fmt.Sprintf("field %s: expected object, got %s", name, truncate(s))
		}
	case "array":
		if s[0] != '[' {
			return fmt.Sprintf("field %s: expected array, got %s", name, truncate(s))
		}
	}
	return ""
}

func truncate(s string) string {
	if len(s) > 30 {
		return s[:30] + "..."
	}
	return s
}
