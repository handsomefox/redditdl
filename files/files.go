// files is a package with utility-like file functions used in redditdl
package files

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var ErrEmpty = errors.New("empty parameter provided")

// File is the structure that is saved to disk later.
type File struct {
	Name, Extension string
	Data            []byte
}

// Write implements io.Writer.
func (f *File) Write(p []byte) (int, error) {
	f.Data = make([]byte, 0, len(p)) // this saves a lot of memory according to benchmarks
	f.Data = append(f.Data, p...)
	return len(p), nil
}

// New returns a pointer to a new File.
func New(name, ext string, data []byte) *File {
	return &File{
		Name:      name,
		Extension: ext,
		Data:      data,
	}
}

// NewFilename generates a valid filename for the media.
func NewFilename(name, extension string) (string, error) {
	formatted, err := format(name, extension)
	if err != nil {
		return "", fmt.Errorf("%w: failed to create filename (name=%s,ext=%s)", err, name, extension)
	}
	// Resolve duplicates
	for i := 0; Exists(formatted); i++ {
		formatted, err = format("("+strconv.Itoa(i)+") "+name, extension)
		if err != nil {
			return "", fmt.Errorf("%w: failed to create filename (name=%s,ext=%s)", err, name, extension)
		}
	}
	return formatted, nil
}

// Exists returns whether the file exists.
func Exists(filename string) bool {
	f, err := os.Stat(filename)
	if err != nil {
		return os.IsExist(err)
	}
	return !f.IsDir()
}

const MaxFilenameLength = 200

// format ensures that the filename is valid for NTFS and has the right extension.
func format(filename, extension string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("%w: filename can not be empty", ErrEmpty)
	}
	if extension == "" {
		return "", fmt.Errorf("%w: extension can not be empty", ErrEmpty)
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

// Save saves the file to the provided path/filename.
func Save(filename string, b []byte) error {
	if err := os.WriteFile(filename, b, 0o600); err != nil {
		return fmt.Errorf("%w: couldn't write file(name=%s)", err, filename)
	}
	return nil
}
