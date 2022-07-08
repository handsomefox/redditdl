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

// getFilteredImages fetches and then returns a slice of filtered images according to the given configuration.
func getFilteredImages(c config.Configuration) ([]finalImage, error) {
	logger.Debug("Fetching posts")
	posts, err := getPosts(c)
	if err != nil {
		return nil, fmt.Errorf("error gettings posts: %v", err)
	}

	images := filterImages(posts, c.MinWidth, c.MinHeight)

	// Continue fetching data until we get the required amount of images
	if len(images) < c.Count {
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

			nextPageImages := filterImages(posts, c.MinWidth, c.MinHeight)

			// Fill until we get the desired amount
			for _, image := range nextPageImages {
				if len(images) >= c.Count {
					finished = true
				}
				// Ignore duplicates
				if slices.Contains(images, image) {
					continue
				}
				images = append(images, image)
			}

			// We will sleep for 5 seconds after each iteration to ensure that we don't hit the rate limiting
			time.Sleep(5 * time.Second)
		}
	}
	return ensureLength(images, c.Count), nil
}

// downloadImages takes a slice of FinalImages and a directory string and tries to download every image
// to the specified directory, it does not stop if a single download fails.
func downloadImages(images []finalImage, c config.Configuration) (int, error) {
	// Create and move to the specified directory
	err := navigateToDirectory(c.Directory)
	if err != nil {
		return 0, fmt.Errorf("failed to navigate to directory, error: %v, directory: %v", err, c.Directory)
	}

	var total, finished, failed uint32
	total = uint32(len(images))

	if c.ShowProgress {
		printDownloadStatus(&total, &finished, &failed)
	}

	wg := new(sync.WaitGroup)
	// The loop which downloads the images.
	for i, v := range images {
		wg.Add(1)
		go func(client *http.Client, index int, imageData finalImage) {
			if err := downloadImage(index, imageData); err != nil {
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

// downloadImage downloads the image and stores it in the specified directory.
func downloadImage(i int, v finalImage) error {
	response, err := client.Get(v.Data.URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status in response: %v", http.StatusText(response.StatusCode))
	}

	filenameAndExtension := strings.Split(response.Request.URL.Path, ".")
	extension := "jpg"
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

// filterImages converts images inside the posts to FinalImages and filters them by specified resolution.
func filterImages(posts *posts, minWidth, minHeight int) []finalImage {
	images := make([]finalImage, 0)
	for _, post := range posts.Data.Children {
		for _, image := range post.Data.Preview.Images {
			if image.Source.Width < int64(minWidth) || image.Source.Height < int64(minHeight) {
				continue
			}
			image.Source.URL = strings.Replace(image.Source.URL, "&amp;s", "&s", 1)
			images = append(images, finalImage{
				Name: post.Data.Title,
				Data: imageData{
					URL:    image.Source.URL,
					Width:  image.Source.Width,
					Height: image.Source.Width,
				},
			})
		}
	}
	return images
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
func ensureLength(posts []finalImage, requiredLength int) []finalImage {
	if len(posts) > requiredLength {
		return posts[:requiredLength]
	}
	return posts
}

// navigateToDirectory moves to the provided directory and creates it if necessary.
func navigateToDirectory(directory string) error {
	if err := os.Mkdir(directory, os.ModePerm); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("error creating a directory for images, %v", err)
		} else {
			logger.Debug("Directory already exists, but we will still continue")
		}
	}
	if err := os.Chdir(directory); err != nil {
		return fmt.Errorf("error navigating to directory, %v", err)
	}
	return nil
}
