package downloader_test

import (
	"context"
	"os"
	"strconv"
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

	finished := int64(0)

	for message := range statusCh {
		status, err := message.Status, message.Error
		if err != nil {
			t.Log(err)
		}
		t.Log(status)

		if status == downloader.StatusFinished {
			finished++
		}
	}

	if finished != clientConfig.Count {
		t.Error("Failed to download requested amount", finished, clientConfig.Count)
	}
}

func setupConfig(count int64) (*downloader.Config, *client.Config) {
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
		Directory:    strconv.Itoa(int(count)),
		WorkerCount:  downloader.DefaultWorkerCount,
		ShowProgress: false,
		ContentType:  downloader.ContentImages,
	}
	return downloaderConfig, clientConfig
}

func BenchmarkDownload1(b *testing.B) {
	dcfg, ccfg := setupConfig(1)
	dl := downloader.New(dcfg, ccfg, downloader.DefaultFilters()...)
	for i := 0; i < b.N; i++ {
		statusCh := dl.Download(context.TODO())

		for message := range statusCh {
			_, err := message.Status, message.Error
			if err != nil {
				b.Error(err)
			}
		}
	}
}

func BenchmarkDownload10(b *testing.B) {
	dcfg, ccfg := setupConfig(10)
	dl := downloader.New(dcfg, ccfg, downloader.DefaultFilters()...)
	for i := 0; i < b.N; i++ {
		statusCh := dl.Download(context.TODO())

		for message := range statusCh {
			_, err := message.Status, message.Error
			if err != nil {
				b.Error(err)
			}
		}
	}
}

func BenchmarkDownload25(b *testing.B) {
	dcfg, ccfg := setupConfig(25)
	dl := downloader.New(dcfg, ccfg, downloader.DefaultFilters()...)
	for i := 0; i < b.N; i++ {
		statusCh := dl.Download(context.TODO())

		for message := range statusCh {
			_, err := message.Status, message.Error
			if err != nil {
				b.Error(err)
			}
		}
	}
}

func BenchmarkDownload50(b *testing.B) {
	dcfg, ccfg := setupConfig(50)
	dl := downloader.New(dcfg, ccfg, downloader.DefaultFilters()...)
	for i := 0; i < b.N; i++ {
		statusCh := dl.Download(context.TODO())

		for message := range statusCh {
			_, err := message.Status, message.Error
			if err != nil {
				b.Error(err)
			}
		}
	}
}

func BenchmarkDownload100(b *testing.B) {
	dcfg, ccfg := setupConfig(100)
	dl := downloader.New(dcfg, ccfg, downloader.DefaultFilters()...)
	for i := 0; i < b.N; i++ {
		statusCh := dl.Download(context.TODO())

		for message := range statusCh {
			_, err := message.Status, message.Error
			if err != nil {
				b.Error(err)
			}
		}
	}
}
