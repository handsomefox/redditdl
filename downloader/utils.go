package downloader

import (
	"github.com/handsomefox/redditdl/configuration"
	"github.com/handsomefox/redditdl/filter"
	"github.com/handsomefox/redditdl/structs"
)

// newDownloadError is a handy thing to create errors faster.
func newDownloadError(err error, filename string) *DownloadError {
	return &DownloadError{
		err:      err,
		filename: filename,
	}
}

// newFetchError is a handy thing to create errors faster.
func newFetchError(err error, url string) *FetchError {
	return &FetchError{
		err: err,
		url: url,
	}
}

func isFiltered(cfg *configuration.Data, item structs.Content, filters ...filter.Filter) bool {
	for _, f := range filters {
		if filtered := f.Filters(item, cfg); filtered {
			return true
		}
	}

	return false
}
