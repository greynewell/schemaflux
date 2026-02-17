---
title: Architecture
description: "How SchemaFlux works internally: frontend, compiler passes, IR, and backends."
section: "Core Concepts"
tags:
  - architecture
  - compiler
order: 3
---

SchemaFlux is structured as a compiler. It reads source files through a frontend, transforms them through a series of ordered passes, and emits output through a backend. This design makes each stage independently testable and allows the system to be extended with new passes or backends without modifying existing code.

## Pipeline Overview

The build pipeline follows a strict linear flow. Data enters through the frontend, passes through each compiler pass in order, and exits through the backend:

```
  ┌─────────┐    ┌──────────────────────────────────┐    ┌─────────┐
  │         │    │         Compiler Passes           │    │         │
  │ Frontend│───▶│                                   │───▶│ Backend │
  │         │    │  slug → sort → enrich → taxonomy  │    │         │
  │ Markdown│    │  related → graph → content_analysis│   │  HTML   │
  │   +     │    │  url → schema → favorites         │    │  Pages  │
  │ YAML FM │    │  affiliate → validate             │    │         │
  └─────────┘    └──────────────────────────────────┘    └─────────┘
```

This architecture is intentionally simple. There are no plugins to install, no middleware chains, no event systems. Each pass receives the full intermediate representation, performs its transformation, and returns the modified result.

## Frontend

The frontend is responsible for reading source files from disk and parsing them into a structured intermediate representation. SchemaFlux includes one built-in frontend that reads markdown files with YAML frontmatter.

The frontend performs the following steps for each file in the content directory:

1. Read the file from disk
2. Split the file into frontmatter and body sections
3. Parse the YAML frontmatter into a metadata map
4. Parse the markdown body into HTML
5. Construct an Entity struct with metadata and rendered body

The frontend does not perform any enrichment or transformation. Its only job is to produce a clean, structured representation of the source content. All enrichment happens in the compiler passes.

## Intermediate Representation

The intermediate representation (IR) is the central data structure in SchemaFlux. It holds the complete state of the build, including all entities, taxonomy indices, graph relationships, and computed metadata. Every compiler pass reads from and writes to the IR.

The IR contains:

- **Entities**: The list of all parsed entities with their metadata, body content, and computed fields
- **Taxonomy indices**: Maps from taxonomy terms to lists of entity references
- **Graph data**: Relationship edges between entities based on shared taxonomy terms
- **Site configuration**: The parsed configuration from schemaflux.yaml
- **Computed metadata**: Slugs, URLs, JSON-LD schemas, content analysis results

After all passes have completed, the IR becomes effectively immutable. The backend reads from it but does not modify it. This separation ensures that the backend can safely generate output without worrying about data consistency.

## Compiler Passes

The compiler runs 12 passes in a fixed order. Each pass implements a simple interface: it receives the IR, performs its transformation, and returns the modified IR. Passes are not independently configurable beyond what the site configuration provides.

The fixed ordering is critical. Each pass depends on the output of previous passes. For example, the `url_resolve` pass depends on `slug_resolve` having already generated slugs, and the `schema` pass depends on `url_resolve` having generated absolute URLs. Changing the pass order would break the build.

See the [Passes](/docs/passes/) documentation for a detailed description of each pass.

## Backend

The backend receives the completed IR and emits output files. SchemaFlux includes one built-in backend: the HTML backend, which generates a complete static site.

The HTML backend produces:

- **Entity pages**: One HTML page per entity, rendered using the entity template
- **Index pages**: Paginated listing pages showing all entities
- **Taxonomy hub pages**: Pages listing all terms for each taxonomy
- **Taxonomy index pages**: Pages listing entities for a specific taxonomy term
- **Letter index pages**: A-Z alphabetical index pages
- **Sitemap**: XML sitemap listing all generated pages
- **RSS feed**: RSS 2.0 feed of recent entities
- **robots.txt**: Search engine directives
- **llms.txt**: LLM-friendly site summary
- **JSON-LD**: Structured data embedded in each entity page
- **Open Graph tags**: Social sharing metadata in each page

The backend writes all output to the configured output directory. Static assets from the static directory are copied as-is.

## Package Structure

SchemaFlux is organized into Go packages that mirror the pipeline architecture:

```
cmd/schemaflux/     CLI entry point
internal/
  config/           YAML configuration parsing
  frontend/         Markdown + frontmatter parsing
  ir/               Intermediate representation types
  passes/           All 12 compiler passes
    slug_resolve/
    sort/
    enrichment/
    taxonomy/
    related/
    graph_enrich/
    content_analysis/
    url_resolve/
    schema/
    favorites/
    affiliate/
    validate/
  backend/          Output backends
    html/
  pipeline/         Pass orchestration
```

Every package uses only the Go standard library. There are no external dependencies anywhere in the codebase. This is a deliberate design choice that keeps the binary small, eliminates supply chain risks, and ensures long-term stability.

## Performance

SchemaFlux is designed for fast builds. The entire pipeline runs in a single process with no disk I/O between passes. Entities are stored in memory as slices, and taxonomy indices are hash maps. The benchmark for a real-world site with 1,997 entities produces 2,328 output pages in approximately 500 milliseconds on commodity hardware.

The performance comes from the simplicity of the architecture. There is no template compilation step, no dependency graph resolution, no incremental build system. Every build processes all entities from scratch. For sites with fewer than 10,000 entities, this brute-force approach is faster than the overhead of incremental build tracking.
