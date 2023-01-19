package downloader

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

var ErrEmptyFileParams = errors.New("empty parameters provided, can't create a filename")

// File is the structure that is saved to disk later.
type File struct {
	Name      string
	Extension string
	Data      []byte
}

// NewFile returns a pointer to a new File.
//
// Example:
//
//	f := files.NewFile("image", "jpg", []byte{12,23,54})
func NewFile(name, ext string, data []byte) *File {
	return &File{
		Name:      name,
		Extension: ext,
		Data:      data,
	}
}

// Save saves the file to the provided path/filename.
func (f *File) Save() error {
	filename, err := NewFilename(f.Name, f.Extension)
	if err != nil {
		return fmt.Errorf("%w: couldn't save file", err)
	}
	if err := os.WriteFile(filename, f.Data, 0o600); err != nil {
		return fmt.Errorf("%w: couldn't save file(name=%s)", err, filename)
	}
	return nil
}

// NewFilename generates a valid filename for the media.
//
// It accounts for:
//   - Invalid characters;
//   - Names that exceed allowed length;
//   - Name collisions.
//
// It returns an error when:
//   - Name or Extensions arguments are empty.
func NewFilename(name, extension string) (string, error) {
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

// formatFilename ensures that the filename is valid for NTFS and has the right extension.
func formatFilename(filename, extension string) (string, error) {
	const MaxFilenameLength = 255 // This really only accounts for NTFS.
	if filename == "" {
		return "", fmt.Errorf("%w: filename can not be empty", ErrEmptyFileParams)
	}
	if extension == "" {
		return "", fmt.Errorf("%w: extension can not be empty", ErrEmptyFileParams)
	}

	filename = removeForbiddenChars(filename)
	extension = removeForbiddenChars(extension)

	totalLength := len(filename) + len(extension) + 1
	if totalLength > MaxFilenameLength {
		requiredLength := MaxFilenameLength - len(extension) - 1
		filename = filename[:requiredLength]
	}
	return fmt.Sprintf("%s.%s", filename, extension), nil
}

// removeForbiddenChars removes invalid characters for Linux/Windows filenames.
func removeForbiddenChars(name string) string {
	// Most of the characters are forbidden on Windows only.
	const forbiddenChars = "/<>\":\\|?*"
	for _, c := range forbiddenChars {
		name = strings.ReplaceAll(name, string(c), "")
	}
	return name
}

// NavigateTo moves to the provided directory and creates it if necessary.
func NavigateTo(dir string, createDir bool) error {
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
