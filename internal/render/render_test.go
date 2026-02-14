package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/greynewell/pssg/internal/config"
	"github.com/greynewell/pssg/internal/entity"
)

func setupTestEngine(t *testing.T, templates map[string]string, entityTemplate string) *Engine {
	t.Helper()
	tmpDir := t.TempDir()

	for name, content := range templates {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write template %s: %v", name, err)
		}
	}

	cfg := &config.Config{
		Paths: config.PathsConfig{Templates: tmpDir},
		Templates: config.TemplatesConfig{
			Entity:   entityTemplate,
			Homepage: "index.html",
		},
	}

	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	return engine
}

func TestRenderEntityDefaultTemplate(t *testing.T) {
	engine := setupTestEngine(t, map[string]string{
		"entity.html": "<html><body><h1>{{.Entity.GetString \"title\"}}</h1></body></html>",
	}, "entity.html")

	e := &entity.Entity{
		Slug:   "test",
		Fields: map[string]interface{}{"title": "Test Page"},
	}

	ctx := EntityPageContext{Entity: e, Slug: "test"}
	html, err := engine.RenderEntity(ctx)
	if err != nil {
		t.Fatalf("RenderEntity() error: %v", err)
	}

	if html == "" {
		t.Error("RenderEntity() returned empty HTML")
	}
	if !contains(html, "<h1>Test Page</h1>") {
		t.Errorf("Expected title in output, got: %s", html)
	}
}

func TestRenderEntityTemplateOverride(t *testing.T) {
	engine := setupTestEngine(t, map[string]string{
		"entity.html":  "<html><body><div class=\"default\">{{.Entity.GetString \"title\"}}</div></body></html>",
		"article.html": "<html><body><article>{{.Entity.GetString \"title\"}}</article></body></html>",
	}, "entity.html")

	e := &entity.Entity{
		Slug:   "test",
		Fields: map[string]interface{}{"title": "My Article", "template": "article.html"},
	}

	ctx := EntityPageContext{Entity: e, Slug: "test"}
	html, err := engine.RenderEntity(ctx)
	if err != nil {
		t.Fatalf("RenderEntity() error: %v", err)
	}

	if !contains(html, "<article>My Article</article>") {
		t.Errorf("Expected article template, got: %s", html)
	}
	if contains(html, "default") {
		t.Errorf("Should not use default template when override is set")
	}
}

func TestRenderEntityInvalidTemplateOverride(t *testing.T) {
	engine := setupTestEngine(t, map[string]string{
		"entity.html": "<html><body>default</body></html>",
	}, "entity.html")

	e := &entity.Entity{
		Slug:   "test",
		Fields: map[string]interface{}{"title": "Test", "template": "nonexistent.html"},
	}

	ctx := EntityPageContext{Entity: e, Slug: "test"}
	_, err := engine.RenderEntity(ctx)
	if err == nil {
		t.Error("Expected error for nonexistent template, got nil")
	}
}

func TestRenderEntityEmptyTemplateFieldUsesDefault(t *testing.T) {
	engine := setupTestEngine(t, map[string]string{
		"entity.html": "<html><body><div class=\"default\">ok</div></body></html>",
	}, "entity.html")

	e := &entity.Entity{
		Slug:   "test",
		Fields: map[string]interface{}{"title": "Test", "template": ""},
	}

	ctx := EntityPageContext{Entity: e, Slug: "test"}
	html, err := engine.RenderEntity(ctx)
	if err != nil {
		t.Fatalf("RenderEntity() error: %v", err)
	}

	if !contains(html, "default") {
		t.Errorf("Expected default template for empty override, got: %s", html)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
