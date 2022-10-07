package filter

import (
	"github.com/handsomefox/redditdl/configuration"
	"github.com/handsomefox/redditdl/structs"
	"github.com/handsomefox/redditdl/utils"
)

func Default() []Filter {
	return []Filter{
		WidthHeight(),
		URLs(),
		Orientation(),
	}
}

// Filter is an interface that filters the given item and returns the result of filtering (true/false).
type Filter interface {
	// Filters returns whether the applied filters say that the item should be filtered out.
	Filters(structs.Content, *configuration.Data) bool
}

// FilterFunc implements filter interface and expects the function to return a boolean.
type FilterFunc func(structs.Content, *configuration.Data) bool

func (f FilterFunc) Filters(c structs.Content, d *configuration.Data) bool {
	return f(c, d)
}

// WidthHeight filters images by specified width and height from settings.
func WidthHeight() FilterFunc {
	return func(item structs.Content, cfg *configuration.Data) bool {
		if item.Width >= cfg.MinWidth && item.Height >= cfg.MinHeight {
			return false
		}

		return true
	}
}

// URLs filters out invalid URLs.
func URLs() FilterFunc {
	return func(item structs.Content, cfg *configuration.Data) bool {
		if len(item.URL) > 0 && utils.IsURL(item.URL) {
			return false
		}

		return true
	}
}

// Orientation filters images by specified orientation.
func Orientation() FilterFunc {
	return func(item structs.Content, cfg *configuration.Data) bool {
		if cfg.Orientation == "" || len(cfg.Orientation) > 1 {
			return false
		}

		var landscape, portrait bool
		if cfg.Orientation == "l" {
			landscape = true
		} else if cfg.Orientation == "p" {
			portrait = true
		}

		if landscape && item.Width > item.Height {
			return false
		}
		if portrait && item.Width < item.Height {
			return false
		}

		return true
	}
}
