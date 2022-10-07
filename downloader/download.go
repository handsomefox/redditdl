package downloader

import (
	"fmt"
	"sync"
	"time"

	"github.com/handsomefox/redditdl/configuration"
	"github.com/handsomefox/redditdl/fetch"
	"github.com/handsomefox/redditdl/filter"
	"github.com/handsomefox/redditdl/structs"
	"github.com/handsomefox/redditdl/utils"

	"go.uber.org/zap"
)

// this ensures that the downloader implements the Downloader interface.
var _ Downloader = &downloader{}

type downloader struct {
	Config  *configuration.Data
	Logger  *zap.SugaredLogger
	Stats   *Stats
	Filters []filter.Filter
}

func (dl *downloader) Download() *Stats {
	var (
		contentChan = make(chan structs.Content)
		filesChan   = make(chan structs.File)
		wg          sync.WaitGroup
	)

	// Fetching posts to the content channel for further download.
	wg.Add(1)
	go func(out chan structs.Content) {
		defer wg.Done()
		defer close(out)
		dl.FetchPosts(out)
	}(contentChan)

	// Downloading posts from the content channel and storing data in files channel.
	wg.Add(1)
	go func(out chan structs.File, in chan structs.Content) {
		defer wg.Done()
		defer close(out)
		dl.DownloadRoutine(filesChan, contentChan)
	}(filesChan, contentChan)

	// Saving data from files channel to disk.
	wg.Add(1)
	go func(in chan structs.File) {
		defer wg.Done()
		dl.SaveFiles(in)
	}(filesChan)

	exitChan := make(chan bool)
	if dl.Config.ShowProgress {
		go dl.ShowProgress(exitChan)
	}

	wg.Wait()

	if dl.Config.ShowProgress {
		exitChan <- true
	}

	close(exitChan)

	return dl.Stats
}

// FetchPosts is fetching, filtering and sending posts to outChan.
func (dl *downloader) FetchPosts(outChan chan structs.Content) {
	var (
		count int64
		after string
	)

	for count < dl.Config.Count {
		url := fetch.FormatURL(dl.Config, after)
		dl.Logger.Debugf("fetching posts from: %v", url)

		posts, err := fetch.Posts(url)
		if err != nil {
			dl.Stats.append(newFetchError(err, url))

			continue
		}

		dl.Logger.Debug("converting posts")
		content := postsToContent(dl.Config.ContentType, posts.Data.Children)

		dl.Logger.Debug("filtering posts")
		for _, c := range content {
			if count == dl.Config.Count {
				break
			}

			if isFiltered(dl.Config, c, dl.Filters...) {
				continue
			}

			dl.Stats.Queued.Add(1)
			count++
			outChan <- c
		}

		// another check prevents us from going to sleep for SleepTime if we have enough links.
		if count == dl.Config.Count {
			break
		}

		if len(posts.Data.Children) == 0 || posts.Data.After == after || posts.Data.After == "" {
			dl.Logger.Info("no more posts to fetch (or rate limited)")

			break
		}

		after = posts.Data.After

		dl.Logger.Debugf("fetching goroutine sleeping")
		time.Sleep(dl.Config.SleepTime)
	}
}

// DownloadRoutine is downloading the files from content chan to files chan using multiple goroutines.
func (dl *downloader) DownloadRoutine(outChan chan structs.File, inChan chan structs.Content) {
	var wg sync.WaitGroup

	for i := 0; i < dl.Config.WorkerCount; i++ {
		wg.Add(1)
		go func(f chan structs.File, c chan structs.Content) {
			defer wg.Done()
			dl.DownloadFiles(f, c)
		}(outChan, inChan)
	}
	wg.Wait()
}

// DownloadFiles gets files from the inChan, fetches their data and stores it in outChan.
func (dl *downloader) DownloadFiles(outChan chan structs.File, inChan chan structs.Content) {
	for content := range inChan {
		content := content

		file, err := fetch.File(&content)
		if err != nil {
			dl.Stats.append(newFetchError(err, content.URL))

			continue
		}

		outChan <- *file
	}
}

// SaveFiles gets data from filesChan and stores it on disk.
func (dl *downloader) SaveFiles(filesChan chan structs.File) {
	if err := utils.NavigateToDirectory(dl.Config.Directory, true); err != nil {
		dl.Stats.Failed.Store(dl.Stats.Queued.Load())
		dl.Stats.append(fmt.Errorf("failed to navigate to directory, error: %w, directory: %v", err, dl.Config.Directory))

		return
	}

	for file := range filesChan {
		file := file
		filename, err := utils.CreateFilename(file.Name, file.Extension)
		if err != nil {
			dl.Logger.Debugf("error saving file: %v", err)
			dl.Stats.appendIncr(newDownloadError(err, filename))

			continue
		}

		if err := utils.SaveFile(filename, &file); err != nil {
			dl.Stats.appendIncr(newDownloadError(err, filename))

			continue
		}

		dl.Stats.Finished.Add(1)
		dl.Logger.Debugf("saved file: %v", file.Name)
	}
}

// ShowProgress prints the current progress of download every two seconds.
func (dl *downloader) ShowProgress(exit chan bool) {
	fStr := "Current progress: queued=%d, finished=%d, failed=%d"
	for {
		select {
		case <-exit:
			return
		default:
			dl.Logger.Infof(fStr, dl.Stats.Queued.Load(), dl.Stats.Finished.Load(), dl.Stats.Failed.Load())
			time.Sleep(time.Second)
		}
	}
}
