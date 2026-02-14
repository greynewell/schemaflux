package build

import (
	"sort"
	"time"

	"github.com/greynewell/pssg/internal/config"
	"github.com/greynewell/pssg/internal/entity"
)

// SortEntities sorts entities according to the sort config.
// If no sort config is set, entities maintain their original order.
func SortEntities(entities []*entity.Entity, cfg config.SortConfig) {
	if cfg.Field == "" {
		return
	}

	order := cfg.Order
	if order == "" {
		order = "asc"
	}
	desc := order == "desc"

	sort.SliceStable(entities, func(i, j int) bool {
		a := entities[i].GetString(cfg.Field)
		b := entities[j].GetString(cfg.Field)

		// Entities missing the sort field sort last
		aEmpty := a == ""
		bEmpty := b == ""
		if aEmpty && bEmpty {
			return false
		}
		if aEmpty {
			return false // a goes after b
		}
		if bEmpty {
			return true // a goes before b
		}

		// Try date parsing (YYYY-MM-DD)
		aTime, aErr := time.Parse("2006-01-02", a)
		bTime, bErr := time.Parse("2006-01-02", b)

		if aErr == nil && bErr == nil {
			if desc {
				return aTime.After(bTime)
			}
			return aTime.Before(bTime)
		}

		// Fall back to string comparison
		if desc {
			return a > b
		}
		return a < b
	})
}
