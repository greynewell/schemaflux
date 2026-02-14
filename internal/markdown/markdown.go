// Package markdown provides a minimal Markdown-to-HTML renderer using only
// the Go standard library. It covers the subset of Markdown used by musegpt
// content: headings, paragraphs, bold/italic, links, images, inline code,
// fenced code blocks, lists, blockquotes, horizontal rules, tables, and raw
// HTML passthrough.
package markdown

import (
	"regexp"
	"strings"
)

// Render converts a Markdown string to an HTML string.
func Render(markdown string) string {
	// Normalize line endings.
	markdown = strings.ReplaceAll(markdown, "\r\n", "\n")
	markdown = strings.ReplaceAll(markdown, "\r", "\n")

	lines := strings.Split(markdown, "\n")
	var out strings.Builder

	i := 0
	for i < len(lines) {
		line := lines[i]

		// --- Fenced code blocks ---
		if lang, ok := fencedCodeStart(line); ok {
			i++
			var code strings.Builder
			for i < len(lines) {
				if isFencedCodeEnd(lines[i]) {
					i++
					break
				}
				if code.Len() > 0 {
					code.WriteByte('\n')
				}
				code.WriteString(lines[i])
				i++
			}
			if lang != "" {
				out.WriteString(`<pre><code class="language-` + lang + `">`)
			} else {
				out.WriteString("<pre><code>")
			}
			out.WriteString(escapeHTML(code.String()))
			out.WriteString("</code></pre>\n")
			continue
		}

		// --- Raw HTML passthrough ---
		if isRawHTML(line) {
			out.WriteString(line)
			out.WriteByte('\n')
			i++
			continue
		}

		// --- Blank line ---
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}

		// --- Horizontal rule ---
		if isHorizontalRule(line) {
			out.WriteString("<hr>\n")
			i++
			continue
		}

		// --- Heading ---
		if level, text, ok := parseHeading(line); ok {
			out.WriteString("<h")
			out.WriteByte('0' + byte(level))
			out.WriteByte('>')
			out.WriteString(renderInline(text))
			out.WriteString("</h")
			out.WriteByte('0' + byte(level))
			out.WriteString(">\n")
			i++
			continue
		}

		// --- Blockquote ---
		if strings.HasPrefix(strings.TrimSpace(line), "> ") || strings.TrimSpace(line) == ">" {
			var bqLines []string
			for i < len(lines) {
				trimmed := strings.TrimSpace(lines[i])
				if strings.HasPrefix(trimmed, "> ") {
					bqLines = append(bqLines, trimmed[2:])
					i++
				} else if trimmed == ">" {
					bqLines = append(bqLines, "")
					i++
				} else {
					break
				}
			}
			inner := Render(strings.Join(bqLines, "\n"))
			out.WriteString("<blockquote>\n")
			out.WriteString(inner)
			out.WriteString("</blockquote>\n")
			continue
		}

		// --- Table ---
		if isTableRow(line) && i+1 < len(lines) && isTableSeparator(lines[i+1]) {
			i = renderTable(&out, lines, i)
			continue
		}

		// --- Unordered list ---
		if isUnorderedListItem(line) {
			i = renderUnorderedList(&out, lines, i)
			continue
		}

		// --- Ordered list ---
		if isOrderedListItem(line) {
			i = renderOrderedList(&out, lines, i)
			continue
		}

		// --- Paragraph (default) ---
		var paraLines []string
		for i < len(lines) {
			l := lines[i]
			if strings.TrimSpace(l) == "" {
				break
			}
			if _, ok := fencedCodeStart(l); ok {
				break
			}
			if isRawHTML(l) {
				break
			}
			if isHorizontalRule(l) {
				break
			}
			if _, _, ok := parseHeading(l); ok {
				break
			}
			trimmed := strings.TrimSpace(l)
			if strings.HasPrefix(trimmed, "> ") || trimmed == ">" {
				break
			}
			if isTableRow(l) && i+1 < len(lines) && isTableSeparator(lines[i+1]) {
				break
			}
			if isUnorderedListItem(l) {
				break
			}
			if isOrderedListItem(l) {
				break
			}
			paraLines = append(paraLines, l)
			i++
		}
		if len(paraLines) > 0 {
			out.WriteString("<p>")
			out.WriteString(renderInline(strings.Join(paraLines, "\n")))
			out.WriteString("</p>\n")
		}
	}

	return out.String()
}

// ---------------------------------------------------------------------------
// Block-level helpers
// ---------------------------------------------------------------------------

var fenceRegex = regexp.MustCompile("^```(\\w*)\\s*$")

func fencedCodeStart(line string) (lang string, ok bool) {
	trimmed := strings.TrimSpace(line)
	m := fenceRegex.FindStringSubmatch(trimmed)
	if m == nil {
		return "", false
	}
	return m[1], true
}

func isFencedCodeEnd(line string) bool {
	return strings.TrimSpace(line) == "```"
}

var htmlTagRegex = regexp.MustCompile(`^</?[a-zA-Z][a-zA-Z0-9]*[\s>!/]`)

func isRawHTML(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	// Also handle HTML comments.
	if strings.HasPrefix(trimmed, "<!--") {
		return true
	}
	return htmlTagRegex.MatchString(trimmed)
}

func isHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	// Must be all the same char (-, *, _) optionally separated by spaces.
	clean := strings.ReplaceAll(trimmed, " ", "")
	if len(clean) < 3 {
		return false
	}
	ch := clean[0]
	if ch != '-' && ch != '*' && ch != '_' {
		return false
	}
	for _, c := range clean {
		if byte(c) != ch {
			return false
		}
	}
	return true
}

func parseHeading(line string) (level int, text string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return 0, "", false
	}
	lvl := 0
	for lvl < len(trimmed) && trimmed[lvl] == '#' {
		lvl++
	}
	if lvl > 6 || lvl == 0 {
		return 0, "", false
	}
	// Must be followed by a space (or be the entire line for an empty heading).
	rest := trimmed[lvl:]
	if rest != "" && rest[0] != ' ' {
		return 0, "", false
	}
	return lvl, strings.TrimSpace(rest), true
}

var unorderedListRegex = regexp.MustCompile(`^(\s*)[-*]\s+(.*)`)

func isUnorderedListItem(line string) bool {
	return unorderedListRegex.MatchString(line)
}

func extractUnorderedListItem(line string) string {
	m := unorderedListRegex.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	return m[2]
}

var orderedListRegex = regexp.MustCompile(`^(\s*)\d+\.\s+(.*)`)

func isOrderedListItem(line string) bool {
	return orderedListRegex.MatchString(line)
}

func extractOrderedListItem(line string) string {
	m := orderedListRegex.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	return m[2]
}

func renderUnorderedList(out *strings.Builder, lines []string, i int) int {
	out.WriteString("<ul>\n")
	for i < len(lines) && isUnorderedListItem(lines[i]) {
		text := extractUnorderedListItem(lines[i])
		out.WriteString("<li>")
		out.WriteString(renderInline(text))
		out.WriteString("</li>\n")
		i++
	}
	out.WriteString("</ul>\n")
	return i
}

func renderOrderedList(out *strings.Builder, lines []string, i int) int {
	out.WriteString("<ol>\n")
	for i < len(lines) && isOrderedListItem(lines[i]) {
		text := extractOrderedListItem(lines[i])
		out.WriteString("<li>")
		out.WriteString(renderInline(text))
		out.WriteString("</li>\n")
		i++
	}
	out.WriteString("</ol>\n")
	return i
}

func isTableRow(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") && strings.Count(trimmed, "|") >= 2
}

var tableSepRegex = regexp.MustCompile(`^\|[\s:-]+(\|[\s:-]+)+\|$`)

func isTableSeparator(line string) bool {
	return tableSepRegex.MatchString(strings.TrimSpace(line))
}

func parseTableCells(line string) []string {
	trimmed := strings.TrimSpace(line)
	// Strip leading and trailing pipes.
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(p)
	}
	return cells
}

func renderTable(out *strings.Builder, lines []string, i int) int {
	// Header row
	headers := parseTableCells(lines[i])
	i++ // skip header
	i++ // skip separator

	out.WriteString("<table>\n<thead>\n<tr>\n")
	for _, h := range headers {
		out.WriteString("<th>")
		out.WriteString(renderInline(h))
		out.WriteString("</th>\n")
	}
	out.WriteString("</tr>\n</thead>\n<tbody>\n")

	for i < len(lines) && isTableRow(lines[i]) {
		cells := parseTableCells(lines[i])
		out.WriteString("<tr>\n")
		for _, c := range cells {
			out.WriteString("<td>")
			out.WriteString(renderInline(c))
			out.WriteString("</td>\n")
		}
		out.WriteString("</tr>\n")
		i++
	}

	out.WriteString("</tbody>\n</table>\n")
	return i
}

// ---------------------------------------------------------------------------
// Inline rendering
// ---------------------------------------------------------------------------

func renderInline(text string) string {
	// We process the text character by character, handling markdown constructs.
	var out strings.Builder
	runes := []rune(text)
	n := len(runes)

	for i := 0; i < n; {
		ch := runes[i]

		// --- Inline code ---
		if ch == '`' {
			end := indexRune(runes, '`', i+1)
			if end != -1 {
				code := string(runes[i+1 : end])
				out.WriteString("<code>")
				out.WriteString(escapeHTML(code))
				out.WriteString("</code>")
				i = end + 1
				continue
			}
		}

		// --- Images: ![alt](url) ---
		if ch == '!' && i+1 < n && runes[i+1] == '[' {
			altEnd := indexRune(runes, ']', i+2)
			if altEnd != -1 && altEnd+1 < n && runes[altEnd+1] == '(' {
				urlEnd := indexRune(runes, ')', altEnd+2)
				if urlEnd != -1 {
					alt := string(runes[i+2 : altEnd])
					href := string(runes[altEnd+2 : urlEnd])
					out.WriteString(`<img src="`)
					out.WriteString(escapeHTMLAttr(href))
					out.WriteString(`" alt="`)
					out.WriteString(escapeHTML(alt))
					out.WriteString(`">`)
					i = urlEnd + 1
					continue
				}
			}
		}

		// --- Links: [text](url) ---
		if ch == '[' {
			textEnd := indexRune(runes, ']', i+1)
			if textEnd != -1 && textEnd+1 < n && runes[textEnd+1] == '(' {
				urlEnd := indexRune(runes, ')', textEnd+2)
				if urlEnd != -1 {
					linkText := string(runes[i+1 : textEnd])
					href := string(runes[textEnd+2 : urlEnd])
					out.WriteString(`<a href="`)
					out.WriteString(escapeHTMLAttr(href))
					out.WriteString(`">`)
					out.WriteString(renderInline(linkText))
					out.WriteString("</a>")
					i = urlEnd + 1
					continue
				}
			}
		}

		// --- Bold+Italic: ***text*** ---
		if ch == '*' && i+2 < n && runes[i+1] == '*' && runes[i+2] == '*' {
			end := indexRuneSeq(runes, "***", i+3)
			if end != -1 {
				inner := string(runes[i+3 : end])
				out.WriteString("<strong><em>")
				out.WriteString(renderInline(inner))
				out.WriteString("</em></strong>")
				i = end + 3
				continue
			}
		}

		// --- Bold: **text** ---
		if ch == '*' && i+1 < n && runes[i+1] == '*' {
			end := indexRuneSeq(runes, "**", i+2)
			if end != -1 {
				inner := string(runes[i+2 : end])
				out.WriteString("<strong>")
				out.WriteString(renderInline(inner))
				out.WriteString("</strong>")
				i = end + 2
				continue
			}
		}

		// --- Italic: *text* ---
		if ch == '*' {
			end := indexRune(runes, '*', i+1)
			if end != -1 {
				inner := string(runes[i+1 : end])
				out.WriteString("<em>")
				out.WriteString(renderInline(inner))
				out.WriteString("</em>")
				i = end + 1
				continue
			}
		}

		// --- Bold: __text__ ---
		if ch == '_' && i+1 < n && runes[i+1] == '_' {
			end := indexRuneSeq(runes, "__", i+2)
			if end != -1 {
				inner := string(runes[i+2 : end])
				out.WriteString("<strong>")
				out.WriteString(renderInline(inner))
				out.WriteString("</strong>")
				i = end + 2
				continue
			}
		}

		// --- Italic: _text_ ---
		if ch == '_' {
			end := indexRune(runes, '_', i+1)
			if end != -1 {
				inner := string(runes[i+1 : end])
				out.WriteString("<em>")
				out.WriteString(renderInline(inner))
				out.WriteString("</em>")
				i = end + 1
				continue
			}
		}

		// --- Newline within a paragraph becomes a space ---
		if ch == '\n' {
			out.WriteByte(' ')
			i++
			continue
		}

		// --- Escape HTML entities in regular text ---
		out.WriteString(escapeHTMLChar(ch))
		i++
	}

	return out.String()
}

// ---------------------------------------------------------------------------
// Utility functions
// ---------------------------------------------------------------------------

func indexRune(runes []rune, target rune, start int) int {
	for i := start; i < len(runes); i++ {
		if runes[i] == target {
			return i
		}
	}
	return -1
}

func indexRuneSeq(runes []rune, seq string, start int) int {
	seqRunes := []rune(seq)
	seqLen := len(seqRunes)
	for i := start; i <= len(runes)-seqLen; i++ {
		match := true
		for j := 0; j < seqLen; j++ {
			if runes[i+j] != seqRunes[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func escapeHTMLAttr(s string) string {
	return escapeHTML(s)
}

func escapeHTMLChar(ch rune) string {
	switch ch {
	case '&':
		return "&amp;"
	case '<':
		return "&lt;"
	case '>':
		return "&gt;"
	case '"':
		return "&quot;"
	default:
		return string(ch)
	}
}
