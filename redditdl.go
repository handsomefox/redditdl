package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flytam/filenamify"
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
	sub := flag.String("sub", "wallpaper", "Subreddit name")
	sort := flag.String("sort", "top", "How to sort (controversial, best, hot, new, random, rising, top)")
	tf := flag.String("tf", "all", "Timeframe from which to get the posts (hour, day, week, month, year, all)")
	dir := flag.String("dir", "images", "Specifies the directory where to download the images")

	cnt := flag.Int("count", 1, "Amount of images to download")
	minX := flag.Int("x", 1920, "minimal width of the image to download")
	minY := flag.Int("y", 1080, "minimal height of the image to download")

	flag.Parse()

	subreddit = *sub
	sorting = *sort
	timeframe = *tf
	directory = *dir

	count = *cnt
	minWidth = *minX
	minHeight = *minY
}

// filteredImage represents the image information which is required to filter by resolution, download and store it.
type filteredImage struct {
	url    string
	name   string
	width  int64
	height int64
}

// createClient returns a pointer to http.Client configured to work with reddit.
func createClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
		},
		Timeout: 60 * time.Second,
	}
}

func main() {
	fmt.Printf("Using flags:\nSubreddit=%s, Limit=%d, Listing=%s, Timeframe=%s, Directory=%s, Min Width=%v, Min Height=%v\n",
		subreddit, count, sorting, timeframe, directory, minWidth, minHeight)

	url := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s",
		subreddit, sorting, count, timeframe)

	resp, err := fetchFromReddit(url)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error, status code: %d\nHeaders: %+v\n", resp.StatusCode, resp.Header)
	}
	defer resp.Body.Close()

	fmt.Println("Decoding json...")

	var result Posts
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&result)
	if err != nil {
		log.Fatal(err)
	}

	images := toImages(result.Data.Children)

	if len(images) < int(count) {
		// If we didn't get the desired amount of images,
		// we will continue fetching images from the next page,
		// until we run out of pages
		lastAfter := result.Data.After
		finished := false
		for {
			currURL := url + "&after=" + lastAfter + "&count=" + strconv.Itoa(count)

			resp, err := fetchFromReddit(currURL)
			if err != nil {
				fmt.Printf("Couldn't fetch more images, error: %v\n", err)
				continue

			}
			if resp.StatusCode != http.StatusOK {
				fmt.Printf("Couldn't fetch more images, status code: %v\n", resp.StatusCode)
				continue
			}
			defer resp.Body.Close()

			fmt.Println("Decoding json from the next page...")

			var result Posts
			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&result)
			if err != nil {
				fmt.Printf("Couldn't decode this page json: %v\n", err)
				continue
			}
			if len(result.Data.Children) == 0 || result.Data.After == lastAfter {
				fmt.Println("We can't load any more posts :/")
			}
			lastAfter = result.Data.After

			if len(lastAfter) == 0 {
				fmt.Println("There's probably something wrong with the request")
				break
			}

			newImages := toImages(result.Data.Children)
			for _, v := range newImages {
				if len(images) >= int(count) {
					finished = true
					break
				}
				if slices.Contains(images, v) {
					continue
				}
				images = append(images, v)
			}
			if finished {
				break
			}
		}
	}

	os.Mkdir(directory, os.ModePerm)
	os.Chdir(directory)

	fmt.Println("Downloading images:")
	total := uint64(len(images)) - 1

	var (
		finished, failed uint64
	)

	go timer(&total, &finished, &failed)

	client := createClient()
	wg := sync.WaitGroup{}
	for i, v := range images {
		wg.Add(1)
		go func(client *http.Client, i int, v filteredImage) {
			err := saveImage(client, i, v)
			if err != nil {
				atomic.AddUint64(&failed, 1)
			} else {
				atomic.AddUint64(&finished, 1)
			}
			wg.Done()
		}(client, i, v)
	}

	wg.Wait()
	fmt.Println("\nFinished downloading all the images!")
}

// fetchFromReddit fetches a json file from reddit containing information about posts using the given url.
func fetchFromReddit(url string) (*http.Response, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("User-Agent", "go:getter")

	fmt.Println("Requesting .json from reddit...")

	client := createClient()
	return client.Do(request)
}

// toImages converts an array of children that contain images to an array of filteredImage(s)
// it also filters by specified resolution, everything smaller than the specified resolution is not downloaded
func toImages(children []Child) []filteredImage {
	images := make([]filteredImage, 0)

	for _, v := range children {
		for _, v2 := range v.Data.Preview.Images {
			if v2.Source.Width < int64(minWidth) || v2.Source.Height < int64(minHeight) {
				continue
			}
			v2.Source.URL = strings.Replace(v2.Source.URL, "&amp;s", "&s", 1)
			images = append(images, filteredImage{
				url:    v2.Source.URL,
				name:   v.Data.Title,
				width:  v2.Source.Width,
				height: v2.Source.Width,
			})
		}
	}
	return images
}

// saveImage downloads the image and stores it in the specified directory.
func saveImage(client *http.Client, i int, v filteredImage) error {
	resp, err := client.Get(v.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("status code is not 200")
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

// timer is a background timer waiting for all the images to finish loading
func timer(total *uint64, finished *uint64, failed *uint64) {
	for {
		fmt.Printf("\rTotal images found in posts: %d, Finished: %d, Failed: %d", *total, *finished, *failed)
		if *total == (*finished + *failed) {
			return
		}
		time.Sleep(time.Millisecond * 100)
	}
}

// createFilename generates a valid filename for the image.
func createFilename(name string, idx int) string {
	str, err := filenamify.Filenamify(name, filenamify.Options{
		MaxLength: 250,
	})
	str += ".png"

	if err != nil {
		str = strconv.Itoa(idx+1) + ".png"
	}

	for fileExists(str) {
		newName := name + strconv.Itoa(idx)
		str, err = filenamify.Filenamify(newName, filenamify.Options{
			MaxLength: 240,
		})
		str += ".png"

		if err != nil {
			str = strconv.Itoa(idx+1) + ".png"
		}
	}

	return str
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
