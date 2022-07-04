package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"redditdl/config"
	"redditdl/utils"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/exp/slices"
)

var client = utils.CreateClient()

func main() {
	// Print the configuration
	log.Printf("Using parameters: Subreddit=%s, Limit=%d, Listing=%s, Timeframe=%s, Directory=%s, Min Width=%v, Min Height=%v\n",
		config.Subreddit, config.Count, config.Sorting, config.Timeframe,
		config.Directory, config.MinWidth, config.MinHeight)

	// Format URL
	url := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s",
		config.Subreddit, config.Sorting, config.Count, config.Timeframe)

	// Get required amount of images
	images, err := getImages(url, config.Count)
	if err != nil {
		log.Fatalf("Error getting images from reddit: %v\n", err)
	}

	// Download the images
	log.Println("Started downloading images")
	if err := downloadImages(images, config.Directory); err != nil {
		log.Fatalf("\nError downloading images: %v\n", err)
	}
	fmt.Println("\nFinished downloading!")
}

// getImages takes in a formatted reddit URL and an amount of images to download and returns
// a slice of FinalImages that were filtered
func getImages(URL string, count int) ([]FinalImage, error) {
	// Fetch required data
	posts, err := getPosts(URL)
	if err != nil {
		return nil, fmt.Errorf("error gettings posts: %v, URL: %v", err, URL)
	}

	// Filter the response
	images := filterImages(posts, config.MinWidth, config.MinHeight)

	// Continue fetching data until we get the required amount of images
	if len(images) < int(count) {
		// after is the id of last post
		lastAfter := posts.Data.After
		finished := false

		for !finished {
			currentURL := URL + "&after=" + lastAfter + "&count=" + strconv.Itoa(count)
			posts, err := getPosts(currentURL)
			if err != nil {
				return nil, fmt.Errorf("error gettings posts for the next page: %v, URL: %v", err, currentURL)
			}

			if len(posts.Data.Children) == 0 || posts.Data.After == lastAfter {
				log.Println("There's no more posts to load")
			}

			lastAfter = posts.Data.After
			if len(lastAfter) == 0 {
				log.Println("There's probably something wrong with the request, maybe we got rate limited")
				finished = true
			}

			nextPageImages := filterImages(posts, config.MinWidth, config.MinHeight)

			// Fill until we get the desired amount
			for _, image := range nextPageImages {
				if len(images) >= int(count) {
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
	return ensureLength(images, config.Count), nil
}

// downloadImages takes a slice of FinalImages and a directory string and tries to download every image
// to the specificed directory, it does not stop if a single download fails
func downloadImages(images []FinalImage, directory string) error {
	// Create and move to the specified directory
	err := navigateToDirectory(directory)
	if err != nil {
		return fmt.Errorf("failed to navigate to directory, error: %v, directory: %v", err, directory)
	}

	var (
		total    = uint32(len(images))
		finished uint32
		failed   uint32
	)

	wg := new(sync.WaitGroup)
	// this will print the download progress in the background
	printDownloadStatus(&total, &finished, &failed)

	// this is the loop which actually downloads the images
	for i, v := range images {
		wg.Add(1)
		go func(client *http.Client, index int, imageData FinalImage) {
			if err := downloadImage(index, imageData); err != nil {
				atomic.AddUint32(&failed, 1)
			} else {
				atomic.AddUint32(&finished, 1)
			}
			wg.Done()
		}(client, i, v)
	}
	wg.Wait()
	return nil
}

// getPosts fetches a json file from reddit containing information about the posts.
func getPosts(url string) (*Posts, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating a request: %v", err)
	}

	request.Header.Add("User-Agent", "go:getter")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetching from reddit failed, error: %v, URL: %v", err, url)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status in response: %v", http.StatusText(response.StatusCode))
	}

	posts := new(Posts)
	if err := json.NewDecoder(response.Body).Decode(posts); err != nil {
		return nil, fmt.Errorf("error decoding posts: %v", err)
	}

	return posts, nil
}

// downloadImage downloads the image and stores it in the specified directory.
func downloadImage(i int, v FinalImage) error {
	response, err := client.Get(v.Data.URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status in response: %v", http.StatusText(response.StatusCode))
	}

	filenameAndExtension := strings.Split(response.Request.URL.Path, ".")
	extension := "jpeg"
	if len(filenameAndExtension) == 2 {
		extension = filenameAndExtension[1]
	}

	filename, err := utils.CreateFilename(v.Name+"."+extension, i)
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
		os.Remove(filename)
		return fmt.Errorf("error when copying the file to disk: %v", err)
	}
	return nil
}

// filterImages converts images inside the posts to FinalImages and filters them by specified resolution
func filterImages(posts *Posts, minWidth, minHeight int) []FinalImage {
	images := make([]FinalImage, 0)
	for _, post := range posts.Data.Children {
		for _, image := range post.Data.Preview.Images {
			if image.Source.Width < int64(minWidth) || image.Source.Height < int64(minHeight) {
				continue
			}
			image.Source.URL = strings.Replace(image.Source.URL, "&amp;s", "&s", 1)
			images = append(images, FinalImage{
				Name: post.Data.Title,
				Data: ImageData{
					URL:    image.Source.URL,
					Width:  image.Source.Width,
					Height: image.Source.Width,
				},
			})
		}
	}
	return images
}

// printDownloadStatus is a background thread that prints the download status
func printDownloadStatus(total *uint32, finished *uint32, failed *uint32) {
	go func(total, finished, failed *uint32) {
		for {
			fmt.Printf("\rTotal images: %d, Finished: %d, Failed: %d", *total, *finished, *failed)
			if *total == (*finished + *failed) {
				return
			}
		}
	}(total, finished, failed)
}

// ensureLength ensures that the total amount of posts is the same as the one specified in the config
func ensureLength(posts []FinalImage, requiredLength int) []FinalImage {
	if len(posts) > requiredLength {
		return posts[:requiredLength]
	}
	return posts
}

// navigateToDirectory moves to the provided directory and creates it if neccessary
func navigateToDirectory(directory string) error {
	if err := os.Mkdir(directory, os.ModePerm); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("error creating a directory for images, %v", err)
		} else {
			log.Print("Directory already exists, but we will still continue")
		}
	}
	if err := os.Chdir(directory); err != nil {
		return fmt.Errorf("error navigating to directory, %v", err)
	}
	return nil
}
