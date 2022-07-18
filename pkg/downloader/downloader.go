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

// DownloaderSettings is the configuration for the Downloader
type DownloaderSettings struct {
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

// Use New() to create a Downloader
type Downloader struct {
	fs []Filter
	ds *DownloaderSettings

	client *http.Client
	log    *zap.SugaredLogger

	after string // After is a post ID, which is used to fetch posts after that ID
}

// New gives a new instance of Downloader
func New(cfg DownloaderSettings, filters []Filter) *Downloader {
	return &Downloader{
		fs:     filters,
		ds:     &cfg,
		client: utils.CreateClient(),
		log:    logging.GetLogger(cfg.Verbose),
		after:  "",
	}
}

// Download downloads the images according to the given configuration
func (dler *Downloader) Download() (uint32, error) {
	media, err := dler.fetchLoop()
	if err != nil {
		return 0, fmt.Errorf("error getting media from reddit: %v", err)
	}

	return dler.saveLoop(media)
}

// this is the fetch loop, where we get needed amount of filtered posts.
func (dler *Downloader) fetchLoop() ([]toDownload, error) {
	media := make([]toDownload, 0, dler.ds.Count)

	for len(media) < dler.ds.Count {
		dler.log.Debug("Fetching posts")

		posts, err := dler.getPosts()
		if err != nil {
			return nil, fmt.Errorf("error gettings posts: %v", err)
		}

		values := dler.postsToMedia(posts)
		values = dler.applyFilters(values, dler.fs)

		// Fill until we get the desired amount
		for _, v := range values {
			if len(media) >= dler.ds.Count {
				break
			}
			if slices.Contains(media, v) {
				continue
			}
			media = append(media, v)
		}

		if len(posts.Data.Children) == 0 || posts.Data.After == dler.after || len(posts.Data.After) == 0 {
			dler.log.Info("There's no more posts to load (or we got rate limited)")
			break
		}
		dler.after = posts.Data.After

		dler.log.Debug("Current post count: ", len(media))
		// We will sleep for 5 seconds after each iteration to ensure that we don't hit the rate limiting
		time.Sleep(5 * time.Second)
	}

	return media, nil
}

// applyFilters applies every filter from the slice of []Filter and returns the mutated slice
// if there are no filters, the original slice is returned
func (dler *Downloader) applyFilters(dl []toDownload, fs []Filter) []toDownload {
	if len(fs) == 0 { // return the original posts if there are no filters
		return dl
	}
	dler.log.Debug("Filtering posts...")

	f := make([]toDownload, 0, dler.ds.Count)
	for _, ff := range fs {
		f = append(f, ff.Filter(dl, dler.ds)...)
	}
	return f
}

// saveLoop is the loop for saving files to disk. It takes a slice of `downloadable` and a directory string and tries
//  to download every media file to the specified directory.
func (dler *Downloader) saveLoop(media []toDownload) (uint32, error) {
	if err := utils.NavigateToDirectory(dler.ds.Directory, true); err != nil {
		return 0, fmt.Errorf("failed to navigate to directory, error: %v, directory: %v", err, dler.ds.Directory)
	}

	var total, finished, failed uint32
	total = uint32(len(media))

	if dler.ds.ShowProgress {
		go func(total, finished, failed *uint32) {
			for {
				fmt.Printf("\rTotal images: %d, Finished: %d, Failed: %d", *total, *finished, *failed)
				if *total == (*finished + *failed) {
					fmt.Println()
					return
				}
			}
		}(&total, &finished, &failed)
	}

	wg := new(sync.WaitGroup)
	for _, v := range media {
		wg.Add(1)
		go func(client *http.Client, data toDownload) {
			if err := dler.downloadMedia(&data); err != nil {
				atomic.AddUint32(&failed, 1)
			} else {
				atomic.AddUint32(&finished, 1)
			}
			wg.Done()
		}(dler.client, v)
	}
	wg.Wait()
	return finished, nil
}

// getPosts fetches a json file from reddit containing information about the posts using the given configuration.
func (dler *Downloader) getPosts() (*posts, error) {
	URL := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s",
		dler.ds.Subreddit, dler.ds.Sorting, dler.ds.Count, dler.ds.Timeframe)

	if len(dler.after) > 0 {
		URL = fmt.Sprintf("%s&after=%s&count=%d", URL, dler.after, dler.ds.Count)
	}

	request, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating a request: %v, URL: %v", err, URL)
	}
	request.Header.Add("User-Agent", "go:getter")

	response, err := dler.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetching from reddit failed, error: %v, URL: %v", err, URL)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			dler.log.Error(err)
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

// downloadMedia downloads a single media file and stores it in the specified directory.
func (dler *Downloader) downloadMedia(v *toDownload) error {
	response, err := dler.client.Get(v.Data.URL)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			dler.log.Error(err)
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
			dler.log.Error(err)
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
func (dler *Downloader) postsToMedia(posts *posts) []toDownload {
	dler.log.Debug("Converting posts to media...")
	media := make([]toDownload, 0)
	for _, post := range posts.Data.Children {
		if post.Data.IsVideo && dler.ds.IncludeVideo {
			media = append(media, toDownload{
				Name:    post.Data.Title,
				Data:    newVideo(&post.Data.Media.RedditVideo),
				IsVideo: true,
			})
		} else {
			for _, img := range post.Data.Preview.Images {
				img.Source.URL = strings.ReplaceAll(img.Source.URL, "&amp;s", "&s")
				media = append(media, toDownload{
					Name:    post.Data.Title,
					Data:    newImage(&img),
					IsVideo: false,
				})
			}
		}
	}
	return media
}

func newVideo(v *RedditVideo) imgData {
	v.ScrubberMediaURL = strings.ReplaceAll(v.ScrubberMediaURL, "&amp;s", "&s")
	d := imgData{
		URL:    v.ScrubberMediaURL,
		Width:  v.Width,
		Height: v.Height,
	}
	return d
}

func newImage(i *image) imgData {
	i.Source.URL = strings.ReplaceAll(i.Source.URL, "&amp;s", "&s")
	d := imgData{
		URL:    i.Source.URL,
		Width:  i.Source.Width,
		Height: i.Source.Height,
	}
	return d
}
