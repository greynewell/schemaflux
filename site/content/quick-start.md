---
title: Quick Start
description: "Install SchemaFlux and build your first site in under five minutes."
section: "Getting Started"
tags:
  - install
  - cli
order: 1
---

SchemaFlux compiles structured data from markdown files into a full static site. This guide walks you through installing the binary, setting up a minimal project, and running your first build.

## Installation

SchemaFlux is distributed as a single static binary with zero external dependencies. Install it directly using Go:

```
go install github.com/greynewell/schemaflux/cmd/schemaflux@latest
```

This downloads, compiles, and installs the `schemaflux` binary into your `$GOPATH/bin` directory. Make sure that directory is in your system `PATH`. You can verify the installation by running:

```
schemaflux --help
```

If you do not have Go installed, you can download prebuilt binaries from the [GitHub releases page](https://github.com/greynewell/schemaflux/releases).

## Create a Configuration File

Every SchemaFlux project starts with a `schemaflux.yaml` configuration file in the project root. This file defines your site metadata, content paths, taxonomies, and output settings. Here is a minimal configuration to get started:

```yaml
site:
  name: "My Site"
  base_url: "https://example.com"
  description: "A site built with SchemaFlux"
  language: "en"

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
```

The `site` section defines global metadata that gets embedded into every page. The `paths` section tells SchemaFlux where to find your content, templates, and where to write the output. Taxonomies define how your entities are grouped and organized.

## Create Content

Create a `content/` directory and add your first entity as a markdown file. Each markdown file represents one entity in the SchemaFlux system. The file should have YAML frontmatter followed by a markdown body:

```markdown
---
title: "Hello World"
description: "My first SchemaFlux entity"
date: "2025-01-15"
categories:
  - "guides"
tags:
  - "getting-started"
  - "tutorial"
---

Welcome to SchemaFlux. This is your first entity. It will be processed
through the compiler pipeline and emitted as a fully rendered HTML page
with JSON-LD structured data, Open Graph tags, and more.
```

You can add as many markdown files as you want to the content directory. SchemaFlux processes all of them in a single build pass. Each entity gets its own page, and entities are automatically linked through shared taxonomy terms.

## Create Templates

SchemaFlux uses Go's standard `html/template` package for rendering. Create a `templates/` directory and add at least an entity template. The template receives the entity data and site configuration as context:

```html
<!DOCTYPE html>
<html lang="{{ "{{" }} .Site.Language {{ "}}" }}">
<head>
  <title>{{ "{{" }} .Entity.Title {{ "}}" }}</title>
  {{ "{{" }} template "_head" . {{ "}}" }}
</head>
<body>
  {{ "{{" }} template "_header" . {{ "}}" }}
  <main>
    <h1>{{ "{{" }} .Entity.Title {{ "}}" }}</h1>
    {{ "{{" }} .Entity.Body {{ "}}" }}
  </main>
  {{ "{{" }} template "_footer" . {{ "}}" }}
</body>
</html>
```

SchemaFlux provides several template partials out of the box, including `_head` for meta tags and structured data, `_header` for site navigation, `_footer` for the footer, and `_styles` for default CSS.

## Build Your Site

With your configuration, content, and templates in place, run the build command:

```
schemaflux build --config schemaflux.yaml
```

SchemaFlux reads all entities from the content directory, runs them through the 12-pass compiler pipeline, and emits the output to your configured output directory. The build process generates individual entity pages, taxonomy hub pages, index pages, pagination, A-Z letter indices, XML sitemaps, RSS feeds, robots.txt, and llms.txt.

The output directory is a complete static site that can be deployed to any static hosting provider, including GitHub Pages, Netlify, Vercel, or a simple web server like Nginx or Caddy.

## Next Steps

Now that you have a working site, explore the rest of the documentation to learn about the full [configuration reference](/docs/configuration/), the [compiler architecture](/docs/architecture/), and the [12 compiler passes](/docs/passes/) that transform your data.
