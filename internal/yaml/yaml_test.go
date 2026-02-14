package yaml

import (
	"reflect"
	"testing"
)

// ---------- helpers ----------

func mustUnmarshalMap(t *testing.T, input string) map[string]interface{} {
	t.Helper()
	m, err := UnmarshalMap([]byte(input))
	if err != nil {
		t.Fatalf("UnmarshalMap error: %v", err)
	}
	return m
}

// ---------- UnmarshalMap tests ----------

func TestSimpleKeyValue(t *testing.T) {
	input := `
name: hello
version: 1.0
`
	m := mustUnmarshalMap(t, input)
	if m["name"] != "hello" {
		t.Errorf("name = %v, want hello", m["name"])
	}
	// "1.0" contains a dot so it is parsed as float64.
	if m["version"] != 1.0 {
		t.Errorf("version = %v (%T), want 1.0", m["version"], m["version"])
	}
}

func TestQuotedStrings(t *testing.T) {
	input := `
double: "hello world"
single: 'foo bar'
`
	m := mustUnmarshalMap(t, input)
	if m["double"] != "hello world" {
		t.Errorf("double = %v", m["double"])
	}
	if m["single"] != "foo bar" {
		t.Errorf("single = %v", m["single"])
	}
}

func TestBooleans(t *testing.T) {
	input := `
enabled: true
disabled: false
`
	m := mustUnmarshalMap(t, input)
	if m["enabled"] != true {
		t.Errorf("enabled = %v", m["enabled"])
	}
	if m["disabled"] != false {
		t.Errorf("disabled = %v", m["disabled"])
	}
}

func TestIntegers(t *testing.T) {
	input := `
count: 42
negative: -3
zero: 0
`
	m := mustUnmarshalMap(t, input)
	if m["count"] != 42 {
		t.Errorf("count = %v (%T)", m["count"], m["count"])
	}
	if m["negative"] != -3 {
		t.Errorf("negative = %v", m["negative"])
	}
	if m["zero"] != 0 {
		t.Errorf("zero = %v", m["zero"])
	}
}

func TestEmptyValues(t *testing.T) {
	input := `
empty: ""
blank:
`
	m := mustUnmarshalMap(t, input)
	if m["empty"] != "" {
		t.Errorf("empty = %v", m["empty"])
	}
	if m["blank"] != "" {
		t.Errorf("blank = %v (%T)", m["blank"], m["blank"])
	}
}

func TestComments(t *testing.T) {
	input := `
# this is a comment
name: hello # inline comment
# another comment
age: 30
`
	m := mustUnmarshalMap(t, input)
	if m["name"] != "hello" {
		t.Errorf("name = %v", m["name"])
	}
	if m["age"] != 30 {
		t.Errorf("age = %v", m["age"])
	}
	if len(m) != 2 {
		t.Errorf("len = %d, want 2", len(m))
	}
}

func TestNestedMaps(t *testing.T) {
	input := `
site:
  name: My Site
  url: https://example.com
paths:
  data: ./data
  output: ./docs
`
	m := mustUnmarshalMap(t, input)
	site, ok := m["site"].(map[string]interface{})
	if !ok {
		t.Fatalf("site is not a map: %T", m["site"])
	}
	if site["name"] != "My Site" {
		t.Errorf("site.name = %v", site["name"])
	}
	if site["url"] != "https://example.com" {
		t.Errorf("site.url = %v", site["url"])
	}
	paths, ok := m["paths"].(map[string]interface{})
	if !ok {
		t.Fatalf("paths is not a map: %T", m["paths"])
	}
	if paths["data"] != "./data" {
		t.Errorf("paths.data = %v", paths["data"])
	}
}

func TestListOfStrings(t *testing.T) {
	input := `
extra_keywords:
  - "Claude Chef"
  - "AI Cooking"
  - "Home Cooking"
`
	m := mustUnmarshalMap(t, input)
	kw, ok := m["extra_keywords"].([]interface{})
	if !ok {
		t.Fatalf("extra_keywords type = %T", m["extra_keywords"])
	}
	if len(kw) != 3 {
		t.Fatalf("len = %d", len(kw))
	}
	if kw[0] != "Claude Chef" {
		t.Errorf("[0] = %v", kw[0])
	}
	if kw[2] != "Home Cooking" {
		t.Errorf("[2] = %v", kw[2])
	}
}

func TestListOfMaps(t *testing.T) {
	input := `
taxonomies:
  - name: category
    label: Categories
    multi_value: false
    min_entities: 1
  - name: cuisine
    label: Cuisines
    multi_value: true
    min_entities: 3
`
	m := mustUnmarshalMap(t, input)
	taxs, ok := m["taxonomies"].([]interface{})
	if !ok {
		t.Fatalf("taxonomies type = %T", m["taxonomies"])
	}
	if len(taxs) != 2 {
		t.Fatalf("len = %d", len(taxs))
	}
	first, ok := taxs[0].(map[string]interface{})
	if !ok {
		t.Fatalf("first type = %T", taxs[0])
	}
	if first["name"] != "category" {
		t.Errorf("first.name = %v", first["name"])
	}
	if first["multi_value"] != false {
		t.Errorf("first.multi_value = %v", first["multi_value"])
	}
	second := taxs[1].(map[string]interface{})
	if second["name"] != "cuisine" {
		t.Errorf("second.name = %v", second["name"])
	}
	if second["min_entities"] != 3 {
		t.Errorf("second.min_entities = %v", second["min_entities"])
	}
}

func TestMultiWordUnquotedStrings(t *testing.T) {
	input := `
description: Delicious tested recipes with AI-powered cooking guidance.
`
	m := mustUnmarshalMap(t, input)
	want := "Delicious tested recipes with AI-powered cooking guidance."
	if m["description"] != want {
		t.Errorf("description = %v", m["description"])
	}
}

func TestDeeplyNestedMaps(t *testing.T) {
	input := `
data:
  format: markdown
  entity_slug:
    source: filename
`
	m := mustUnmarshalMap(t, input)
	data := m["data"].(map[string]interface{})
	if data["format"] != "markdown" {
		t.Errorf("data.format = %v", data["format"])
	}
	slug := data["entity_slug"].(map[string]interface{})
	if slug["source"] != "filename" {
		t.Errorf("slug.source = %v", slug["source"])
	}
}

func TestMapOfStrings(t *testing.T) {
	input := `
priorities:
  homepage: "1.0"
  entity: "0.8"
  hub: "0.6"
`
	m := mustUnmarshalMap(t, input)
	p := m["priorities"].(map[string]interface{})
	if p["homepage"] != "1.0" {
		t.Errorf("homepage = %v (%T)", p["homepage"], p["homepage"])
	}
	if p["entity"] != "0.8" {
		t.Errorf("entity = %v", p["entity"])
	}
}

// ---------- Unmarshal (struct) tests ----------

func TestUnmarshalSimpleStruct(t *testing.T) {
	type S struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
		Count   int    `yaml:"count"`
		Enabled bool   `yaml:"enabled"`
	}
	input := `
name: hello
version: "1.0"
count: 42
enabled: true
`
	var s S
	if err := Unmarshal([]byte(input), &s); err != nil {
		t.Fatal(err)
	}
	if s.Name != "hello" {
		t.Errorf("Name = %q", s.Name)
	}
	if s.Version != "1.0" {
		t.Errorf("Version = %q", s.Version)
	}
	if s.Count != 42 {
		t.Errorf("Count = %d", s.Count)
	}
	if s.Enabled != true {
		t.Errorf("Enabled = %v", s.Enabled)
	}
}

func TestUnmarshalNestedStruct(t *testing.T) {
	type Inner struct {
		Data   string `yaml:"data"`
		Output string `yaml:"output"`
	}
	type Outer struct {
		Name  string `yaml:"name"`
		Paths Inner  `yaml:"paths"`
	}
	input := `
name: test
paths:
  data: ./data
  output: ./docs
`
	var o Outer
	if err := Unmarshal([]byte(input), &o); err != nil {
		t.Fatal(err)
	}
	if o.Name != "test" {
		t.Errorf("Name = %q", o.Name)
	}
	if o.Paths.Data != "./data" {
		t.Errorf("Paths.Data = %q", o.Paths.Data)
	}
	if o.Paths.Output != "./docs" {
		t.Errorf("Paths.Output = %q", o.Paths.Output)
	}
}

func TestUnmarshalSliceOfStrings(t *testing.T) {
	type S struct {
		Tags []string `yaml:"tags"`
	}
	input := `
tags:
  - go
  - yaml
  - parser
`
	var s S
	if err := Unmarshal([]byte(input), &s); err != nil {
		t.Fatal(err)
	}
	want := []string{"go", "yaml", "parser"}
	if !reflect.DeepEqual(s.Tags, want) {
		t.Errorf("Tags = %v, want %v", s.Tags, want)
	}
}

func TestUnmarshalSliceOfStructs(t *testing.T) {
	type Tax struct {
		Name       string `yaml:"name"`
		Label      string `yaml:"label"`
		MultiValue bool   `yaml:"multi_value"`
		MinEntities int   `yaml:"min_entities"`
	}
	type Cfg struct {
		Taxonomies []Tax `yaml:"taxonomies"`
	}
	input := `
taxonomies:
  - name: category
    label: Categories
    multi_value: false
    min_entities: 1
  - name: cuisine
    label: Cuisines
    multi_value: true
    min_entities: 3
`
	var cfg Cfg
	if err := Unmarshal([]byte(input), &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Taxonomies) != 2 {
		t.Fatalf("len = %d", len(cfg.Taxonomies))
	}
	if cfg.Taxonomies[0].Name != "category" {
		t.Errorf("[0].Name = %q", cfg.Taxonomies[0].Name)
	}
	if cfg.Taxonomies[0].MultiValue != false {
		t.Errorf("[0].MultiValue = %v", cfg.Taxonomies[0].MultiValue)
	}
	if cfg.Taxonomies[1].MinEntities != 3 {
		t.Errorf("[1].MinEntities = %d", cfg.Taxonomies[1].MinEntities)
	}
}

func TestUnmarshalMapField(t *testing.T) {
	type S struct {
		Priorities map[string]string `yaml:"priorities"`
	}
	input := `
priorities:
  homepage: "1.0"
  entity: "0.8"
`
	var s S
	if err := Unmarshal([]byte(input), &s); err != nil {
		t.Fatal(err)
	}
	if s.Priorities["homepage"] != "1.0" {
		t.Errorf("homepage = %q", s.Priorities["homepage"])
	}
	if s.Priorities["entity"] != "0.8" {
		t.Errorf("entity = %q", s.Priorities["entity"])
	}
}

func TestUnmarshalIgnoresUnknownFields(t *testing.T) {
	type S struct {
		Name string `yaml:"name"`
	}
	input := `
name: hello
unknown_field: whatever
another: 123
`
	var s S
	if err := Unmarshal([]byte(input), &s); err != nil {
		t.Fatal(err)
	}
	if s.Name != "hello" {
		t.Errorf("Name = %q", s.Name)
	}
}

func TestUnmarshalDashTag(t *testing.T) {
	type S struct {
		Name   string `yaml:"name"`
		Secret string `yaml:"-"`
	}
	input := `
name: hello
`
	var s S
	s.Secret = "keep"
	if err := Unmarshal([]byte(input), &s); err != nil {
		t.Fatal(err)
	}
	if s.Secret != "keep" {
		t.Errorf("Secret should be unchanged, got %q", s.Secret)
	}
}

func TestUnmarshalEmptyInput(t *testing.T) {
	m, err := UnmarshalMap([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestUnmarshalFloat(t *testing.T) {
	type S struct {
		Rate float64 `yaml:"rate"`
	}
	input := `rate: 3.14`
	var s S
	if err := Unmarshal([]byte(input), &s); err != nil {
		t.Fatal(err)
	}
	if s.Rate != 3.14 {
		t.Errorf("Rate = %f", s.Rate)
	}
}

func TestUnmarshalMapInterface(t *testing.T) {
	// This simulates the frontmatter use-case: parsing into map[string]interface{}.
	input := `
title: Chicken Tikka Masala
description: A classic Indian dish
prep_time: PT30M
cook_time: PT45M
servings: 4
recipe_category: Main Course
cuisine: Indian
`
	var fields map[string]interface{}
	if err := Unmarshal([]byte(input), &fields); err != nil {
		t.Fatal(err)
	}
	if fields["title"] != "Chicken Tikka Masala" {
		t.Errorf("title = %v", fields["title"])
	}
	if fields["servings"] != 4 {
		t.Errorf("servings = %v (%T)", fields["servings"], fields["servings"])
	}
	if fields["cuisine"] != "Indian" {
		t.Errorf("cuisine = %v", fields["cuisine"])
	}
}

// ---------- Full pssg.yaml integration test ----------

func TestFullPssgConfig(t *testing.T) {
	// Minimal reproduction of the actual pssg.yaml structure.
	input := `
site:
  name: "Claude Chef"
  base_url: "https://claudechef.com"
  description: "Delicious, tested recipes with AI-powered cooking guidance."
  language: "en"
  version: "0.3.0"
  author: "Grey Newell"

paths:
  data: "/home/user/recipes"
  templates: "templates"
  output: "docs"
  cache: ".cache"
  static: ""

data:
  format: "markdown"
  entity_type: "recipe"
  entity_slug:
    source: "filename"
  body_sections:
    - name: "ingredients"
      header: "Ingredients"
      type: "unordered_list"
    - name: "instructions"
      header: "Instructions"
      type: "ordered_list"

taxonomies:
  - name: category
    label: Categories
    label_singular: Category
    field: recipe_category
    multi_value: false
    min_entities: 1
    letter_page_threshold: 50
    hub_title: "{{.Name}} Recipes"
  - name: cuisine
    label: Cuisines
    label_singular: Cuisine
    field: cuisine
    multi_value: false
    min_entities: 1
    letter_page_threshold: 50

pagination:
  entities_per_page: 48

structured_data:
  entity_type: "Recipe"
  field_mappings:
    name: "title"
    description: "description"
  extra_keywords:
    - "Claude Chef"
    - "AI Cooking"
  date_published: "2025-01-01"
  homepage_schemas:
    - "WebSite"
    - "ItemList"
  entity_schemas:
    - "Recipe"
    - "BreadcrumbList"

rss:
  enabled: true
  main_feed: "feed.xml"
  category_feeds: true
  category_taxonomy: "category"

robots:
  allow_all: true
  extra_bots:
    - "GPTBot"
    - "ClaudeBot"

sitemap:
  max_urls_per_file: 50000
  priorities:
    homepage: "1.0"
    entity: "0.8"
  change_freqs:
    homepage: "daily"
    entity: "weekly"

templates:
  entity: "recipe.html"
  homepage: "index.html"

output:
  clean_build: false
  minify: false
  extract_css: "styles.css"
  extract_js: "main.js"
`

	// ---- Struct-based Unmarshal ----
	type EntitySlug struct {
		Source string `yaml:"source"`
	}
	type BodySection struct {
		Name   string `yaml:"name"`
		Header string `yaml:"header"`
		Type   string `yaml:"type"`
	}
	type SiteConfig struct {
		Name        string `yaml:"name"`
		BaseURL     string `yaml:"base_url"`
		Description string `yaml:"description"`
		Language    string `yaml:"language"`
		Version     string `yaml:"version"`
		Author      string `yaml:"author"`
	}
	type PathsConfig struct {
		Data      string `yaml:"data"`
		Templates string `yaml:"templates"`
		Output    string `yaml:"output"`
		Cache     string `yaml:"cache"`
		Static    string `yaml:"static"`
	}
	type DataConfig struct {
		Format       string        `yaml:"format"`
		EntityType   string        `yaml:"entity_type"`
		EntitySlug   EntitySlug    `yaml:"entity_slug"`
		BodySections []BodySection `yaml:"body_sections"`
	}
	type TaxonomyConfig struct {
		Name                string `yaml:"name"`
		Label               string `yaml:"label"`
		LabelSingular       string `yaml:"label_singular"`
		Field               string `yaml:"field"`
		MultiValue          bool   `yaml:"multi_value"`
		MinEntities         int    `yaml:"min_entities"`
		LetterPageThreshold int    `yaml:"letter_page_threshold"`
		HubTitle            string `yaml:"hub_title"`
	}
	type PaginationConfig struct {
		EntitiesPerPage int `yaml:"entities_per_page"`
	}
	type SchemaConfig struct {
		EntityType     string            `yaml:"entity_type"`
		FieldMappings  map[string]string `yaml:"field_mappings"`
		ExtraKeywords  []string          `yaml:"extra_keywords"`
		DatePublished  string            `yaml:"date_published"`
		HomepageSchema []string          `yaml:"homepage_schemas"`
		EntitySchema   []string          `yaml:"entity_schemas"`
	}
	type RSSConfig struct {
		Enabled          bool   `yaml:"enabled"`
		MainFeed         string `yaml:"main_feed"`
		CategoryFeeds    bool   `yaml:"category_feeds"`
		CategoryTaxonomy string `yaml:"category_taxonomy"`
	}
	type RobotsConfig struct {
		AllowAll  bool     `yaml:"allow_all"`
		ExtraBots []string `yaml:"extra_bots"`
	}
	type SitemapConfig struct {
		MaxURLsPerFile int               `yaml:"max_urls_per_file"`
		Priorities     map[string]string `yaml:"priorities"`
		ChangeFreqs    map[string]string `yaml:"change_freqs"`
	}
	type TemplatesConfig struct {
		Entity   string `yaml:"entity"`
		Homepage string `yaml:"homepage"`
	}
	type OutputConfig struct {
		CleanBuild bool   `yaml:"clean_build"`
		Minify     bool   `yaml:"minify"`
		ExtractCSS string `yaml:"extract_css"`
		ExtractJS  string `yaml:"extract_js"`
	}
	type Config struct {
		Site       SiteConfig       `yaml:"site"`
		Paths      PathsConfig      `yaml:"paths"`
		Data       DataConfig       `yaml:"data"`
		Taxonomies []TaxonomyConfig `yaml:"taxonomies"`
		Pagination PaginationConfig `yaml:"pagination"`
		Schema     SchemaConfig     `yaml:"structured_data"`
		RSS        RSSConfig        `yaml:"rss"`
		Robots     RobotsConfig     `yaml:"robots"`
		Sitemap    SitemapConfig    `yaml:"sitemap"`
		Templates  TemplatesConfig  `yaml:"templates"`
		Output     OutputConfig     `yaml:"output"`
	}

	var cfg Config
	if err := Unmarshal([]byte(input), &cfg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Site
	if cfg.Site.Name != "Claude Chef" {
		t.Errorf("Site.Name = %q", cfg.Site.Name)
	}
	if cfg.Site.BaseURL != "https://claudechef.com" {
		t.Errorf("Site.BaseURL = %q", cfg.Site.BaseURL)
	}
	if cfg.Site.Language != "en" {
		t.Errorf("Site.Language = %q", cfg.Site.Language)
	}

	// Paths
	if cfg.Paths.Data != "/home/user/recipes" {
		t.Errorf("Paths.Data = %q", cfg.Paths.Data)
	}
	if cfg.Paths.Static != "" {
		t.Errorf("Paths.Static = %q, want empty", cfg.Paths.Static)
	}

	// Data nested struct
	if cfg.Data.Format != "markdown" {
		t.Errorf("Data.Format = %q", cfg.Data.Format)
	}
	if cfg.Data.EntitySlug.Source != "filename" {
		t.Errorf("Data.EntitySlug.Source = %q", cfg.Data.EntitySlug.Source)
	}
	if len(cfg.Data.BodySections) != 2 {
		t.Fatalf("BodySections len = %d", len(cfg.Data.BodySections))
	}
	if cfg.Data.BodySections[0].Name != "ingredients" {
		t.Errorf("BodySections[0].Name = %q", cfg.Data.BodySections[0].Name)
	}
	if cfg.Data.BodySections[1].Type != "ordered_list" {
		t.Errorf("BodySections[1].Type = %q", cfg.Data.BodySections[1].Type)
	}

	// Taxonomies (slice of structs)
	if len(cfg.Taxonomies) != 2 {
		t.Fatalf("Taxonomies len = %d", len(cfg.Taxonomies))
	}
	if cfg.Taxonomies[0].Name != "category" {
		t.Errorf("[0].Name = %q", cfg.Taxonomies[0].Name)
	}
	if cfg.Taxonomies[0].MultiValue != false {
		t.Errorf("[0].MultiValue = %v", cfg.Taxonomies[0].MultiValue)
	}
	if cfg.Taxonomies[0].MinEntities != 1 {
		t.Errorf("[0].MinEntities = %d", cfg.Taxonomies[0].MinEntities)
	}
	if cfg.Taxonomies[0].LetterPageThreshold != 50 {
		t.Errorf("[0].LetterPageThreshold = %d", cfg.Taxonomies[0].LetterPageThreshold)
	}
	if cfg.Taxonomies[0].HubTitle != "{{.Name}} Recipes" {
		t.Errorf("[0].HubTitle = %q", cfg.Taxonomies[0].HubTitle)
	}

	// Pagination
	if cfg.Pagination.EntitiesPerPage != 48 {
		t.Errorf("Pagination.EntitiesPerPage = %d", cfg.Pagination.EntitiesPerPage)
	}

	// StructuredData
	if cfg.Schema.EntityType != "Recipe" {
		t.Errorf("Schema.EntityType = %q", cfg.Schema.EntityType)
	}
	if cfg.Schema.FieldMappings["name"] != "title" {
		t.Errorf("Schema.FieldMappings[name] = %q", cfg.Schema.FieldMappings["name"])
	}
	if len(cfg.Schema.ExtraKeywords) != 2 || cfg.Schema.ExtraKeywords[0] != "Claude Chef" {
		t.Errorf("Schema.ExtraKeywords = %v", cfg.Schema.ExtraKeywords)
	}
	if len(cfg.Schema.HomepageSchema) != 2 || cfg.Schema.HomepageSchema[1] != "ItemList" {
		t.Errorf("Schema.HomepageSchema = %v", cfg.Schema.HomepageSchema)
	}
	if cfg.Schema.DatePublished != "2025-01-01" {
		t.Errorf("Schema.DatePublished = %q", cfg.Schema.DatePublished)
	}

	// RSS
	if cfg.RSS.Enabled != true {
		t.Errorf("RSS.Enabled = %v", cfg.RSS.Enabled)
	}
	if cfg.RSS.MainFeed != "feed.xml" {
		t.Errorf("RSS.MainFeed = %q", cfg.RSS.MainFeed)
	}

	// Robots
	if cfg.Robots.AllowAll != true {
		t.Errorf("Robots.AllowAll = %v", cfg.Robots.AllowAll)
	}
	if len(cfg.Robots.ExtraBots) != 2 || cfg.Robots.ExtraBots[0] != "GPTBot" {
		t.Errorf("Robots.ExtraBots = %v", cfg.Robots.ExtraBots)
	}

	// Sitemap
	if cfg.Sitemap.MaxURLsPerFile != 50000 {
		t.Errorf("Sitemap.MaxURLsPerFile = %d", cfg.Sitemap.MaxURLsPerFile)
	}
	if cfg.Sitemap.Priorities["homepage"] != "1.0" {
		t.Errorf("Sitemap.Priorities = %v", cfg.Sitemap.Priorities)
	}
	if cfg.Sitemap.ChangeFreqs["homepage"] != "daily" {
		t.Errorf("Sitemap.ChangeFreqs = %v", cfg.Sitemap.ChangeFreqs)
	}

	// Templates
	if cfg.Templates.Entity != "recipe.html" {
		t.Errorf("Templates.Entity = %q", cfg.Templates.Entity)
	}

	// Output
	if cfg.Output.CleanBuild != false {
		t.Errorf("Output.CleanBuild = %v", cfg.Output.CleanBuild)
	}
	if cfg.Output.ExtractCSS != "styles.css" {
		t.Errorf("Output.ExtractCSS = %q", cfg.Output.ExtractCSS)
	}
}

// ---------- Frontmatter integration test ----------

func TestFrontmatterParsing(t *testing.T) {
	// Simulates what markdown.go does: parse frontmatter into map[string]interface{}.
	input := `
title: Chicken Tikka Masala
description: A creamy, spiced tomato sauce with tender chicken
prep_time: PT30M
cook_time: PT45M
servings: 4
recipe_category: Main Course
cuisine: Indian
author: Grey Newell
image: chicken-tikka-masala.jpg
keywords: chicken, tikka, masala, indian, curry
skill_level: Intermediate
flavors:
  - Savory
  - Spicy
  - Creamy
tools:
  - Skillet
  - Oven
`
	var fields map[string]interface{}
	if err := Unmarshal([]byte(input), &fields); err != nil {
		t.Fatal(err)
	}
	if fields["title"] != "Chicken Tikka Masala" {
		t.Errorf("title = %v", fields["title"])
	}
	if fields["servings"] != 4 {
		t.Errorf("servings = %v (%T)", fields["servings"], fields["servings"])
	}
	flavors := fields["flavors"].([]interface{})
	if len(flavors) != 3 {
		t.Fatalf("flavors len = %d", len(flavors))
	}
	if flavors[0] != "Savory" {
		t.Errorf("flavors[0] = %v", flavors[0])
	}
	tools := fields["tools"].([]interface{})
	if len(tools) != 2 || tools[0] != "Skillet" {
		t.Errorf("tools = %v", tools)
	}
}

// ---------- Edge cases ----------

func TestColonInValue(t *testing.T) {
	input := `url: "https://example.com:8080/path"`
	m := mustUnmarshalMap(t, input)
	if m["url"] != "https://example.com:8080/path" {
		t.Errorf("url = %v", m["url"])
	}
}

func TestColonInUnquotedValue(t *testing.T) {
	input := `url: https://example.com`
	m := mustUnmarshalMap(t, input)
	if m["url"] != "https://example.com" {
		t.Errorf("url = %v", m["url"])
	}
}

func TestTemplateStringsInValues(t *testing.T) {
	input := `hub_title: "{{.Name}} Recipes"`
	m := mustUnmarshalMap(t, input)
	if m["hub_title"] != "{{.Name}} Recipes" {
		t.Errorf("hub_title = %v", m["hub_title"])
	}
}

func TestURLTemplateWithBraces(t *testing.T) {
	input := `url_template: "https://www.amazon.com/s?k={{term}}&tag={{tag}}"`
	m := mustUnmarshalMap(t, input)
	if m["url_template"] != "https://www.amazon.com/s?k={{term}}&tag={{tag}}" {
		t.Errorf("url_template = %v", m["url_template"])
	}
}

func TestNullValue(t *testing.T) {
	input := `nothing: null`
	m := mustUnmarshalMap(t, input)
	if m["nothing"] != nil {
		t.Errorf("nothing = %v, want nil", m["nothing"])
	}
}

func TestNonPointerError(t *testing.T) {
	var s struct{}
	err := Unmarshal([]byte("name: hello"), s)
	if err == nil {
		t.Error("expected error for non-pointer")
	}
}

func TestListItemsWithQuotes(t *testing.T) {
	input := `
items:
  - "quoted item"
  - 'single quoted'
  - unquoted item
`
	m := mustUnmarshalMap(t, input)
	items := m["items"].([]interface{})
	if items[0] != "quoted item" {
		t.Errorf("[0] = %v", items[0])
	}
	if items[1] != "single quoted" {
		t.Errorf("[1] = %v", items[1])
	}
	if items[2] != "unquoted item" {
		t.Errorf("[2] = %v", items[2])
	}
}

func TestListOfMapsWithProviders(t *testing.T) {
	// This mirrors the affiliates.providers section.
	type Provider struct {
		Name          string `yaml:"name"`
		URLTemplate   string `yaml:"url_template"`
		EnvVar        string `yaml:"env_var"`
		AlwaysInclude bool   `yaml:"always_include"`
	}
	type Aff struct {
		Providers       []Provider `yaml:"providers"`
		SearchTermPaths []string   `yaml:"search_term_paths"`
	}
	input := `
providers:
  - name: "Amazon"
    url_template: "https://www.amazon.com/s?k={{term}}&tag={{tag}}"
    env_var: "AMAZON_AFFILIATE_TAG"
  - name: "Walmart"
    url_template: "https://www.walmart.com/search?q={{term}}"
    always_include: true
search_term_paths:
  - "ingredients[].searchTerm"
  - "gear[].searchTerm"
`
	var aff Aff
	if err := Unmarshal([]byte(input), &aff); err != nil {
		t.Fatal(err)
	}
	if len(aff.Providers) != 2 {
		t.Fatalf("providers len = %d", len(aff.Providers))
	}
	if aff.Providers[0].Name != "Amazon" {
		t.Errorf("[0].Name = %q", aff.Providers[0].Name)
	}
	if aff.Providers[0].URLTemplate != "https://www.amazon.com/s?k={{term}}&tag={{tag}}" {
		t.Errorf("[0].URLTemplate = %q", aff.Providers[0].URLTemplate)
	}
	if aff.Providers[1].AlwaysInclude != true {
		t.Errorf("[1].AlwaysInclude = %v", aff.Providers[1].AlwaysInclude)
	}
	if len(aff.SearchTermPaths) != 2 {
		t.Fatalf("search_term_paths len = %d", len(aff.SearchTermPaths))
	}
	if aff.SearchTermPaths[0] != "ingredients[].searchTerm" {
		t.Errorf("SearchTermPaths[0] = %q", aff.SearchTermPaths[0])
	}
}

func TestOnlyComments(t *testing.T) {
	input := `
# just comments
# nothing else
`
	m, err := UnmarshalMap([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestMixedIndentLevels(t *testing.T) {
	input := `
a:
  b:
    c: deep
  d: shallow
e: top
`
	m := mustUnmarshalMap(t, input)
	a := m["a"].(map[string]interface{})
	b := a["b"].(map[string]interface{})
	if b["c"] != "deep" {
		t.Errorf("a.b.c = %v", b["c"])
	}
	if a["d"] != "shallow" {
		t.Errorf("a.d = %v", a["d"])
	}
	if m["e"] != "top" {
		t.Errorf("e = %v", m["e"])
	}
}
