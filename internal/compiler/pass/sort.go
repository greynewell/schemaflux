package pass

import (
	"sort"
	"time"

	"github.com/greynewell/schemaflux/internal/compiler/ir"
)

// SortPass sorts entities according to the config sort settings.
type SortPass struct{}

func (s *SortPass) Name() string { return "Sort" }

func (s *SortPass) Run(p *ir.Program) error {
	cfg := p.Config.Sort
	if cfg.Field == "" {
		return nil
	}

	order := cfg.Order
	if order == "" {
		order = "asc"
	}
	desc := order == "desc"

	sort.SliceStable(p.Entities, func(i, j int) bool {
		a := p.Entities[i].Raw.GetString(cfg.Field)
		b := p.Entities[j].Raw.GetString(cfg.Field)

		aEmpty := a == ""
		bEmpty := b == ""
		if aEmpty && bEmpty {
			return false
		}
		if aEmpty {
			return false
		}
		if bEmpty {
			return true
		}

		aTime, aErr := time.Parse("2006-01-02", a)
		bTime, bErr := time.Parse("2006-01-02", b)
		if aErr == nil && bErr == nil {
			if desc {
				return aTime.After(bTime)
			}
			return aTime.Before(bTime)
		}

		if desc {
			return a > b
		}
		return a < b
	})
	return nil
}
