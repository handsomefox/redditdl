package fetch_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/handsomefox/redditdl/fetch"
)

func TestNewClient(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want *http.Client
	}{
		{},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fetch.NewClient(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsURL(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "http://",
			args: args{
				str: "http://",
			},
			want: false,
		},
		{
			name: "google.com",
			args: args{
				str: "google.com",
			},
			want: false,
		},
		{
			name: "google",
			args: args{
				str: "google",
			},
			want: false,
		},
		{
			name: "www.google",
			args: args{
				str: "www.google",
			},
			want: false,
		},
		{
			name: "http://google.com",
			args: args{
				str: "http://google.com",
			},
			want: true,
		},
		{
			name: "https://google.com",
			args: args{
				str: "https://google.com",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fetch.IsURL(tt.args.str); got != tt.want {
				t.Errorf("IsURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
