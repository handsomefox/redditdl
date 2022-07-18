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

func (f FilterFunc) Filter(td []toDownload, ds *DownloaderSettings) []toDownload {
	return f(td, ds)
}

var (
	whFilter FilterFunc = func(td []toDownload, ds *DownloaderSettings) []toDownload {
		f := make([]toDownload, 0)
		for _, m := range td {
			if m.Data.Width >= ds.MinWidth && m.Data.Height >= ds.MinHeight {
				f = append(f, m)
			}
		}
		return f
	}

	urlFilter FilterFunc = func(td []toDownload, ds *DownloaderSettings) []toDownload {
		f := make([]toDownload, 0)
		for _, m := range td {
			if len(m.Data.URL) > 0 && utils.IsURL(m.Data.URL) {
				f = append(f, m)
			}
		}
		return f
	}
)
