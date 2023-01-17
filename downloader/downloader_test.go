package downloader_test

import (
	"context"
	"os"
	"testing"

	"github.com/handsomefox/redditdl/client"
	"github.com/handsomefox/redditdl/downloader"
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

	downloaderConfig := &downloader.Config{
		Directory:    os.TempDir(),
		WorkerCount:  downloader.DefaultWorkerCount,
		ShowProgress: false,
		ContentType:  downloader.ContentAny,
	}

	dl := downloader.New(downloaderConfig, clientConfig, downloader.DefaultFilters()...)

	statusCh := dl.Download(context.TODO())

	total := int64(0)

	for message := range statusCh {
		status, err := message.Status, message.Error
		if err != nil {
			t.Log(err)
		}
		t.Log(status)

		if status == downloader.StatusFinished || status == downloader.StatusFailed {
			total++
		}
	}

	if total != clientConfig.Count {
		t.Error("Failed to download requested amount", total, clientConfig.Count)
	}
}

func setupConfig(dir string, count int64) (*downloader.Config, *client.Config) {
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
	downloaderConfig := &downloader.Config{
		Directory:    dir,
		WorkerCount:  downloader.DefaultWorkerCount,
		ShowProgress: false,
		ContentType:  downloader.ContentImages,
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
	dl := downloader.New(dcfg, ccfg, downloader.DefaultFilters()...)

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
	dl := downloader.New(dcfg, ccfg, downloader.DefaultFilters()...)

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
	dl := downloader.New(dcfg, ccfg, downloader.DefaultFilters()...)

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
