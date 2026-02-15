---
title: "Configuration Reference"
description: "Complete reference for the schemaflux.yaml configuration file. Covers site settings, paths, data schema, taxonomies, pagination, structured data, and output options."
section: "Reference"
tags:
  - "config"
  - "yaml"
  - "reference"
order: 3
author: "Grey Newell"
---

## Overview

SchemaFlux is configured via a single `schemaflux.yaml` file. The config is parsed by an internal zero-dependency YAML parser and validated at load time. Relative paths are resolved against the directory containing the config file.

## Site

Top-level metadata for the generated site:

```yaml
site:
  name: "My Site"           # Required. Site title.
  base_url: "https://..."   # Required. Canonical base URL (no trailing slash).
  description: "..."        # Site description for meta tags and feeds.
  language: "en"            # BCP-47 language code. Default: "en".
  version: "1.0.0"          # Version string displayed in footer/meta.
  author: "Name"            # Author name for structured data.
  author_url: "https://..." # Author URL for attribution links.
  license: "MIT"            # License identifier.
  cname: "example.com"      # Writes a CNAME file for GitHub Pages.
  repo_url: "https://..."   # Repository URL for source links.
```

## Paths

Directories for content, templates, and build output:

```yaml
paths:
  data: "./content"          # Required. Directory containing .md entity files.
  templates: "./templates"   # Template directory. Default: "templates".
  output: "./output"         # Build output directory. Default: "docs".
  cache: "./.cache"          # Cache directory. Default: ".cache".
  static: "./static"         # Static assets copied to output root.
  source_dir: "./src"        # Source code directory for code entity linking.
```

## Data

Schema for entity markdown files:

```yaml
data:
  format: "markdown"         # Content format. Default: "markdown".
  entity_type: "recipe"      # Label for entities (used in page titles).
  entity_slug:
    source: "filename"       # Slug source: "filename" or "field:<name>".
  body_sections:             # Named sections parsed from markdown body.
    - name: "ingredients"
      header: "Ingredients"
      type: "unordered_list"  # unordered_list, ordered_list, faq, markdown
    - name: "instructions"
      header: "Instructions"
      type: "ordered_list"
  fields:                     # Optional field schema for validation.
    - name: "title"
      type: "string"
      required: true
    - name: "status"
      type: "enum"
      allowed:
        - "draft"
        - "published"
      default: "published"
```

### Field Types

| Type | Description |
|------|-------------|
| `string` | Text value |
| `int` | Integer |
| `float` | Floating-point number |
| `bool` | Boolean (true/false) |
| `date` | Date string |
| `list` | Array of values |
| `enum` | Constrained to `allowed` values |

## Taxonomies

Define how entities are grouped into browsable categories:

```yaml
taxonomies:
  - name: "category"                    # URL-safe identifier.
    label: "Categories"                 # Display name (plural).
    label_singular: "Category"          # Singular form.
    field: "category"                   # Frontmatter field to group by.
    multi_value: false                  # True if field is a list.
    min_entities: 1                     # Min entities to show a hub page.
    letter_page_threshold: 50           # Entry count to enable A-Z pages.
    hub_title: "{{.Name}} Recipes"      # Go template for hub page title.
    hub_meta_description: "Browse..."   # Go template for hub meta desc.
    index_description: "Browse by..."   # Description on taxonomy index.
```

Each taxonomy generates:

- An **index page** listing all values
- A **hub page** for each value (with pagination)
- **Letter pages** if entries exceed `letter_page_threshold`

## Pagination

```yaml
pagination:
  entities_per_page: 48    # Entities per page on hub/listing pages.
```

## Structured Data

JSON-LD and Open Graph configuration:

```yaml
structured_data:
  entity_type: "Recipe"          # Schema.org type for entities.
  field_mappings:                 # Map Schema.org fields to frontmatter.
    name: "title"
    description: "description"
    author: "author"
  extra_keywords:
    - "keyword1"
  date_published: "2025-01-01"
  homepage_schemas:
    - "WebSite"
    - "ItemList"
  entity_schemas:
    - "Recipe"
    - "BreadcrumbList"
  hub_schemas:
    - "CollectionPage"
    - "BreadcrumbList"
  index_schemas:
    - "ItemList"
    - "BreadcrumbList"
```

## Templates

Map page types to template filenames in the templates directory:

```yaml
templates:
  entity: "entity.html"
  homepage: "index.html"
  hub: "hub.html"
  taxonomy_index: "taxonomy_index.html"
  letter: "letter.html"
  all_entities: "all.html"
  static_pages:
    "about.html": "about.html"
```

## Output

Build output options:

```yaml
output:
  clean_build: false       # Delete output dir before building.
  minify: false            # Minify HTML output.
  extract_css: "styles.css" # Extract _styles.css to a standalone file.
  extract_js: "main.js"    # Extract _main.js to a standalone file.
```

## Sitemap

```yaml
sitemap:
  max_urls_per_file: 50000
  priorities:
    homepage: "1.0"
    entity: "0.8"
    taxonomy_index: "0.7"
    hub_page_1: "0.6"
    hub_page_n: "0.4"
    letter_page: "0.5"
  change_freqs:
    homepage: "daily"
    entity: "weekly"
    taxonomy_index: "weekly"
    hub: "weekly"
    letter_page: "weekly"
```

## RSS

```yaml
rss:
  enabled: true
  main_feed: "feed.xml"
  category_feeds: true
  category_taxonomy: "category"
```

## Robots

```yaml
robots:
  allow_all: true
  extra_bots:
    - "GPTBot"
    - "ClaudeBot"
```

## LLMs.txt

```yaml
llms_txt:
  enabled: true
  tagline: "Site description for AI crawlers."
  taxonomies:
    - "category"
```

## Search

```yaml
search:
  enabled: true
  fields:
    - "title"
    - "description"
    - "tags"
```

## Sort

```yaml
sort:
  field: "title"      # Frontmatter field to sort entities by.
  order: "asc"        # "asc" or "desc".
```

## Related Entities

```yaml
related_entities:
  enabled: true
  max: 3              # Maximum related entities per page.
```

## Affiliates

```yaml
affiliates:
  providers:
    - name: "Amazon"
      url_template: "https://amazon.com/s?k={{term}}&tag={{tag}}"
      env_var: "AMAZON_AFFILIATE_TAG"
    - name: "Walmart"
      url_template: "https://walmart.com/search?q={{term}}"
      always_include: true
  search_term_paths:
    - "ingredients[].searchTerm"
```

## Extra

Arbitrary extra configuration accessible in templates via `.Extra`:

```yaml
extra:
  cta:
    enabled: true
    heading: "Get Started"
    description: "Try SchemaFlux today."
    button_text: "Install"
    button_url: "/getting-started.html"
  data:
    custom_key: "custom_value"
```
