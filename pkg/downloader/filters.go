package downloader

import (
	"redditdl/pkg/config"
	"redditdl/utils"
)

var filters = []filter{resolutionFilter, invalidURLFilter}

type filter interface {
	Filter([]downloadable, *config.Configuration)
}

// filterFunc implements filter interface and expects the function to change the contents of
// []downloadable according to its own logic.
type filterFunc func([]downloadable, *config.Configuration)

func (f filterFunc) Filter(d []downloadable, c *config.Configuration) {
	f(d, c)
}

func applyFilters(original []downloadable, filters []filter, c config.Configuration) {
	logger.Debug("Filtering posts...")
	for _, f := range filters {
		f.Filter(original, &c)
	}
}

var resolutionFilter filterFunc = func(media []downloadable, c *config.Configuration) {
	filtered := make([]downloadable, 0)
	for _, m := range media {
		if m.Data.Width >= c.MinWidth && m.Data.Height >= c.MinHeight {
			filtered = append(filtered, m)
		}
	}
	media = filtered
}

var invalidURLFilter filterFunc = func(media []downloadable, c *config.Configuration) {
	filtered := make([]downloadable, 0)
	for _, m := range media {
		if len(m.Data.URL) > 0 && utils.IsURL(m.Data.URL) {
			filtered = append(filtered, m)
		}
	}
	media = filtered
}
