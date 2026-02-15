---
title: "Markdown Renderer"
description: "SchemaFlux includes a custom markdown-to-HTML renderer built entirely with the Go standard library. No external markdown libraries are used."
section: "Internals"
tags:
  - "markdown"
  - "renderer"
  - "zero-dependencies"
order: 4
author: "Grey Newell"
---

## Overview

SchemaFlux ships with a custom markdown-to-HTML renderer implemented in the `internal/markdown` package. The renderer is built using only the Go standard library -- no third-party markdown libraries like goldmark or blackfriday are used. This supports the project's zero-dependency design goal.

The renderer covers the subset of Markdown used by structured content: headings, paragraphs, bold/italic, links, images, inline code, fenced code blocks, lists, blockquotes, horizontal rules, tables, and raw HTML passthrough.

## Usage

The renderer is called internally by the compiler frontend when parsing entity body content, and is also exposed as a template function:

```go
// In Go code
html := markdown.Render(markdownString)

// In templates
{{ markdown .SomeField }}
```

## Supported Syntax

### Headings

ATX-style headings (h1 through h6):

```markdown
# Heading 1
## Heading 2
### Heading 3
#### Heading 4
##### Heading 5
###### Heading 6
```

### Paragraphs

Consecutive lines of text form a paragraph. Blank lines separate paragraphs.

### Inline Formatting

- **Bold**: `**text**` or `__text__`
- *Italic*: `*text*` or `_text_`
- **Bold italic**: `***text***`
- `Inline code`: backtick-wrapped text

### Links and Images

```markdown
[Link text](https://example.com)
![Alt text](https://example.com/image.png)
```

### Fenced Code Blocks

Triple-backtick fenced blocks with optional language annotation:

````markdown
```go
func main() {
    fmt.Println("Hello")
}
```
````

The language annotation is rendered as a CSS class (`language-go`) for syntax highlighting.

### Lists

Unordered lists use `-` or `*` prefixes:

```markdown
- Item one
- Item two
- Item three
```

Ordered lists use numbered prefixes:

```markdown
1. First step
2. Second step
3. Third step
```

### Blockquotes

Lines prefixed with `>`:

```markdown
> This is a blockquote.
> It can span multiple lines.
```

Blockquotes are rendered recursively, so nested markdown inside blockquotes is fully processed.

### Tables

Pipe-delimited tables with a separator row:

```markdown
| Column A | Column B |
|----------|----------|
| Cell 1   | Cell 2   |
| Cell 3   | Cell 4   |
```

### Horizontal Rules

Three or more `-`, `*`, or `_` characters on a line:

```markdown
---
```

### Raw HTML Passthrough

Lines starting with HTML tags are passed through unmodified:

```markdown
<div class="custom">
  Custom HTML content
</div>
```

HTML comments (`<!-- ... -->`) are also passed through.

## HTML Escaping

All text content is HTML-escaped to prevent injection. Special characters `&`, `<`, `>`, and `"` are converted to their entity equivalents. This escaping is applied to paragraph text, heading text, code block content, and all inline elements.

## Table of Contents

The template engine includes a `TOC` field on entity pages. The table of contents is extracted from markdown headings (h2 through h6) and each heading receives an auto-generated `id` attribute for anchor linking.

## Limitations

The following Markdown features are not supported:

- Setext-style headings (underline with `===` or `---`)
- Reference-style links (`[text][id]`)
- Definition lists
- Footnotes
- Task lists (checkboxes)
- Nested lists (only single-level lists are supported)
- Strikethrough (`~~text~~`)

These limitations are by design -- the renderer covers the markdown subset needed for structured content, keeping the implementation simple and dependency-free.
