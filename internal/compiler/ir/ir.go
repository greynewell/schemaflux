package ir

import (
	"html/template"

	"github.com/greynewell/schemaflux/internal/affiliate"
	"github.com/greynewell/schemaflux/internal/config"
	"github.com/greynewell/schemaflux/internal/entity"
	"github.com/greynewell/schemaflux/internal/render"
	"github.com/greynewell/schemaflux/internal/taxonomy"
)

// Program is the complete intermediate representation produced by the compiler
// frontend and passes. Backends consume it read-only.
type Program struct {
	Config           *config.Config
	Site             config.SiteConfig
	Entities         []*ResolvedEntity          // Final sorted order
	EntityBySlug     map[string]*ResolvedEntity // O(1) lookup
	Taxonomies       []TaxonomyGroup            // Grouped + paginated
	Favorites        []*ResolvedEntity
	Contributors     map[string]interface{}
	Diagnostics      []Diagnostic
	TotalEntityCount int
	BuildDate        string
}

// ResolvedEntity wraps entity.Entity with all pre-computed data needed by backends.
// The Raw field preserves template compatibility — existing template functions that
// access .Entity.GetString still work.
type ResolvedEntity struct {
	Raw          *entity.Entity // Untyped AST (preserves template compat)
	Slug         string
	URL          string
	CanonicalURL string

	Pairings []*ResolvedEntity
	Related  []*ResolvedEntity

	Enrichment     map[string]interface{}
	AffiliateLinks []affiliate.Link

	JsonLD      string // Pre-marshaled <script> blocks
	OG          render.OGMeta
	Breadcrumbs []render.Breadcrumb
	ChartData   map[string]interface{}

	TOC         []render.TOCEntry
	ReadingTime int
	WordCount   int

	SourceCode string
	SourceLang string

	CookModePrompt string
	ShareImageSVG  string
	ShareImageURL  string // Deterministic: {baseURL}/images/share/{slug}.svg

	// EnrichedGraphData holds the graph_data JSON after cross-entity enrichment.
	// Passes read this instead of mutating Raw.Fields["graph_data"].
	EnrichedGraphData string

	TaxonomyMemberships map[string][]string // taxonomy name -> list of entry slugs

	// ValidSlugs maps taxonomy name -> set of entry slugs (for link validation in templates)
	ValidSlugs map[string]map[string]bool
}

// TaxonomyGroup holds a taxonomy with pre-computed pagination, chart data, and schema.
type TaxonomyGroup struct {
	Taxonomy     taxonomy.Taxonomy
	ValidSlugs   map[string]bool // slug -> true for entries in this taxonomy
	IndexJsonLD  string
	IndexOG      render.OGMeta
	IndexChartJS template.JS
	IndexSVG     string
	IndexURL     string
}

// Diagnostic records a validation warning or error.
type Diagnostic struct {
	Level   DiagLevel
	Message string
	Entity  string // slug, if entity-specific
}

// DiagLevel indicates the severity of a diagnostic.
type DiagLevel int

const (
	DiagWarning DiagLevel = iota
	DiagError
)

// String returns the level as a string.
func (d DiagLevel) String() string {
	switch d {
	case DiagWarning:
		return "warning"
	case DiagError:
		return "error"
	default:
		return "unknown"
	}
}

// NewProgram creates an empty Program with initialized maps.
func NewProgram(cfg *config.Config) *Program {
	return &Program{
		Config:       cfg,
		Site:         cfg.Site,
		EntityBySlug: make(map[string]*ResolvedEntity),
	}
}

// NewResolvedEntity wraps a raw entity into a ResolvedEntity.
func NewResolvedEntity(e *entity.Entity) *ResolvedEntity {
	return &ResolvedEntity{
		Raw:                 e,
		Slug:                e.Slug,
		TaxonomyMemberships: make(map[string][]string),
		ValidSlugs:          make(map[string]map[string]bool),
		ChartData:           make(map[string]interface{}),
	}
}

// EntityBySlugRaw returns the raw entity for a slug, or nil.
func (p *Program) EntityBySlugRaw(slug string) *entity.Entity {
	if re, ok := p.EntityBySlug[slug]; ok {
		return re.Raw
	}
	return nil
}

// RawEntities returns a slice of raw entity pointers in the same order as Entities.
// Useful for passing to library functions that expect []*entity.Entity.
func (p *Program) RawEntities() []*entity.Entity {
	result := make([]*entity.Entity, len(p.Entities))
	for i, re := range p.Entities {
		result[i] = re.Raw
	}
	return result
}

// RawEntitySlice converts a ResolvedEntity slice to a raw entity slice.
func RawEntitySlice(resolved []*ResolvedEntity) []*entity.Entity {
	result := make([]*entity.Entity, len(resolved))
	for i, re := range resolved {
		result[i] = re.Raw
	}
	return result
}

// AddDiagnostic appends a diagnostic to the program.
func (p *Program) AddDiagnostic(level DiagLevel, message, entitySlug string) {
	p.Diagnostics = append(p.Diagnostics, Diagnostic{
		Level:   level,
		Message: message,
		Entity:  entitySlug,
	})
}

// HasErrors returns true if there are any error-level diagnostics.
func (p *Program) HasErrors() bool {
	for _, d := range p.Diagnostics {
		if d.Level == DiagError {
			return true
		}
	}
	return false
}
