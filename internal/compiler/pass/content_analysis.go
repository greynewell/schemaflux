package pass

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
	"github.com/greynewell/schemaflux/internal/render"
)

// ContentAnalysisPass extracts TOC, reading time, word count, and source code snippets.
type ContentAnalysisPass struct{}

func (c *ContentAnalysisPass) Name() string { return "ContentAnalysis" }

func (c *ContentAnalysisPass) Run(p *ir.Program) error {
	for _, re := range p.Entities {
		e := re.Raw

		// TOC, reading time, word count
		re.TOC = render.ExtractTOC(e.Body)
		re.ReadingTime = render.ReadingTime(e.Body)
		re.WordCount = render.WordCount(e.Body)

		// Source code extraction
		srcDir := p.Config.Paths.SourceDir
		if srcDir == "" {
			continue
		}
		filePath := e.GetString("file_path")
		if filePath == "" {
			continue
		}
		startLine := e.GetInt("start_line")
		endLine := e.GetInt("end_line")
		re.SourceLang = e.GetString("language")
		absPath := filepath.Join(srcDir, filePath)
		data, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		if startLine > 0 && endLine > 0 && startLine <= len(lines) {
			if endLine > len(lines) {
				endLine = len(lines)
			}
			if endLine-startLine > 80 {
				endLine = startLine + 80
			}
			re.SourceCode = strings.Join(lines[startLine-1:endLine], "\n")
		} else if e.GetString("node_type") == "File" && len(lines) <= 120 {
			re.SourceCode = string(data)
		} else if e.GetString("node_type") == "File" && len(lines) > 120 {
			re.SourceCode = strings.Join(lines[:60], "\n") + "\n// ... (" + fmt.Sprintf("%d", len(lines)-60) + " more lines)"
		}
	}
	return nil
}
