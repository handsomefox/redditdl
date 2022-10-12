// Package filter is a package that is used to implement functions
// that act upon reddit's content and can filter out
// things that user does not need to download
// depending on content parameters (like resolution)
package filter

import (
	"github.com/handsomefox/redditdl/configuration"
	"github.com/handsomefox/redditdl/fetch"
	"github.com/handsomefox/redditdl/fetch/api"
)

// Default returns a slice of the filters included in this package.
func Default() []Filter {
	return []Filter{
		WidthHeight(),
		URLs(),
		Orientation(),
	}
}

// IsFiltered returns a boolean that indicates whether applying filters to the given item
// indicate that the item is unwanted.
func IsFiltered(cfg *configuration.Config, item api.Content, filters ...Filter) bool {
	for _, f := range filters {
		if filtered := f.Filters(item, cfg); filtered {
			return true
		}
	}
	return false
}

// Filter is an interface that filters the given item and returns the result of filtering (true/false).
type Filter interface {
	// Filters returns whether the item should be filtered out.
	Filters(api.Content, *configuration.Config) bool
}

// DeciderFunc implements filter interface and expects the function to return a boolean.
type DeciderFunc func(api.Content, *configuration.Config) bool

func (fn DeciderFunc) Filters(c api.Content, d *configuration.Config) bool {
	return fn(c, d)
}

// WidthHeight is a filter that filters images by specified width and height from settings.
func WidthHeight() DeciderFunc {
	return func(item api.Content, cfg *configuration.Config) bool {
		if item.Width >= cfg.MinWidth && item.Height >= cfg.MinHeight {
			return false
		}
		return true
	}
}

// URLs is a filter that filters out invalid URLs.
func URLs() DeciderFunc {
	return func(item api.Content, cfg *configuration.Config) bool {
		if len(item.URL) > 0 && fetch.IsURL(item.URL) {
			return false
		}
		return true
	}
}

// Orientation is a filter that filters images by specified orientation.
func Orientation() DeciderFunc {
	return func(item api.Content, cfg *configuration.Config) bool {
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
