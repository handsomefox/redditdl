package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"redditdl/pkg/utils"

	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

type downloader struct {
	client  *http.Client
	log     *zap.SugaredLogger
	after   string
	counter counter
}

// this is saved to disk later.
type file struct {
	name      string
	extension string
	bytes     []byte
}

// used for showing progress.
type counter struct {
	queued, finished, failed atomic.Int64
}

// Content contains information which is required to filter by resolution,
// Content and store a video or an image.
type Content struct {
	Name          string
	URL           string
	Width, Height int
	IsVideo       bool
}

func (dl *downloader) download(settings *Settings, filters []Filter) (int64, error) {
	var (
		contentChan = make(chan Content)
		filesChan   = make(chan file)

		err error
		wg  sync.WaitGroup
	)

	wg.Add(1)
	// fetching files to content chan
	go func(ss *Settings, fs []Filter, c chan Content) {
		defer wg.Done()
		defer close(c)

		if err := dl.fetchPosts(ss, fs, c); err != nil {
			dl.log.Debugf("error fetching posts: %s", err.Error())
		}
	}(settings, filters, contentChan)

	wg.Add(1)
	// downloading the files from content chan to files chan using multiple goroutines
	go func(f chan file, c chan Content) {
		defer close(f)
		defer wg.Done()

		var wg sync.WaitGroup

		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func(f chan file, c chan Content) {
				defer wg.Done()
				dl.downloadFiles(f, c)
			}(f, c)
		}
		wg.Wait()
	}(filesChan, contentChan)

	wg.Add(1)
	// saving files to disk
	go func(ss *Settings, c chan file) {
		defer wg.Done()
		err = dl.saveFiles(ss, c)
	}(settings, filesChan)

	terminate := make(chan int8)
	// start the progress tracking goroutine
	if settings.ShowProgress {
		go dl.showProgress(terminate)
	}
	wg.Wait()
	if settings.ShowProgress {
		terminate <- 1
	}
	close(terminate)

	return dl.counter.finished.Load(), err
}

func (dl *downloader) fetchPosts(settings *Settings, filters []Filter, contentChan chan Content) error {
	downloads := make([]Content, 0, settings.Count)

	var count int64
	for count < settings.Count {
		dl.log.Debug("fetching posts")
		posts, err := dl.getPostsFromReddit(settings)
		if err != nil {
			return fmt.Errorf("error fetching posts: %w", err)
		}

		dl.log.Debug("Converting posts to content...")
		content := postsToContent(settings, posts.Data.Children)

		dl.log.Debug("Filtering posts...")
		content = applyFilters(settings, content, filters)

		for _, c := range content {
			if slices.Contains(downloads, c) {
				continue
			}
			downloads = append(downloads, c)
		}

		for _, d := range downloads {
			if count == settings.Count {
				break
			}
			dl.counter.queued.Add(1)
			count++
			contentChan <- d
		}

		if len(posts.Data.Children) == 0 || posts.Data.After == dl.after || posts.Data.After == "" {
			dl.log.Info("no more posts to fetch (or rate limited)")
			break
		}

		dl.after = posts.Data.After
		dl.log.Debugf("fetching goroutine sleeping")
		time.Sleep(SleepTime)
	}

	return nil
}

func (dl *downloader) downloadFiles(filesChan chan file, contentChan chan Content) {
	for content := range contentChan {
		request, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, content.URL, http.NoBody)
		if err != nil {
			dl.log.Debug(err)
		}

		response, err := dl.client.Do(request)
		if err != nil {
			dl.log.Debug(err)
			continue
		}

		if response.StatusCode != http.StatusOK {
			dl.log.Debugf("unexpected status code in response: %v", http.StatusText(response.StatusCode))
			continue
		}

		var extension string
		if content.IsVideo {
			extension = "mp4" // if we didn't manage to figure out the video extension, assume mp4
		} else {
			extension = "jpg" // if we didn't manage to figure out the image extension, assume jpg
		}

		// the URL path is usually equal to something like "randomid.extension",
		// this way we can get the actual file extension
		split := strings.Split(response.Request.URL.Path, ".")
		if len(split) == 2 {
			extension = split[1]
		}

		b, err := io.ReadAll(response.Body)
		if err != nil {
			dl.log.Debugf("error copying data from body: %v", err)
			continue
		}

		response.Body.Close()
		filesChan <- file{
			bytes:     b,
			name:      content.Name,
			extension: extension,
		}
	}
}

func (dl *downloader) saveFiles(settings *Settings, filesChan chan file) error {
	if err := utils.NavigateToDirectory(settings.Directory, true); err != nil {
		dl.counter.failed.Store(dl.counter.queued.Load())
		return fmt.Errorf("failed to navigate to directory, error: %w, directory: %v", err, settings.Directory)
	}

	for file := range filesChan {
		filename, err := utils.CreateFilename(file.name, file.extension)
		if err != nil {
			dl.log.Debug(err)
			continue
		}

		f, err := os.Create(filename)
		if err != nil {
			dl.log.Debugf("error creating a file: %v", err)
			dl.counter.failed.Add(1)
			continue
		}

		r := bytes.NewReader(file.bytes)
		if _, err := r.WriteTo(f); err != nil {
			if err := os.Remove(filename); err != nil {
				dl.log.Debugf("error removing file after a failed copy: %v", err)
				dl.counter.failed.Add(1)
				continue
			}

			dl.log.Debugf("error copying file to disk: %v", err)
			dl.counter.failed.Add(1)
			continue
		}

		dl.counter.finished.Add(1)
		dl.log.Debugf("saved file: %v to disk", filename)
	}

	return nil
}

// getPostsFromReddit fetches a json file from reddit containing information
// about the posts using the given configuration.
func (dl *downloader) getPostsFromReddit(settings *Settings) (*posts, error) {
	URL := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s",
		settings.Subreddit, settings.Sorting, settings.Count, settings.Timeframe)

	if len(dl.after) > 0 {
		URL = fmt.Sprintf("%s&after=%s&count=%d", URL, dl.after, settings.Count)
	}

	request, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, URL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("error creating a request: %w, URL: %v", err, URL)
	}
	request.Header.Add("User-Agent", "go:getter")

	response, err := dl.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetching from reddit failed, error: %w, URL: %v", err, URL)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %v", ErrUnexpectedStatus, http.StatusText(response.StatusCode))
	}

	posts := &posts{}
	if err := json.NewDecoder(response.Body).Decode(posts); err != nil {
		return nil, fmt.Errorf("error decoding posts: %w", err)
	}

	return posts, nil
}

// Converts posts to content depending on the configuration, leaving only the required types of media in.
func postsToContent(settings *Settings, child []child) []Content {
	media := make([]Content, 0)

	for i := 0; i < len(child); i++ {
		value := &child[i]
		if value.Data.IsVideo && settings.IncludeVideo { // Append video
			media = append(media, Content{
				Name:    value.Data.Title,
				URL:     strings.ReplaceAll(value.Data.Media.RedditVideo.ScrubberMediaURL, "&amp;s", "&s"),
				Width:   value.Data.Media.RedditVideo.Width,
				Height:  value.Data.Media.RedditVideo.Height,
				IsVideo: true,
			})
		} else { // Append images
			for _, img := range value.Data.Preview.Images {
				media = append(media, Content{
					Name:    value.Data.Title,
					URL:     strings.ReplaceAll(img.Source.URL, "&amp;s", "&s"),
					Width:   img.Source.Width,
					Height:  img.Source.Height,
					IsVideo: false,
				})
			}
		}
	}

	return media
}

func (dl *downloader) showProgress(terminate chan int8) {
	end := false
	for !end {
		select {
		case <-terminate:
			end = true
		default:
			dl.log.Infof("Current progress: queued=%d, finished=%d, failed=%d",
				dl.counter.queued.Load(), dl.counter.finished.Load(), dl.counter.failed.Load())
			time.Sleep(time.Second)
		}
	}
}
