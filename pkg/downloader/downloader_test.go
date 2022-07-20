package downloader

import (
	"os"
	"path"
	"strconv"
	"testing"
)

func TestDownload(t *testing.T) {
	s := Settings{
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

	count, err := Download(s, Filters)
	if err != nil {
		t.Fatalf("Download(%#v) error: %v", s, err)
	}

	if count != int64(s.Count) {
		t.Fatalf("Download(%#v) loaded %v media, expected %v", s, count, s.Count)
	}
}

func BenchmarkDownload(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s := Settings{
			Verbose:      false,
			ShowProgress: false,
			IncludeVideo: true,
			Subreddit:    "wallpaper",
			Sorting:      "best",
			Timeframe:    "all",
			Directory:    path.Join(os.TempDir(), strconv.Itoa(i)),
			Count:        35,
			MinWidth:     1920,
			MinHeight:    1080,
		}
		Download(s, Filters)
	}
}
