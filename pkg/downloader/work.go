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

	"golang.org/x/exp/slices"
)

// getFilteredMedia fetches and then returns a slice of downloadable posts according to the given configuration.
func getFilteredMedia(c config.Configuration) ([]downloadable, error) {
	logger.Debug("Fetching posts")
	posts, err := getPosts(c)
	if err != nil {
		return nil, fmt.Errorf("error gettings posts: %v", err)
	}

	media := postsToMedia(posts, c)
	filtered := filterByResolution(media, c.MinWidth, c.MinHeight)

	// Continue fetching data until we get the required amount of media
	if len(filtered) < c.Count {
		finished := false
		c.After = posts.Data.After

		for !finished {
			logger.Debug("Fetching posts...")
			posts, err := getPosts(c)
			if err != nil {
				return nil, fmt.Errorf("error gettings posts for the next page: %v", err)
			}

			if len(posts.Data.Children) == 0 || posts.Data.After == c.After {
				logger.Debug("There's no more posts to load")
			}

			c.After = posts.Data.After
			if len(c.After) == 0 {
				logger.Debug("There's probably something wrong with the request, maybe we got rate limited")
				finished = true
			}

			nextPageMedia := filterByResolution(postsToMedia(posts, c), c.MinWidth, c.MinHeight)

			// Fill until we get the desired amount
			for _, m := range nextPageMedia {
				if len(filtered) >= c.Count {
					finished = true
				}
				// Ignore duplicates
				if slices.Contains(filtered, m) {
					continue
				}
				filtered = append(filtered, m)
			}

			// We will sleep for 5 seconds after each iteration to ensure that we don't hit the rate limiting
			time.Sleep(5 * time.Second)
		}
	}
	return ensureLength(filtered, c.Count), nil
}

// downloadMedia takes a slice of `downloadable` and a directory string and tries to download every media file
// to the specified directory, it does not stop if a single download fails.
func downloadMedia(media []downloadable, c config.Configuration) (int, error) {
	// Create and move to the specified directory
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
	// The loop which does the download.
	for i, v := range media {
		wg.Add(1)
		go func(client *http.Client, index int, data downloadable) {
			if err := downloadPost(index, data); err != nil {
				atomic.AddUint32(&failed, 1)
			} else {
				atomic.AddUint32(&finished, 1)
			}
			wg.Done()
		}(client, i, v)
	}
	wg.Wait()

	return int(finished), nil
}

// getPosts fetches a json file from reddit containing information about the posts using the given configuration.
func getPosts(c config.Configuration) (*posts, error) {
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

// downloadPost downloads a single media file and stores it in the specified directory.
func downloadPost(i int, v downloadable) error {
	URL := ""
	if v.IsVideo {
		URL = v.VideoData.ScrubberMediaURL
	} else {
		URL = v.ImageData.URL
	}
	response, err := client.Get(URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status in response: %v", http.StatusText(response.StatusCode))
	}

	filenameAndExtension := strings.Split(response.Request.URL.Path, ".")
	extension := ""
	if v.IsVideo {
		extension = "mp4"
	} else {
		extension = "jpg"
	}
	if len(filenameAndExtension) == 2 {
		extension = filenameAndExtension[1]
	}

	filename, err := utils.CreateFilename(v.Name, extension, i)
	if err != nil {
		return fmt.Errorf("error when downloading to disk: %v", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error when creating a file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("error when deleting the created file after a failed copy: %v", err)
		}
		return fmt.Errorf("error when copying the file to disk: %v", err)
	}
	return nil
}

// Converts posts to media depeding on the configuration, leaving only the required types of media in
func postsToMedia(posts *posts, c config.Configuration) []downloadable {
	media := make([]downloadable, 0)

	for _, post := range posts.Data.Children {
		if post.Data.IsVideo {
			// Add the video
			if c.IncludeVideo {
				post.Data.Media.RedditVideo.ScrubberMediaURL = strings.ReplaceAll(post.Data.Media.RedditVideo.ScrubberMediaURL, "&amp;s", "&s")
				media = append(media, downloadable{
					Name:      post.Data.Title,
					IsVideo:   true,
					ImageData: nil,
					VideoData: &post.Data.Media.RedditVideo,
				})
			}
		} else {
			// Add the images
			for _, img := range post.Data.Preview.Images {
				img.Source.URL = strings.ReplaceAll(img.Source.URL, "&amp;s", "&s")
				media = append(media, downloadable{
					Name:      post.Data.Title,
					IsVideo:   false,
					ImageData: &img.Source,
					VideoData: nil,
				})
			}
		}

	}

	return media
}

// filterByResolution filters posts by specified resolution.
func filterByResolution(media []downloadable, minWidth, minHeight int) []downloadable {
	filtered := make([]downloadable, 0)
	for _, m := range media {
		if m.IsVideo && m.VideoData != nil {
			if m.VideoData.Width >= minWidth && m.VideoData.Height >= minHeight {
				filtered = append(filtered, m)
			}
		} else if m.ImageData != nil {
			if m.ImageData.Width >= int64(minWidth) && m.ImageData.Height >= int64(minHeight) {
				filtered = append(filtered, m)
			}
		}
	}
	return filtered
}

// printDownloadStatus is a background thread that prints the download status.
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

// ensureLength ensures that the total amount of posts is the same as the one specified in the configuration.
func ensureLength(posts []downloadable, requiredLength int) []downloadable {
	if len(posts) > requiredLength {
		return posts[:requiredLength]
	}
	return posts
}

// navigateToDirectory moves to the provided directory and creates it if necessary.
func navigateToDirectory(directory string) error {
	if err := os.Mkdir(directory, os.ModePerm); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("error creating a directory for media, %v", err)
		} else {
			logger.Debug("Directory already exists, but we will still continue")
		}
	}
	if err := os.Chdir(directory); err != nil {
		return fmt.Errorf("error navigating to directory, %v", err)
	}
	return nil
}
