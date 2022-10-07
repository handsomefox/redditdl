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

// FetchError is an errowr which contains data about errors that occurred when fetching data from some url.
type FetchError struct {
	err error
	url string
}

func (e DownloadError) Error() string {
	return fmt.Errorf("downloading file named %v failed: %w", e.filename, e.err).Error()
}

func (e FetchError) Error() string {
	return fmt.Errorf("fetching file from %v failed: %w", e.url, e.err).Error()
}
