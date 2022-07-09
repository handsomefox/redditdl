package downloader

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"redditdl/pkg/config"
	"redditdl/utils"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

// getFilteredMedia fetches and then returns a slice of downloadable posts according to the given configuration.
func getFilteredMedia(c config.Configuration) ([]downloadable, error) {
	media := make([]downloadable, 0, c.Count)
	for len(media) < c.Count {
		logger.Debug("Fetching posts")
		posts, err := getPosts(&c)
		if err != nil {
			return nil, fmt.Errorf("error gettings posts: %v", err)
		}
		converted := postsToMedia(posts, &c)
		applyFilters(converted, filters, c)

		// Fill until we get the desired amount
		for _, m := range converted {
			if len(media) >= c.Count {
				break
			}
			if slices.Contains(media, m) {
				continue
			}
			media = append(media, m)
		}

		if len(posts.Data.Children) == 0 || posts.Data.After == c.After {
			logger.Debug("There's no more posts to load")
			break
		}
		c.After = posts.Data.After
		if len(c.After) == 0 {
			logger.Debug("We might have got rate limited")
		}
		logger.Debug("Current post count", zap.Int("count", len(media)))
		// We will sleep for 5 seconds after each iteration to ensure that we don't hit the rate limiting
		time.Sleep(5 * time.Second)
	}
	return media, nil
}

// downloadMedia takes a slice of `downloadable` and a directory string and tries to download every media file
// to the specified directory, it does not stop if a single download fails.
func downloadMedia(media []downloadable, c config.Configuration) (uint32, error) {
	err := navigateToDirectory(c.Directory)
	if err != nil {
		return 0, fmt.Errorf("failed to navigate to directory, error: %v, directory: %v", err, c.Directory)
	}

	var total, finished, failed uint32
	total = uint32(len(media))

	if c.ShowProgress {
		printDownloadStatus(&total, &finished, &failed)
	}

	wg := new(sync.WaitGroup)
	for _, v := range media {
		wg.Add(1)
		go func(client *http.Client, data downloadable) {
			if err := downloadPost(data); err != nil {
				atomic.AddUint32(&failed, 1)
			} else {
				atomic.AddUint32(&finished, 1)
			}
			wg.Done()
		}(client, v)
	}
	wg.Wait()
	return finished, nil
}

// getPosts fetches a json file from reddit containing information about the posts using the given configuration.
func getPosts(c *config.Configuration) (*posts, error) {
	URL := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s",
		c.Subreddit, c.Sorting, c.Count, c.Timeframe)

	if len(c.After) > 0 {
		URL = fmt.Sprintf("%s&after=%s&count=%d", URL, c.After, c.Count)
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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error(err)
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
func downloadPost(v downloadable) error {
	response, err := client.Get(v.Data.URL)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error(err)
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
			logger.Error(err)
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
func postsToMedia(posts *posts, c *config.Configuration) []downloadable {
	logger.Debug("Converting posts to media...")
	media := make([]downloadable, 0)
	for _, post := range posts.Data.Children {
		if post.Data.IsVideo && c.IncludeVideo {
			media = append(media, downloadable{
				Name:    post.Data.Title,
				Data:    createVideo(&post.Data.Media.RedditVideo),
				IsVideo: true,
			})
		} else {
			for _, img := range post.Data.Preview.Images {
				img.Source.URL = strings.ReplaceAll(img.Source.URL, "&amp;s", "&s")
				media = append(media, downloadable{
					Name:    post.Data.Title,
					Data:    createImage(&img),
					IsVideo: false,
				})
			}
		}
	}
	return media
}

func createVideo(v *RedditVideo) data {
	v.ScrubberMediaURL = strings.ReplaceAll(v.ScrubberMediaURL, "&amp;s", "&s")
	d := data{
		URL:    v.ScrubberMediaURL,
		Width:  v.Width,
		Height: v.Height,
	}
	return d
}

func createImage(i *image) data {
	i.Source.URL = strings.ReplaceAll(i.Source.URL, "&amp;s", "&s")
	d := data{
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

// navigateToDirectory moves to the provided directory and creates it if necessary.
func navigateToDirectory(dir string) error {
	if err := os.Mkdir(dir, os.ModePerm); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("error creating a directory for media, %v", err)
		} else {
			logger.Debug("Directory already exists, but we will still continue")
		}
	}
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("error navigating to directory, %v", err)
	}
	return nil
}
