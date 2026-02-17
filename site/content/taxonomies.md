---
title: Taxonomies
description: "Grouping entities with taxonomies: categories, tags, and custom classification systems."
section: "Reference"
tags:
  - taxonomies
  - categories
order: 7
---

Taxonomies in SchemaFlux are classification systems that group entities by shared attributes. Every taxonomy generates its own set of pages: a hub page listing all terms, individual index pages for each term, and optional A-Z letter pages for alphabetical browsing.

## What Are Taxonomies?

A taxonomy is a named grouping system that maps frontmatter field values to sets of entities. The most common examples are categories and tags, but you can define any number of custom taxonomies for your domain.

When you assign taxonomy values to an entity through its frontmatter, SchemaFlux automatically groups that entity with all other entities sharing the same values. These groupings drive the generation of listing pages, navigation structures, and relationship scoring.

For a site reviewing consumer electronics, you might define taxonomies for categories (headphones, speakers, cameras), brands (Sony, Bose, Apple), and price ranges (budget, mid-range, premium). Each of these taxonomies generates its own navigation hierarchy.

## Configuration

Taxonomies are defined in the `taxonomies` section of `schemaflux.yaml`:

```yaml
taxonomies:
  - name: "categories"
    field: "categories"
    plural: "categories"
    singular: "category"
  - name: "tags"
    field: "tags"
    plural: "tags"
    singular: "tag"
  - name: "brands"
    field: "brand"
    plural: "brands"
    singular: "brand"
    multi_value: false
```

Each taxonomy definition requires these fields:

| Field | Description |
|-------|-------------|
| `name` | Internal name used to reference the taxonomy |
| `field` | The frontmatter field name to read values from |
| `plural` | Plural label used in URLs and page titles |
| `singular` | Singular label used in breadcrumbs and descriptions |
| `multi_value` | Whether the field is an array (default: true) |

The `name` is used internally and in URLs. The `field` maps to the frontmatter key in your entity files. The `plural` and `singular` forms are used in generated page titles and navigation labels.

## Multi-Value vs Single-Value

By default, taxonomy fields are treated as multi-value, meaning they expect an array of values in the frontmatter. An entity can belong to multiple categories or have multiple tags:

```yaml
---
title: "Sony WH-1000XM5 Review"
categories:
  - "reviews"
  - "headphones"
tags:
  - "wireless"
  - "noise-cancelling"
  - "bluetooth"
---
```

For single-value fields, set `multi_value: false` in the taxonomy configuration. This is appropriate for fields like "brand" or "author" where each entity has exactly one value:

```yaml
---
title: "Sony WH-1000XM5 Review"
brand: "Sony"
---
```

The difference affects how the taxonomy pass processes the field. Multi-value fields are iterated as arrays, with the entity added to the index for each value. Single-value fields are read as a single string.

## Generated Pages

Each taxonomy generates three types of pages:

### Hub Pages

The hub page is the top-level page for a taxonomy, listing all terms with their entity counts. For a "categories" taxonomy, the hub page lives at `/categories/` and shows all categories like "reviews (45 entities)", "guides (12 entities)", etc.

Hub pages are rendered using the `hub` template and provide a navigational entry point into each taxonomy.

### Index Pages

Each taxonomy term gets its own index page listing all entities belonging to that term. For the "reviews" category, the index page lives at `/categories/reviews/` and lists all review entities in sort order.

Index pages support pagination when the number of entities exceeds the configured page size. Paginated pages use URLs like `/categories/reviews/page/2/`.

### Letter Pages

Letter pages provide A-Z alphabetical navigation within a taxonomy. For the "categories" taxonomy, letter pages live at `/categories/letter/a/`, `/categories/letter/b/`, etc. Each page lists all terms starting with that letter.

Letter pages are especially useful for taxonomies with many terms, like tags or brands, where alphabetical browsing is more practical than scrolling through a long list.

## Taxonomy Terms in Templates

In entity templates, you can access an entity's taxonomy values to render tag lists, category badges, or breadcrumb navigation:

```html
{{ "{{" }} if .Entity.Categories {{ "}}" }}
<div class="categories">
  {{ "{{" }} range .Entity.Categories {{ "}}" }}
    <a href="/categories/{{ "{{" }} . | slugify {{ "}}" }}/">{{ "{{" }} . {{ "}}" }}</a>
  {{ "{{" }} end {{ "}}" }}
</div>
{{ "{{" }} end {{ "}}" }}
```

In hub and taxonomy index templates, you access the full taxonomy data:

```html
<h1>{{ "{{" }} .Taxonomy.Plural {{ "}}" }}</h1>
{{ "{{" }} range .Taxonomy.Terms {{ "}}" }}
  <a href="{{ "{{" }} .URL {{ "}}" }}">{{ "{{" }} .Name {{ "}}" }} ({{ "{{" }} .Count {{ "}}" }})</a>
{{ "{{" }} end {{ "}}" }}
```

## Taxonomies and Relationships

Taxonomies are the foundation of the relationship system in SchemaFlux. The `related` pass uses shared taxonomy terms to compute relatedness scores between entities. Two entities sharing a category are considered more related than two entities sharing only a tag, because categories typically represent broader topical groupings.

The relationship weights can be configured so that different taxonomies contribute differently to relatedness scores. This allows you to tune the "Related Articles" sections on your entity pages to surface the most relevant content.

## Best Practices

Define taxonomies that reflect how your audience thinks about your content. Categories should be broad, mutually exclusive groupings that help users navigate to content areas. Tags should be specific, descriptive labels that help users find content on narrow topics. Custom taxonomies like brands or formats should map to real-world classification schemes that users already understand.

Keep taxonomy term names consistent. Use lowercase, avoid abbreviations, and prefer plural forms for multi-entity groupings. SchemaFlux normalizes terms to lowercase during processing, but consistent naming in your source files makes the content easier to maintain.
