package fetch_test

import (
	"testing"

	"github.com/handsomefox/redditdl/fetch"
)

func TestNewClient(t *testing.T) {
	t.Parallel()
	if client := fetch.NewClient(); client == nil {
		t.Error("Failed to create client") // this will never happen
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
		got := fetch.IsURL(test.url)
		if got != test.want {
			t.Errorf("TestIsURL(%#v) unexpected result, got: %v, want: %v", test, got, test.want)
		}
	}
}
