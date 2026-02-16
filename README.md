# schemaflux

Structured data compiler. Part of the [MIST stack](https://github.com/greynewell/mist-go).

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Dependencies](https://img.shields.io/badge/Dependencies-0-brightgreen)](#)

```
markdown + frontmatter  ->  frontend  ->  12 passes  ->  backend  ->  output
```

Zero external deps. Single static binary.

## How it works

Entities (structured data with fields, taxonomies, relationships) go in. A config defines the schema. Passes resolve slugs, sort, enrich, group, score relationships, validate. Backends emit output from the finalized IR.

```
1,997 entities -> 2,328 pages in ~500ms
```

1. **Frontend** parses markdown + YAML frontmatter into IR
2. **Passes** transform IR: slugs, sorting, enrichment, taxonomy grouping, related scoring, graph enrichment, content analysis, URL resolution, schema generation, validation
3. **Backend** emits output. IR is immutable at this point.

The built-in HTML backend produces a complete static site: taxonomy pages, pagination, A-Z indices, search index, JSON-LD, Open Graph, sitemaps, RSS, `llms.txt`.

## Install

```bash
go install github.com/greynewell/schemaflux/cmd/schemaflux@latest
schemaflux build --config schemaflux.yaml
```

## Config

```yaml
site:
  name: "My Dataset"
  base_url: "https://example.com"
paths:
  content: "./content"
  output: "./output"
  templates: "./templates"
taxonomies:
  - name: category
    label: Categories
    field: category
templates:
  entity: entity.html
  homepage: index.html
```

## Architecture

```
compiler.Compile(cfg)
  -> frontend.Parse()        # markdown + YAML -> IR
  -> pass.Registry.RunAll()  # 12 ordered passes
  -> backend.Emit()          # IR -> output

internal/
  compiler/
    frontend/    Parse structured data into IR
    ir/          Program, ResolvedEntity, TaxonomyGroup
    pass/        12 passes with declared dependencies
    backend/     Pluggable output (html/ ships built-in)
  config/        YAML config types
  entity/        Untyped AST
  markdown/      Markdown-to-HTML renderer
  yaml/          YAML parser
```

## Badge

[![Compiled with SchemaFlux](https://img.shields.io/badge/compiled%20with-SchemaFlux-5B7B5E)](https://schemaflux.dev)

```markdown
[![Compiled with SchemaFlux](https://img.shields.io/badge/compiled%20with-SchemaFlux-5B7B5E)](https://schemaflux.dev)
```
