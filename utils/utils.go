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
func CreateFilename(name string, idx int) (string, error) {
	formatted, err := formatFilename(name)
	if err != nil {
		return "", fmt.Errorf("error creating filename (%v): %v", name, err)
	}

	// Resoulve dupicates
	for i := 0; FileExists(formatted); i++ {
		formatted, err = formatFilename("(" + strconv.Itoa(i) + ") " + name)
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
func formatFilename(fullFilename string) (string, error) {
	nameAndExt := strings.Split(fullFilename, ".")
	if len(nameAndExt) != 2 {
		return "", fmt.Errorf("%v: %v", ErrInvalidFilenameSyntax, fullFilename)
	}

	filename := nameAndExt[0]
	extension := nameAndExt[1]

	if len(filename) == 0 {
		return "", ErrEmptyFilename
	}

	totalLength := len(filename) + len(extension) + 1
	if totalLength > NTFS_MAX_FILENAME_LENGTH {
		fmt.Println(totalLength)
		requiredLength := NTFS_MAX_FILENAME_LENGTH - len(extension) - 1
		fmt.Println(requiredLength)
		filename = filename[:requiredLength]
	}

	return filename + "." + extension, nil
}
