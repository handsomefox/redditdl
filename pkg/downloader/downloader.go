// Package downloader is a package that can be
// used to download media files from reddit.com
package downloader

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/handsomefox/redditdl/pkg/downloader/config"
	"github.com/handsomefox/redditdl/pkg/downloader/filters"
	"github.com/handsomefox/redditdl/pkg/downloader/models"
	"github.com/handsomefox/redditdl/pkg/downloader/models/fetch"
	"github.com/handsomefox/redditdl/pkg/files"
	"github.com/handsomefox/redditdl/pkg/logging"
	"go.uber.org/zap"
)

// New returns a new Downloader instance with the specified configuration.
func New(cfg *config.Config, fs ...filters.Filter) *Downloader {
	return &Downloader{
		cfg:    cfg,
		logger: logging.Get(cfg.Verbose),
		stats:  &collectedStats{},
		fs:     fs,
	}
}

type Downloader struct {
	cfg    *config.Config
	logger *zap.SugaredLogger
	stats  *collectedStats
	fs     []filters.Filter
}

// Download downloads the files using the given parameters to downloader in
// a concurrent fashion to maximize download speeds.
func (dl *Downloader) Download() *Stats {
	var (
		contentChan = make(chan *models.Content)
		filesChan   = make(chan *files.File)
	)
	// Fetching posts to the content channel for further download.
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(c chan<- *models.Content) {
		defer wg.Done()
		defer close(c)
		dl.fetchPosts(c)
	}(contentChan)
	// Downloading posts from the content channel and storing data in files channel.
	wg.Add(1)
	go func(f chan<- *files.File, c <-chan *models.Content) {
		defer wg.Done()
		defer close(f)
		dl.downloadRoutine(f, c)
	}(filesChan, contentChan)
	// Saving data from files channel to disk.
	wg.Add(1)
	go func(f <-chan *files.File) {
		defer wg.Done()
		dl.saveFiles(f)
	}(filesChan)

	exitChan := make(chan struct{})
	if dl.cfg.ShowProgress {
		go dl.showProgress(exitChan)
	}

	wg.Wait()

	if dl.cfg.ShowProgress {
		exitChan <- struct{}{}
	}
	close(exitChan)
	return &Stats{st: dl.stats}
}

// fetchPosts is fetching, filtering and sending posts to outChan.
func (dl *Downloader) fetchPosts(contentChan chan<- *models.Content) {
	var (
		count int64
		after string
	)
	for count < dl.cfg.Count {
		url := fetch.FormatURL(dl.cfg, after)
		dl.logger.Debugf("Fetching posts from: %s", url)

		posts, err := fetch.Posts(url)
		if err != nil {
			dl.stats.append(newFetchError(err, url))
			continue
		}

		dl.logger.Debug("Converting response")
		content := postsToContent(dl.cfg.ContentType, posts.Data.Children)

		dl.logger.Debug("Filtering posts")
		for _, c := range content {
			c := c
			if count == dl.cfg.Count {
				break
			}
			if filters.IsFiltered(dl.cfg, c, dl.fs...) {
				continue
			}
			dl.stats.queued.Add(1)
			count++
			contentChan <- &c
		}
		// another check prevents us from going to sleep for SleepTime if we have enough links.
		if count == dl.cfg.Count {
			break
		}
		if len(posts.Data.Children) == 0 || posts.Data.After == after || posts.Data.After == "" {
			dl.logger.Info("There's no more posts to fetch")
			break
		}

		after = posts.Data.After

		dl.logger.Debugf("Fetch is sleeping...")
		time.Sleep(dl.cfg.SleepTime)
	}
}

// downloadRoutine is downloading the files from content chan to files chan using multiple goroutines.
func (dl *Downloader) downloadRoutine(fileChan chan<- *files.File, contentChan <-chan *models.Content) {
	wg := &sync.WaitGroup{}
	for i := 0; i < dl.cfg.WorkerCount; i++ {
		wg.Add(1)
		go func(f chan<- *files.File, c <-chan *models.Content) {
			defer wg.Done()
			dl.downloadFiles(f, c)
		}(fileChan, contentChan)
	}
	wg.Wait()
}

// downloadFiles gets files from the inChan, fetches their data and stores it in outChan.
func (dl *Downloader) downloadFiles(fileChan chan<- *files.File, contentChan <-chan *models.Content) {
	for content := range contentChan {
		file, err := fetch.File(content)
		if err != nil {
			dl.stats.append(newFetchError(err, content.URL))
			continue
		}
		fileChan <- file
	}
}

// saveFiles gets data from filesChan and stores it on disk.
func (dl *Downloader) saveFiles(filesChan <-chan *files.File) {
	if err := files.NavigateTo(dl.cfg.Directory, true); err != nil {
		dl.stats.failed.Store(dl.stats.queued.Load())
		dl.stats.append(fmt.Errorf("%w: failed to navigate to directory %s", err, dl.cfg.Directory))
		return
	}
	for file := range filesChan {
		filename, err := files.NewFilename(file.Name, file.Extension)
		if err != nil {
			dl.logger.Debugf("%s: failed to save file", err)
			dl.stats.appendIncr(newDownloadError(err, filename))
			continue
		}
		if err := files.Save(filename, file.Data); err != nil {
			dl.stats.appendIncr(newDownloadError(err, filename))
			continue
		}
		dl.stats.finished.Add(1)
		dl.logger.Debugf("saved file: %s", file.Name)
	}
}

// showProgress prints the current progress of download every two seconds.
func (dl *Downloader) showProgress(exit <-chan struct{}) {
	fStr := "Current progress: queued=%d, finished=%d, failed=%d"
	for {
		select {
		case <-exit:
			return
		default:
			dl.logger.Infof(fStr, dl.stats.queued.Load(), dl.stats.finished.Load(), dl.stats.failed.Load())
			time.Sleep(time.Second)
		}
	}
}

// Converts posts to content depending on the configuration, leaving only the required types of media in.
func postsToContent(typ config.ContentType, children []models.Child) []models.Content {
	data := make([]models.Content, 0, len(children))
	for i := 0; i < len(children); i++ {
		v := &children[i].Data
		if !v.IsVideo && (typ == config.ContentAny || typ == config.ContentImages) {
			if len(v.Preview.Images) != 1 {
				continue
			}
			img := &v.Preview.Images[0]
			data = append(data, models.Content{
				Name:    v.Title,
				URL:     strings.ReplaceAll(img.Source.URL, "&amp;s", "&s"),
				Width:   img.Source.Width,
				Height:  img.Source.Height,
				IsVideo: false,
			})
		} else if v.IsVideo && (typ == config.ContentAny || typ == config.ContentVideos) {
			data = append(data, models.Content{
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
