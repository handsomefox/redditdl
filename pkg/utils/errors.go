package utils

import (
	"errors"
)

var (
	ErrEmptyFilename  = errors.New("empty filename")
	ErrEmptyExtension = errors.New("empty extension")
)
