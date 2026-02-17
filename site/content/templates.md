---
title: Templates
description: "Go template syntax, available functions, and template types in SchemaFlux."
section: "Reference"
tags:
  - templates
  - html
order: 6
---

SchemaFlux uses Go's standard `html/template` package for rendering HTML output. Templates are plain HTML files with Go template directives embedded in double curly braces. This approach provides automatic HTML escaping, strong typing, and the full power of Go's template language without any external dependencies.

## Template Syntax

Go templates use `{{ "{{" }}` and `{{ "}}" }}` delimiters for dynamic content. Inside these delimiters you can access variables, call functions, and use control structures.

### Accessing Data

Templates receive a context object with the current entity, site configuration, and computed data:

```html
<h1>{{ "{{" }} .Entity.Title {{ "}}" }}</h1>
<p>{{ "{{" }} .Entity.Description {{ "}}" }}</p>
<div>{{ "{{" }} .Entity.Body {{ "}}" }}</div>
<time>{{ "{{" }} .Entity.Date {{ "}}" }}</time>
```

### Conditionals

```html
{{ "{{" }} if .Entity.Image {{ "}}" }}
  <img src="{{ "{{" }} .Entity.Image {{ "}}" }}" alt="{{ "{{" }} .Entity.Title {{ "}}" }}">
{{ "{{" }} end {{ "}}" }}

{{ "{{" }} if .Entity.Rating {{ "}}" }}
  <span class="rating">{{ "{{" }} .Entity.Rating {{ "}}" }} / 5</span>
{{ "{{" }} else {{ "}}" }}
  <span class="rating">Not rated</span>
{{ "{{" }} end {{ "}}" }}
```

### Loops

```html
<ul>
{{ "{{" }} range .Entity.Tags {{ "}}" }}
  <li>{{ "{{" }} . {{ "}}" }}</li>
{{ "{{" }} end {{ "}}" }}
</ul>

{{ "{{" }} range .Related {{ "}}" }}
  <a href="{{ "{{" }} .URL {{ "}}" }}">{{ "{{" }} .Title {{ "}}" }}</a>
{{ "{{" }} end {{ "}}" }}
```

### Template Inclusion

Templates can include other templates (partials) using the `template` directive:

```html
{{ "{{" }} template "_head" . {{ "}}" }}
{{ "{{" }} template "_header" . {{ "}}" }}
{{ "{{" }} template "_footer" . {{ "}}" }}
```

The dot `.` passes the current context to the included template. Always pass context to partials so they can access the same data.

## Available Template Functions

SchemaFlux provides several built-in template functions in addition to Go's default template functions:

| Function | Description | Example |
|----------|-------------|---------|
| `safeHTML` | Marks a string as safe HTML (no escaping) | `{{ "{{" }} .Entity.Body \| safeHTML {{ "}}" }}` |
| `lower` | Converts string to lowercase | `{{ "{{" }} .Entity.Title \| lower {{ "}}" }}` |
| `upper` | Converts string to uppercase | `{{ "{{" }} .Entity.Title \| upper {{ "}}" }}` |
| `truncate` | Truncates string to N characters | `{{ "{{" }} .Entity.Description \| truncate 150 {{ "}}" }}` |
| `dateFormat` | Formats a date string | `{{ "{{" }} .Entity.Date \| dateFormat "Jan 2, 2006" {{ "}}" }}` |
| `slugify` | Converts string to URL slug | `{{ "{{" }} .Entity.Title \| slugify {{ "}}" }}` |
| `jsonLD` | Outputs the entity JSON-LD script tag | `{{ "{{" }} .Entity \| jsonLD {{ "}}" }}` |
| `openGraph` | Outputs Open Graph meta tags | `{{ "{{" }} .Entity \| openGraph {{ "}}" }}` |
| `add` | Adds two numbers | `{{ "{{" }} add .Page 1 {{ "}}" }}` |
| `sub` | Subtracts two numbers | `{{ "{{" }} sub .Total 1 {{ "}}" }}` |
| `seq` | Generates a number sequence | `{{ "{{" }} range seq 1 .TotalPages {{ "}}" }}` |

## Template Types

SchemaFlux uses five distinct template types, each receiving different context data.

### Entity Template

The entity template renders individual entity pages. It receives the richest context, including the full entity data, related entities, and taxonomy information.

```html
<!DOCTYPE html>
<html lang="{{ "{{" }} .Site.Language {{ "}}" }}">
<head>
  <title>{{ "{{" }} .Entity.Title {{ "}}" }} - {{ "{{" }} .Site.Name {{ "}}" }}</title>
  {{ "{{" }} template "_head" . {{ "}}" }}
</head>
<body>
  {{ "{{" }} template "_header" . {{ "}}" }}
  <main>
    <article>
      <h1>{{ "{{" }} .Entity.Title {{ "}}" }}</h1>
      <p class="meta">{{ "{{" }} .Entity.Date {{ "}}" }} &middot; {{ "{{" }} .Entity.ReadingTime {{ "}}" }} min read</p>
      {{ "{{" }} .Entity.Body \| safeHTML {{ "}}" }}
    </article>
    {{ "{{" }} if .Related {{ "}}" }}
    <aside>
      <h2>Related</h2>
      {{ "{{" }} range .Related {{ "}}" }}
        <a href="{{ "{{" }} .URL {{ "}}" }}">{{ "{{" }} .Title {{ "}}" }}</a>
      {{ "{{" }} end {{ "}}" }}
    </aside>
    {{ "{{" }} end {{ "}}" }}
  </main>
  {{ "{{" }} template "_footer" . {{ "}}" }}
</body>
</html>
```

### Index Template

The index template renders paginated listing pages showing all entities. It receives pagination data including page number, total pages, and the slice of entities for the current page.

### Hub Template

The hub template renders taxonomy hub pages that list all terms for a taxonomy. For example, a categories hub page lists all categories with entity counts. It receives the taxonomy name and a map of terms to entity counts.

### Taxonomy Index Template

The taxonomy index template renders pages for a specific taxonomy term, listing all entities belonging to that term. It receives the taxonomy name, term name, and the list of matching entities.

### Letter Template

The letter template renders A-Z index pages that list entities starting with a specific letter. It receives the letter and the filtered list of entities.

## Partials

Partials are reusable template fragments that are included by other templates. They follow the convention of starting with an underscore.

### _head

The `_head` partial outputs the contents of the HTML `<head>` element, including meta tags, canonical URL, Open Graph tags, Twitter card tags, JSON-LD structured data, and CSS references.

### _header

The `_header` partial renders the site navigation header with the site name, main navigation links, and optional search functionality.

### _footer

The `_footer` partial renders the site footer with copyright information, navigation links, and any configured footer content.

### _styles

The `_styles` partial outputs the CSS styles for the site. SchemaFlux includes a default stylesheet that provides responsive layout, typography, and dark/light mode support.

### _main

The `_main` partial provides a reusable main content wrapper with consistent layout and spacing. It is used by index and hub templates to maintain layout consistency.

## Custom Templates

You can customize any template by creating your own version in the templates directory. SchemaFlux will use your template instead of the built-in default. The template receives the same context data regardless of whether it is a built-in or custom template, so you have full control over the HTML output while still accessing all computed entity data, taxonomy indices, and site configuration.
