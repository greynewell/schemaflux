package markdown

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Headings
// ---------------------------------------------------------------------------

func TestHeadings(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"H1", "# Hello", "<h1>Hello</h1>\n"},
		{"H2", "## Hello", "<h2>Hello</h2>\n"},
		{"H3", "### Hello", "<h3>Hello</h3>\n"},
		{"H4", "#### Hello", "<h4>Hello</h4>\n"},
		{"H5", "##### Hello", "<h5>Hello</h5>\n"},
		{"H6", "###### Hello", "<h6>Hello</h6>\n"},
		{"H1 with inline", "# Hello **world**", "<h1>Hello <strong>world</strong></h1>\n"},
		{"not a heading (no space)", "#nope", "<p>#nope</p>\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.in)
			if got != tt.want {
				t.Errorf("Render(%q)\n  got:  %q\n  want: %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Paragraphs
// ---------------------------------------------------------------------------

func TestParagraphs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"single paragraph",
			"Hello world",
			"<p>Hello world</p>\n",
		},
		{
			"two paragraphs",
			"Para one\n\nPara two",
			"<p>Para one</p>\n<p>Para two</p>\n",
		},
		{
			"multiline paragraph",
			"Line one\nLine two",
			"<p>Line one Line two</p>\n",
		},
		{
			"multiple blank lines between paragraphs",
			"Para one\n\n\nPara two",
			"<p>Para one</p>\n<p>Para two</p>\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.in)
			if got != tt.want {
				t.Errorf("Render(%q)\n  got:  %q\n  want: %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Bold
// ---------------------------------------------------------------------------

func TestBold(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"asterisks", "This is **bold** text", "<p>This is <strong>bold</strong> text</p>\n"},
		{"underscores", "This is __bold__ text", "<p>This is <strong>bold</strong> text</p>\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.in)
			if got != tt.want {
				t.Errorf("Render(%q)\n  got:  %q\n  want: %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Italic
// ---------------------------------------------------------------------------

func TestItalic(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"asterisks", "This is *italic* text", "<p>This is <em>italic</em> text</p>\n"},
		{"underscores", "This is _italic_ text", "<p>This is <em>italic</em> text</p>\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.in)
			if got != tt.want {
				t.Errorf("Render(%q)\n  got:  %q\n  want: %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Bold + Italic
// ---------------------------------------------------------------------------

func TestBoldItalic(t *testing.T) {
	in := "This is ***bold and italic*** text"
	want := "<p>This is <strong><em>bold and italic</em></strong> text</p>\n"
	got := Render(in)
	if got != want {
		t.Errorf("Render(%q)\n  got:  %q\n  want: %q", in, got, want)
	}
}

// ---------------------------------------------------------------------------
// Links
// ---------------------------------------------------------------------------

func TestLinks(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"simple link",
			"Visit [Google](https://google.com) today",
			`<p>Visit <a href="https://google.com">Google</a> today</p>` + "\n",
		},
		{
			"link with bold text",
			"[**Bold Link**](https://example.com)",
			`<p><a href="https://example.com"><strong>Bold Link</strong></a></p>` + "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.in)
			if got != tt.want {
				t.Errorf("Render(%q)\n  got:  %q\n  want: %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Images
// ---------------------------------------------------------------------------

func TestImages(t *testing.T) {
	in := "![Alt text](https://example.com/img.png)"
	want := `<p><img src="https://example.com/img.png" alt="Alt text"></p>` + "\n"
	got := Render(in)
	if got != want {
		t.Errorf("Render(%q)\n  got:  %q\n  want: %q", in, got, want)
	}
}

func TestImageWithSpecialChars(t *testing.T) {
	in := `![Photo & "art"](https://example.com/img.png?a=1&b=2)`
	want := `<p><img src="https://example.com/img.png?a=1&amp;b=2" alt="Photo &amp; &quot;art&quot;"></p>` + "\n"
	got := Render(in)
	if got != want {
		t.Errorf("Render(%q)\n  got:  %q\n  want: %q", in, got, want)
	}
}

// ---------------------------------------------------------------------------
// Inline code
// ---------------------------------------------------------------------------

func TestInlineCode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"simple",
			"Use `fmt.Println` to print",
			"<p>Use <code>fmt.Println</code> to print</p>\n",
		},
		{
			"with HTML inside",
			"Use `<div>` tag",
			"<p>Use <code>&lt;div&gt;</code> tag</p>\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.in)
			if got != tt.want {
				t.Errorf("Render(%q)\n  got:  %q\n  want: %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Fenced code blocks
// ---------------------------------------------------------------------------

func TestFencedCodeBlock(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"with language",
			"```go\nfmt.Println(\"hello\")\n```",
			"<pre><code class=\"language-go\">fmt.Println(&quot;hello&quot;)</code></pre>\n",
		},
		{
			"without language",
			"```\nhello\nworld\n```",
			"<pre><code>hello\nworld</code></pre>\n",
		},
		{
			"HTML entities escaped",
			"```html\n<div class=\"foo\">&amp;</div>\n```",
			"<pre><code class=\"language-html\">&lt;div class=&quot;foo&quot;&gt;&amp;amp;&lt;/div&gt;</code></pre>\n",
		},
		{
			"multiline code",
			"```python\ndef hello():\n    print(\"hi\")\n```",
			"<pre><code class=\"language-python\">def hello():\n    print(&quot;hi&quot;)</code></pre>\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.in)
			if got != tt.want {
				t.Errorf("Render(%q)\n  got:  %q\n  want: %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Unordered lists
// ---------------------------------------------------------------------------

func TestUnorderedList(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"dash items",
			"- one\n- two\n- three",
			"<ul>\n<li>one</li>\n<li>two</li>\n<li>three</li>\n</ul>\n",
		},
		{
			"asterisk items",
			"* alpha\n* beta",
			"<ul>\n<li>alpha</li>\n<li>beta</li>\n</ul>\n",
		},
		{
			"with inline formatting",
			"- **bold** item\n- *italic* item",
			"<ul>\n<li><strong>bold</strong> item</li>\n<li><em>italic</em> item</li>\n</ul>\n",
		},
		{
			"with inline code",
			"- Use `code` here",
			"<ul>\n<li>Use <code>code</code> here</li>\n</ul>\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.in)
			if got != tt.want {
				t.Errorf("Render(%q)\n  got:  %q\n  want: %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Ordered lists
// ---------------------------------------------------------------------------

func TestOrderedList(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"simple",
			"1. first\n2. second\n3. third",
			"<ol>\n<li>first</li>\n<li>second</li>\n<li>third</li>\n</ol>\n",
		},
		{
			"with inline formatting",
			"1. **bold** step\n2. Go to [link](http://x.com)",
			"<ol>\n<li><strong>bold</strong> step</li>\n<li>Go to <a href=\"http://x.com\">link</a></li>\n</ol>\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.in)
			if got != tt.want {
				t.Errorf("Render(%q)\n  got:  %q\n  want: %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Blockquotes
// ---------------------------------------------------------------------------

func TestBlockquotes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"simple",
			"> This is a quote",
			"<blockquote>\n<p>This is a quote</p>\n</blockquote>\n",
		},
		{
			"multiline",
			"> Line one\n> Line two",
			"<blockquote>\n<p>Line one Line two</p>\n</blockquote>\n",
		},
		{
			"with inline formatting",
			"> **Bold** quote",
			"<blockquote>\n<p><strong>Bold</strong> quote</p>\n</blockquote>\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.in)
			if got != tt.want {
				t.Errorf("Render(%q)\n  got:  %q\n  want: %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Horizontal rules
// ---------------------------------------------------------------------------

func TestHorizontalRule(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"dashes", "---", "<hr>\n"},
		{"asterisks", "***", "<hr>\n"},
		{"underscores", "___", "<hr>\n"},
		{"long dashes", "----------", "<hr>\n"},
		{"spaced dashes", "- - -", "<hr>\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.in)
			if got != tt.want {
				t.Errorf("Render(%q)\n  got:  %q\n  want: %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tables
// ---------------------------------------------------------------------------

func TestTables(t *testing.T) {
	in := "| Name | Age |\n| --- | --- |\n| Alice | 30 |\n| Bob | 25 |"
	want := "<table>\n<thead>\n<tr>\n<th>Name</th>\n<th>Age</th>\n</tr>\n</thead>\n<tbody>\n<tr>\n<td>Alice</td>\n<td>30</td>\n</tr>\n<tr>\n<td>Bob</td>\n<td>25</td>\n</tr>\n</tbody>\n</table>\n"
	got := Render(in)
	if got != want {
		t.Errorf("Render(table)\n  got:  %q\n  want: %q", got, want)
	}
}

func TestTableWithInline(t *testing.T) {
	in := "| Feature | Status |\n| --- | --- |\n| **Bold** | `done` |"
	got := Render(in)
	if !strings.Contains(got, "<strong>Bold</strong>") {
		t.Errorf("expected bold in table cell, got: %s", got)
	}
	if !strings.Contains(got, "<code>done</code>") {
		t.Errorf("expected code in table cell, got: %s", got)
	}
}

func TestTableHeaderOnly(t *testing.T) {
	// Table with headers and separator but no body rows.
	in := "| Col1 | Col2 |\n| --- | --- |"
	got := Render(in)
	if !strings.Contains(got, "<thead>") {
		t.Errorf("expected thead in output, got: %s", got)
	}
	if !strings.Contains(got, "<tbody>") {
		t.Errorf("expected tbody in output, got: %s", got)
	}
}

// ---------------------------------------------------------------------------
// Raw HTML passthrough
// ---------------------------------------------------------------------------

func TestRawHTML(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"div tag",
			"<div class=\"foo\">bar</div>",
			"<div class=\"foo\">bar</div>\n",
		},
		{
			"self-closing",
			"<br>",
			"<br>\n",
		},
		{
			"self-closing slash",
			"<hr/>",
			"<hr/>\n",
		},
		{
			"closing tag",
			"</section>",
			"</section>\n",
		},
		{
			"HTML comment",
			"<!-- comment -->",
			"<!-- comment -->\n",
		},
		{
			"mixed with markdown",
			"# Title\n\n<div>raw</div>\n\nParagraph",
			"<h1>Title</h1>\n<div>raw</div>\n<p>Paragraph</p>\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.in)
			if got != tt.want {
				t.Errorf("Render(%q)\n  got:  %q\n  want: %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Raw HTML block passthrough (pre, script, style)
// ---------------------------------------------------------------------------

func TestRawHTMLBlockPre(t *testing.T) {
	in := "<pre class=\"mermaid\">\nsequenceDiagram\n    A->>B: hello\n    B-->>A: world\n</pre>"
	want := "<pre class=\"mermaid\">\nsequenceDiagram\n    A->>B: hello\n    B-->>A: world\n</pre>\n"
	got := Render(in)
	if got != want {
		t.Errorf("Render(pre block)\n  got:  %q\n  want: %q", got, want)
	}
}

func TestRawHTMLBlockPreWithBlankLines(t *testing.T) {
	in := "<pre class=\"mermaid\">\nsequenceDiagram\n    participant UI as UI Thread\n\n    UI->>BG: message\n</pre>"
	want := "<pre class=\"mermaid\">\nsequenceDiagram\n    participant UI as UI Thread\n\n    UI->>BG: message\n</pre>\n"
	got := Render(in)
	if got != want {
		t.Errorf("Render(pre block with blanks)\n  got:  %q\n  want: %q", got, want)
	}
}

func TestRawHTMLBlockScript(t *testing.T) {
	in := "<script>\nconsole.log(\"hello\");\n</script>"
	want := "<script>\nconsole.log(\"hello\");\n</script>\n"
	got := Render(in)
	if got != want {
		t.Errorf("Render(script block)\n  got:  %q\n  want: %q", got, want)
	}
}

func TestRawHTMLBlockStyle(t *testing.T) {
	in := "<style>\nbody { color: red; }\n</style>"
	want := "<style>\nbody { color: red; }\n</style>\n"
	got := Render(in)
	if got != want {
		t.Errorf("Render(style block)\n  got:  %q\n  want: %q", got, want)
	}
}

func TestRawHTMLBlockPreClosingOnSameLine(t *testing.T) {
	in := "<pre>code</pre>"
	want := "<pre>code</pre>\n"
	got := Render(in)
	if got != want {
		t.Errorf("Render(pre same line close)\n  got:  %q\n  want: %q", got, want)
	}
}

func TestRawHTMLBlockFollowedByMarkdown(t *testing.T) {
	in := "<pre class=\"mermaid\">\nA->>B: hello\n</pre>\n\n## Next Section\n\nSome text"
	got := Render(in)
	if !strings.Contains(got, "A->>B: hello") {
		t.Errorf("expected raw >> in pre block, got: %s", got)
	}
	if strings.Contains(got, "&gt;") {
		t.Errorf(">> should not be escaped inside pre block, got: %s", got)
	}
	if !strings.Contains(got, "<h2>Next Section</h2>") {
		t.Errorf("expected h2 after pre block, got: %s", got)
	}
	if !strings.Contains(got, "<p>Some text</p>") {
		t.Errorf("expected paragraph after pre block, got: %s", got)
	}
}

// ---------------------------------------------------------------------------
// HTML entity escaping
// ---------------------------------------------------------------------------

func TestHTMLEntityEscaping(t *testing.T) {
	in := "Use < and > and & in text"
	want := "<p>Use &lt; and &gt; and &amp; in text</p>\n"
	got := Render(in)
	if got != want {
		t.Errorf("Render(%q)\n  got:  %q\n  want: %q", in, got, want)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestEmptyInput(t *testing.T) {
	got := Render("")
	if got != "" {
		t.Errorf("Render(empty) = %q, want empty string", got)
	}
}

func TestOnlyBlankLines(t *testing.T) {
	got := Render("\n\n\n")
	if got != "" {
		t.Errorf("Render(blank lines) = %q, want empty string", got)
	}
}

func TestCRLFLineEndings(t *testing.T) {
	in := "# Hello\r\n\r\nWorld"
	want := "<h1>Hello</h1>\n<p>World</p>\n"
	got := Render(in)
	if got != want {
		t.Errorf("Render(CRLF)\n  got:  %q\n  want: %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Complex document
// ---------------------------------------------------------------------------

func TestComplexDocument(t *testing.T) {
	md := `# My Article

This is the **introduction** with a [link](https://example.com).

## Code Example

` + "```go" + `
func main() {
    fmt.Println("hello")
}
` + "```" + `

## Features

- Fast
- Simple
- **Powerful**

> A wise quote

---

| Header | Value |
| --- | --- |
| Key | 42 |

1. Step one
2. Step two

<div class="custom">HTML block</div>

Final paragraph with *emphasis*.`

	got := Render(md)

	checks := []struct {
		label string
		want  string
	}{
		{"h1", "<h1>My Article</h1>"},
		{"bold intro", "<strong>introduction</strong>"},
		{"link", `<a href="https://example.com">link</a>`},
		{"h2", "<h2>Code Example</h2>"},
		{"code block", `<pre><code class="language-go">`},
		{"escaped code", `fmt.Println(&quot;hello&quot;)`},
		{"ul", "<ul>"},
		{"li", "<li>Fast</li>"},
		{"bold li", "<li><strong>Powerful</strong></li>"},
		{"blockquote", "<blockquote>"},
		{"hr", "<hr>"},
		{"table", "<table>"},
		{"th", "<th>Header</th>"},
		{"td", "<td>42</td>"},
		{"ol", "<ol>"},
		{"ol li", "<li>Step one</li>"},
		{"raw html", `<div class="custom">HTML block</div>`},
		{"final paragraph", "<em>emphasis</em>"},
	}

	for _, c := range checks {
		t.Run(c.label, func(t *testing.T) {
			if !strings.Contains(got, c.want) {
				t.Errorf("complex doc missing %q:\n%s", c.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Regression: list item containing a link
// ---------------------------------------------------------------------------

func TestListItemWithLink(t *testing.T) {
	in := "- See [docs](https://docs.example.com) for details"
	got := Render(in)
	want := `<ul>` + "\n" + `<li>See <a href="https://docs.example.com">docs</a> for details</li>` + "\n" + `</ul>` + "\n"
	if got != want {
		t.Errorf("Render(%q)\n  got:  %q\n  want: %q", in, got, want)
	}
}

// ---------------------------------------------------------------------------
// Heading followed by paragraph with no blank line
// ---------------------------------------------------------------------------

func TestHeadingFollowedByParagraph(t *testing.T) {
	in := "# Title\nSome text"
	got := Render(in)
	if !strings.Contains(got, "<h1>Title</h1>") {
		t.Errorf("expected h1, got: %s", got)
	}
	if !strings.Contains(got, "<p>Some text</p>") {
		t.Errorf("expected paragraph, got: %s", got)
	}
}

// ---------------------------------------------------------------------------
// Code block with empty lines
// ---------------------------------------------------------------------------

func TestCodeBlockWithEmptyLines(t *testing.T) {
	in := "```\nline1\n\nline3\n```"
	want := "<pre><code>line1\n\nline3</code></pre>\n"
	got := Render(in)
	if got != want {
		t.Errorf("Render(%q)\n  got:  %q\n  want: %q", in, got, want)
	}
}

// ---------------------------------------------------------------------------
// Horizontal rule should not conflict with list items
// ---------------------------------------------------------------------------

func TestHRNotConfusedWithListDash(t *testing.T) {
	// "- item" is a list, "---" is HR
	got := Render("- item")
	if strings.Contains(got, "<hr>") {
		t.Errorf("single dash item should not be HR: %s", got)
	}
	if !strings.Contains(got, "<li>item</li>") {
		t.Errorf("expected list item, got: %s", got)
	}
}

// ---------------------------------------------------------------------------
// Paragraph should not absorb heading
// ---------------------------------------------------------------------------

func TestParagraphDoesNotAbsorbHeading(t *testing.T) {
	in := "Para text\n\n## Heading"
	got := Render(in)
	if !strings.Contains(got, "<p>Para text</p>") {
		t.Errorf("expected paragraph, got: %s", got)
	}
	if !strings.Contains(got, "<h2>Heading</h2>") {
		t.Errorf("expected h2, got: %s", got)
	}
}

// ---------------------------------------------------------------------------
// Multiple inline formats in one line
// ---------------------------------------------------------------------------

func TestMultipleInlineFormats(t *testing.T) {
	in := "This is **bold** and *italic* and `code`"
	got := Render(in)
	want := "<p>This is <strong>bold</strong> and <em>italic</em> and <code>code</code></p>\n"
	if got != want {
		t.Errorf("Render(%q)\n  got:  %q\n  want: %q", in, got, want)
	}
}

// ---------------------------------------------------------------------------
// Blockquote with empty line marker
// ---------------------------------------------------------------------------

func TestBlockquoteEmptyLine(t *testing.T) {
	in := "> Line one\n>\n> Line two"
	got := Render(in)
	if !strings.Contains(got, "<blockquote>") {
		t.Errorf("expected blockquote, got: %s", got)
	}
	if !strings.Contains(got, "Line one") && !strings.Contains(got, "Line two") {
		t.Errorf("expected both lines in blockquote, got: %s", got)
	}
}
