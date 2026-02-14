# SchemaFlux

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Dependencies](https://img.shields.io/badge/Dependencies-0-brightgreen)](#)

**Unified Schema Transformation Pipeline | The Data-to-View Compiler**

A fast, zero-dependency static site generator and structured data transformation engine. SchemaFlux compiles schema-driven content into deterministic, type-safe views.

**Zero external dependencies** — SchemaFlux uses only the Go standard library.
- Single static binary, compiles anywhere Go does
- No supply-chain risk from third-party modules
- Smaller binary, faster builds

```
1,997 entities -> 2,328 pages in ~500ms
```

## Why

Most static site generators are built for blogs. SchemaFlux is built for **structured datasets** — collections of entities with rich metadata, taxonomies, and relationships. Feed it a directory of markdown files with frontmatter and a YAML config, and it generates a full site with:

- Taxonomy pages with automatic categorization and pagination
- A-Z letter indices
- Client-side search (generated at build time, zero server dependencies)
- D3.js chart data for visualizations
- SEO: sitemaps, robots.txt, JSON-LD, Open Graph, `llms.txt`
- RSS feeds
- Enrichment data via JSON sidecar files

## Quick Start

```bash
go install github.com/greynewell/schemaflux/cmd/schemaflux@latest

schemaflux build --config schemaflux.yaml
```

## Config

SchemaFlux is driven by a single `schemaflux.yaml`:

```yaml
site:
  name: "My Site"
  base_url: "https://example.com"

paths:
  content: "./content"
  output: "./output"
  templates: "./templates"

taxonomies:
  - name: category
    label: Categories
    field: category
    template: hub.html

templates:
  entity: entity.html
  homepage: index.html
```

## Architecture

```
cmd/schemaflux/    CLI entrypoint
internal/
  build/           Build pipeline orchestration
  config/          YAML config parsing
  entity/          Entity loading and metadata
  loader/          Markdown + frontmatter parsing
  render/          Template engine and page contexts
  taxonomy/        Taxonomy generation, pagination, A-Z indices
  enrichment/      JSON sidecar enrichment data
  affiliate/       Affiliate link matching
  output/          File writing
  schema/          JSON-LD structured data
```

~7,100 lines of Go. Zero external dependencies.
