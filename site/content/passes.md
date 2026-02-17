---
title: Compiler Passes
description: "The 12 ordered compiler passes that transform entities from raw data to enriched output."
section: "Core Concepts"
tags:
  - passes
  - pipeline
order: 5
---

SchemaFlux processes entities through a fixed sequence of 12 compiler passes. Each pass reads from the intermediate representation (IR), performs a specific transformation, and writes the results back. The passes run in a strict order because later passes depend on the output of earlier ones.

## Pass Order

The passes execute in this exact sequence on every build:

```
 1. slug_resolve      Generate URL slugs from titles
 2. sort              Order entities by configured criteria
 3. enrichment        Enrich entity metadata
 4. taxonomy          Build taxonomy indices
 5. related           Score related entities
 6. graph_enrich      Build relationship graph
 7. content_analysis  Analyze content metrics
 8. url_resolve       Resolve absolute URLs
 9. schema            Generate JSON-LD schemas
10. favorites         Process user favorites
11. affiliate         Process affiliate links
12. validate          Validate output completeness
```

This order is not configurable. Changing it would break the dependency chain between passes. For example, `url_resolve` depends on slugs from `slug_resolve`, and `schema` depends on absolute URLs from `url_resolve`.

## 1. SlugResolve

The `slug_resolve` pass generates URL-safe slugs for every entity. It takes the entity title, converts it to lowercase, replaces spaces and special characters with hyphens, and strips any remaining non-alphanumeric characters.

For example, the title "Best Wireless Headphones (2025 Edition)" becomes the slug `best-wireless-headphones-2025-edition`. If a slug collision is detected (two entities producing the same slug), a numeric suffix is appended to make it unique.

The generated slug is stored in the entity metadata and used by all subsequent passes and templates to construct URLs.

## 2. Sort

The `sort` pass orders all entities according to the configured sort criteria. By default, entities are sorted by date in descending order, placing the most recent entities first. This affects the order of entities on index pages, in RSS feeds, and in taxonomy listings.

The sort is stable, meaning entities with the same sort key retain their original relative order. This pass modifies the entity slice in the IR in place.

## 3. Enrichment

The `enrichment` pass adds computed metadata to each entity. This includes normalizing date formats, computing word counts, estimating reading time, extracting excerpt text from the body, and setting default values for any missing standard fields.

For example, if an entity has a body of 1,200 words, the enrichment pass sets `word_count: 1200` and `reading_time: 6` (assuming 200 words per minute). If the entity has no explicit description, the pass extracts the first paragraph as an auto-generated description.

This pass ensures that every entity has a consistent, complete set of metadata fields that templates and later passes can rely on.

## 4. Taxonomy

The `taxonomy` pass reads taxonomy field values from each entity and builds taxonomy indices. For each taxonomy defined in the configuration, it creates a map from term to list of entities.

For example, if three entities have the category "reviews", the taxonomy pass creates an index entry mapping "reviews" to those three entities. These indices are stored in the IR and used by the backend to generate hub pages, taxonomy index pages, and letter index pages.

The pass also normalizes taxonomy terms by lowercasing them and trimming whitespace, ensuring that "Go" and "go" map to the same term.

## 5. Related

The `related` pass computes relatedness scores between all pairs of entities based on shared taxonomy terms. For each entity, it finds all other entities that share at least one taxonomy term and scores them by the number of shared terms.

The scoring algorithm counts each shared term as one point, with optional weighting by taxonomy. Category matches can be weighted more heavily than tag matches, for example. The pass stores the top N related entities (configurable, default 5) on each entity for use in templates.

This pass enables "Related Articles" sections and internal linking structures that improve both user navigation and SEO.

## 6. GraphEnrich

The `graph_enrich` pass builds a full relationship graph from the pairwise relationships computed by the `related` pass. While the `related` pass only stores the top N neighbors per entity, the graph pass constructs a complete graph data structure with edges between all related entities.

This graph is used for advanced features like detecting content clusters, identifying hub entities (entities that connect many others), and computing graph-based metrics like centrality scores. The graph data is stored in the IR and available to templates.

## 7. ContentAnalysis

The `content_analysis` pass examines the rendered body content of each entity and computes content quality metrics. These metrics include heading structure analysis, link density, image count, code block count, and content depth scoring.

The pass checks for common content issues like missing headings, very short content, excessive link density, or missing images. The results are stored as metadata on each entity and can be used in templates to display quality indicators or in the validate pass to flag content issues.

## 8. URLResolve

The `url_resolve` pass generates absolute URLs for every entity by combining the site's `base_url` from the configuration with the entity slug. It also resolves relative URLs within entity body content to absolute URLs.

For example, with a base URL of `https://example.com` and a slug of `best-wireless-headphones`, the resolved URL becomes `https://example.com/best-wireless-headphones/`. This absolute URL is stored on the entity and used by the schema pass for JSON-LD, by the backend for canonical tags, and in sitemaps and feeds.

## 9. Schema

The `schema` pass generates JSON-LD structured data for each entity. It reads the entity metadata and the structured data configuration to produce a schema.org-compliant JSON-LD object.

The pass maps entity types to schema.org types using the `type_map` configuration. A review entity might get a `Review` schema with `reviewRating`, while a product entity gets a `Product` schema with `offers`. The generated JSON-LD is stored on the entity and embedded in the HTML page head by the backend.

## 10. Favorites

The `favorites` pass processes user-defined favorite or featured entities. If entities are marked with a `favorite: true` frontmatter field, this pass collects them into a separate list in the IR. This list can be used by templates to display featured content sections or highlighted entity lists.

## 11. Affiliate

The `affiliate` pass processes affiliate link metadata. For entities with `affiliate_url` frontmatter fields, this pass validates the URLs, adds tracking parameters if configured, and marks the entity as having affiliate content. This enables templates to render appropriate disclosures and call-to-action buttons.

## 12. Validate

The `validate` pass is the final step before the backend. It checks the completeness and consistency of the IR, ensuring that all entities have required fields, all URLs are properly resolved, all taxonomy references are valid, and no data integrity issues exist.

If validation errors are found, they are reported as warnings in the build output. The build does not fail on validation warnings by default, but a strict mode can be enabled to treat warnings as errors.

The validate pass acts as a safety net, catching any issues that might have been introduced by earlier passes or by malformed source content. It ensures that the backend receives a clean, consistent IR to generate output from.
