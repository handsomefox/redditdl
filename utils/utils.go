package utils

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/handsomefox/redditdl/structs"
)

const (
	clientTimeout = time.Minute
)

// CreateClient returns a pointer to http.Client configured to work with reddit.
func CreateClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
		},
		Timeout: clientTimeout,
	}
}

// CreateFilename generates a valid filename for the media.
func CreateFilename(name, extension string) (string, error) {
	formatted, err := formatFilename(name, extension)
	if err != nil {
		return "", fmt.Errorf("error creating filename (%v): %w", name, err)
	}

	// Resolve duplicates
	for i := 0; FileExists(formatted); i++ {
		formatted, err = formatFilename("("+strconv.Itoa(i)+") "+name, extension)
		if err != nil {
			return "", fmt.Errorf("error creating filename (%v): %w", name, err)
		}
	}

	return formatted, nil
}

// FileExists returns whether the file exists.
func FileExists(filename string) bool {
	f, err := os.Stat(filename)
	if err != nil {
		return os.IsExist(err)
	}

	return !f.IsDir()
}

const MaxFilenameLength = 200

// formatFilename ensures that the filename is valid for NTFS and has the right extension.
func formatFilename(filename, extension string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("empty filename provided")
	}

	if extension == "" {
		return "", fmt.Errorf("empty extension provided")
	}

	filename = removeForbiddenChars(filename)
	extension = removeForbiddenChars(extension)

	totalLength := len(filename) + len(extension) + 1
	if totalLength > MaxFilenameLength {
		requiredLength := MaxFilenameLength - len(extension) - 1
		filename = filename[:requiredLength]
	}

	return filename + "." + extension, nil
}

// removeForbiddenChars removes invalid characters for Linux/Windows filenames.
func removeForbiddenChars(name string) string {
	var (
		forbiddenChars = []string{"/", "<", ">", ":", "\"", "\\", "|", "?", "*", "(", ")"}
		result         = name
	)

	for _, c := range forbiddenChars {
		result = strings.ReplaceAll(result, c, "")
	}

	return result
}

// IsURL checks if the URL is valid.
func IsURL(str string) bool {
	u, err := url.ParseRequestURI(str)

	return err == nil && u.Host != "" && u.Scheme != ""
}

// NavigateToDirectory moves to the provided directory and creates it if necessary.
func NavigateToDirectory(dir string, createDir bool) error {
	if createDir {
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			if !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("error creating a directory, %w", err)
			}
		}
	}

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("error navigating to directory, %w", err)
	}

	return nil
}

func SaveFile(filename string, file *structs.File) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating a file: %w", err)
	}

	r := bytes.NewReader(file.Data)
	if _, err := r.WriteTo(f); err != nil {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("error removing a file: %w", err)
		}

		return fmt.Errorf("erorr writing to a file: %w", err)
	}

	return nil
}
