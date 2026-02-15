package frontend

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greynewell/schemaflux/internal/config"
	"github.com/greynewell/schemaflux/internal/entity"
	"github.com/greynewell/schemaflux/internal/yaml"
)

// markdownLoader loads entities from markdown files with YAML frontmatter.
type markdownLoader struct {
	cfg *config.Config
}

// load reads all .md files from the data directory and parses them into entities.
func (l *markdownLoader) load() ([]*entity.Entity, error) {
	dataDir := l.cfg.Paths.Data
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, fmt.Errorf("reading data dir %s: %w", dataDir, err)
	}

	var entities []*entity.Entity
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(dataDir, entry.Name())
		e, err := l.parseFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", entry.Name(), err)
			continue
		}
		entities = append(entities, e)
	}

	return entities, nil
}

func (l *markdownLoader) parseFile(path string) (*entity.Entity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)

	frontmatter, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("splitting frontmatter: %w", err)
	}

	fields, positions, err := yaml.UnmarshalMapWithPositions([]byte(frontmatter))
	if err != nil {
		return nil, fmt.Errorf("parsing frontmatter YAML: %w", err)
	}

	// Compute frontmatter offset: the opening "---" line number.
	// Frontmatter fields are relative to the start of the frontmatter block;
	// add the offset to get absolute file line numbers.
	fmOffset := frontmatterOffset(content)
	fieldPositions := make(map[string]int, len(positions))
	for k, fp := range positions {
		fieldPositions[k] = fmOffset + fp.Line
	}

	slug := l.deriveSlug(path, fields)
	sections := l.parseSections(body)

	return &entity.Entity{
		Slug:           slug,
		SourceFile:     path,
		Fields:         fields,
		FieldPositions: fieldPositions,
		Sections:       sections,
		Body:           body,
	}, nil
}

// frontmatterOffset returns the 1-based line number of the opening "---".
// Fields inside frontmatter are relative to the line after "---", so adding
// this offset converts them to absolute file line numbers.
func frontmatterOffset(content string) int {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			return i + 1 // 1-based
		}
	}
	return 0
}

func splitFrontmatter(content string) (string, string, error) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return "", content, nil
	}

	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", content, fmt.Errorf("no closing --- found for frontmatter")
	}

	fm := strings.TrimSpace(rest[:idx])
	body := strings.TrimSpace(rest[idx+4:])
	return fm, body, nil
}

func (l *markdownLoader) deriveSlug(path string, fields map[string]interface{}) string {
	source := l.cfg.Data.EntitySlug.Source
	if strings.HasPrefix(source, "field:") {
		fieldName := source[6:]
		if v, ok := fields[fieldName]; ok {
			if s, ok := v.(string); ok {
				return entity.ToSlug(s)
			}
		}
	}
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func (l *markdownLoader) parseSections(body string) map[string]interface{} {
	sections := make(map[string]interface{})

	for _, sectionCfg := range l.cfg.Data.BodySections {
		content := extractSection(body, sectionCfg.Header)
		if content == "" {
			continue
		}

		switch sectionCfg.Type {
		case "unordered_list":
			sections[sectionCfg.Name] = parseUnorderedList(content)
		case "ordered_list":
			sections[sectionCfg.Name] = parseOrderedList(content)
		case "faq":
			sections[sectionCfg.Name] = parseFAQs(content)
		case "markdown":
			sections[sectionCfg.Name] = content
		default:
			sections[sectionCfg.Name] = content
		}
	}

	return sections
}

func extractSection(body, header string) string {
	marker := "## " + header
	idx := strings.Index(body, marker)
	if idx < 0 {
		return ""
	}

	start := idx + len(marker)
	nlIdx := strings.Index(body[start:], "\n")
	if nlIdx < 0 {
		return ""
	}
	start += nlIdx + 1

	rest := body[start:]
	nextH2 := strings.Index(rest, "\n## ")
	if nextH2 >= 0 {
		rest = rest[:nextH2]
	}

	return strings.TrimSpace(rest)
}

func parseUnorderedList(content string) []string {
	var items []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			items = append(items, strings.TrimPrefix(line, "- "))
		} else if strings.HasPrefix(line, "* ") {
			items = append(items, strings.TrimPrefix(line, "* "))
		}
	}
	return items
}

func parseOrderedList(content string) []string {
	var items []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if len(line) >= 3 {
			dotIdx := strings.Index(line, ". ")
			if dotIdx > 0 && dotIdx <= 4 {
				prefix := line[:dotIdx]
				allDigits := true
				for _, c := range prefix {
					if c < '0' || c > '9' {
						allDigits = false
						break
					}
				}
				if allDigits {
					items = append(items, line[dotIdx+2:])
				}
			}
		}
	}
	return items
}

func parseFAQs(content string) []entity.FAQ {
	var faqs []entity.FAQ

	parts := strings.Split("\n"+content, "\n### ")
	for _, part := range parts[1:] {
		lines := strings.SplitN(part, "\n", 2)
		question := strings.TrimSpace(lines[0])
		answer := ""
		if len(lines) > 1 {
			answer = strings.TrimSpace(lines[1])
		}
		if question != "" {
			faqs = append(faqs, entity.FAQ{
				Question: question,
				Answer:   answer,
			})
		}
	}

	return faqs
}
