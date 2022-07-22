package downloader

import (
	"redditdl/pkg/utils"
	"strconv"
	"strings"
)

// You can mutate this slice to contain your own filters.
var Filters = []Filter{whFilter, urlFilter, aspectRatioFilter}

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
	aspectRatioFilter FilterFunc = func(c []content, s *Settings) []content {
		if s.AspectRatio == "" {
			return c
		}

		split := strings.Split(s.AspectRatio, ":")
		if len(split) != 2 {
			return c
		}

		wStr, hStr := split[0], split[1]
		width, err := strconv.ParseFloat(wStr, 64)
		if err != nil {
			return c
		}
		height, err := strconv.ParseFloat(hStr, 64)
		if err != nil {
			return c
		}

		ratio := width / height

		f := make([]content, 0)
		for _, m := range c {
			imgRatio := float64(m.Width) / float64(m.Height)
			if utils.CompareFloat64(ratio, imgRatio) {
				f = append(f, m)
			}
		}
		return f
	}
)
