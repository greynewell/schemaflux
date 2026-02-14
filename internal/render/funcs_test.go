package render

import (
	"testing"
)

func TestReadingTime(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantTime int
		wantWC   int
	}{
		{"empty body", "", 1, 0},
		{"single word", "hello", 1, 1},
		{"short paragraph", "This is a short paragraph with a few words.", 1, 9},
		{"200 words = 1 min", generateWords(200), 1, 200},
		{"201 words = 2 min", generateWords(201), 2, 201},
		{"400 words = 2 min", generateWords(400), 2, 400},
		{"1000 words = 5 min", generateWords(1000), 5, 1000},
		{"markdown with headings", "## Heading\n\nSome text here with **bold** and *italic* words.\n\n### Another heading\n\nMore content.", 1, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTime := ReadingTime(tt.body)
			if gotTime != tt.wantTime {
				t.Errorf("ReadingTime() = %d, want %d", gotTime, tt.wantTime)
			}
			gotWC := WordCount(tt.body)
			if gotWC != tt.wantWC {
				t.Errorf("WordCount() = %d, want %d", gotWC, tt.wantWC)
			}
		})
	}
}

func TestReadingTimeTemplateFunc(t *testing.T) {
	// Test the template func version that takes an interface{}
	got := readingTimeFunc("This is a test with several words in it")
	if got != 1 {
		t.Errorf("readingTimeFunc() = %d, want 1", got)
	}

	// Non-string returns 1
	got = readingTimeFunc(42)
	if got != 1 {
		t.Errorf("readingTimeFunc(int) = %d, want 1", got)
	}
}

func generateWords(n int) string {
	result := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			result += " "
		}
		result += "word"
	}
	return result
}
