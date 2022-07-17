package downloader

import "redditdl/pkg/utils"

// You can mutate this slice to contain your own filters.
var Filters = []Filter{whFilter, urlFilter}

// Interface that filters the given slice and returns the mutated version of it.
type Filter interface {
	Filter([]toDownload, *DownloaderSettings) []toDownload
}

// []downloadable according to its own logic.
// FilterFunc implements filter interface and expects the function to return a new slice.
type FilterFunc func([]toDownload, *DownloaderSettings) []toDownload

func (f FilterFunc) Filter(d []toDownload, c *DownloaderSettings) []toDownload {
	return f(d, c)
}

var (
	whFilter FilterFunc = func(media []toDownload, c *DownloaderSettings) []toDownload {
		f := make([]toDownload, 0)
		for _, m := range media {
			if m.Data.Width >= c.MinWidth && m.Data.Height >= c.MinHeight {
				f = append(f, m)
			}
		}
		return f
	}

	urlFilter FilterFunc = func(media []toDownload, c *DownloaderSettings) []toDownload {
		f := make([]toDownload, 0)
		for _, m := range media {
			if len(m.Data.URL) > 0 && utils.IsURL(m.Data.URL) {
				f = append(f, m)
			}
		}
		return f
	}
)
