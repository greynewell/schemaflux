package pass

import (
	"encoding/json"
	"log"
	"os"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
)

// FavoritesPass reads the favorites JSON file and resolves p.Favorites.
type FavoritesPass struct{}

func (f *FavoritesPass) Name() string        { return "Favorites" }
func (f *FavoritesPass) Requires() []string { return []string{"SlugResolution"} }

func (f *FavoritesPass) Run(p *ir.Program) error {
	path := p.Config.Extra.Favorites
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Warning: failed to load favorites: %v", err)
		return nil
	}

	var slugs []string
	if err := json.Unmarshal(data, &slugs); err != nil {
		log.Printf("Warning: failed to parse favorites: %v", err)
		return nil
	}

	for _, slug := range slugs {
		if re, ok := p.EntityBySlug[slug]; ok {
			p.Favorites = append(p.Favorites, re)
		}
	}

	if len(p.Favorites) > 0 {
		log.Printf("Resolved %d favorites", len(p.Favorites))
	}

	return nil
}
