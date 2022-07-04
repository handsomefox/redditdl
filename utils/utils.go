package utils

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidFilenameSyntax = errors.New("invalid filename syntax")
	ErrEmptyFilename         = errors.New("empty filename")
	ErrEmptyExtension        = errors.New("empty extension")
)

const NTFS_MAX_FILENAME_LENGTH = 256

// CreateClient returns a pointer to http.Client configured to work with reddit.
func CreateClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
		},
		Timeout: 60 * time.Second,
	}
}

// CreateFilename generates a valid filename for the image.
func CreateFilename(name, extension string, idx int) (string, error) {
	formatted, err := formatFilename(name, extension)
	if err != nil {
		return "", fmt.Errorf("error creating filename (%v): %v", name, err)
	}

	// Resoulve dupicates
	for i := 0; FileExists(formatted); i++ {
		formatted, err = formatFilename("("+strconv.Itoa(idx)+") "+name, extension)
		if err != nil {
			return "", fmt.Errorf("error creating filename (%v): %v", name, err)
		}
	}

	return formatted, nil
}

// FileExists returns whether the file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return os.IsExist(err)
}

// formatFilename ensures that the filename is valid for NTFS and has the right extension
func formatFilename(filename, extension string) (string, error) {
	if len(filename) == 0 {
		return "", ErrEmptyFilename
	}

	if len(extension) == 0 {
		return "", ErrEmptyExtension
	}

	filename = removeForbiddenChars(filename)

	totalLength := len(filename) + len(extension) + 1
	if totalLength > NTFS_MAX_FILENAME_LENGTH {
		requiredLength := NTFS_MAX_FILENAME_LENGTH - len(extension) - 1
		filename = filename[:requiredLength]
	}

	return filename + "." + extension, nil
}

var forbiddenChars = []string{"/", "<", ">", ":", "\"", "\\", "|", "?", "*"}

// removeForbiddenChars removes invalid charactes for Linux/Windows filenames
func removeForbiddenChars(name string) string {
	result := name
	for _, c := range forbiddenChars {
		result = strings.ReplaceAll(result, c, " ")
	}
	return result
}
