package util

import (
	"os"
	"path"
	"testing"
)

func TestIsValidURL(t *testing.T) {
	t.Parallel()
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsValidURL(tt.args.str); got != tt.want {
				t.Errorf("IsURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFilename(t *testing.T) {
	t.Parallel()
	type args struct {
		name      string
		extension string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Create file.jpg",
			args: args{
				name:      "file",
				extension: "jpg",
			},
			want:    "file.jpg",
			wantErr: false,
		}, {
			name: "Create a file with invalid characters",
			args: args{
				name:      "/<>:file",
				extension: "/<>:jpg",
			},
			want:    "file.jpg",
			wantErr: false,
		}, {
			name: "Empty name",
			args: args{
				name:      "",
				extension: "jpg",
			},
			want:    "",
			wantErr: true,
		}, {
			name: "Empty extension",
			args: args{
				name:      "file",
				extension: "",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewFilename(tt.args.name, tt.args.extension)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFilename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NewFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExists(t *testing.T) {
	t.Parallel()
	exec, err := os.Executable()
	if err != nil {
		t.Fatalf("couldn't find the running executable")
	}
	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Check existing file",
			args: args{
				filename: exec,
			},
			want: true,
		}, {
			name: "Check non-existing file",
			args: args{
				filename: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Exists(tt.args.filename); got != tt.want {
				t.Errorf("Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNavigateTo(t *testing.T) {
	t.Parallel()
	type args struct {
		dir       string
		createDir bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Navigate to created directory",
			args: args{
				dir:       path.Join(os.TempDir(), "test_dir"),
				createDir: true,
			},
			wantErr: false,
		}, {
			name: "Navigate to non-existing directory",
			args: args{
				dir:       "/<>:",
				createDir: false,
			},
			wantErr: true,
		}, {
			name: "Navigate to existing directory",
			args: args{
				dir:       os.TempDir(),
				createDir: false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := NavigateTo(tt.args.dir, tt.args.createDir); (err != nil) != tt.wantErr {
				t.Errorf("NavigateTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
