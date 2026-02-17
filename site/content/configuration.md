---
title: Configuration
description: "Complete reference for the schemaflux.yaml configuration file."
section: "Getting Started"
tags:
  - config
  - yaml
order: 2
---

SchemaFlux is configured through a single YAML file, typically named `schemaflux.yaml`, located in the root of your project. This file controls every aspect of the build process, from site metadata to taxonomy definitions to structured data schemas.

## Site Configuration

The `site` section defines global metadata for your site. These values are available in templates and are used to generate meta tags, structured data, and feed metadata.

```yaml
site:
  name: "My Site"
  base_url: "https://example.com"
  description: "A brief description of the site"
  language: "en"
  author: "Your Name"
  author_url: "https://yoursite.com"
```

The `base_url` field is critical because it is used to generate absolute URLs in sitemaps, RSS feeds, canonical tags, and Open Graph metadata. Always use the full URL including the protocol, without a trailing slash.

## Paths

The `paths` section tells SchemaFlux where to find inputs and where to write outputs.

```yaml
paths:
  content: "content/"
  templates: "templates/"
  output: "dist/"
  static: "static/"
```

| Field | Description | Default |
|-------|-------------|---------|
| `content` | Directory containing markdown entity files | `content/` |
| `templates` | Directory containing Go HTML templates | `templates/` |
| `output` | Directory where the built site is written | `dist/` |
| `static` | Directory of static assets copied as-is | `static/` |

All paths are relative to the directory containing the configuration file. The output directory is completely replaced on each build, so do not store any hand-edited files there.

## Taxonomies

Taxonomies define how entities are grouped and categorized. Each taxonomy creates hub pages that list all terms, index pages for each term listing its entities, and optional letter-based indices for alphabetical browsing.

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

The `field` property maps to the frontmatter key in your entity files. By default, taxonomy fields are treated as multi-value (arrays), meaning an entity can belong to multiple terms. Set `multi_value: false` for single-value fields like "brand" or "author" where each entity has exactly one value.

## Templates

The `templates` section configures which template files are used for different page types.

```yaml
templates:
  entity: "entity.html"
  index: "index.html"
  hub: "hub.html"
  taxonomy_index: "taxonomy.html"
  letter: "letter.html"
  partials:
    - "_head.html"
    - "_header.html"
    - "_footer.html"
    - "_styles.html"
    - "_main.html"
```

Each template type serves a specific purpose. The `entity` template renders individual entity pages. The `index` template renders paginated listing pages. The `hub` template renders taxonomy hub pages showing all terms. The `taxonomy_index` template renders pages listing entities for a specific term. The `letter` template renders A-Z index pages.

## Structured Data

The `structured_data` section controls JSON-LD schema generation for your entities.

```yaml
structured_data:
  default_type: "Article"
  type_map:
    reviews: "Review"
    products: "Product"
    recipes: "Recipe"
  organization:
    name: "My Organization"
    url: "https://example.com"
    logo: "https://example.com/logo.png"
```

The `type_map` lets you assign different schema.org types based on taxonomy terms. If an entity belongs to the "reviews" category, it gets a `Review` schema. Otherwise, the `default_type` is used. The `organization` block is embedded in the schema as the publisher.

## Sitemap and Feeds

Configure sitemap and RSS feed generation:

```yaml
sitemap:
  enabled: true
  changefreq: "weekly"
  priority: 0.8

feeds:
  rss:
    enabled: true
    title: "My Site RSS Feed"
    limit: 50
```

The sitemap is generated as `sitemap.xml` in the output root, listing all entity pages and taxonomy pages with the configured change frequency and priority. RSS feeds are generated at `/feed.xml` and include the most recent entities up to the configured limit.

## SEO Configuration

The `seo` section provides additional control over search engine optimization features:

```yaml
seo:
  robots_txt: true
  llms_txt: true
  open_graph: true
  twitter_cards: true
  canonical_urls: true
```

When `llms_txt` is enabled, SchemaFlux generates an `llms.txt` file in the output root, providing a machine-readable summary of your site content for large language models.

## CTA Configuration

The `cta` section lets you define call-to-action elements that are injected into entity pages:

```yaml
cta:
  enabled: true
  text: "Check the latest price"
  position: "after_content"
```

This is useful for affiliate sites or product review sites where you want a consistent call-to-action across all entity pages.

## Full Example

Here is a complete configuration file showing all available options:

```yaml
site:
  name: "My Review Site"
  base_url: "https://reviews.example.com"
  description: "In-depth reviews and comparisons"
  language: "en"
  author: "Your Name"

paths:
  content: "content/"
  templates: "templates/"
  output: "dist/"
  static: "static/"

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

templates:
  entity: "entity.html"
  index: "index.html"
  hub: "hub.html"
  taxonomy_index: "taxonomy.html"
  letter: "letter.html"

structured_data:
  default_type: "Article"
  type_map:
    reviews: "Review"

sitemap:
  enabled: true

feeds:
  rss:
    enabled: true
    limit: 50

seo:
  robots_txt: true
  llms_txt: true
  open_graph: true
  canonical_urls: true
```
