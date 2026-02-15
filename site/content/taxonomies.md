---
title: "Taxonomies"
description: "Taxonomies group entities by shared field values into browsable categories with automatic pagination, A-Z indices, and hub pages."
section: "Concepts"
tags:
  - "taxonomies"
  - "grouping"
  - "pagination"
order: 6
author: "Grey Newell"
---

## What Are Taxonomies?

Taxonomies are the primary organizational mechanism in SchemaFlux. A taxonomy groups entities by the value of a specific frontmatter field. For example, a "category" taxonomy groups entities by their `category` field, creating hub pages like `/category/tutorial.html`.

Each taxonomy automatically generates:

- A **taxonomy index page** listing all values (e.g., `/category/`)
- A **hub page** for each value with paginated entity listings (e.g., `/category/desserts.html`)
- **Letter pages** for A-Z navigation when entries exceed the threshold (e.g., `/category/letter-a.html`)

## Configuration

Taxonomies are declared in `schemaflux.yaml`:

```yaml
taxonomies:
  - name: "category"
    label: "Categories"
    label_singular: "Category"
    field: "category"
    multi_value: false
    min_entities: 1
    letter_page_threshold: 50
    hub_title: "{{.Name}} Recipes"
    hub_meta_description: "Browse {{.Name}} recipes."
    index_description: "Browse by category."
```

### Fields

| Field | Description |
|-------|-------------|
| `name` | URL-safe identifier used in paths (e.g., `/category/`) |
| `label` | Human-readable plural label |
| `label_singular` | Singular form of the label |
| `field` | Frontmatter field to group entities by |
| `multi_value` | If true, the field is a list and entities can belong to multiple values |
| `min_entities` | Minimum entities required for a hub page to be generated |
| `letter_page_threshold` | Number of entries at which A-Z letter navigation is enabled |

### Template Strings

Hub titles and descriptions support Go template syntax with the following variables:

- `{{.Name}}` -- the taxonomy value name (e.g., "Desserts")
- `{{.Count}}` -- number of entities in this group

## Single vs. Multi-Value

**Single-value** taxonomies (`multi_value: false`) expect the frontmatter field to contain a single string:

```yaml
category: "Tutorial"
```

**Multi-value** taxonomies (`multi_value: true`) expect the frontmatter field to contain a list:

```yaml
tags:
  - "go"
  - "static-site"
  - "compiler"
```

An entity with multiple values appears in each corresponding hub page.

## Pagination

Hub pages are paginated based on `pagination.entities_per_page`. Each hub page shows a slice of entities with prev/next navigation and numbered page links.

Page URLs follow the pattern:

- Page 1: `/category/desserts.html`
- Page 2: `/category/desserts-page-2.html`
- Page N: `/category/desserts-page-N.html`

## Letter Pages

When a taxonomy has more entries than `letter_page_threshold`, the taxonomy index page shows A-Z navigation instead of a flat list. Each letter gets its own page listing entries that start with that letter.

Letter page URLs: `/category/letter-a.html`, `/category/letter-b.html`, etc. Entries starting with numbers are grouped under a `#` page.

## Taxonomy Pass

The taxonomy grouping is handled by the `TaxonomyPass` in the compiler pipeline. It runs after slug resolution and sorting, and creates `TaxonomyGroup` entries in the IR. Each group contains:

- The `Taxonomy` metadata (name, label, entries with entity counts)
- A `ValidSlugs` map for fast slug lookups
- Computed letter groups for A-Z navigation

## Custom Templates

Each taxonomy can override the default templates:

```yaml
taxonomies:
  - name: "category"
    template: "category-hub.html"
    index_template: "category-index.html"
    letter_template: "category-letter.html"
```

If not specified, taxonomies fall back to `hub.html`, `taxonomy_index.html`, and `letter.html` respectively.
