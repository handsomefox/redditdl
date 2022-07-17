package utils

import (
	"os"
	"testing"
)

func TestCreateClient(t *testing.T) {
	client := CreateClient()
	if client == nil {
		t.Fatal("Failed to create client") // this will never happen
	}
}

func TestRemoveForbiddenChars(t *testing.T) {
	s := ""
	for _, v := range forbiddenChars {
		s += v
	}

	s = removeForbiddenChars(s)
	if s != "" {
		t.Fatal("removeForbiddenChars() unexpected test result")
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
			t.Fatalf("CreateFilename(%#v) unexpected result, got: %v, want: %v", test, got, test.want)
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
			t.Fatalf("TestIsURL(%#v) unexpected result, got: %v, want: %v", test, got, test.want)
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
			t.Fatalf("NavigateToDirectory(%#v) unexpected result, got: %v, want: %#v", test, err, test.shouldError)
		}

		if !test.shouldError && err != nil {
			t.Fatalf("NavigateToDirectory(%#v) unexpected result, got: %v, want: %#v", test, err, test.shouldError)
		}

	}
}
