package downloader_test

import (
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/handsomefox/redditdl/downloader"
)

func TestDownload(t *testing.T) {
	t.Parallel()

	settings := downloader.Settings{
		Directory:    os.TempDir(),
		Subreddit:    "wallpaper",
		Sorting:      "best",
		Timeframe:    "all",
		Orientation:  "",
		Count:        5,
		MinWidth:     0,
		MinHeight:    0,
		Verbose:      false,
		ShowProgress: false,
		IncludeVideo: false,
	}

	count, err := downloader.Download(&settings, downloader.DefaultFilters())
	if err != nil {
		t.Fatalf("Download(%#v) error: %v", settings, err)
	}

	if count != settings.Count {
		t.Fatalf("Download(%#v) loaded %v media, expected %v", settings, count, settings.Count)
	}
}

func BenchmarkDownload(b *testing.B) {
	settings := downloader.Settings{
		Directory:    "",
		Subreddit:    "wallpaper",
		Sorting:      "best",
		Timeframe:    "all",
		Orientation:  "",
		Count:        35,
		MinWidth:     1920,
		MinHeight:    1080,
		Verbose:      false,
		ShowProgress: false,
		IncludeVideo: true,
	}

	filters := downloader.DefaultFilters()

	for i := 0; i < b.N; i++ {
		settings.Directory = path.Join(os.TempDir(), strconv.Itoa(i))
		if _, err := downloader.Download(&settings, filters); err != nil {
			b.Error(err)
		}
	}
}
