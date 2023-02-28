package downloader

import (
	"net/url"

	"github.com/handsomefox/redditdl/client/media"
	"github.com/handsomefox/redditdl/cmd/params"
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
func IsFiltered(p *params.CLIParameters, item media.Content, fs ...Filter) bool {
	for _, f := range fs {
		if filtered := f.Filters(item, p); filtered {
			return true
		}
	}
	return false
}

// Filter is an interface that filters the given item and returns the result of filtering (true/false).
type Filter interface {
	// Filters returns whether the item should be filtered out.
	Filters(media.Content, *params.CLIParameters) bool
}

// FilterFunc implements filter interface and expects the function to return a boolean.
type FilterFunc func(media.Content, *params.CLIParameters) bool

// Filters is the implementation of Filter interface.
func (fn FilterFunc) Filters(c media.Content, p *params.CLIParameters) bool {
	return fn(c, p)
}

// FilterWidthHeight is a filter that filters images by specified width and height from settings.
func FilterWidthHeight() FilterFunc {
	return func(item media.Content, p *params.CLIParameters) bool {
		if item.Width >= p.MediaMinWidth && item.Height >= p.MediaMinHeight {
			return false
		}
		return true
	}
}

// FilterInvalidURLs is a filter that filters out invalid FilterInvalidURLs.
func FilterInvalidURLs() FilterFunc {
	return func(item media.Content, p *params.CLIParameters) bool {
		if len(item.URL) > 0 && isValidURL(item.URL) {
			return false
		}
		return true
	}
}

// FilterOrientation is a filter that filters images by specified orientation.
func FilterOrientation() FilterFunc {
	return func(item media.Content, p *params.CLIParameters) bool {
		switch p.MediaOrientation {
		case params.RequiredOrientationAny:
			return false
		case params.RequiredOrientationLandscape:
			if item.Orientation != media.OrientationLandscape {
				return true
			}
		case params.RequiredOrientationPortrait:
			if item.Orientation != media.OrientationPortrait {
				return true
			}
		}
		return false
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
