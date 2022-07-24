package downloader

import "redditdl/pkg/utils"

func DefaultFilters() []Filter {
	return []Filter{
		FilterWidthHeight(),
		FilterURLs(),
		FilterOrientation(),
	}
}

// Interface that filters the given slice and returns the mutated version of it.
type Filter interface {
	Filter([]content, *Settings) []content
}

// FilterFunc implements filter interface and expects the function to return a new slice.
type FilterFunc func([]content, *Settings) []content

func (f FilterFunc) Filter(c []content, s *Settings) []content {
	return f(c, s)
}

// FilterWidthHeight filters images by specified width and height from settings.
func FilterWidthHeight() FilterFunc {
	return func(cont []content, settings *Settings) []content {
		filtered := make([]content, 0)

		for _, m := range cont {
			if m.Width >= settings.MinWidth && m.Height >= settings.MinHeight {
				filtered = append(filtered, m)
			}
		}

		return filtered
	}
}

// FilterURLs filters out invalid URLs.
func FilterURLs() FilterFunc {
	return func(cont []content, settings *Settings) []content {
		filtered := make([]content, 0)

		for _, m := range cont {
			if len(m.URL) > 0 && utils.IsURL(m.URL) {
				filtered = append(filtered, m)
			}
		}

		return filtered
	}
}

// FilterOrientation filters images by specified orientation.
func FilterOrientation() FilterFunc {
	return func(cont []content, settings *Settings) []content {
		if settings.Orientation == "" || len(settings.Orientation) > 1 {
			return cont
		}

		var landscape, portrait bool
		if settings.Orientation == "l" {
			landscape = true
		} else if settings.Orientation == "p" {
			portrait = true
		}

		filtered := make([]content, 0)

		for _, m := range cont {
			if landscape && m.Width > m.Height {
				filtered = append(filtered, m)
			} else if portrait && m.Width < m.Height {
				filtered = append(filtered, m)
			}
		}

		return filtered
	}
}

// applyFilters applies every filter from the slice of []Filter and returns the mutated slice
// if there are no filters, the original slice is returned.
func applyFilters(settings *Settings, cont []content, filters []Filter) []content {
	if len(filters) == 0 { // return the original posts if there are no filters
		return cont
	}

	filtered := make([]content, 0, len(cont))
	filtered = append(filtered, cont...)

	for _, ff := range filters {
		filtered = ff.Filter(filtered, settings)
	}

	return filtered
}
