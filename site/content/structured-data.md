---
title: Structured Data
description: "JSON-LD schemas, Open Graph, sitemaps, RSS feeds, and SEO features."
section: "Reference"
tags:
  - seo
  - json-ld
  - schema
order: 8
---

SchemaFlux automatically generates structured data and SEO metadata for every page it produces. This includes JSON-LD schemas embedded in page headers, Open Graph tags for social sharing, XML sitemaps for search engines, RSS feeds for subscribers, and specialized files like robots.txt and llms.txt.

For a live example of SchemaFlux-style structured data output, see [mcpbr.org](https://mcpbr.org) — a benchmark runner site with JSON-LD schemas on every page including SoftwareApplication, HowTo, FAQPage, and BreadcrumbList.

## JSON-LD Generation

JSON-LD (JavaScript Object Notation for Linked Data) is the structured data format recommended by Google for search engine optimization. SchemaFlux generates JSON-LD schema blocks for every entity page based on the `structured_data` configuration.

The `schema` compiler pass reads the entity metadata and produces a schema.org-compliant JSON-LD object. The schema type is determined by the `type_map` configuration, which maps taxonomy terms to schema types:

```yaml
structured_data:
  default_type: "Article"
  type_map:
    reviews: "Review"
    products: "Product"
    recipes: "Recipe"
  organization:
    name: "My Site"
    url: "https://example.com"
    logo: "https://example.com/logo.png"
```

For an entity in the "reviews" category, SchemaFlux generates a JSON-LD block like this:

```json
{
  "@context": "https://schema.org",
  "@type": "Review",
  "name": "Sony WH-1000XM5 Review",
  "description": "A comprehensive review of the Sony WH-1000XM5...",
  "url": "https://example.com/sony-wh-1000xm5-review/",
  "datePublished": "2025-06-15",
  "dateModified": "2025-07-01",
  "author": {
    "@type": "Person",
    "name": "Your Name"
  },
  "publisher": {
    "@type": "Organization",
    "name": "My Site",
    "url": "https://example.com",
    "logo": {
      "@type": "ImageObject",
      "url": "https://example.com/logo.png"
    }
  },
  "reviewRating": {
    "@type": "Rating",
    "ratingValue": "4.5",
    "bestRating": "5"
  }
}
```

The JSON-LD is embedded as a `<script type="application/ld+json">` tag in the HTML `<head>` element. You can output it in templates using the `jsonLD` function: `{{ "{{" }} .Entity | jsonLD {{ "}}" }}`.

## Supported Schema Types

SchemaFlux maps entity data to these schema.org types:

| Schema Type | Description | Key Properties |
|-------------|-------------|----------------|
| `Article` | Default for general content | headline, datePublished, author |
| `Review` | For review entities with ratings | reviewRating, itemReviewed |
| `Product` | For product entities | offers, brand, sku |
| `Recipe` | For recipe entities | prepTime, cookTime, ingredients |
| `HowTo` | For tutorial/guide entities | step, totalTime |
| `FAQPage` | For FAQ entities | mainEntity with Q&A pairs |

The schema pass automatically extracts the relevant properties from entity metadata. For example, a `Review` schema includes `reviewRating` built from the entity's `rating` field, and a `Product` schema includes `offers` built from the entity's `price` field.

## Open Graph Tags

Open Graph tags control how your pages appear when shared on social media platforms like Facebook, LinkedIn, and Twitter. SchemaFlux generates Open Graph meta tags for every entity page:

```html
<meta property="og:type" content="article">
<meta property="og:title" content="Sony WH-1000XM5 Review">
<meta property="og:description" content="A comprehensive review...">
<meta property="og:url" content="https://example.com/sony-wh-1000xm5-review/">
<meta property="og:site_name" content="My Site">
<meta property="og:image" content="https://example.com/images/sony-xm5.jpg">
```

Twitter Card tags are also generated for enhanced display on Twitter:

```html
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="Sony WH-1000XM5 Review">
<meta name="twitter:description" content="A comprehensive review...">
<meta name="twitter:image" content="https://example.com/images/sony-xm5.jpg">
```

The image tag is only included if the entity has an `image` field in its frontmatter. Open Graph and Twitter Card generation can be individually enabled or disabled in the `seo` configuration.

## Sitemaps

SchemaFlux generates an XML sitemap at `/sitemap.xml` listing all generated pages. The sitemap includes entity pages, taxonomy hub pages, taxonomy index pages, and the main index page.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/</loc>
    <changefreq>daily</changefreq>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>https://example.com/sony-wh-1000xm5-review/</loc>
    <lastmod>2025-07-01</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.8</priority>
  </url>
</urlset>
```

The sitemap uses the configured `changefreq` and `priority` values from the `sitemap` section of the configuration. If an entity has an `updated` date, it is used as the `lastmod` value.

## RSS Feeds

SchemaFlux generates an RSS 2.0 feed at `/feed.xml` containing the most recent entities. The feed includes entity titles, descriptions, links, and publication dates.

The number of entities included in the feed is controlled by the `feeds.rss.limit` configuration option (default: 50). Entities are included in date order with the most recent first.

RSS feeds allow users to subscribe to your site's content using feed readers and enable syndication by other platforms.

## robots.txt

When enabled in the configuration, SchemaFlux generates a `robots.txt` file that controls search engine crawler behavior:

```
User-agent: *
Allow: /

Sitemap: https://example.com/sitemap.xml
```

The default configuration allows all crawlers to access all pages and points them to the sitemap. You can customize the robots.txt content through the configuration if you need to disallow specific paths or user agents.

## llms.txt

SchemaFlux generates an `llms.txt` file at the site root, following the emerging convention for providing machine-readable site summaries to large language models. The file contains the site name, description, and a structured list of all content pages with their URLs and descriptions.

This file helps AI assistants understand the scope and content of your site, improving the quality of AI-generated responses that reference your content. The format is plain text with a simple structure that any text-processing system can parse.

## Canonical URLs

Every generated page includes a canonical URL meta tag in the `<head>`:

```html
<link rel="canonical" href="https://example.com/sony-wh-1000xm5-review/">
```

Canonical URLs prevent duplicate content issues when the same page is accessible through multiple URLs (for example, with and without trailing slashes, or through taxonomy listings). The canonical URL always points to the primary entity URL generated by the `url_resolve` pass.
