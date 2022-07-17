package downloader

import (
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
)

type DownloaderSettings struct {
	// Verbose turns the logging on or off
	Verbose bool
	// ShowProgress indicates whether the application will show the download progress
	ShowProgress bool
	// IncludeVideo indicates whether the application should download videos as well
	IncludeVideo bool
	// Subreddit name
	Subreddit string
	// Sorting How to sort the subreddit
	Sorting string
	// Timeframe of the posts
	Timeframe string
	// Directory to download media to
	Directory string
	// Count Amount of media to download
	Count int
	// MinWidth Minimal width of the media
	MinWidth int
	// MinHeight Minimal height of the media
	MinHeight int
	// After is a post ID, which is used to fetch posts after that ID
	After string
}

// Use New() to create a Downloader
type Downloader struct {
	fs  []Filter
	s   DownloaderSettings
	cl  *http.Client
	log *zap.SugaredLogger
}

// New gives a new instance of Downloader
func New(cfg DownloaderSettings, filters []Filter) *Downloader {
	return &Downloader{
		fs:  filters,
		s:   cfg,
		cl:  utils.CreateClient(),
		log: logging.GetLogger(cfg.Verbose),
	}
}

// Download downloads the images according to the given configuration
func (d *Downloader) Download() (uint32, error) {
	media, err := d.getFilteredMedia()
	if err != nil {
		return 0, fmt.Errorf("error getting media from reddit: %v", err)
	}

	count, err := d.downloadMedia(media)
	if err != nil {
		return 0, fmt.Errorf("error downloading the media from reddit: %v", err)
	}

	return count, nil
}

// getFilteredMedia fetches and then returns a slice of downloadable posts according to the given configuration.
func (d *Downloader) getFilteredMedia() ([]toDownload, error) {
	media := make([]toDownload, 0, d.s.Count)
	for len(media) < d.s.Count {
		d.log.Debug("Fetching posts")
		posts, err := d.getPosts()
		if err != nil {
			return nil, fmt.Errorf("error gettings posts: %v", err)
		}
		converted := d.postsToMedia(posts)
		converted = d.applyFilters(converted, d.fs)

		// Fill until we get the desired amount
		for _, m := range converted {
			if len(media) >= d.s.Count {
				break
			}
			if slices.Contains(media, m) {
				continue
			}
			media = append(media, m)
		}

		if len(posts.Data.Children) == 0 || posts.Data.After == d.s.After {
			d.log.Debug("There's no more posts to load")
			break
		}
		d.s.After = posts.Data.After
		if len(d.s.After) == 0 {
			d.log.Debug("We might have got rate limited")
		}
		d.log.Debug("Current post count", zap.Int("count", len(media)))
		// We will sleep for 5 seconds after each iteration to ensure that we don't hit the rate limiting
		time.Sleep(5 * time.Second)
	}
	return media, nil
}

func (d *Downloader) applyFilters(dl []toDownload, fs []Filter) []toDownload {
	d.log.Debug("Filtering posts...")
	if len(fs) == 0 {
		return dl
	}

	f := make([]toDownload, 0)
	for _, ff := range fs {
		f = append(f, ff.Filter(dl, &d.s)...)
	}
	return f
}

// downloadMedia takes a slice of `downloadable` and a directory string and tries to download every media file
// to the specified directory, it does not stop if a single download fails.
func (d *Downloader) downloadMedia(media []toDownload) (uint32, error) {
	err := utils.NavigateToDirectory(d.s.Directory, true)
	if err != nil {
		return 0, fmt.Errorf("failed to navigate to directory, error: %v, directory: %v", err, d.s.Directory)
	}

	var total, finished, failed uint32
	total = uint32(len(media))

	if d.s.ShowProgress {
		printDownloadStatus(&total, &finished, &failed)
	}

	wg := new(sync.WaitGroup)
	for _, v := range media {
		wg.Add(1)
		go func(client *http.Client, data toDownload) {
			if err := d.downloadPost(data); err != nil {
				atomic.AddUint32(&failed, 1)
			} else {
				atomic.AddUint32(&finished, 1)
			}
			wg.Done()
		}(d.cl, v)
	}
	wg.Wait()
	return finished, nil
}

// getPosts fetches a json file from reddit containing information about the posts using the given configuration.
func (d *Downloader) getPosts() (*posts, error) {
	URL := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s",
		d.s.Subreddit, d.s.Sorting, d.s.Count, d.s.Timeframe)

	if len(d.s.After) > 0 {
		URL = fmt.Sprintf("%s&after=%s&count=%d", URL, d.s.After, d.s.Count)
	}

	request, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating a request: %v, URL: %v", err, URL)
	}
	request.Header.Add("User-Agent", "go:getter")

	response, err := d.cl.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetching from reddit failed, error: %v, URL: %v", err, URL)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			d.log.Error(err)
		}
	}(response.Body)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status in response: %v", http.StatusText(response.StatusCode))
	}

	posts := new(posts)
	if err := json.NewDecoder(response.Body).Decode(posts); err != nil {
		return nil, fmt.Errorf("error decoding posts: %v", err)
	}

	return posts, nil
}

// downloadPost downloads a single media file and stores it in the specified directory.
func (d *Downloader) downloadPost(v toDownload) error {
	response, err := d.cl.Get(v.Data.URL)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			d.log.Error(err)
		}
	}(response.Body)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status in response: %v", http.StatusText(response.StatusCode))
	}
	extension := ""
	if v.IsVideo {
		extension = "mp4" // if we didn't manage to figure out the video extension, assume mp4
	} else {
		extension = "jpg" // if we didn't manage to figure out the image extension, assume jpg
	}

	nameAndExt := strings.Split(response.Request.URL.Path, ".")
	if len(nameAndExt) == 2 {
		extension = nameAndExt[1]
	}

	filename, err := utils.CreateFilename(v.Name, extension)
	if err != nil {
		return fmt.Errorf("error when downloading to disk: %v", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error when creating a file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			d.log.Error(err)
		}
	}(file)

	_, err = io.Copy(file, response.Body)
	if err != nil {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("error when deleting the created file after a failed copy: %v", err)
		}
		return fmt.Errorf("error when copying the file to disk: %v", err)
	}
	return nil
}

// Converts posts to media depending on the configuration, leaving only the required types of media in
func (d *Downloader) postsToMedia(posts *posts) []toDownload {
	d.log.Debug("Converting posts to media...")
	media := make([]toDownload, 0)
	for _, post := range posts.Data.Children {
		if post.Data.IsVideo && d.s.IncludeVideo {
			media = append(media, toDownload{
				Name:    post.Data.Title,
				Data:    createVideo(&post.Data.Media.RedditVideo),
				IsVideo: true,
			})
		} else {
			for _, img := range post.Data.Preview.Images {
				img.Source.URL = strings.ReplaceAll(img.Source.URL, "&amp;s", "&s")
				media = append(media, toDownload{
					Name:    post.Data.Title,
					Data:    createImage(&img),
					IsVideo: false,
				})
			}
		}
	}
	return media
}

func createVideo(v *RedditVideo) imgData {
	v.ScrubberMediaURL = strings.ReplaceAll(v.ScrubberMediaURL, "&amp;s", "&s")
	d := imgData{
		URL:    v.ScrubberMediaURL,
		Width:  v.Width,
		Height: v.Height,
	}
	return d
}

func createImage(i *image) imgData {
	i.Source.URL = strings.ReplaceAll(i.Source.URL, "&amp;s", "&s")
	d := imgData{
		URL:    i.Source.URL,
		Width:  i.Source.Width,
		Height: i.Source.Height,
	}
	return d
}

// printDownloadStatus runs a background goroutine that prints the download status.
func printDownloadStatus(total *uint32, finished *uint32, failed *uint32) {
	go func(total, finished, failed *uint32) {
		for {
			fmt.Printf("\rTotal images: %d, Finished: %d, Failed: %d", *total, *finished, *failed)
			if *total == (*finished + *failed) {
				fmt.Println()
				return
			}
		}
	}(total, finished, failed)
}
