package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateClient(t *testing.T) {
	client := CreateClient()
	if client == nil {
		t.Error("Failed to create client") // this will never happen
	}
}

func TestRemoveForbiddenChars(t *testing.T) {
	s := ""
	for _, v := range forbiddenChars {
		s += v
	}

	s = removeForbiddenChars(s)
	if s != "" {
		t.Error("removeForbiddenChars() unexpected test result")
	}
}

func TestCreateFilename(t *testing.T) {
	tests := []struct {
		name, extension, want string
	}{
		{"file", "jpg", "file.jpg"},
		{"//file//", "|||extension", "file.extension"},
		{"", "png", ""},
		{"file", "", ""},
	}

	for _, test := range tests {
		got, _ := CreateFilename(test.name, test.extension)
		if got != test.want {
			t.Errorf("CreateFilename(%#v) unexpected result, got: %v, want: %v", test, got, test.want)
		}
	}
}

func TestIsURL(t *testing.T) {
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
		got := IsURL(test.url)
		if got != test.want {
			t.Errorf("TestIsURL(%#v) unexpected result, got: %v, want: %v", test, got, test.want)
		}
	}
}

func TestNavigateToDirectory(t *testing.T) {
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
		err := NavigateToDirectory(test.dir, test.create)

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

	_ = NavigateToDirectory(os.TempDir(), false)
	for i, test := range tests {
		if test.create {
			f, err := os.CreateTemp(os.TempDir(), test.name)
			if err != nil {
				t.Errorf("Couldn't create file, name %v, dir %v", test.name, os.TempDir())
				continue
			}

			test.name = filepath.Base(f.Name())
			tests[i].name = filepath.Base(f.Name())
			defer f.Close()
		}

		if got := FileExists(test.name); got != test.want {
			t.Errorf("FileExists(%#v) unexpected output, want %v, got %v", test, test.want, got)
		}

	}

	for _, test := range tests {
		if test.create {
			os.Remove(test.name)
		}
	}
}

func TestCompareFloat64(t *testing.T) {
	tests := []struct {
		a, b float64
	}{
		{16.0 / 9.0, 1920.0 / 1080.0},
		{16.0 / 9.0, 1280.0 / 720.0},
		{16.0 / 9.0, 2560.0 / 1440.0},
		{16.0 / 10.0, 1280.0 / 800.0},
		{16.0 / 10.0, 1440.0 / 900.0},
		{16.0 / 10.0, 1920.0 / 1200.0},
		{4.0 / 3.0, 640.0 / 480.0},
		{4.0 / 3.0, 800.0 / 600.0},
		{4.0 / 3.0, 1600.0 / 1200.0},
	}

	for _, test := range tests {
		if !CompareFloat64(test.a, test.b) {
			t.Errorf("CompareFloat64(%v, %v) not true", test.a, test.b)
		}
	}
}
