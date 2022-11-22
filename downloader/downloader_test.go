package downloader_test

import (
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/handsomefox/redditdl/downloader"
	"github.com/handsomefox/redditdl/downloader/configuration"
	"github.com/handsomefox/redditdl/downloader/filter"
)

func TestDownload(t *testing.T) {
	t.Parallel()

	cfg := configuration.Config{
		Directory:    os.TempDir(),
		Subreddit:    "wallpaper",
		Sorting:      "best",
		Timeframe:    "all",
		Orientation:  "",
		Count:        25,
		MinWidth:     0,
		MinHeight:    0,
		WorkerCount:  configuration.DefaultWorkerCount,
		SleepTime:    configuration.DefaultSleepTime,
		Verbose:      true,
		ShowProgress: true,
		ContentType:  configuration.ContentAny,
	}

	client := downloader.New(&cfg, filter.Default()...)
	stats := client.Download()

	if len(stats.Errors()) != 0 {
		t.Fatalf("Download(%#v) errors: %v", cfg, stats.Errors())
	}
	if stats.Finished() != cfg.Count {
		t.Fatalf("Download(%#v) loaded %v media, expected %v", cfg, stats.Finished(), cfg.Count)
	}
}

func setupConfig(count int64) configuration.Config {
	return configuration.Config{
		Directory:    strconv.Itoa(int(count)),
		Subreddit:    "wallpaper",
		Sorting:      "best",
		Timeframe:    "all",
		Orientation:  "",
		Count:        count,
		MinWidth:     0,
		MinHeight:    0,
		WorkerCount:  configuration.DefaultWorkerCount,
		SleepTime:    configuration.DefaultSleepTime,
		Verbose:      false,
		ShowProgress: false,
		ContentType:  configuration.ContentImages,
	}
}

func BenchmarkDownload1(b *testing.B) {
	cfg := setupConfig(1)
	client := downloader.New(&cfg, filter.Default()...)
	for i := 0; i < b.N; i++ {
		cfg.Directory = path.Join(os.TempDir(), strconv.Itoa(i))
		if stats := client.Download(); len(stats.Errors()) != 0 {
			for _, err := range stats.Errors() {
				b.Error(err)
			}
		}
	}
}

func BenchmarkDownload10(b *testing.B) {
	cfg := setupConfig(10)
	client := downloader.New(&cfg, filter.Default()...)
	for i := 0; i < b.N; i++ {
		cfg.Directory = path.Join(os.TempDir(), strconv.Itoa(i))
		if stats := client.Download(); len(stats.Errors()) != 0 {
			for _, err := range stats.Errors() {
				b.Error(err)
			}
		}
	}
}

func BenchmarkDownload25(b *testing.B) {
	cfg := setupConfig(25)
	client := downloader.New(&cfg, filter.Default()...)
	for i := 0; i < b.N; i++ {
		cfg.Directory = path.Join(os.TempDir(), strconv.Itoa(i))
		if stats := client.Download(); len(stats.Errors()) != 0 {
			for _, err := range stats.Errors() {
				b.Error(err)
			}
		}
	}
}

func BenchmarkDownload50(b *testing.B) {
	cfg := setupConfig(50)
	client := downloader.New(&cfg, filter.Default()...)
	for i := 0; i < b.N; i++ {
		cfg.Directory = path.Join(os.TempDir(), strconv.Itoa(i))
		if stats := client.Download(); len(stats.Errors()) != 0 {
			for _, err := range stats.Errors() {
				b.Error(err)
			}
		}
	}
}

func BenchmarkDownload100(b *testing.B) {
	cfg := setupConfig(100)
	client := downloader.New(&cfg, filter.Default()...)
	for i := 0; i < b.N; i++ {
		cfg.Directory = path.Join(os.TempDir(), strconv.Itoa(i))
		if stats := client.Download(); len(stats.Errors()) != 0 {
			for _, err := range stats.Errors() {
				b.Error(err)
			}
		}
	}
}
