# AGENTS.md

Guidelines for AI agents and contributors working on SchemaFlux.

## Hard Rules

1. **Zero external dependencies.** SchemaFlux uses only the Go standard library. Do not add any `require` directives to `go.mod`. This is enforced by a unit test (`TestZeroDependencies`). YAML parsing and markdown rendering are implemented internally.

2. **Test-first.** Write failing tests before implementing features. All new code must have unit tests.

3. **No generated code.** No code generators, no protobuf, no build-time codegen.

## Design Goals

- Single binary, zero config defaults, fast builds
- Every feature earns its complexity — if stdlib can do it, use stdlib
- Backward-compatible changes only; existing sites must not break

## Architecture

SchemaFlux is structured as a data-to-view compiler with frontend/middle-end/backend separation:

```
compiler.Compile(cfg)
  -> frontend.Parse()          # reads .md files, produces IR Program
  -> pass.Registry.RunAll()    # 11 ordered passes transform the IR
  -> backend.Emit()            # HTML backend maps IR to render contexts
```

**Compiler pipeline** (`internal/compiler/`):
- `frontend/` — parses markdown + YAML frontmatter into `[]*ir.ResolvedEntity`
- `ir/` — IR types: `Program`, `ResolvedEntity`, `TaxonomyGroup`, `Diagnostic`
- `pass/` — 11 ordered passes (slug resolution, sort, enrichment, affiliate, taxonomy, related, graph enrichment, content analysis, URL resolution, schema, validation)
- `backend/html/` — maps IR to render contexts, writes HTML files + sitemap/RSS/etc.

**Stable foundations** (unchanged):
- `config/` — config types + YAML loading
- `entity/` — untyped AST (`Entity` struct with map-based fields)
- `markdown/` — custom markdown-to-HTML renderer
- `yaml/` — custom zero-dependency YAML parser

**Utility libraries** (called by passes/backend):
- `enrichment/` — JSON cache reading
- `affiliate/` — affiliate link generation
- `taxonomy/` — grouping logic
- `render/` — template engine + context types
- `output/` — sitemap, RSS, robots.txt, manifest writers
- `schema/` — JSON-LD generator

Templates use Go's `html/template`. Config uses YAML (parsed by `internal/yaml`). Content uses markdown with YAML frontmatter.

The IR is immutable once passes complete — backends read but never modify.

## Implementation Order

When adding features:
1. Add config types to `internal/config/types.go`
2. Write tests for the new behavior
3. Implement as a new pass in `internal/compiler/pass/`
4. Register the pass in `internal/compiler/compiler.go`
5. Run `go test ./...` — all tests must pass
