---
title: "Template Engine"
description: "SchemaFlux uses Go's html/template with 60+ custom functions for rendering entity pages, homepages, taxonomy hubs, and more."
section: "Reference"
tags:
  - "templates"
  - "rendering"
  - "reference"
order: 5
author: "Grey Newell"
---

## Overview

SchemaFlux uses Go's built-in `html/template` package for rendering. Templates are loaded from the directory specified by `paths.templates` in your config. Files with `.html`, `.css`, and `.js` extensions are loaded as named templates.

Partials are template files whose names start with `_` (e.g., `_head.html`, `_header.html`). They are defined using the `{{define "_name.html"}}...{{end}}` syntax and included via `{{template "_name.html"}}`.

## Template Types

### Entity Template

Renders individual entity pages. Receives an `EntityPageContext` with:

| Field | Type | Description |
|-------|------|-------------|
| `.Site` | SiteConfig | Site-level config (name, base URL, etc.) |
| `.Entity` | Entity | The entity data with fields and sections |
| `.Slug` | string | URL-safe entity identifier |
| `.URL` | string | Relative URL for this entity |
| `.CanonicalURL` | string | Full canonical URL |
| `.Breadcrumbs` | []Breadcrumb | Navigation breadcrumbs |
| `.Related` | []Entity | Related entities |
| `.JsonLD` | HTML | JSON-LD structured data script tag |
| `.OG` | OGMeta | Open Graph metadata |
| `.TOC` | []TOCEntry | Table of contents extracted from body |
| `.ReadingTime` | int | Estimated reading time in minutes |
| `.WordCount` | int | Word count of the body |
| `.Taxonomies` | []Taxonomy | All site taxonomies |
| `.Extra` | map | Extra config data |

### Homepage Template

Renders the site homepage. Receives a `HomepageContext` with:

| Field | Type | Description |
|-------|------|-------------|
| `.Site` | SiteConfig | Site-level config |
| `.Entities` | []Entity | All entities |
| `.Taxonomies` | []Taxonomy | All taxonomies with entries |
| `.Favorites` | []Entity | Favorited entities |
| `.EntityCount` | int | Total entity count |
| `.JsonLD` | HTML | JSON-LD structured data |
| `.OG` | OGMeta | Open Graph metadata |
| `.Extra` | map | Extra config data |

### Hub Template

Renders taxonomy hub pages (e.g., a specific category). Receives a `HubPageContext` with:

| Field | Type | Description |
|-------|------|-------------|
| `.Site` | SiteConfig | Site-level config |
| `.Taxonomy` | Taxonomy | The parent taxonomy |
| `.Entry` | Entry | The specific taxonomy value |
| `.Entities` | []Entity | Entities on this page |
| `.Pagination` | PaginationInfo | Page numbers, prev/next URLs |
| `.Breadcrumbs` | []Breadcrumb | Navigation breadcrumbs |
| `.JsonLD` | HTML | JSON-LD structured data |
| `.OG` | OGMeta | Open Graph metadata |

### Taxonomy Index Template

Renders the index page for a taxonomy (e.g., list of all categories). Receives a `TaxonomyIndexContext` with:

| Field | Type | Description |
|-------|------|-------------|
| `.Site` | SiteConfig | Site-level config |
| `.Taxonomy` | Taxonomy | The taxonomy |
| `.Entries` | []Entry | All taxonomy values |
| `.TopEntries` | []Entry | Top entries by entity count |
| `.HasLetters` | bool | Whether A-Z navigation is shown |
| `.Letters` | []string | Available letter links |
| `.LetterGroups` | []LetterGroup | Entries grouped by first letter |

## Template Functions

SchemaFlux registers 60+ custom template functions:

### String Functions

| Function | Description |
|----------|-------------|
| `slug` / `slugify` | Convert string to URL slug |
| `lower` | Lowercase |
| `upper` | Uppercase |
| `title` | Title case |
| `join` | Join slice with separator |
| `split` | Split string by separator |
| `replace` | Replace all occurrences |
| `contains` | Check substring |
| `hasPrefix` / `hasSuffix` | Prefix/suffix check |
| `trimSpace` | Trim whitespace |
| `truncate` | Truncate with ellipsis |
| `urlencode` | URL-encode a string |

### Number Functions

| Function | Description |
|----------|-------------|
| `formatNumber` | Add thousands separators (e.g., 1,234) |
| `add` / `sub` / `mul` / `div` / `mod` | Integer arithmetic |
| `addf` / `mulf` | Float arithmetic |

### Collection Functions

| Function | Description |
|----------|-------------|
| `first` / `last` | First/last element of a slice |
| `seq` | Generate integer sequence 1..n |
| `dict` | Create map from key/value pairs |
| `slice` | Slice a list by start/end index |
| `len` | Length of slice, map, or string |
| `sort` / `reverse` | Sort/reverse string slices |
| `min` / `max` | Integer min/max |

### Entity Functions

| Function | Description |
|----------|-------------|
| `field` | Access entity field by name |
| `section` | Access parsed body section |
| `getStringSlice` | Get string slice from entity field |
| `hasField` | Check if entity has a field |
| `getInt` / `getFloat` | Get typed field value |

### HTML/JSON Functions

| Function | Description |
|----------|-------------|
| `jsonMarshal` | Marshal value to JSON |
| `toJSON` | Convert to JSON string |
| `safeHTML` | Mark string as safe HTML |
| `safeJS` / `safeCSS` / `safeURL` | Context-safe wrappers |
| `noescape` | Skip HTML escaping |
| `markdown` | Render markdown to HTML |

### Comparison Functions

| Function | Description |
|----------|-------------|
| `eq` / `ne` | Equality/inequality |
| `lt` / `le` / `gt` / `ge` | Integer comparison |
| `default` | Default value if nil/empty |
| `ternary` | Conditional expression |
| `hasKey` | Check map key existence |

### Content Functions

| Function | Description |
|----------|-------------|
| `readingTime` | Estimated reading time in minutes |
| `wordCount` | Word count |
| `formatDuration` | ISO 8601 duration to human-readable |
| `durationMinutes` | ISO 8601 duration to minutes |

## Example

A minimal entity template:

```html
<!DOCTYPE html>
<html lang="{{.Site.Language}}">
<head>
{{template "_head.html"}}
<title>{{field .Entity "title"}} - {{.Site.Name}}</title>
</head>
<body>
{{template "_header.html"}}
<main>
  <h1>{{field .Entity "title"}}</h1>
  <p>{{field .Entity "description"}}</p>
  {{markdown (field .Entity "body")}}
</main>
{{template "_footer.html" .}}
</body>
</html>
```
