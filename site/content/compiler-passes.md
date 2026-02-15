---
title: "Compiler Passes"
description: "SchemaFlux runs 12 ordered passes that transform the intermediate representation: slug resolution, sorting, enrichment, taxonomy grouping, related entity scoring, and more."
section: "Internals"
tags:
  - "compiler"
  - "passes"
  - "architecture"
order: 9
author: "Grey Newell"
---

## Overview

After the frontend parses markdown files into an intermediate representation (IR), SchemaFlux runs a pipeline of 12 ordered passes. Each pass transforms or enriches the IR. Passes may add data but must never remove data set by a previous pass.

Passes implement the `Pass` interface:

```go
type Pass interface {
    Name() string
    Run(p *ir.Program) error
}
```

Passes can optionally declare dependencies by implementing `DependencyAware`:

```go
type DependencyAware interface {
    Requires() []string
}
```

The registry validates ordering before execution begins -- if a pass's dependencies haven't run, the build fails.

## Pass Order

The passes execute in this fixed order:

| # | Pass | Description |
|---|------|-------------|
| 1 | SlugResolution | Generate URL-safe slugs from filenames or field values |
| 2 | Favorites | Resolve favorited entities from a JSON file |
| 3 | Sort | Order entities by configurable field and direction |
| 4 | Enrichment | Load external enrichment data from a JSON cache |
| 5 | Affiliate | Generate affiliate shopping links for configured providers |
| 6 | Taxonomy | Group entities by taxonomy fields, compute entries and letter groups |
| 7 | RelatedEntities | Score and rank related entities by shared taxonomy overlap |
| 8 | GraphEnrichment | Compute graph-level metadata (connections, clusters) |
| 9 | ContentAnalysis | Analyze body content: word count, reading time, TOC extraction |
| 10 | URLResolution | Compute canonical URLs, breadcrumbs, and relative paths |
| 11 | Schema | Generate JSON-LD structured data and Open Graph metadata |
| 12 | Validation | Validate required fields, field types, and enum constraints |

## Pass Details

### 1. SlugResolution

Generates a URL-safe slug for each entity. The slug source is configured by `data.entity_slug.source`:

- `"filename"` -- derive the slug from the markdown filename (default)
- `"field:<name>"` -- derive from a frontmatter field

### 2. Favorites

Reads a JSON file specified by `extra.favorites` and marks matching entities as favorites. These appear prominently on the homepage.

### 3. Sort

Sorts entities by a frontmatter field. Configured by `sort.field` and `sort.order` (asc/desc). Default sort is alphabetical by title.

### 4. Enrichment

Loads cached enrichment data from JSON files in the enrichment cache directory. This allows pre-computed data (e.g., AI-generated descriptions, nutritional info) to be merged into entity metadata without re-processing.

### 5. Affiliate

Generates affiliate links for configured providers. Scans entity fields at configured JSON paths (e.g., `ingredients[].searchTerm`) and builds URLs using provider templates with tag substitution from environment variables.

### 6. Taxonomy

The core grouping pass. For each configured taxonomy:

- Collects entity field values
- Groups entities into taxonomy entries
- Filters entries below `min_entities`
- Sorts entries alphabetically
- Computes letter groups for A-Z navigation
- Builds valid slug maps for fast lookup

### 7. RelatedEntities

When `related_entities.enabled` is true, this pass computes related entities for each entity by scoring shared taxonomy membership. Entities that share more taxonomy values rank higher. The top N (default 3) are stored on the IR.

### 8. GraphEnrichment

Computes graph-level metadata: entity connections through shared taxonomy values, hierarchical relationships (domain/subdomain), and cluster membership. This data powers architecture visualizations on the homepage.

### 9. ContentAnalysis

Analyzes entity body content:

- Word count
- Reading time estimate (200 WPM)
- Table of contents extraction from headings
- Source code extraction (for code entity types)

### 10. URLResolution

Computes canonical URLs, relative URLs, and breadcrumb navigation for each entity. URLs are based on the entity slug and the site's base URL.

### 11. Schema

Generates JSON-LD structured data and Open Graph metadata for each entity. Uses the configured `structured_data.field_mappings` to map frontmatter fields to Schema.org properties. Also generates share image SVGs.

### 12. Validation

The final pass validates all entities against the declared field schema:

- Required fields must be present
- Field values must match declared types
- Enum fields must use allowed values
- Diagnostics are emitted for validation failures

## IR Immutability

After all passes complete, the IR is considered frozen. The HTML backend reads the IR to generate output but never modifies it. This separation ensures backends can be swapped without affecting the transformation logic.

## Timing

Each pass reports its execution duration. The compiler logs these timings at the end of a build, making it easy to identify performance bottlenecks.
