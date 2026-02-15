---
title: "Structured Data and SEO"
description: "SchemaFlux generates JSON-LD, Open Graph tags, sitemaps, RSS feeds, robots.txt, and llms.txt for comprehensive search engine optimization."
section: "Concepts"
tags:
  - "seo"
  - "json-ld"
  - "structured-data"
  - "sitemap"
order: 7
author: "Grey Newell"
---

## Overview

SchemaFlux generates a comprehensive suite of SEO artifacts as part of every build. These are produced by the compiler pipeline passes and the HTML backend, ensuring every page has proper metadata for search engines, social media crawlers, and AI systems.

## JSON-LD

Every page includes JSON-LD structured data in a `<script type="application/ld+json">` tag. The schema types are configurable per page type:

### Homepage Schemas

Default: `WebSite` + `ItemList`

The WebSite schema includes the site name, URL, description, and a social sharing image. The ItemList schema lists featured or recent entities.

### Entity Schemas

Default: configurable (e.g., `Recipe`, `TechArticle`, `Article`)

Entity schemas map frontmatter fields to Schema.org properties via `field_mappings`:

```yaml
structured_data:
  entity_type: "Recipe"
  field_mappings:
    name: "title"
    description: "description"
    author: "author"
    prepTime: "prep_time"
    cookTime: "cook_time"
```

### Hub Page Schemas

Default: `CollectionPage` + `BreadcrumbList`

Hub pages include a collection schema listing all entities in the group, plus breadcrumb navigation.

### Index Page Schemas

Default: `ItemList` + `BreadcrumbList`

Taxonomy index pages list all taxonomy values as an ItemList.

## Open Graph

Every page includes Open Graph meta tags for social media sharing:

```html
<meta property="og:title" content="...">
<meta property="og:description" content="...">
<meta property="og:url" content="...">
<meta property="og:type" content="...">
<meta property="og:site_name" content="...">
<meta property="og:image" content="...">
```

The `og:type` is set to `website` for the homepage and `article` for all other pages. Share images are generated as SVG files in `/images/share/`.

## Sitemaps

SchemaFlux generates XML sitemaps compliant with the sitemap protocol. Every page in the site is included with configurable priority and change frequency values:

```xml
<url>
  <loc>https://example.com/entity.html</loc>
  <lastmod>2025-02-14</lastmod>
  <changefreq>weekly</changefreq>
  <priority>0.8</priority>
</url>
```

Large sites automatically split into multiple sitemap files with a sitemap index, controlled by `sitemap.max_urls_per_file`.

## RSS Feeds

When RSS is enabled, SchemaFlux generates:

- A **main feed** (`feed.xml`) containing all entities
- **Category feeds** (one per taxonomy value) when `category_feeds` is true

Feeds include entity titles, descriptions, URLs, and publication dates.

## Robots.txt

SchemaFlux generates a `robots.txt` file. When `allow_all` is true, all crawlers are permitted:

```
User-agent: *
Allow: /
Sitemap: https://example.com/sitemap.xml
```

Additional bot directives can be added via `extra_bots` to explicitly allow AI crawlers like GPTBot, ClaudeBot, and PerplexityBot.

## LLMs.txt

When `llms_txt.enabled` is true, SchemaFlux generates an `llms.txt` file following the emerging standard for AI-readable site summaries. The file includes:

- Site name and tagline
- A structured listing of all entities grouped by taxonomy
- Direct URLs to entity pages

This helps AI systems understand the site content without crawling every page.

## Manifest.json

A PWA-compatible `manifest.json` is generated with the site name, description, and basic configuration.

## CNAME

When `site.cname` is set, a `CNAME` file is written to the output directory for GitHub Pages custom domain configuration.

## Search Index

When `search.enabled` is true, a `search-index.json` file is generated containing a compact JSON array of entity metadata (title, description, slug, and configured fields). This powers client-side search functionality.

## Share Images

The HTML backend generates SVG share images for:

- The homepage
- Each taxonomy hub page
- Each taxonomy index page
- Each letter page
- Each entity page (when applicable)

These are written to `/images/share/` and referenced in Open Graph `og:image` tags.
