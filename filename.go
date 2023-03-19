package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// NewFormattedFilename generates a valid filename for the media.
//
// It accounts for:
//   - Invalid characters;
//   - Names that exceed allowed length;
//   - Name collisions.
//
// It returns an error when:
//   - Name or Extensions arguments are empty.
func NewFormattedFilename(name, extension string) (string, error) {
	formatted, err := formatFilename(name, extension)
	if err != nil {
		return "", fmt.Errorf("%w: failed to create filename (name=%s,ext=%s)", err, name, extension)
	}
	// Resolve duplicates
	for i := 0; FileExists(formatted); i++ {
		formatted, err = formatFilename(fmt.Sprintf("(%d) %s", i, name), extension)
		if err != nil {
			return "", fmt.Errorf("%w: failed to create filename (name=%s,ext=%s)", err, name, extension)
		}
	}

	return formatted, nil
}

// FileExists returns whether the file exists.
func FileExists(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		return false
	}

	return true
}

// ChdirOrCreate moves to the provided directory and creates it if necessary.
func ChdirOrCreate(dir string, createDir bool) error {
	if createDir {
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			if !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("%w: couldn't create directory(name=%s)", err, dir)
			}
		}
	}

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("%w: couldn't navigate to directory(name=%s)", err, dir)
	}

	return nil
}

type FilenameError struct {
	err         error
	explanation string
}

func (se *FilenameError) Error() string {
	if se.err != nil {
		return se.explanation + ": " + se.err.Error()
	}

	return se.explanation
}

// formatFilename ensures that the filename is valid for NTFS and has the right extension.
func formatFilename(filename, extension string) (string, error) {
	const MaxFilenameLength = 255 // This really only accounts for NTFS.

	if filename == "" {
		return "", &FilenameError{
			err:         nil,
			explanation: "filename can not be empty",
		}
	}

	if extension == "" {
		return "", &FilenameError{
			err:         nil,
			explanation: "extension can not be empty",
		}
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
	// Most of the characters are forbidden on Windows only.
	forbiddenChars := []rune{
		'#', '%', '&', '{', '}', '\\', '<', '>', '*', '?', '/', '$',
		'!', '\'', '"', ':', '@', '+', '`', '|', '=',
	}
	for _, c := range forbiddenChars {
		name = strings.ReplaceAll(name, string(c), "")
	}

	return name
}
