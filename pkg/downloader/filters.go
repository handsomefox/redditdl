package downloader

import (
	"redditdl/pkg/utils"
)

// You can mutate this slice to contain your own filters.
var Filters = []Filter{whFilter, urlFilter, orientationFilter}

// Interface that filters the given slice and returns the mutated version of it.
type Filter interface {
	Filter([]content, *Settings) []content
}

// []downloadable according to its own logic.
// FilterFunc implements filter interface and expects the function to return a new slice.
type FilterFunc func([]content, *Settings) []content

func (f FilterFunc) Filter(c []content, s *Settings) []content {
	return f(c, s)
}

var (
	whFilter FilterFunc = func(c []content, s *Settings) []content {
		f := make([]content, 0)
		for _, m := range c {
			if m.Width >= s.MinWidth && m.Height >= s.MinHeight {
				f = append(f, m)
			}
		}
		return f
	}

	urlFilter FilterFunc = func(c []content, s *Settings) []content {
		f := make([]content, 0)
		for _, m := range c {
			if len(m.URL) > 0 && utils.IsURL(m.URL) {
				f = append(f, m)
			}
		}
		return f
	}
	orientationFilter FilterFunc = func(c []content, s *Settings) []content {
		if s.Orientation == "" || len(s.Orientation) > 1 {
			return c
		}

		var landscape, portrait bool
		if s.Orientation == "l" {
			landscape = true
		} else if s.Orientation == "p" {
			portrait = true
		}

		log.Debugf("%#v, %#v", landscape, portrait)

		f := make([]content, 0)
		for _, m := range c {
			if landscape && m.Width > m.Height {
				f = append(f, m)
			} else if portrait && m.Width < m.Height {
				f = append(f, m)
			}
		}
		return f
	}
)
