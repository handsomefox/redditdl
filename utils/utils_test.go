package utils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/handsomefox/redditdl/utils"
)

func TestCreateClient(t *testing.T) {
	t.Parallel()

	if client := utils.CreateClient(); client == nil {
		t.Error("Failed to create client") // this will never happen
	}
}

func TestCreateFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, extension, want string
	}{
		{"file", "jpg", "file.jpg"},
		{"//file//", "|||extension", "file.extension"},
		{"", "png", ""},
		{"file", "", ""},
	}

	for _, test := range tests {
		got, _ := utils.CreateFilename(test.name, test.extension)
		if got != test.want {
			t.Errorf("CreateFilename(%#v) unexpected result, got: %v, want: %v", test, got, test.want)
		}
	}
}

func TestIsURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url  string
		want bool
	}{
		{"google.com", false},
		{"google", false},
		{"www.google", false},
		{"http://google.com", true},
		{"https://google.com", true},
		{"http://", false},
	}

	for _, test := range tests {
		got := utils.IsURL(test.url)
		if got != test.want {
			t.Errorf("TestIsURL(%#v) unexpected result, got: %v, want: %v", test, got, test.want)
		}
	}
}

func TestNavigateToDirectory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		dir         string
		create      bool
		shouldError bool
	}{
		{os.TempDir(), false, false},
		{"dir", true, false},
		{"zzzzzzzzzzzzzzzzz", false, true},
	}

	for _, test := range tests {
		err := utils.NavigateToDirectory(test.dir, test.create)

		if test.shouldError && err == nil {
			t.Errorf("NavigateToDirectory(%#v) unexpected result, got: %v, want: %#v", test, err, test.shouldError)
		}

		if !test.shouldError && err != nil {
			t.Errorf("NavigateToDirectory(%#v) unexpected result, got: %v, want: %#v", test, err, test.shouldError)
		}
	}
}

func TestFileExists(t *testing.T) {
	tests := []struct {
		name         string
		create, want bool
	}{
		{"randomfilename", false, false},
		{"anotherrandomfilename.exe", false, false},
		{"randomfilename", true, true},
		{"anotherrandomfilename.exe", true, true},
	}

	_ = utils.NavigateToDirectory(os.TempDir(), false)

	for index, test := range tests {
		if !test.create {
			continue
		}

		file, err := os.CreateTemp(os.TempDir(), test.name)
		if err != nil {
			t.Errorf("Couldn't create file, name %v, dir %v", test.name, os.TempDir())

			continue
		}

		test.name = filepath.Base(file.Name())
		tests[index].name = filepath.Base(file.Name())

		file.Close()
	}

	for _, test := range tests {
		if got := utils.FileExists(test.name); got != test.want {
			t.Errorf("FileExists(%#v) unexpected output, want %v, got %v", test, test.want, got)
		}
	}

	for _, test := range tests {
		if test.create {
			os.Remove(test.name)
		}
	}
}
