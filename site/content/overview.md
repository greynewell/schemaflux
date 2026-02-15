---
title: "Overview"
description: "SchemaFlux is a compiler for structured data. It reads entities with metadata, enriches them through an ordered pass pipeline, and emits output through pluggable backends."
section: "Getting Started"
tags:
  - "architecture"
  - "compiler"
  - "overview"
order: 1
author: "Grey Newell"
---

## What is SchemaFlux?

SchemaFlux is a compiler for structured data. You define entities as markdown files with YAML frontmatter, declare a schema in a config file, and SchemaFlux handles the transformation into a complete static site.

```
markdown + frontmatter  ->  frontend  ->  12 passes  ->  backend  ->  output
```

Zero external dependencies. Single static binary. Go standard library only.

## How It Works

SchemaFlux operates on **entities** -- units of structured data with fields, taxonomies, and relationships. A config file defines the schema; a pipeline of passes resolves slugs, sorts, enriches, groups, computes relationships, and validates. Backends consume the resulting intermediate representation (IR) to produce output.

```
1,997 entities -> 2,328 pages in ~500ms
```

The compiler pipeline has three stages:

1. **Frontend** parses markdown files with YAML frontmatter into an intermediate representation
2. **Passes** transform the IR: slug resolution, sorting, enrichment, taxonomy grouping, related entity scoring, graph enrichment, content analysis, URL resolution, schema generation, validation
3. **Backend** emits output from the finalized IR

The IR is immutable once passes complete -- backends read but never modify.

## Key Features

- **Zero dependencies** -- only the Go standard library, no third-party packages
- **Single binary** -- compiles to a single static executable
- **Pluggable backends** -- the HTML backend ships built-in; others can be added
- **12-pass pipeline** -- ordered, dependency-aware transformations
- **Taxonomy system** -- automatic grouping, pagination, A-Z indices, letter pages
- **Structured data** -- JSON-LD, Open Graph, sitemaps, RSS feeds, robots.txt
- **Custom markdown renderer** -- headings, lists, tables, code blocks, inline formatting
- **Custom YAML parser** -- zero-dependency frontmatter and config parsing
- **Template engine** -- Go `html/template` with 60+ helper functions
- **Search index** -- JSON search index generation for client-side search
- **LLMs.txt** -- automatic generation of llms.txt for AI crawlers
- **Concurrent rendering** -- 32-goroutine pool for entity page rendering

## Architecture

```
compiler.Compile(cfg)
  -> frontend.Parse()        # markdown + YAML -> IR
  -> pass.Registry.RunAll()  # 12 ordered passes
  -> backend.Emit()          # IR -> output
```

The codebase is organized under `internal/`:

| Package | Purpose |
|---------|---------|
| `compiler/frontend/` | Parse structured data into IR |
| `compiler/ir/` | IR types: Program, ResolvedEntity, TaxonomyGroup |
| `compiler/pass/` | 12 passes with declared dependencies |
| `compiler/backend/html/` | Built-in HTML static site backend |
| `config/` | YAML config types and loading |
| `entity/` | Untyped AST (Entity struct with map-based fields) |
| `markdown/` | Custom markdown-to-HTML renderer |
| `yaml/` | Custom zero-dependency YAML parser |
| `render/` | Template engine and context types |
| `output/` | Sitemap, RSS, robots.txt, manifest writers |
| `schema/` | JSON-LD structured data generator |
| `taxonomy/` | Grouping, pagination, letter pages |

## Use Case: Static Sites

The built-in HTML backend compiles structured data into a complete static site with taxonomy pages, pagination, A-Z indices, search index, JSON-LD, Open Graph, sitemaps, RSS, and llms.txt.
