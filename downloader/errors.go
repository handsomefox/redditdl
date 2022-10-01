package downloader

import "errors"

var (
	ErrSaveGoroutine    = errors.New("error waiting for the saving goroutine to finish")
	ErrUnexpectedStatus = errors.New("unexpected status in response")
)
