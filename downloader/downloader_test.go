package downloader

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/handsomefox/redditdl/client"
)

func TestDownload(t *testing.T) {
	t.Parallel()

	clientConfig := &client.Config{
		Subreddit:   "wallpaper",
		Sorting:     "best",
		Timeframe:   "all",
		Orientation: "",
		Count:       25,
		MinWidth:    0,
		MinHeight:   0,
	}

	downloaderConfig := &Config{
		Directory:    os.TempDir(),
		WorkerCount:  DefaultWorkerCount,
		ShowProgress: false,
		ContentType:  ContentAny,
	}

	dl := New(downloaderConfig, clientConfig, DefaultFilters()...)

	statusCh := dl.Download(context.TODO())

	total := int64(0)

	for message := range statusCh {
		status, err := message.Status, message.Error
		if err != nil {
			t.Log(err)
		}
		t.Log(status)

		if status == StatusFinished || status == StatusFailed {
			total++
		}
	}

	if total != clientConfig.Count {
		t.Error("Failed to download requested amount", total, clientConfig.Count)
	}
}

func setupConfig(dir string, count int64) (*Config, *client.Config) {
	os.Setenv("ENVIRONMENT", "PRODUCTION")
	clientConfig := &client.Config{
		Subreddit:   "wallpaper",
		Sorting:     "best",
		Timeframe:   "all",
		Orientation: "",
		Count:       count,
		MinWidth:    0,
		MinHeight:   0,
	}
	downloaderConfig := &Config{
		Directory:    dir,
		WorkerCount:  DefaultWorkerCount,
		ShowProgress: false,
		ContentType:  ContentImages,
	}
	return downloaderConfig, clientConfig
}

func BenchmarkDownload1(b *testing.B) {
	b.StopTimer()

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		b.Fatal(err)
	}

	dcfg, ccfg := setupConfig(dir, 1)
	dl := New(dcfg, ccfg, DefaultFilters()...)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		statusCh := dl.Download(context.TODO())
		for {
			_, more := <-statusCh
			if !more {
				break
			}
		}
	}
	b.StopTimer()
	os.RemoveAll(dir)
}

func BenchmarkDownload25(b *testing.B) {
	b.StopTimer()

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		b.Fatal(err)
	}

	dcfg, ccfg := setupConfig(dir, 25)
	dl := New(dcfg, ccfg, DefaultFilters()...)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		statusCh := dl.Download(context.TODO())
		for {
			_, more := <-statusCh
			if !more {
				break
			}
		}
	}
	b.StopTimer()
	os.RemoveAll(dir)
}

func BenchmarkDownload100(b *testing.B) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		b.Fatal(err)
	}

	dcfg, ccfg := setupConfig(dir, 100)
	dl := New(dcfg, ccfg, DefaultFilters()...)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		statusCh := dl.Download(context.TODO())
		for {
			_, more := <-statusCh
			if !more {
				break
			}
		}
	}
	b.StopTimer()
	os.RemoveAll(dir)
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
			if got := FileExists(tt.args.filename); got != tt.want {
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
			if got := isValidURL(tt.args.str); got != tt.want {
				t.Errorf("IsURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
