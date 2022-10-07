package downloader_test

import (
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/handsomefox/redditdl/configuration"
	"github.com/handsomefox/redditdl/downloader"
	"github.com/handsomefox/redditdl/filter"
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
		ContentType:  configuration.MediaAny,
	}

	client := downloader.New(&cfg, filter.Default()...)
	stats := client.Download()

	if len(stats.Errors) != 0 {
		t.Fatalf("Download(%#v) errors: %v", cfg, stats.Errors)
	}
	if stats.Finished.Load() != cfg.Count {
		t.Fatalf("Download(%#v) loaded %v media, expected %v", cfg, stats.Finished.Load(), cfg.Count)
	}
}

func BenchmarkDownload(b *testing.B) {
	b.StopTimer()

	cfg := configuration.Config{
		Directory:    "",
		Subreddit:    "wallpaper",
		Sorting:      "best",
		Timeframe:    "all",
		Orientation:  "",
		Count:        35,
		MinWidth:     1920,
		MinHeight:    1080,
		WorkerCount:  configuration.DefaultWorkerCount,
		SleepTime:    configuration.DefaultSleepTime,
		Verbose:      false,
		ShowProgress: false,
		ContentType:  configuration.MediaAny,
	}

	client := downloader.New(&cfg, filter.Default()...)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		cfg.Directory = path.Join(os.TempDir(), strconv.Itoa(i))
		if stats := client.Download(); len(stats.Errors) != 0 {
			for _, err := range stats.Errors {
				b.Error(err)
			}
		}
	}
}
