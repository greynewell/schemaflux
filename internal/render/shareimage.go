package render

import (
	"fmt"
	"strings"
)

// Share image constants
const (
	svgW       = 1200
	svgH       = 630
	svgBg      = "#0f1117"
	svgCard    = "#1a1d27"
	svgBorder  = "#2a2e3e"
	svgText    = "#e4e4e7"
	svgMuted   = "#9ca3af"
	svgAccent  = "#6366f1"
	svgAccentL = "#818cf8"
	svgFont    = "Inter,system-ui,sans-serif"
)

// escXML escapes XML special characters.
func escXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// truncLabel truncates a label to maxLen characters.
func truncLabel(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-2] + ".."
}

// svgHeader returns the opening SVG tag.
func svgHeader() string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, svgW, svgH, svgW, svgH)
}

// svgBackground renders the dark background with accent gradient bar at bottom.
func svgBackground() string {
	return fmt.Sprintf(
		`<rect width="%d" height="%d" fill="%s"/>`+
			`<defs><linearGradient id="grad" x1="0" y1="0" x2="1" y2="0">`+
			`<stop offset="0%%" stop-color="%s"/><stop offset="100%%" stop-color="#a855f7"/>`+
			`</linearGradient></defs>`+
			`<rect y="%d" width="%d" height="6" fill="url(#grad)"/>`,
		svgW, svgH, svgBg,
		svgAccent,
		svgH-6, svgW,
	)
}

// svgBrand renders the site name in the top-left.
func svgBrand(siteName string) string {
	return fmt.Sprintf(`<text x="60" y="60" fill="%s" font-size="18" font-weight="700" font-family="%s">%s</text>`,
		svgMuted, svgFont, escXML(siteName))
}

// svgTitle renders the main title centered.
func svgTitle(title string, y int) string {
	t := truncLabel(title, 60)
	return fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" fill="%s" font-size="36" font-weight="700" font-family="%s">%s</text>`,
		svgW/2, y, svgText, svgFont, escXML(t))
}

// svgSubtitle renders a subtitle below the title.
func svgSubtitle(sub string, y int) string {
	s := truncLabel(sub, 80)
	return fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" fill="%s" font-size="18" font-family="%s">%s</text>`,
		svgW/2, y, svgMuted, svgFont, escXML(s))
}

// renderBarsSVG renders horizontal bars as SVG elements.
func renderBarsSVG(bars []NameCount, x, y, maxW, barH, gap int) string {
	if len(bars) == 0 {
		return ""
	}
	maxVal := bars[0].Count
	for _, b := range bars {
		if b.Count > maxVal {
			maxVal = b.Count
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	var sb strings.Builder
	for i, b := range bars {
		by := y + i*(barH+gap)
		bw := (b.Count * maxW) / maxVal
		if bw < 4 {
			bw = 4
		}
		label := truncLabel(b.Name, 24)

		// Label
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-size="13" font-family="%s">%s</text>`,
			x, by+barH/2+4, svgMuted, svgFont, escXML(label)))
		// Bar
		barX := x + 200
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="3" fill="%s" opacity="0.8"/>`,
			barX, by, bw, barH, svgAccent))
		// Count
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-size="12" font-family="%s">%d</text>`,
			barX+bw+8, by+barH/2+4, svgMuted, svgFont, b.Count))
	}
	return sb.String()
}

// GenerateHomepageShareSVG generates a share image for the homepage.
func GenerateHomepageShareSVG(siteName, description string, taxStats []NameCount, totalEntities int) string {
	var sb strings.Builder
	sb.WriteString(svgHeader())
	sb.WriteString(svgBackground())
	sb.WriteString(svgBrand(siteName))
	sb.WriteString(svgTitle(siteName, 160))
	sb.WriteString(svgSubtitle(description, 200))

	// Total entities count
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="260" text-anchor="middle" fill="%s" font-size="48" font-weight="700" font-family="%s">%d</text>`,
		svgW/2, svgAccentL, svgFont, totalEntities))
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="290" text-anchor="middle" fill="%s" font-size="16" font-family="%s">Total Entities</text>`,
		svgW/2, svgMuted, svgFont))

	// Taxonomy bars
	limit := len(taxStats)
	if limit > 8 {
		limit = 8
	}
	sb.WriteString(renderBarsSVG(taxStats[:limit], 60, 320, 700, 24, 8))

	sb.WriteString(`</svg>`)
	return sb.String()
}

// GenerateEntityShareSVG generates a share image for an entity page.
func GenerateEntityShareSVG(siteName, title, nodeType, language, domain string) string {
	var sb strings.Builder
	sb.WriteString(svgHeader())
	sb.WriteString(svgBackground())
	sb.WriteString(svgBrand(siteName))
	sb.WriteString(svgTitle(title, 200))

	// Pills
	pillX := svgW/2 - 200
	pillY := 240
	pills := []struct{ label, color string }{
		{nodeType, svgAccent},
		{language, "#3b82f6"},
		{domain, "#22c55e"},
	}
	for _, p := range pills {
		if p.label == "" {
			continue
		}
		w := len(p.label)*9 + 24
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="30" rx="15" fill="%s" opacity="0.2"/>`,
			pillX, pillY, w, p.color))
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="30" rx="15" fill="none" stroke="%s" stroke-width="1"/>`,
			pillX, pillY, w, p.color))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-size="14" font-weight="500" font-family="%s">%s</text>`,
			pillX+12, pillY+20, p.color, svgFont, escXML(p.label)))
		pillX += w + 12
	}

	// Architecture docs badge
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="400" text-anchor="middle" fill="%s" font-size="16" font-family="%s">Architecture Documentation</text>`,
		svgW/2, svgMuted, svgFont))

	sb.WriteString(`</svg>`)
	return sb.String()
}

// GenerateHubShareSVG generates a share image for a hub (category) page.
func GenerateHubShareSVG(siteName, entryName, taxLabel string, count int, topTypes []NameCount) string {
	var sb strings.Builder
	sb.WriteString(svgHeader())
	sb.WriteString(svgBackground())
	sb.WriteString(svgBrand(siteName))
	sb.WriteString(svgTitle(entryName, 160))
	sb.WriteString(svgSubtitle(fmt.Sprintf("%s — %d entities", taxLabel, count), 200))

	// Bar chart of type distribution
	limit := len(topTypes)
	if limit > 8 {
		limit = 8
	}
	if limit > 0 {
		sb.WriteString(renderBarsSVG(topTypes[:limit], 60, 250, 700, 28, 10))
	}

	sb.WriteString(`</svg>`)
	return sb.String()
}

// GenerateTaxIndexShareSVG generates a share image for a taxonomy index page.
func GenerateTaxIndexShareSVG(siteName, taxLabel string, topEntries []NameCount) string {
	var sb strings.Builder
	sb.WriteString(svgHeader())
	sb.WriteString(svgBackground())
	sb.WriteString(svgBrand(siteName))
	sb.WriteString(svgTitle(taxLabel, 160))
	sb.WriteString(svgSubtitle(fmt.Sprintf("%d categories", len(topEntries)), 200))

	// Bar chart of top entries
	limit := len(topEntries)
	if limit > 10 {
		limit = 10
	}
	if limit > 0 {
		sb.WriteString(renderBarsSVG(topEntries[:limit], 60, 240, 700, 26, 8))
	}

	sb.WriteString(`</svg>`)
	return sb.String()
}

// GenerateAllEntitiesShareSVG generates a share image for the all-entities page.
func GenerateAllEntitiesShareSVG(siteName string, totalCount int, typeDist []NameCount) string {
	var sb strings.Builder
	sb.WriteString(svgHeader())
	sb.WriteString(svgBackground())
	sb.WriteString(svgBrand(siteName))
	sb.WriteString(svgTitle("All Entities", 160))

	// Big count
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="250" text-anchor="middle" fill="%s" font-size="64" font-weight="700" font-family="%s">%d</text>`,
		svgW/2, svgAccentL, svgFont, totalCount))
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="285" text-anchor="middle" fill="%s" font-size="18" font-family="%s">entities documented</text>`,
		svgW/2, svgMuted, svgFont))

	// Proportional bar segments
	if len(typeDist) > 0 {
		barY := 340
		barW := 1080
		barX := 60
		total := 0
		for _, t := range typeDist {
			total += t.Count
		}
		if total == 0 {
			total = 1
		}

		colors := []string{svgAccent, "#3b82f6", "#22c55e", "#f59e0b", "#ef4444", "#a855f7", "#6b7280", "#ec4899"}
		cx := barX
		for i, t := range typeDist {
			if i >= 8 {
				break
			}
			w := (t.Count * barW) / total
			if w < 2 {
				w = 2
			}
			color := colors[i%len(colors)]
			sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="32" fill="%s" rx="%s"/>`,
				cx, barY, w, color, func() string {
					if i == 0 {
						return "4"
					}
					if i == len(typeDist)-1 || i == 7 {
						return "4"
					}
					return "0"
				}()))
			cx += w
		}

		// Legend
		ly := barY + 56
		lx := 60
		for i, t := range typeDist {
			if i >= 8 {
				break
			}
			color := colors[i%len(colors)]
			label := truncLabel(t.Name, 20)
			sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="12" height="12" rx="2" fill="%s"/>`,
				lx, ly, color))
			sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-size="12" font-family="%s">%s (%d)</text>`,
				lx+18, ly+10, svgMuted, svgFont, escXML(label), t.Count))
			lx += len(label)*7 + 60
			if lx > 1100 {
				lx = 60
				ly += 24
			}
		}
	}

	sb.WriteString(`</svg>`)
	return sb.String()
}

// GenerateLetterShareSVG generates a share image for a letter page.
func GenerateLetterShareSVG(siteName, taxLabel, letter string, entryCount int) string {
	var sb strings.Builder
	sb.WriteString(svgHeader())
	sb.WriteString(svgBackground())
	sb.WriteString(svgBrand(siteName))

	// Large decorative letter
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="360" text-anchor="middle" fill="%s" font-size="200" font-weight="700" font-family="%s" opacity="0.15">%s</text>`,
		svgW/2, svgAccentL, svgFont, escXML(letter)))

	sb.WriteString(svgTitle(fmt.Sprintf("%s — %s", taxLabel, letter), 260))
	sb.WriteString(svgSubtitle(fmt.Sprintf("%d entries", entryCount), 300))

	sb.WriteString(`</svg>`)
	return sb.String()
}
