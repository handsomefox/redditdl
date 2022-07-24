package downloader

import (
	"bytes"
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
	queued, finished, failed int64
}

// content contains information which is required to filter by resolution,
// content and store a video or an image.
type content struct {
	Name, URL     string
	Width, Height int
	IsVideo       bool
}

func (dl *downloader) download(settings *Settings, filters []Filter) (int64, error) {
	var (
		err error = nil

		fetched = make(chan content)
		files   = make(chan file)
		wg      = new(sync.WaitGroup)
	)

	wg.Add(1)
	// fetching files to fetched chan
	go func(settings *Settings, filters []Filter, fetched chan content) {
		defer wg.Done()
		defer close(fetched)

		if err := dl.fetchPosts(settings, filters, fetched); err != nil {
			dl.log.Debugf("error fetching posts: %w", err)
		}
	}(settings, filters, fetched)

	wg.Add(1)
	// downloading the files from fetched chan to files chan using multiple goroutines
	go func(files chan file, fetched chan content) {
		defer close(files)
		defer wg.Done()

		wg := new(sync.WaitGroup)

		for i := 0; i < workerCount; i++ {
			wg.Add(1)

			go func(files chan file, fetched chan content) {
				defer wg.Done()
				dl.downloadFiles(files, fetched)
			}(files, fetched)
		}

		wg.Wait()
	}(files, fetched)

	wg.Add(1)
	// saving files to disk
	go func(settings *Settings, files chan file) {
		defer wg.Done()

		err = dl.saveFiles(settings, files)
	}(settings, files)

	wg.Wait()

	return dl.counter.finished, err
}

func (dl *downloader) fetchPosts(settings *Settings, filters []Filter, fetched chan content) error {
	dls := make([]content, 0, settings.Count)

	count := 0
	for count < settings.Count {
		dl.log.Debug("fetching posts")

		posts, err := dl.getPostsFromReddit(settings)
		if err != nil {
			return fmt.Errorf("error fetching posts: %w", err)
		}

		dl.log.Debug("Converting posts to content...")

		cont := postsToContent(settings, posts.Data.Children)

		dl.log.Debug("Filtering posts...")

		cont = applyFilters(settings, cont, filters)

		for _, v := range cont {
			if slices.Contains(dls, v) {
				continue
			}

			dls = append(dls, v)
		}

		for _, value := range dls {
			if count == settings.Count {
				break
			}

			atomic.AddInt64(&dl.counter.queued, 1)
			count++
			fetched <- value
		}

		if len(posts.Data.Children) == 0 || posts.Data.After == dl.after || len(posts.Data.After) == 0 {
			dl.log.Info("no more posts to fetch (or rate limited)")

			break
		}

		dl.after = posts.Data.After
		dl.log.Debugf("fetching goroutine sleeping")
		time.Sleep(SleepTime)
	}

	return nil
}

func (dl *downloader) downloadFiles(files chan file, fetched chan content) {
	for value := range fetched {
		response, err := dl.client.Get(value.URL)
		if err != nil {
			dl.log.Debugf("failed to GET the post URL: %v", err)

			continue
		}

		if response.StatusCode != http.StatusOK {
			dl.log.Debugf("unexpected status code in response: %v", http.StatusText(response.StatusCode))

			continue
		}

		var extension string
		if value.IsVideo {
			extension = "mp4" // if we didn't manage to figure out the video extension, assume mp4
		} else {
			extension = "jpg" // if we didn't manage to figure out the image extension, assume jpg
		}

		// the URL path is usually equal to something like "randomid.extension",
		// this way we can get the actual file extension
		nameAndExt := strings.Split(response.Request.URL.Path, ".")
		requiredLength := 2

		if len(nameAndExt) == requiredLength {
			extension = nameAndExt[1]
		}

		bytes, err := io.ReadAll(response.Body)
		if err != nil {
			dl.log.Debugf("error copying data from body: %v", err)

			continue
		}

		response.Body.Close()
		files <- file{
			bytes:     bytes,
			name:      value.Name,
			extension: extension,
		}
	}
}

func (dl *downloader) saveFiles(settings *Settings, files chan file) error {
	if err := utils.NavigateToDirectory(settings.Directory, true); err != nil {
		atomic.StoreInt64(&dl.counter.failed, dl.counter.queued)

		return fmt.Errorf("failed to navigate to directory, error: %w, directory: %v", err, settings.Directory)
	}

	for value := range files {
		filename, err := utils.CreateFilename(value.name, value.extension)
		if err != nil {
			dl.log.Debugf("error creating a filename: %v", err)

			continue
		}

		file, err := os.Create(filename)
		if err != nil {
			dl.log.Debugf("error creating a file: %v", err)
			atomic.AddInt64(&dl.counter.failed, 1)

			continue
		}

		r := bytes.NewReader(value.bytes)
		if _, err := io.Copy(file, r); err != nil {
			if err := os.Remove(filename); err != nil {
				dl.log.Debugf("error removing file after a failed copy: %v", err)
				atomic.AddInt64(&dl.counter.failed, 1)

				continue
			}

			dl.log.Debugf("error copying file to disk: %v", err)
			atomic.AddInt64(&dl.counter.failed, 1)

			continue
		}

		atomic.AddInt64(&dl.counter.finished, 1)
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

	request, err := http.NewRequest("GET", URL, nil)
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

	posts := new(posts)
	if err := json.NewDecoder(response.Body).Decode(posts); err != nil {
		return nil, fmt.Errorf("error decoding posts: %w", err)
	}

	return posts, nil
}

// Converts posts to content depending on the configuration, leaving only the required types of media in.
func postsToContent(s *Settings, cd []child) []content {
	media := make([]content, 0)

	for _, value := range cd {
		if value.Data.IsVideo && s.IncludeVideo { // Append video
			media = append(media, content{
				Name:    value.Data.Title,
				URL:     strings.ReplaceAll(value.Data.Media.RedditVideo.ScrubberMediaURL, "&amp;s", "&s"),
				Width:   value.Data.Media.RedditVideo.Width,
				Height:  value.Data.Media.RedditVideo.Height,
				IsVideo: true,
			})
		} else { // Append images
			for _, img := range value.Data.Preview.Images {
				media = append(media, content{
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
