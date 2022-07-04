package utils

import "errors"

var (
	ErrInvalidFilenameSyntax = errors.New("invalid filename syntax")
	ErrEmptyFilename         = errors.New("empty filename")
)
