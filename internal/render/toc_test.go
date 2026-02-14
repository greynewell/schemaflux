package render

import (
	"strings"
	"testing"
)

func TestExtractTOC(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     []TOCEntry
	}{
		{
			"empty body",
			"",
			nil,
		},
		{
			"no headings",
			"Just some text without any headings.",
			nil,
		},
		{
			"h1 ignored",
			"# Title\n\nSome text.",
			nil,
		},
		{
			"single h2",
			"## Getting Started\n\nSome text.",
			[]TOCEntry{{Level: 2, Text: "Getting Started", ID: "getting-started"}},
		},
		{
			"mixed levels",
			"## Introduction\n\nText.\n\n### Setup\n\nMore text.\n\n#### Advanced\n\nDetails.\n\n## Conclusion\n\nEnd.",
			[]TOCEntry{
				{Level: 2, Text: "Introduction", ID: "introduction"},
				{Level: 3, Text: "Setup", ID: "setup"},
				{Level: 4, Text: "Advanced", ID: "advanced"},
				{Level: 2, Text: "Conclusion", ID: "conclusion"},
			},
		},
		{
			"duplicate headings get suffix",
			"## Setup\n\nFirst.\n\n## Setup\n\nSecond.\n\n## Setup\n\nThird.",
			[]TOCEntry{
				{Level: 2, Text: "Setup", ID: "setup"},
				{Level: 2, Text: "Setup", ID: "setup-1"},
				{Level: 2, Text: "Setup", ID: "setup-2"},
			},
		},
		{
			"h6 supported",
			"###### Deep Heading\n\nText.",
			[]TOCEntry{{Level: 6, Text: "Deep Heading", ID: "deep-heading"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTOC(tt.markdown)
			if len(got) != len(tt.want) {
				t.Fatalf("ExtractTOC() returned %d entries, want %d", len(got), len(tt.want))
			}
			for i, entry := range got {
				if entry.Level != tt.want[i].Level {
					t.Errorf("entry[%d].Level = %d, want %d", i, entry.Level, tt.want[i].Level)
				}
				if entry.Text != tt.want[i].Text {
					t.Errorf("entry[%d].Text = %q, want %q", i, entry.Text, tt.want[i].Text)
				}
				if entry.ID != tt.want[i].ID {
					t.Errorf("entry[%d].ID = %q, want %q", i, entry.ID, tt.want[i].ID)
				}
			}
		})
	}
}

func TestInjectHeadingIDs(t *testing.T) {
	toc := []TOCEntry{
		{Level: 2, Text: "Getting Started", ID: "getting-started"},
		{Level: 3, Text: "Installation", ID: "installation"},
	}

	html := "<h2>Getting Started</h2>\n<p>Some text.</p>\n<h3>Installation</h3>\n<p>More text.</p>"
	got := InjectHeadingIDs(html, toc)

	if !strings.Contains(got, `<h2 id="getting-started">Getting Started</h2>`) {
		t.Errorf("Expected h2 with id, got: %s", got)
	}
	if !strings.Contains(got, `<h3 id="installation">Installation</h3>`) {
		t.Errorf("Expected h3 with id, got: %s", got)
	}
}

func TestInjectHeadingIDsEmpty(t *testing.T) {
	html := "<p>No headings here.</p>"
	got := InjectHeadingIDs(html, nil)
	if got != html {
		t.Errorf("Expected unchanged HTML, got: %s", got)
	}
}
