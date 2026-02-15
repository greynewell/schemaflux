# SchemaFlux

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Dependencies](https://img.shields.io/badge/Dependencies-0-brightgreen)](#)

**A compiler for structured data.**

SchemaFlux reads entities with metadata, enriches them through an ordered pass pipeline, and emits output through pluggable backends. You define the schema. SchemaFlux handles the transformation.

```
markdown + frontmatter  ->  frontend  ->  12 passes  ->  backend  ->  output
```

Zero external dependencies. Single static binary. Go standard library only.

## How it works

SchemaFlux operates on **entities** — units of structured data with fields, taxonomies, and relationships. A config file defines the schema; a pipeline of passes resolves slugs, sorts, enriches, groups, computes relationships, and validates. Backends consume the resulting IR to produce output.

```
1,997 entities -> 2,328 pages in ~500ms
```

The compiler pipeline:

1. **Frontend** parses markdown files with YAML frontmatter into an intermediate representation
2. **Passes** transform the IR: slug resolution, sorting, enrichment, taxonomy grouping, related entity scoring, graph enrichment, content analysis, URL resolution, schema generation, validation
3. **Backend** emits output from the finalized IR

The IR is immutable once passes complete — backends read but never modify.

## Use case: static sites

The built-in HTML backend compiles structured data into a complete static site with taxonomy pages, pagination, A-Z indices, search index, JSON-LD, Open Graph, sitemaps, RSS, and `llms.txt`.

## Quick start

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
