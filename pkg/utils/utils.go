package utils

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// CreateClient returns a pointer to http.Client configured to work with reddit.
func CreateClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
		},
		Timeout: 60 * time.Second,
	}
}

// CreateFilename generates a valid filename for the media.
func CreateFilename(name, extension string) (string, error) {
	formatted, err := formatFilename(name, extension)
	if err != nil {
		return "", fmt.Errorf("error creating filename (%v): %v", name, err)
	}

	// Resolve duplicates
	for i := 0; FileExists(formatted); i++ {
		formatted, err = formatFilename("("+strconv.Itoa(i)+") "+name, extension)
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

const NtfsMaxFilenameLength = 256

// formatFilename ensures that the filename is valid for NTFS and has the right extension
func formatFilename(filename, extension string) (string, error) {
	if len(filename) == 0 {
		return "", fmt.Errorf("empty filename")
	}
	if len(extension) == 0 {
		return "", fmt.Errorf("file should have an extension")
	}
	filename = removeForbiddenChars(filename)
	extension = removeForbiddenChars(extension)

	totalLength := len(filename) + len(extension) + 1
	if totalLength > NtfsMaxFilenameLength {
		requiredLength := NtfsMaxFilenameLength - len(extension) - 1
		filename = filename[:requiredLength]
	}

	return filename + "." + extension, nil
}

var forbiddenChars = []string{"/", "<", ">", ":", "\"", "\\", "|", "?", "*"}

// removeForbiddenChars removes invalid characters for Linux/Windows filenames
func removeForbiddenChars(name string) string {
	result := name
	for _, c := range forbiddenChars {
		result = strings.ReplaceAll(result, c, "")
	}
	return result
}

// IsURL checks if the URL is valid
func IsURL(str string) bool {
	u, err := url.ParseRequestURI(str)
	return err == nil && u.Host != "" && u.Scheme != ""
}

// NavigateToDirectory moves to the provided directory and creates it if necessary.
func NavigateToDirectory(dir string, createDir bool) error {
	if createDir {
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			if !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("error creating a directory, %v", err)
			}
		}
	}

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("error navigating to directory, %v", err)
	}
	return nil
}
