package downloader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"redditdl/pkg/logging"
	"redditdl/pkg/utils"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
)

var (
	client = utils.CreateClient() // client used for http requests
	log    *zap.SugaredLogger     // logger from logging package
	after  = ""                   // After is a post ID, which is used to fetch posts after that ID
)

// Settings is the configuration for the Downloader
type Settings struct {
	Verbose      bool   // Verbose turns the logging on or off
	ShowProgress bool   // ShowProgress indicates whether the application will show the download progress
	IncludeVideo bool   // IncludeVideo indicates whether the application should download videos as well
	Subreddit    string // Subreddit name
	Sorting      string // Sorting How to sort the subreddit
	Timeframe    string // Timeframe of the posts
	Directory    string // Directory to download media to
	Count        int    // Count Amount of media to download
	MinWidth     int    // MinWidth Minimal width of the media
	MinHeight    int    // MinHeight Minimal height of the media
}

// this is saved to disk later
type file struct {
	bytes           []byte
	name, extension string
}

// used for showing progress
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

// amount of goroutines downloading media
// TODO: maybe make this configurable
const workers = 16

// Download downloads the images according to the given configuration
func Download(s Settings, fs []Filter) (int64, error) {
	log = logging.GetLogger(s.Verbose)

	// TODO: maybe separate this into its own struct and add a constructor
	eg := new(errgroup.Group)
	fetched := make(chan content)
	files := make(chan file)
	c := new(counter)

	// this fetches the posts and sends them to the fetched channel for other goroutines to download
	eg.Go(func() error {
		defer close(fetched)

		dls := make([]content, 0, s.Count)

		count := 0
		for count < s.Count {
			log.Debug("fetching posts")

			posts, err := getPosts(&s)
			if err != nil {
				return fmt.Errorf("error fetching posts: %v", err)
			}

			log.Debug("Converting posts to content...")
			cont := postsToContent(&s, posts.Data.Children)

			log.Debug("Filtering posts...")
			cont = applyFilters(&s, cont, fs)

			for _, v := range cont {
				if slices.Contains(dls, v) {
					continue
				}
				dls = append(dls, v)
			}

			for _, v := range dls {
				if count == s.Count {
					break
				}
				atomic.AddInt64(&c.queued, 1)
				count++
				fetched <- v
			}

			if len(posts.Data.Children) == 0 || posts.Data.After == after || len(posts.Data.After) == 0 {
				log.Info("no more posts to fetch (or rate limited)")
				break
			}
			after = posts.Data.After

			log.Debugf("fetching goroutine sleeping")
			time.Sleep(5 * time.Second)
		}

		return nil
	})

	// this downloads the images/videos to a byte slice, creates a filename for them and sends them to the files channel
	go func() {
		wg := new(sync.WaitGroup)
		defer close(files)

		for i := 0; i < workers; i++ {
			wg.Add(1)

			eg.Go(func() error {
				for v := range fetched {
					response, err := client.Get(v.URL)
					if err != nil {
						log.Debugf("failed to GET the post URL: %v", err)
						continue
					}

					if response.StatusCode != http.StatusOK {
						log.Debugf("unexpected status code in response: %v", http.StatusText(response.StatusCode))
						continue
					}

					var extension string
					if v.IsVideo {
						extension = "mp4" // if we didn't manage to figure out the video extension, assume mp4
					} else {
						extension = "jpg" // if we didn't manage to figure out the image extension, assume jpg
					}

					// TODO: maybe find a better way to get file extension from reddit
					nameAndExt := strings.Split(response.Request.URL.Path, ".")
					if len(nameAndExt) == 2 {
						extension = nameAndExt[1]
					}

					b, err := io.ReadAll(response.Body)
					if err != nil {
						log.Debugf("error copying data from body: %v", err)
						continue
					}
					response.Body.Close()

					files <- file{
						bytes:     b,
						name:      v.Name,
						extension: extension,
					}
					// log.Debugf("queued file %v for saving", v.Name)
				}
				wg.Done()
				return nil
			})
		}
		wg.Wait()
	}()

	// this saves the downloaded files to disk
	eg.Go(func() error {
		if err := utils.NavigateToDirectory(s.Directory, true); err != nil {
			atomic.StoreInt64(&c.failed, c.queued)
			return fmt.Errorf("failed to navigate to directory, error: %v, directory: %v", err, s.Directory)
		}

		for f := range files {
			filename, err := utils.CreateFilename(f.name, f.extension)
			if err != nil {
				log.Debugf("error creating a filename: %v", err)
				continue
			}

			file, err := os.Create(filename)
			if err != nil {
				log.Debugf("error creating a file: %v", err)
				atomic.AddInt64(&c.failed, 1)
				continue
			}

			r := bytes.NewReader(f.bytes)
			if _, err := io.Copy(file, r); err != nil {
				if err := os.Remove(filename); err != nil {
					log.Debugf("error removing file after a failed copy: %v", err)
					atomic.AddInt64(&c.failed, 1)
					continue
				}
				log.Debugf("error copying file to disk: %v", err)
				atomic.AddInt64(&c.failed, 1)
				continue
			}
			atomic.AddInt64(&c.finished, 1)
			log.Debugf("saved file: %v to disk", filename)
		}
		return nil
	})

	// start the progress tracking goroutine
	if s.ShowProgress {
		go func(c *counter) {
			time.Sleep(1 * time.Second)
			for {
				log.Infof("Current progress: queued=%d, finished=%d, failed=%d", c.queued, c.finished, c.failed)
				time.Sleep(2 * time.Second)
			}
		}(c)
	}

	return c.finished, eg.Wait()
}

// applyFilters applies every filter from the slice of []Filter and returns the mutated slice
// if there are no filters, the original slice is returned
func applyFilters(s *Settings, td []content, fs []Filter) []content {
	if len(fs) == 0 { // return the original posts if there are no filters
		return td
	}

	f := make([]content, 0, len(td))
	f = append(f, td...)
	for _, ff := range fs {
		f = ff.Filter(f, s)
	}
	return f
}

// getPosts fetches a json file from reddit containing information about the posts using the given configuration.
func getPosts(s *Settings) (*posts, error) {
	URL := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s",
		s.Subreddit, s.Sorting, s.Count, s.Timeframe)

	if len(after) > 0 {
		URL = fmt.Sprintf("%s&after=%s&count=%d", URL, after, s.Count)
	}

	request, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating a request: %v, URL: %v", err, URL)
	}
	request.Header.Add("User-Agent", "go:getter")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetching from reddit failed, error: %v, URL: %v", err, URL)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status in response: %v", http.StatusText(response.StatusCode))
	}

	posts := new(posts)
	if err := json.NewDecoder(response.Body).Decode(posts); err != nil {
		return nil, fmt.Errorf("error decoding posts: %v", err)
	}
	return posts, nil
}

// Converts posts to content depending on the configuration, leaving only the required types of media in
func postsToContent(s *Settings, cd []child) []content {
	media := make([]content, 0)

	for _, v := range cd {
		if v.Data.IsVideo && s.IncludeVideo { // Append video
			media = append(media, newVideo(v.Data.Title, &v.Data.Media.RedditVideo))
		} else { // Append images
			for _, img := range v.Data.Preview.Images {
				media = append(media, newImage(v.Data.Title, &img.Source))
			}
		}
	}

	return media
}

func newVideo(title string, rv *RedditVideo) content {
	return content{
		Name:    title,
		URL:     strings.ReplaceAll(rv.ScrubberMediaURL, "&amp;s", "&s"),
		Width:   rv.Width,
		Height:  rv.Height,
		IsVideo: true,
	}
}

func newImage(title string, id *imageData) content {
	return content{
		Name:    title,
		URL:     strings.ReplaceAll(id.URL, "&amp;s", "&s"),
		Width:   id.Width,
		Height:  id.Height,
		IsVideo: false,
	}
}
