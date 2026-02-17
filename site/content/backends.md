---
title: Backends
description: "Output backends: the HTML backend, output structure, and building custom backends."
section: "Reference"
tags:
  - backends
  - output
order: 9
---

A backend in SchemaFlux is the final stage of the compiler pipeline. It receives the fully enriched intermediate representation (IR) and emits output files. The backend is responsible for rendering templates, writing HTML files, copying static assets, and generating auxiliary files like sitemaps and feeds.

## The HTML Backend

SchemaFlux ships with one built-in backend: the HTML backend. This backend generates a complete static website from the IR, including all entity pages, taxonomy pages, index pages, and auxiliary files. It is the default and only backend needed for most use cases.

The HTML backend performs the following steps:

1. **Render entity pages**: For each entity in the IR, the backend renders the entity template with the entity data, related entities, taxonomy information, and site configuration. The output is written to `/{slug}/index.html`.

2. **Render index pages**: The backend generates paginated index pages listing all entities in sort order. The first page is written to `/index.html`, and subsequent pages to `/page/{n}/index.html`.

3. **Render taxonomy hub pages**: For each taxonomy, the backend renders a hub page listing all terms. The output is written to `/{taxonomy}/index.html`.

4. **Render taxonomy index pages**: For each term in each taxonomy, the backend renders an index page listing entities for that term. The output is written to `/{taxonomy}/{term}/index.html`.

5. **Render letter pages**: For each taxonomy, the backend generates A-Z letter pages at `/{taxonomy}/letter/{a-z}/index.html`.

6. **Generate sitemap**: The XML sitemap is written to `/sitemap.xml`.

7. **Generate RSS feed**: The RSS feed is written to `/feed.xml`.

8. **Generate robots.txt**: The robots file is written to `/robots.txt`.

9. **Generate llms.txt**: The LLM summary is written to `/llms.txt`.

10. **Copy static assets**: All files in the static directory are copied to the output root.

## Output Structure

A typical SchemaFlux build produces an output directory with this structure:

```
dist/
  index.html                          Main index page
  page/
    2/index.html                      Pagination page 2
    3/index.html                      Pagination page 3
  sony-wh-1000xm5-review/
    index.html                        Entity page
  bose-qc45-review/
    index.html                        Entity page
  categories/
    index.html                        Categories hub
    reviews/
      index.html                      Category: reviews
      page/
        2/index.html                  Paginated category page
    guides/
      index.html                      Category: guides
    letter/
      a/index.html                    A-Z letter index
      b/index.html
  tags/
    index.html                        Tags hub
    wireless/
      index.html                      Tag: wireless
    bluetooth/
      index.html                      Tag: bluetooth
    letter/
      a/index.html
      b/index.html
  sitemap.xml
  feed.xml
  robots.txt
  llms.txt
  static/
    css/
    images/
    js/
```

Every page is rendered as an `index.html` inside a directory, producing clean URLs without file extensions. The URL `/sony-wh-1000xm5-review/` maps to the file `dist/sony-wh-1000xm5-review/index.html`.

## Static Assets

The `static` directory in your project is copied directly to the output directory without any processing. This is where you put CSS files, JavaScript files, images, fonts, and any other assets that should be served alongside the generated HTML.

SchemaFlux does not perform any asset processing like CSS minification, JavaScript bundling, or image optimization. If you need these features, use external tools in your build pipeline before or after running SchemaFlux. The output directory contains the final static site ready for deployment.

## Template Context

The HTML backend passes specific context data to each template type. Understanding this context is essential for writing effective templates.

### Entity Template Context

```
.Site          Site configuration (name, base_url, description, etc.)
.Entity        The current entity (title, description, body, metadata, etc.)
.Related       Slice of related entities with titles and URLs
.Taxonomy      Map of taxonomy data for this entity
.Meta          Computed metadata (JSON-LD, Open Graph data, etc.)
```

### Index Template Context

```
.Site          Site configuration
.Entities      Slice of entities for the current page
.Pagination    Pagination data (page, total_pages, prev_url, next_url)
```

### Hub Template Context

```
.Site          Site configuration
.Taxonomy      Taxonomy metadata (name, plural, singular)
.Terms         Slice of terms with names, counts, and URLs
```

### Taxonomy Index Template Context

```
.Site          Site configuration
.Taxonomy      Taxonomy metadata
.Term          Current term name
.Entities      Slice of entities for this term
.Pagination    Pagination data
```

## Performance Considerations

The HTML backend is optimized for throughput. It renders all pages sequentially in a single goroutine, which avoids the overhead of goroutine scheduling and synchronization for typical site sizes. For a site with 1,997 entities, the entire backend phase (rendering all pages, generating sitemaps, copying assets) completes in under 200 milliseconds.

The backend writes files using buffered I/O and creates directories only as needed. Template execution is fast because Go's template package compiles templates once and reuses the compiled representation for each page.

## Building Custom Backends

While SchemaFlux ships with only the HTML backend, the backend interface is designed to be extensible. A backend implements a single method that receives the completed IR and produces output.

The backend interface is defined in the `internal/backend` package:

```go
type Backend interface {
    Emit(ir *ir.IR) error
}
```

To build a custom backend, implement this interface and register it in the pipeline. A custom backend could emit JSON files for an API, generate PDF documents, produce Markdown files for another system, or push content to a CMS via API calls.

The key constraint is that the backend must not modify the IR. It receives a read-only view of the data and produces output as a side effect. This ensures that backends are composable -- you could run multiple backends on the same IR if needed.

Custom backends have access to the full IR, including all entities with their computed metadata, all taxonomy indices, the relationship graph, site configuration, and template functions. This gives custom backends the same rich data that the HTML backend uses, without needing to reimplement any of the compiler passes.

## Deployment

The output of the HTML backend is a self-contained static site with no server-side dependencies. It can be deployed to any static hosting platform:

- **GitHub Pages**: Push the output directory to a `gh-pages` branch
- **Netlify / Vercel**: Point the build output directory to `dist/`
- **S3 + CloudFront**: Upload the output directory to an S3 bucket
- **Nginx / Caddy**: Serve the output directory directly
- **Any web server**: The output is plain HTML, CSS, and static assets

No server-side rendering, no runtime, no database. The generated files are the complete site.
