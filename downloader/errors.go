package downloader

import "errors"

var (
	ErrNoFileExtension = errors.New("failed to pick a file extension")
	ErrFailedSave      = errors.New("downloader cannot navigate to directory, terminating")
	ErrNoParams        = errors.New("no parameters were provided")
	ErrEmptyFileParams = errors.New("empty parameters provided, can't create a filename")
)
