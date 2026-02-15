---
title: "CLI Usage"
description: "Reference for the schemaflux command-line interface. Build static sites from structured data with a single command."
section: "Reference"
tags:
  - "cli"
  - "reference"
  - "build"
order: 8
author: "Grey Newell"
---

## Overview

SchemaFlux provides a single command-line binary with a `build` subcommand. The CLI reads a YAML config file, runs the compiler pipeline, and writes the output to disk.

## Installation

```bash
go install github.com/greynewell/schemaflux/cmd/schemaflux@latest
```

This requires Go 1.25 or later. The binary is placed in `$GOPATH/bin/schemaflux`.

## Commands

### build

Compile a static site from structured data:

```bash
schemaflux build [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `schemaflux.yaml` | Path to the config file |
| `--force` | `false` | Force a full rebuild (ignore cache) |

### Examples

Build with the default config file in the current directory:

```bash
schemaflux build
```

Build with a specific config file:

```bash
schemaflux build --config path/to/schemaflux.yaml
```

Force a full rebuild:

```bash
schemaflux build --force
```

## Build Output

During a build, the compiler logs each stage:

```
Compiling site: My Site
Running 12 passes...
  SlugResolution       2ms
  Favorites            0ms
  Sort                 1ms
  Enrichment           5ms
  Affiliate            0ms
  Taxonomy             12ms
  RelatedEntities      8ms
  GraphEnrichment      3ms
  ContentAnalysis      15ms
  URLResolution        1ms
  Schema               22ms
  Validation           4ms
Emitting via HTML backend...
Loading templates from templates/...
Rendering 1997 entity pages...
Rendering taxonomy pages...
Rendering homepage...
Generating sitemap (2328 entries)...
Generated 1 sitemap file(s)
Generated 3 RSS feed(s)

Compile complete!
  Entities:   1997
  Taxonomies: 7
  Passes:     12
  Output:     ./output
  Duration:   487ms
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Build succeeded |
| 1 | Build failed (config error, parse error, template error, etc.) |

## Config Resolution

The `--config` path is resolved to an absolute path. All relative paths within the config file are resolved against the directory containing the config file, not the working directory.

For example, if you run:

```bash
schemaflux build --config sites/mysite/schemaflux.yaml
```

And the config contains `paths.data: "./content"`, the content directory resolves to `sites/mysite/content/`.

## Diagnostics

If the compiler encounters issues during parsing or validation, diagnostic messages are logged with severity levels:

- **error** -- fatal issues that prevent compilation
- **warning** -- non-fatal issues that may indicate problems

Diagnostics include source file positions when available:

```
[warning] content/broken-entity.md:5: missing required field "title"
```

## Development Workflow

For local development, build and serve with any static file server:

```bash
schemaflux build && cd output && python3 -m http.server 8000
```

Or use Go's built-in file server for a more robust option:

```bash
schemaflux build && go run -v net/http@latest -addr :8000 -dir output
```

## CI/CD

SchemaFlux works well in CI/CD pipelines. A typical GitHub Actions workflow:

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: '1.25'
- run: go install github.com/greynewell/schemaflux/cmd/schemaflux@latest
- run: schemaflux build --config site/schemaflux.yaml
```

The compiled output directory can then be deployed to GitHub Pages, Netlify, Cloudflare Pages, or any static hosting provider.
