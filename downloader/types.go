package downloader

import "errors"

var ErrNoFileExtension = errors.New("failed to pick a file extension")

// ContentType is the type of media which will be queued for download.
type ContentType uint8

const (
	_ ContentType = iota
	ContentImages
	ContentVideos
	ContentAny
)

type DownloadStatus byte

const (
	_ DownloadStatus = iota
	StatusStarted
	StatusFinished
	StatusFailed
)

type StatusMessage struct {
	Error  error
	Status DownloadStatus
}
