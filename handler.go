package schemaflux

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/greynewell/mist-go/protocol"
	"github.com/greynewell/mist-go/tokentrace"
	"github.com/greynewell/mist-go/trace"
)

// Handler provides HTTP handlers for the SchemaFlux API.
type Handler struct {
	registry *Registry
	reporter *tokentrace.Reporter
}

// NewHandler creates a handler wired to the given registry and reporter.
func NewHandler(registry *Registry, reporter *tokentrace.Reporter) *Handler {
	return &Handler{registry: registry, reporter: reporter}
}

// Ingest handles POST /mist — accepts MIST protocol messages containing
// schema definitions or data entities for validation.
func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg protocol.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "invalid message: "+err.Error(), http.StatusBadRequest)
		return
	}

	switch msg.Type {
	case protocol.TypeDataSchema:
		h.handleSchema(w, r.Context(), &msg)
	case protocol.TypeDataEntities:
		h.handleEntities(w, r.Context(), &msg)
	default:
		http.Error(w, "expected type data.schema or data.entities, got "+msg.Type, http.StatusBadRequest)
	}
}

func (h *Handler) handleSchema(w http.ResponseWriter, ctx context.Context, msg *protocol.Message) {
	var ds protocol.DataSchema
	if err := msg.Decode(&ds); err != nil {
		http.Error(w, "invalid schema payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	schema := &Schema{
		Name:   ds.Name,
		Fields: make([]Field, len(ds.Fields)),
	}
	for i, f := range ds.Fields {
		schema.Fields[i] = Field{Name: f.Name, Type: f.Type, Required: f.Required}
	}

	if err := h.registry.Register(schema); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) handleEntities(w http.ResponseWriter, ctx context.Context, msg *protocol.Message) {
	var de protocol.DataEntities
	if err := msg.Decode(&de); err != nil {
		http.Error(w, "invalid entities payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx, span := trace.Start(ctx, "schemaflux.validate")
	span.SetAttr("schema", de.Schema)
	span.SetAttr("count", de.Count)

	// We acknowledge receipt. In a full implementation, we'd process the
	// entities at de.Path. Here we validate the envelope.
	if de.Schema == "" {
		span.SetAttr("error", "no schema specified")
		span.End("error")
		h.reporter.Report(ctx, span)
		http.Error(w, "schema name required in entities payload", http.StatusBadRequest)
		return
	}

	if _, ok := h.registry.Get(de.Schema); !ok {
		span.SetAttr("error", "unknown schema")
		span.End("error")
		h.reporter.Report(ctx, span)
		http.Error(w, "unknown schema: "+de.Schema, http.StatusBadRequest)
		return
	}

	span.End("ok")
	h.reporter.Report(ctx, span)
	w.WriteHeader(http.StatusAccepted)
}

// ValidateRequest is the JSON body for POST /validate.
type ValidateRequest struct {
	Schema string          `json:"schema"`
	Data   json.RawMessage `json:"data"`
}

// ValidateResponse is the JSON body returned by POST /validate.
type ValidateResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// Validate handles POST /validate — validates a JSON object against a schema.
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx, span := trace.Start(r.Context(), "schemaflux.validate")
	span.SetAttr("schema", req.Schema)

	errs := h.registry.ValidateData(req.Schema, req.Data)
	resp := ValidateResponse{
		Valid:  len(errs) == 0,
		Errors: errs,
	}

	if resp.Valid {
		span.End("ok")
	} else {
		span.SetAttr("validation_errors", len(errs))
		span.End("error")
	}
	h.reporter.Report(ctx, span)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// RegisterSchema handles POST /schemas — registers a schema directly.
func (h *Handler) RegisterSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var schema Schema
	if err := json.NewDecoder(r.Body).Decode(&schema); err != nil {
		http.Error(w, "invalid schema: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.registry.Register(&schema); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// SchemasResponse is the JSON body for GET /schemas.
type SchemasResponse struct {
	Schemas []*Schema `json:"schemas"`
}

// Schemas handles GET /schemas — lists all registered schemas.
func (h *Handler) Schemas(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SchemasResponse{Schemas: h.registry.All()})
}

// SchemaByName handles GET /schemas/{name} — returns a specific schema.
func (h *Handler) SchemaByName(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "schema name required", http.StatusBadRequest)
		return
	}

	s, ok := h.registry.Get(name)
	if !ok {
		http.Error(w, "schema not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}
