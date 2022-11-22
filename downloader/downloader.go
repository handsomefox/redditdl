// Package downloader is a package that can be
// used to download media files from reddit.com
package downloader

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/handsomefox/redditdl/downloader/config"
	"github.com/handsomefox/redditdl/downloader/fetch"
	"github.com/handsomefox/redditdl/downloader/fetch/api"
	"github.com/handsomefox/redditdl/downloader/filters"
	"github.com/handsomefox/redditdl/files"
	"github.com/handsomefox/redditdl/logging"
	"go.uber.org/zap"
)

// Downloader is a single-method interface which takes in configuration and a list of filters
// and returns statistics after it is done downloading all the files.
type Downloader interface {
	Download() Stats
}

// New returns a new Downloader instance with the specified configuration.
func New(cfg *config.Config, fs ...filters.Filter) Downloader {
	return &downloader{
		Config:  cfg,
		Logger:  logging.Get(cfg.Verbose),
		Stats:   &stats{},
		Filters: fs,
	}
}

var _ Downloader = &downloader{}

type downloader struct {
	Config  *config.Config
	Logger  *zap.SugaredLogger
	Stats   *stats
	Filters []filters.Filter
}

// Download downloads the files using the given parameters to downloader in
// a concurrent fashion to maximize download speeds.
func (dl *downloader) Download() Stats {
	var (
		contentChan = make(chan *api.Content)
		filesChan   = make(chan *files.File)
	)
	// Fetching posts to the content channel for further download.
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(c chan<- *api.Content) {
		defer wg.Done()
		defer close(c)
		dl.FetchPosts(c)
	}(contentChan)
	// Downloading posts from the content channel and storing data in files channel.
	wg.Add(1)
	go func(f chan<- *files.File, c <-chan *api.Content) {
		defer wg.Done()
		defer close(f)
		dl.DownloadRoutine(f, c)
	}(filesChan, contentChan)
	// Saving data from files channel to disk.
	wg.Add(1)
	go func(f <-chan *files.File) {
		defer wg.Done()
		dl.SaveFiles(f)
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
func (dl *downloader) FetchPosts(contentChan chan<- *api.Content) {
	var (
		count int64
		after string
	)
	for count < dl.Config.Count {
		url := fetch.FormatURL(dl.Config, after)
		dl.Logger.Debugf("Fetching posts from: %s", url)

		posts, err := fetch.Posts(url)
		if err != nil {
			dl.Stats.append(newFetchError(err, url))
			continue
		}

		dl.Logger.Debug("Converting response")
		content := postsToContent(dl.Config.ContentType, posts.Data.Children)

		dl.Logger.Debug("Filtering posts")
		for _, c := range content {
			c := c
			if count == dl.Config.Count {
				break
			}
			if filters.IsFiltered(dl.Config, c, dl.Filters...) {
				continue
			}
			dl.Stats.queued.Add(1)
			count++
			contentChan <- &c
		}
		// another check prevents us from going to sleep for SleepTime if we have enough links.
		if count == dl.Config.Count {
			break
		}
		if len(posts.Data.Children) == 0 || posts.Data.After == after || posts.Data.After == "" {
			dl.Logger.Info("There's no more posts to fetch")
			break
		}

		after = posts.Data.After

		dl.Logger.Debugf("Fetch is sleeping...")
		time.Sleep(dl.Config.SleepTime)
	}
}

// DownloadRoutine is downloading the files from content chan to files chan using multiple goroutines.
func (dl *downloader) DownloadRoutine(fileChan chan<- *files.File, contentChan <-chan *api.Content) {
	wg := &sync.WaitGroup{}
	for i := 0; i < dl.Config.WorkerCount; i++ {
		wg.Add(1)
		go func(f chan<- *files.File, c <-chan *api.Content) {
			defer wg.Done()
			dl.DownloadFiles(f, c)
		}(fileChan, contentChan)
	}
	wg.Wait()
}

// DownloadFiles gets files from the inChan, fetches their data and stores it in outChan.
func (dl *downloader) DownloadFiles(fileChan chan<- *files.File, contentChan <-chan *api.Content) {
	for content := range contentChan {
		file, err := fetch.File(content)
		if err != nil {
			dl.Stats.append(newFetchError(err, content.URL))
			continue
		}
		fileChan <- file
	}
}

// SaveFiles gets data from filesChan and stores it on disk.
func (dl *downloader) SaveFiles(filesChan <-chan *files.File) {
	if err := files.NavigateTo(dl.Config.Directory, true); err != nil {
		dl.Stats.failed.Store(dl.Stats.queued.Load())
		dl.Stats.append(fmt.Errorf("%w: failed to navigate to directory %s", err, dl.Config.Directory))
		return
	}
	for file := range filesChan {
		filename, err := files.NewFilename(file.Name, file.Extension)
		if err != nil {
			dl.Logger.Debugf("%s: failed to save file", err)
			dl.Stats.appendIncr(newDownloadError(err, filename))
			continue
		}
		if err := files.Save(filename, file.Data); err != nil {
			dl.Stats.appendIncr(newDownloadError(err, filename))
			continue
		}
		dl.Stats.finished.Add(1)
		dl.Logger.Debugf("saved file: %s", file.Name)
	}
}

// ShowProgress prints the current progress of download every two seconds.
func (dl *downloader) ShowProgress(exit <-chan bool) {
	fStr := "Current progress: queued=%d, finished=%d, failed=%d"
	for {
		select {
		case <-exit:
			return
		default:
			dl.Logger.Infof(fStr, dl.Stats.queued.Load(), dl.Stats.finished.Load(), dl.Stats.failed.Load())
			time.Sleep(time.Second)
		}
	}
}

// Converts posts to content depending on the configuration, leaving only the required types of media in.
func postsToContent(typ config.ContentType, children []api.Child) []api.Content {
	data := make([]api.Content, 0, len(children))
	for i := 0; i < len(children); i++ {
		v := &children[i].Data
		if !v.IsVideo && (typ == config.ContentAny || typ == config.ContentImages) {
			if len(v.Preview.Images) != 1 {
				continue
			}
			img := &v.Preview.Images[0]
			data = append(data, api.Content{
				Name:    v.Title,
				URL:     strings.ReplaceAll(img.Source.URL, "&amp;s", "&s"),
				Width:   img.Source.Width,
				Height:  img.Source.Height,
				IsVideo: false,
			})
		} else if v.IsVideo && (typ == config.ContentAny || typ == config.ContentVideos) {
			data = append(data, api.Content{
				Name:    v.Title,
				URL:     strings.ReplaceAll(v.Media.RedditVideo.ScrubberMediaURL, "&amp;s", "&s"),
				Width:   v.Media.RedditVideo.Width,
				Height:  v.Media.RedditVideo.Height,
				IsVideo: true,
			})
		}
	}
	return data
}
