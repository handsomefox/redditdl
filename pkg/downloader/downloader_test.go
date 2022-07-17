package downloader

import (
	"os"
	"testing"
)

func TestDownload(t *testing.T) {
	cfg := DownloaderSettings{
		Verbose:      false,
		ShowProgress: false,
		IncludeVideo: false,
		Subreddit:    "wallpaper",
		Sorting:      "best",
		Timeframe:    "all",
		Directory:    os.TempDir(),
		Count:        5,
		MinWidth:     0,
		MinHeight:    0,
	}

	dl := New(cfg, Filters)
	count, err := dl.Download()
	if err != nil {
		t.Fatalf("Download(%#v) error: %v", cfg, err)
	}

	if count != uint32(cfg.Count) {
		t.Fatalf("Download(%#v) loaded %v media, expected %v", cfg, count, cfg.Count)
	}
}
