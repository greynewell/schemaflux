package schemaflux

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/greynewell/mist-go/protocol"
	"github.com/greynewell/mist-go/tokentrace"
)

// --- Schema tests ---

func TestSchemaValidation(t *testing.T) {
	tests := []struct {
		name    string
		schema  Schema
		wantErr bool
	}{
		{"valid", Schema{Name: "user", Fields: []Field{{Name: "id", Type: "int", Required: true}}}, false},
		{"no name", Schema{Fields: []Field{{Name: "id", Type: "int"}}}, true},
		{"no fields", Schema{Name: "empty"}, true},
		{"field no name", Schema{Name: "s", Fields: []Field{{Type: "string"}}}, true},
		{"invalid type", Schema{Name: "s", Fields: []Field{{Name: "f", Type: "bigint"}}}, true},
		{"object type", Schema{Name: "s", Fields: []Field{{Name: "meta", Type: "object"}}}, false},
		{"array type", Schema{Name: "s", Fields: []Field{{Name: "tags", Type: "array"}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- Registry tests ---

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	err := reg.Register(&Schema{
		Name:   "user",
		Fields: []Field{{Name: "name", Type: "string", Required: true}},
	})
	if err != nil {
		t.Fatal(err)
	}

	s, ok := reg.Get("user")
	if !ok {
		t.Fatal("expected to find user schema")
	}
	if s.Name != "user" {
		t.Errorf("Name = %s, want user", s.Name)
	}
}

func TestRegistryNames(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Schema{Name: "a", Fields: []Field{{Name: "f", Type: "string"}}})
	reg.Register(&Schema{Name: "b", Fields: []Field{{Name: "f", Type: "int"}}})

	if len(reg.Names()) != 2 {
		t.Errorf("Names = %d, want 2", len(reg.Names()))
	}
}

func TestRegistryRejectInvalid(t *testing.T) {
	reg := NewRegistry()
	err := reg.Register(&Schema{Name: ""})
	if err == nil {
		t.Error("expected error for invalid schema")
	}
}

// --- ValidateData tests ---

func TestValidateDataValid(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Schema{
		Name: "user",
		Fields: []Field{
			{Name: "name", Type: "string", Required: true},
			{Name: "age", Type: "int", Required: true},
		},
	})

	errs := reg.ValidateData("user", json.RawMessage(`{"name":"Alice","age":30}`))
	if len(errs) != 0 {
		t.Errorf("expected valid, got errors: %v", errs)
	}
}

func TestValidateDataMissingRequired(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Schema{
		Name: "user",
		Fields: []Field{
			{Name: "name", Type: "string", Required: true},
			{Name: "age", Type: "int", Required: true},
		},
	})

	errs := reg.ValidateData("user", json.RawMessage(`{"name":"Alice"}`))
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

func TestValidateDataWrongType(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Schema{
		Name: "user",
		Fields: []Field{
			{Name: "age", Type: "int", Required: true},
		},
	})

	errs := reg.ValidateData("user", json.RawMessage(`{"age":"not-a-number"}`))
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

func TestValidateDataFloatForInt(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Schema{
		Name: "user",
		Fields: []Field{
			{Name: "age", Type: "int"},
		},
	})

	errs := reg.ValidateData("user", json.RawMessage(`{"age":30.5}`))
	if len(errs) != 1 {
		t.Fatalf("expected 1 error (float for int), got %d: %v", len(errs), errs)
	}
}

func TestValidateDataOptionalField(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Schema{
		Name: "user",
		Fields: []Field{
			{Name: "name", Type: "string", Required: true},
			{Name: "bio", Type: "string", Required: false},
		},
	})

	errs := reg.ValidateData("user", json.RawMessage(`{"name":"Alice"}`))
	if len(errs) != 0 {
		t.Errorf("optional field missing should not error: %v", errs)
	}
}

func TestValidateDataUnknownSchema(t *testing.T) {
	reg := NewRegistry()
	errs := reg.ValidateData("nonexistent", json.RawMessage(`{}`))
	if len(errs) != 1 {
		t.Errorf("expected 1 error for unknown schema, got %d", len(errs))
	}
}

func TestValidateDataInvalidJSON(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Schema{
		Name:   "user",
		Fields: []Field{{Name: "name", Type: "string"}},
	})

	errs := reg.ValidateData("user", json.RawMessage(`not json`))
	if len(errs) != 1 {
		t.Errorf("expected 1 error for invalid JSON, got %d", len(errs))
	}
}

func TestValidateDataBool(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Schema{
		Name:   "config",
		Fields: []Field{{Name: "active", Type: "bool", Required: true}},
	})

	errs := reg.ValidateData("config", json.RawMessage(`{"active":true}`))
	if len(errs) != 0 {
		t.Errorf("expected valid bool, got errors: %v", errs)
	}

	errs = reg.ValidateData("config", json.RawMessage(`{"active":"yes"}`))
	if len(errs) != 1 {
		t.Errorf("expected 1 error for string as bool, got %d", len(errs))
	}
}

func TestValidateDataObjectAndArray(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Schema{
		Name: "complex",
		Fields: []Field{
			{Name: "meta", Type: "object", Required: true},
			{Name: "tags", Type: "array", Required: true},
		},
	})

	errs := reg.ValidateData("complex", json.RawMessage(`{"meta":{"key":"val"},"tags":["a","b"]}`))
	if len(errs) != 0 {
		t.Errorf("expected valid, got errors: %v", errs)
	}

	errs = reg.ValidateData("complex", json.RawMessage(`{"meta":"string","tags":"string"}`))
	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d: %v", len(errs), errs)
	}
}

// --- Handler tests ---

func testHandler() *Handler {
	reg := NewRegistry()
	reg.Register(&Schema{
		Name: "user",
		Fields: []Field{
			{Name: "name", Type: "string", Required: true},
			{Name: "age", Type: "int", Required: true},
		},
	})
	reporter := tokentrace.NewReporter("schemaflux", "")
	return NewHandler(reg, reporter)
}

func TestHandlerValidateValid(t *testing.T) {
	h := testHandler()
	body, _ := json.Marshal(ValidateRequest{
		Schema: "user",
		Data:   json.RawMessage(`{"name":"Alice","age":30}`),
	})

	req := httptest.NewRequest("POST", "/validate", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Validate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp ValidateResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Valid {
		t.Errorf("expected valid, got errors: %v", resp.Errors)
	}
}

func TestHandlerValidateInvalid(t *testing.T) {
	h := testHandler()
	body, _ := json.Marshal(ValidateRequest{
		Schema: "user",
		Data:   json.RawMessage(`{"name":"Alice"}`),
	})

	req := httptest.NewRequest("POST", "/validate", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Validate(w, req)

	var resp ValidateResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Valid {
		t.Error("expected invalid response")
	}
	if len(resp.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(resp.Errors))
	}
}

func TestHandlerRegisterSchema(t *testing.T) {
	reg := NewRegistry()
	reporter := tokentrace.NewReporter("schemaflux", "")
	h := NewHandler(reg, reporter)

	body, _ := json.Marshal(Schema{
		Name:   "event",
		Fields: []Field{{Name: "type", Type: "string", Required: true}},
	})

	req := httptest.NewRequest("POST", "/schemas", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.RegisterSchema(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body: %s", w.Code, w.Body.String())
	}

	// Verify it's registered.
	if _, ok := reg.Get("event"); !ok {
		t.Error("schema should be registered")
	}
}

func TestHandlerSchemas(t *testing.T) {
	h := testHandler()
	req := httptest.NewRequest("GET", "/schemas", nil)
	w := httptest.NewRecorder()
	h.Schemas(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp SchemasResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Schemas) != 1 {
		t.Errorf("expected 1 schema, got %d", len(resp.Schemas))
	}
}

func TestHandlerIngestSchema(t *testing.T) {
	reg := NewRegistry()
	reporter := tokentrace.NewReporter("schemaflux", "")
	h := NewHandler(reg, reporter)

	ds := protocol.DataSchema{
		Name:   "product",
		Fields: []protocol.SchemaField{{Name: "sku", Type: "string", Required: true}},
	}
	msg, _ := protocol.New("test", protocol.TypeDataSchema, ds)
	body, _ := msg.Marshal()

	req := httptest.NewRequest("POST", "/mist", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Ingest(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body: %s", w.Code, w.Body.String())
	}

	if _, ok := reg.Get("product"); !ok {
		t.Error("schema should be registered via MIST protocol")
	}
}

func TestHandlerIngestWrongType(t *testing.T) {
	h := testHandler()
	msg, _ := protocol.New("test", protocol.TypeHealthPing, protocol.HealthPing{From: "test"})
	body, _ := msg.Marshal()

	req := httptest.NewRequest("POST", "/mist", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Ingest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandlerIngestEntitiesKnownSchema(t *testing.T) {
	h := testHandler()
	de := protocol.DataEntities{Count: 10, Format: "json", Schema: "user"}
	msg, _ := protocol.New("test", protocol.TypeDataEntities, de)
	body, _ := msg.Marshal()

	req := httptest.NewRequest("POST", "/mist", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Ingest(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202, body: %s", w.Code, w.Body.String())
	}
}

func TestHandlerIngestEntitiesUnknownSchema(t *testing.T) {
	h := testHandler()
	de := protocol.DataEntities{Count: 10, Format: "json", Schema: "nonexistent"}
	msg, _ := protocol.New("test", protocol.TypeDataEntities, de)
	body, _ := msg.Marshal()

	req := httptest.NewRequest("POST", "/mist", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Ingest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandlerMethodNotAllowed(t *testing.T) {
	h := testHandler()
	req := httptest.NewRequest("GET", "/mist", nil)
	w := httptest.NewRecorder()
	h.Ingest(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}
