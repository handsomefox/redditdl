package downloader

import (
	"net/url"

	"github.com/handsomefox/redditdl/client"
)

// DefaultFilters returns a slice of the filters included in this package.
func DefaultFilters() []Filter {
	return []Filter{
		FilterWidthHeight(),
		FilterInvalidURLs(),
		FilterOrientation(),
	}
}

// IsFiltered returns a boolean that indicates whether applying filters to the given item
// indicate that the item is unwanted.
func IsFiltered(cfg *client.Config, item client.Content, fs ...Filter) bool {
	for _, f := range fs {
		if filtered := f.Filters(item, cfg); filtered {
			return true
		}
	}
	return false
}

// Filter is an interface that filters the given item and returns the result of filtering (true/false).
type Filter interface {
	// Filters returns whether the item should be filtered out.
	Filters(client.Content, *client.Config) bool
}

// FilterFunc implements filter interface and expects the function to return a boolean.
type FilterFunc func(client.Content, *client.Config) bool

// Filters is the implementation of Filter interface.
func (fn FilterFunc) Filters(c client.Content, d *client.Config) bool {
	return fn(c, d)
}

// FilterWidthHeight is a filter that filters images by specified width and height from settings.
func FilterWidthHeight() FilterFunc {
	return func(item client.Content, cfg *client.Config) bool {
		if item.Width >= cfg.MinWidth && item.Height >= cfg.MinHeight {
			return false
		}
		return true
	}
}

// FilterInvalidURLs is a filter that filters out invalid FilterInvalidURLs.
func FilterInvalidURLs() FilterFunc {
	return func(item client.Content, _ *client.Config) bool {
		if len(item.URL) > 0 && isValidURL(item.URL) {
			return false
		}
		return true
	}
}

// FilterOrientation is a filter that filters images by specified orientation.
func FilterOrientation() FilterFunc {
	return func(item client.Content, cfg *client.Config) bool {
		if cfg.Orientation == "" || len(cfg.Orientation) > 1 {
			return false
		}
		var landscape, portrait bool
		switch cfg.Orientation {
		case "l":
			landscape = true
		case "p":
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

// isValidURL checks if the URL is valid.
//
// Example:
//
//	fmt.Println(fetch.isValidURL("www.google.com"))
//	Output: true
//
// Invalid example:
//
//	fmt.Println(fetch.isValidURL("google.com"))
//	Output: false
func isValidURL(str string) bool {
	u, err := url.ParseRequestURI(str)
	return err == nil && u.Host != "" && u.Scheme != ""
}
