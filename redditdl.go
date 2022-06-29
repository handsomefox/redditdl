package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"golang.org/x/exp/slices"
)

// Configuration
var (
	subreddit string
	sorting   string
	timeframe string
	directory string
	count     int
	minWidth  int
	minHeight int
)

func init() {
	subFlag := flag.String("sub", "wallpaper", "Subreddit name")
	sortFlag := flag.String("sort", "top", "How to sort (controversial, best, hot, new, random, rising, top)")
	timeframeFlag := flag.String("tf", "all", "Timeframe from which to get the posts (hour, day, week, month, year, all)")
	directoryFlag := flag.String("dir", "images", "Specifies the directory where to download the images")
	countFlag := flag.Int("count", 1, "Amount of images to download")
	minWidthFlag := flag.Int("x", 1920, "minimal width of the image to download")
	minHeightFlag := flag.Int("y", 1080, "minimal height of the image to download")

	flag.Parse()

	subreddit = *subFlag
	sorting = *sortFlag
	timeframe = *timeframeFlag
	directory = *directoryFlag
	count = *countFlag
	minWidth = *minWidthFlag
	minHeight = *minHeightFlag
}

var client *http.Client = createClient()

func main() {
	log.Printf("Using flags:\nSubreddit=%s, Limit=%d, Listing=%s, Timeframe=%s, Directory=%s, Min Width=%v, Min Height=%v\n",
		subreddit, count, sorting, timeframe, directory, minWidth, minHeight)

	url := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s",
		subreddit, sorting, count, timeframe)

	resp, err := fetchURL(url)
	if err != nil {
		log.Printf("Fetching from reddit failed, error: %v, URL: %v\n", err, url)
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected status in response: %v\n", http.StatusText(resp.StatusCode))
	}
	defer resp.Body.Close()

	log.Println("Decoding json...")

	result := new(Posts)
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		log.Fatal("Error decoding response: " + err.Error())
	}

	images := postsToImages(result)

	if len(images) < int(count) {
		// If we didn't get the desired amount of images,
		// we will continue fetching images from the next page,
		// until we run out of pages or get the needed amount of images.

		// after is the id of last post
		lastAfter := result.Data.After

		finished := false
		for !finished {
			currentURL := url + "&after=" + lastAfter + "&count=" + strconv.Itoa(count)

			resp, err := fetchURL(currentURL)
			if err != nil {
				log.Printf("Fetching from reddit failed, error: %v, URL: %v\n", err, currentURL)
				continue
			}
			if resp.StatusCode != http.StatusOK {
				log.Printf("Unexpected status in response: %v\n", http.StatusText(resp.StatusCode))
				continue
			}
			defer resp.Body.Close()

			log.Println("Decoding json for the next page...")

			result := new(Posts)
			if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
				log.Fatal("Error decoding response: " + err.Error())
				continue
			}

			if len(result.Data.Children) == 0 || result.Data.After == lastAfter {
				log.Println("There's no more posts to load")
			}

			lastAfter = result.Data.After
			if len(lastAfter) == 0 {
				log.Println("There's probably something wrong with the request, maybe we got rate limited")
				finished = true
			}

			nextPageImages := postsToImages(result)

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
		}
	}

	if err := os.Mkdir(directory, os.ModePerm); err != nil {
		if !errors.Is(err, os.ErrExist) {
			log.Fatalf("Error creating a directory for images, error: %v, directory: %v\n", err, directory)
		} else {
			log.Print("Directory already exists, but we will still continue")
		}
	}
	if err := os.Chdir(directory); err != nil {
		log.Fatalf("Error navigating to directory, error: %v, directory: %v\n", err, directory)
	}

	log.Println("Started downloading images")
	var (
		total    = uint32(len(images))
		finished uint32
		failed   uint32
	)

	wg := new(sync.WaitGroup)
	go printDownloadStatus(&total, &finished, &failed)

	for i, v := range images {
		wg.Add(1)
		go func(client *http.Client, index int, imageData filteredImage) {
			if err := downloadToDisk(index, imageData); err != nil {
				atomic.AddUint32(&failed, 1)
			} else {
				atomic.AddUint32(&finished, 1)
			}
			wg.Done()
		}(client, i, v)
	}
	wg.Wait()
}

// fetchURL fetches a json file from reddit containing information about posts using the given url.
func fetchURL(url string) (*http.Response, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("User-Agent", "go:getter")
	// log.Println("Requesting .json from reddit...")

	return client.Do(request)
}

// downloadToDisk downloads the image and stores it in the specified directory.
func downloadToDisk(i int, v filteredImage) error {
	resp, err := client.Get(v.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status in response: %v", http.StatusText(resp.StatusCode))
	}

	filename := createFilename(v.name, i)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(filename)
		return err
	}
	return nil
}
