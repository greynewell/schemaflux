package pass

import (
	"encoding/json"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
)

// GraphEnrichmentPass enriches graph_data nodes with cross-entity metrics
// (line_count, language, call_count, called_by_count).
type GraphEnrichmentPass struct{}

func (g *GraphEnrichmentPass) Name() string        { return "GraphEnrichment" }
func (g *GraphEnrichmentPass) Requires() []string { return []string{"SlugResolution"} }

func (g *GraphEnrichmentPass) Run(p *ir.Program) error {
	for _, re := range p.Entities {
		e := re.Raw
		graphJSON := e.GetString("graph_data")
		if graphJSON == "" {
			continue
		}
		var graphObj struct {
			Nodes []map[string]interface{} `json:"nodes"`
			Edges []struct {
				Source string `json:"source"`
				Target string `json:"target"`
				Type   string `json:"type"`
			} `json:"edges"`
		}
		if json.Unmarshal([]byte(graphJSON), &graphObj) != nil {
			continue
		}
		changed := false
		for i, node := range graphObj.Nodes {
			slugVal, ok := node["slug"].(string)
			if !ok || slugVal == "" {
				continue
			}
			ref, found := p.EntityBySlug[slugVal]
			if !found {
				continue
			}
			if lc := ref.Raw.GetInt("line_count"); lc > 0 {
				graphObj.Nodes[i]["lc"] = lc
				changed = true
			}
			if lang := ref.Raw.GetString("language"); lang != "" {
				graphObj.Nodes[i]["lang"] = lang
				changed = true
			}
			if cc := ref.Raw.GetInt("call_count"); cc > 0 {
				graphObj.Nodes[i]["cc"] = cc
				changed = true
			}
			if cbc := ref.Raw.GetInt("called_by_count"); cbc > 0 {
				graphObj.Nodes[i]["cbc"] = cbc
				changed = true
			}
		}
		if changed {
			enrichedGraph, _ := json.Marshal(graphObj)
			re.EnrichedGraphData = string(enrichedGraph)
		}
	}
	return nil
}
