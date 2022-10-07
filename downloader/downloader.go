package downloader

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/handsomefox/redditdl/configuration"
	"github.com/handsomefox/redditdl/fetch"
	"github.com/handsomefox/redditdl/fetch/api"
	"github.com/handsomefox/redditdl/files"
	"github.com/handsomefox/redditdl/filter"
	"github.com/handsomefox/redditdl/logging"
	"go.uber.org/zap"
)

// Downloader is a single-method interface which takes in configuration and a list of filters
// and returns statistics after it is done downloading all the files.
type Downloader interface {
	Download() *Stats
}

// New returns a new Downloader instance with the specified configuration.
func New(config *configuration.Config, filters ...filter.Filter) Downloader {
	return &downloader{
		Config:  config,
		Logger:  logging.Get(config.Verbose),
		Stats:   &Stats{},
		Filters: filters,
	}
}

// Stats is the struct containing statistics for the download.
// It may or may not be extended with additional data later.
type Stats struct {
	Errors []error
	mu     sync.Mutex

	Queued   atomic.Int64
	Finished atomic.Int64
	Failed   atomic.Int64
}

// append is used to append errors to Stats.
func (s *Stats) append(err error) {
	s.mu.Lock()
	s.Errors = append(s.Errors, err)
	s.mu.Unlock()
}

// appendIncr appends the error and increments Failed count.
func (s *Stats) appendIncr(err error) {
	s.append(err)
	s.Failed.Add(1)
}

// HasErrors returns whether the errors slice is non-empty.
func (s *Stats) HasErrors() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Errors) != 0
}

// this ensures that the downloader implements the Downloader interface.
var _ Downloader = &downloader{}

type downloader struct {
	Config  *configuration.Config
	Logger  *zap.SugaredLogger
	Stats   *Stats
	Filters []filter.Filter
}

// Download downloads the files using the given parameters to downloader in
// a concurrent fashion to maximize download speeds.
func (dl *downloader) Download() *Stats {
	var (
		contentChan = make(chan api.Content)
		filesChan   = make(chan files.File)
		wg          sync.WaitGroup
	)
	// Fetching posts to the content channel for further download.
	wg.Add(1)
	go func(c chan<- api.Content) {
		defer wg.Done()
		defer close(c)
		dl.FetchPosts(c)
	}(contentChan)
	// Downloading posts from the content channel and storing data in files channel.
	wg.Add(1)
	go func(f chan<- files.File, c <-chan api.Content) {
		defer wg.Done()
		defer close(f)
		dl.DownloadRoutine(f, c)
	}(filesChan, contentChan)
	// Saving data from files channel to disk.
	wg.Add(1)
	go func(f <-chan files.File) {
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
func (dl *downloader) FetchPosts(contentChan chan<- api.Content) {
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
			if filter.IsFiltered(dl.Config, c, dl.Filters...) {
				continue
			}
			dl.Stats.Queued.Add(1)
			count++
			contentChan <- c
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
func (dl *downloader) DownloadRoutine(fileChan chan<- files.File, contentChan <-chan api.Content) {
	var wg sync.WaitGroup
	for i := 0; i < dl.Config.WorkerCount; i++ {
		wg.Add(1)
		go func(f chan<- files.File, c <-chan api.Content) {
			defer wg.Done()
			dl.DownloadFiles(f, c)
		}(fileChan, contentChan)
	}
	wg.Wait()
}

// DownloadFiles gets files from the inChan, fetches their data and stores it in outChan.
func (dl *downloader) DownloadFiles(fileChan chan<- files.File, contentChan <-chan api.Content) {
	for content := range contentChan {
		content := content
		file, err := fetch.File(&content)
		if err != nil {
			dl.Stats.append(newFetchError(err, content.URL))
			continue
		}
		fileChan <- *file
	}
}

// SaveFiles gets data from filesChan and stores it on disk.
func (dl *downloader) SaveFiles(filesChan <-chan files.File) {
	if err := files.NavigateTo(dl.Config.Directory, true); err != nil {
		dl.Stats.Failed.Store(dl.Stats.Queued.Load())
		dl.Stats.append(fmt.Errorf("failed to navigate to directory, error: %w, directory: %v", err, dl.Config.Directory))
		return
	}
	for file := range filesChan {
		file := file
		filename, err := files.NewFilename(file.Name, file.Extension)
		if err != nil {
			dl.Logger.Debugf("error saving file: %v", err)
			dl.Stats.appendIncr(newDownloadError(err, filename))
			continue
		}
		if err := files.Save(filename, file.Data); err != nil {
			dl.Stats.appendIncr(newDownloadError(err, filename))
			continue
		}
		dl.Stats.Finished.Add(1)
		dl.Logger.Debugf("saved file: %v", file.Name)
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
			dl.Logger.Infof(fStr, dl.Stats.Queued.Load(), dl.Stats.Finished.Load(), dl.Stats.Failed.Load())
			time.Sleep(time.Second)
		}
	}
}

// Converts posts to content depending on the configuration, leaving only the required types of media in.
func postsToContent(typ configuration.ContentType, children []api.Child) []api.Content {
	data := make([]api.Content, 0, len(children))
	for i := 0; i < len(children); i++ {
		value := &children[i].Data
		if !value.IsVideo && typ == configuration.ContentAny || typ == configuration.ContentImages {
			for _, img := range value.Preview.Images {
				data = append(data, api.Content{
					Name:    value.Title,
					URL:     strings.ReplaceAll(img.Source.URL, "&amp;s", "&s"),
					Width:   img.Source.Width,
					Height:  img.Source.Height,
					IsVideo: false,
				})
			}
		} else if value.IsVideo && typ == configuration.ContentAny || typ == configuration.ContentVideos {
			data = append(data, api.Content{
				Name:    value.Title,
				URL:     strings.ReplaceAll(value.Media.RedditVideo.ScrubberMediaURL, "&amp;s", "&s"),
				Width:   value.Media.RedditVideo.Width,
				Height:  value.Media.RedditVideo.Height,
				IsVideo: true,
			})
		}
	}
	return data
}
