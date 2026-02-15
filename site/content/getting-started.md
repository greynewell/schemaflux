---
title: "Getting Started"
description: "Install SchemaFlux and build your first static site in minutes. Requires Go 1.25 or later."
section: "Getting Started"
tags:
  - "installation"
  - "quickstart"
  - "cli"
order: 2
author: "Grey Newell"
---

## Installation

SchemaFlux requires Go 1.25 or later. Install the CLI with:

```bash
go install github.com/greynewell/schemaflux/cmd/schemaflux@latest
```

This downloads, compiles, and places the `schemaflux` binary in your `$GOPATH/bin` directory.

## Quick Start

Create a project directory with the following structure:

```
my-site/
  schemaflux.yaml
  content/
    my-first-entity.md
  templates/
    entity.html
    index.html
    hub.html
    taxonomy_index.html
    _head.html
    _header.html
    _footer.html
    _styles.css
    _main.js
```

## Minimal Config

Create a `schemaflux.yaml` at the root of your project:

```yaml
site:
  name: "My Dataset"
  base_url: "https://example.com"

paths:
  data: "./content"
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

## Content Files

Each entity is a markdown file with YAML frontmatter:

```markdown
---
title: "My First Entity"
description: "A brief description of this entity."
category: "Tutorial"
---

## Hello World

This is the body content of your entity. It supports **bold**, *italic*, `code`, links, images, lists, tables, blockquotes, and fenced code blocks.
```

## Build

Run the compiler from your project directory:

```bash
schemaflux build --config schemaflux.yaml
```

The compiled site appears in the `output/` directory (or wherever `paths.output` points). You can serve it with any static file server:

```bash
cd output && python3 -m http.server 8000
```

## What Gets Generated

A full build produces:

- **Entity pages** -- one HTML page per markdown file
- **Homepage** -- the `index.html` landing page
- **Taxonomy index pages** -- one per taxonomy (e.g., `/category/`)
- **Taxonomy hub pages** -- one per taxonomy value (e.g., `/category/tutorial.html`)
- **Letter pages** -- A-Z navigation for large taxonomies
- **sitemap.xml** -- XML sitemap for search engines
- **robots.txt** -- crawler directives
- **manifest.json** -- PWA manifest
- **RSS feeds** -- main feed and per-category feeds
- **llms.txt** -- structured content listing for AI crawlers
- **CNAME** -- GitHub Pages custom domain file (if configured)
- **Search index** -- JSON file for client-side search (if enabled)
- **Share images** -- SVG social sharing images
