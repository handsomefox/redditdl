package downloader

import (
	"fmt"
)

var (
	_ error = DownloadError{}
	_ error = FetchError{}
)

// DownloadError is an error which contains data about which file failed to download and why.
type DownloadError struct {
	err      error
	filename string
}

// FetchError is an error which contains data about errors that occurred when fetching data from some url.
type FetchError struct {
	err error
	url string
}

func (e DownloadError) Error() string {
	return fmt.Errorf("%w: couldn't download file (name=%s)", e.err, e.filename).Error()
}

func (e FetchError) Error() string {
	return fmt.Errorf("%w: fetching file from %v failed", e.err, e.url).Error()
}

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
