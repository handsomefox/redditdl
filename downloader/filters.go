package downloader

import "github.com/handsomefox/redditdl/internal/utils"

func DefaultFilters() []Filter {
	return []Filter{
		FilterWidthHeight(),
		FilterURLs(),
		FilterOrientation(),
	}
}

// Filter is an interface that filters the given slice and returns the mutated version of it.
type Filter interface {
	Filter([]Content, *Settings) []Content
}

// FilterFunc implements filter interface and expects the function to return a new slice.
type FilterFunc func([]Content, *Settings) []Content

func (f FilterFunc) Filter(c []Content, s *Settings) []Content {
	return f(c, s)
}

// FilterWidthHeight filters images by specified width and height from settings.
func FilterWidthHeight() FilterFunc {
	return func(c []Content, s *Settings) []Content {
		f := make([]Content, 0)
		for _, m := range c {
			if m.Width >= s.MinWidth && m.Height >= s.MinHeight {
				f = append(f, m)
			}
		}

		return f
	}
}

// FilterURLs filters out invalid URLs.
func FilterURLs() FilterFunc {
	return func(c []Content, s *Settings) []Content {
		f := make([]Content, 0)
		for _, m := range c {
			if len(m.URL) > 0 && utils.IsURL(m.URL) {
				f = append(f, m)
			}
		}

		return f
	}
}

// FilterOrientation filters images by specified orientation.
func FilterOrientation() FilterFunc {
	return func(c []Content, s *Settings) []Content {
		if s.Orientation == "" || len(s.Orientation) > 1 {
			return c
		}

		var landscape, portrait bool
		if s.Orientation == "l" {
			landscape = true
		} else if s.Orientation == "p" {
			portrait = true
		}

		f := make([]Content, 0)
		for _, m := range c {
			if landscape && m.Width > m.Height {
				f = append(f, m)
			} else if portrait && m.Width < m.Height {
				f = append(f, m)
			}
		}

		return f
	}
}

// applyFilters applies every filter from the slice of []Filter and returns the mutated slice
// if there are no filters, the original slice is returned.
func applyFilters(s *Settings, c []Content, fs []Filter) []Content {
	if len(fs) == 0 { // return the original posts if there are no filters
		return c
	}

	f := make([]Content, 0, len(c))
	f = append(f, c...)

	for _, ff := range fs {
		f = ff.Filter(f, s)
	}

	return f
}
