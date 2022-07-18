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

var (
	client = utils.CreateClient() // client used for http requests
	log    *zap.SugaredLogger     // logger from logging package
	after  = ""                   // After is a post ID, which is used to fetch posts after that ID
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

// Download downloads the images according to the given configuration
func Download(ds DownloaderSettings, fs []Filter) (uint32, error) {
	log = logging.GetLogger(ds.Verbose)
	media, err := fetchLoop(&ds, fs)
	if err != nil {
		return 0, fmt.Errorf("error getting media from reddit: %v", err)
	}

	return saveLoop(&ds, media)
}

// this is the fetch loop, where we get needed amount of filtered posts.
func fetchLoop(ds *DownloaderSettings, fs []Filter) ([]toDownload, error) {
	media := make([]toDownload, 0, ds.Count)

	for len(media) < ds.Count {
		log.Debug("Fetching posts")

		posts, err := getPosts(ds)
		if err != nil {
			return nil, fmt.Errorf("error gettings posts: %v", err)
		}
		values := postsToMedia(ds, posts.Data.Children)
		values = applyFilters(ds, values, fs)

		// Fill until we get the desired amount
		for _, v := range values {
			if len(media) >= ds.Count {
				break
			}
			if slices.Contains(media, v) {
				continue
			}
			media = append(media, v)
		}

		if len(posts.Data.Children) == 0 || posts.Data.After == after || len(posts.Data.After) == 0 {
			log.Info("There's no more posts to load (or we got rate limited)")
			break
		}
		after = posts.Data.After

		log.Debug("Current post count: ", len(media))
		// We will sleep for 7 seconds after each iteration to ensure that we don't hit the rate limiting
		time.Sleep(7 * time.Second)
	}
	return media, nil
}

// applyFilters applies every filter from the slice of []Filter and returns the mutated slice
// if there are no filters, the original slice is returned
func applyFilters(ds *DownloaderSettings, td []toDownload, fs []Filter) []toDownload {
	if len(fs) == 0 { // return the original posts if there are no filters
		return td
	}
	log.Debug("Filtering posts...")

	f := make([]toDownload, 0, len(td))
	for _, ff := range fs {
		f = append(f, ff.Filter(td, ds)...)
	}
	return f
}

// saveLoop is the loop for saving files to disk. It takes a slice of `downloadable` and a directory string and tries
//  to download every media file to the specified directory.
func saveLoop(ds *DownloaderSettings, td []toDownload) (uint32, error) {
	if err := utils.NavigateToDirectory(ds.Directory, true); err != nil {
		return 0, fmt.Errorf("failed to navigate to directory, error: %v, directory: %v", err, ds.Directory)
	}

	var (
		finished, failed uint32
		total            = uint32(len(td))
	)

	if ds.ShowProgress {
		go func(t, fn, fl *uint32) {
			for {
				fmt.Printf("\rTotal images: %d, Finished: %d, Failed: %d", *t, *fn, *fl)
				if *t == (*fn + *fl) {
					fmt.Println()
					return
				}
			}
		}(&total, &finished, &failed)
	}

	wg := new(sync.WaitGroup)
	for _, v := range td {
		wg.Add(1)
		go func(data toDownload) {
			if err := downloadMedia(&data); err != nil {
				atomic.AddUint32(&failed, 1)
			} else {
				atomic.AddUint32(&finished, 1)
			}
			wg.Done()
		}(v)
	}
	wg.Wait()
	return finished, nil
}

// getPosts fetches a json file from reddit containing information about the posts using the given configuration.
func getPosts(ds *DownloaderSettings) (*posts, error) {
	URL := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s",
		ds.Subreddit, ds.Sorting, ds.Count, ds.Timeframe)

	if len(after) > 0 {
		URL = fmt.Sprintf("%s&after=%s&count=%d", URL, after, ds.Count)
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

// downloadMedia downloads a single media file and stores it in the specified directory.
func downloadMedia(td *toDownload) error {
	response, err := client.Get(td.Data.URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status in response: %v", http.StatusText(response.StatusCode))
	}

	var extension string
	if td.IsVideo {
		extension = "mp4" // if we didn't manage to figure out the video extension, assume mp4
	} else {
		extension = "jpg" // if we didn't manage to figure out the image extension, assume jpg
	}

	nameAndExt := strings.Split(response.Request.URL.Path, ".")
	if len(nameAndExt) == 2 {
		extension = nameAndExt[1]
	}

	filename, err := utils.CreateFilename(td.Name, extension)
	if err != nil {
		return fmt.Errorf("error when downloading to disk: %v", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error when creating a file: %v", err)
	}
	defer file.Close()

	if _, err = io.Copy(file, response.Body); err != nil {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("error when deleting the created file after a failed copy: %v", err)
		}
		return fmt.Errorf("error when copying the file to disk: %v", err)
	}

	return nil
}

// Converts posts to media depending on the configuration, leaving only the required types of media in
func postsToMedia(ds *DownloaderSettings, cd []child) []toDownload {
	log.Debug("Converting posts to media...")
	media := make([]toDownload, 0)

	for _, v := range cd {
		if v.Data.IsVideo && ds.IncludeVideo { // Append video
			media = append(media, newVideo(v.Data.Title, &v.Data.Media.RedditVideo))
		} else { // Append images
			for _, img := range v.Data.Preview.Images {
				media = append(media, newImage(v.Data.Title, &img.Source))
			}
		}
	}

	return media
}

func newVideo(title string, rv *RedditVideo) toDownload {
	return toDownload{
		Name: title,
		Data: imgData{
			URL:    strings.ReplaceAll(rv.ScrubberMediaURL, "&amp;s", "&s"),
			Width:  rv.Width,
			Height: rv.Height,
		},
		IsVideo: true,
	}
}

func newImage(title string, id *imageData) toDownload {
	return toDownload{
		Name: title,
		Data: imgData{
			URL:    strings.ReplaceAll(id.URL, "&amp;s", "&s"),
			Width:  id.Width,
			Height: id.Height,
		},
		IsVideo: false,
	}
}
