package render

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/greynewell/pssg/internal/entity"
)

var headingRegex = regexp.MustCompile(`(?m)^(#{2,6})\s+(.+)$`)

// ExtractTOC parses heading lines from markdown and returns TOC entries.
func ExtractTOC(markdown string) []TOCEntry {
	matches := headingRegex.FindAllStringSubmatch(markdown, -1)
	if len(matches) == 0 {
		return nil
	}

	idCounts := make(map[string]int)
	var entries []TOCEntry

	for _, m := range matches {
		level := len(m[1]) // number of # characters
		text := strings.TrimSpace(m[2])
		id := entity.ToSlug(text)

		// Handle duplicate IDs
		if count, exists := idCounts[id]; exists {
			idCounts[id] = count + 1
			id = fmt.Sprintf("%s-%d", id, count)
		} else {
			idCounts[id] = 1
		}

		entries = append(entries, TOCEntry{
			Level: level,
			Text:  text,
			ID:    id,
		})
	}

	return entries
}

// InjectHeadingIDs adds id attributes to rendered HTML headings to match TOC anchors.
func InjectHeadingIDs(html string, toc []TOCEntry) string {
	if len(toc) == 0 {
		return html
	}

	for _, entry := range toc {
		// Match <h2>Text</h2> and replace with <h2 id="slug">Text</h2>
		tag := fmt.Sprintf("h%d", entry.Level)
		// Use a regex that matches the heading tag with the exact text
		pattern := fmt.Sprintf(`<%s>(%s)</%s>`, tag, regexp.QuoteMeta(entry.Text), tag)
		re := regexp.MustCompile(pattern)
		replacement := fmt.Sprintf(`<%s id="%s">%s</%s>`, tag, entry.ID, entry.Text, tag)
		html = re.ReplaceAllString(html, replacement)
	}

	return html
}
