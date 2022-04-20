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
)

type RedditResponse struct {
	Kind string `json:"kind"`
	Data Data   `json:"data"`
}

type Data struct {
	Children []Child `json:"children"`
}

type Child struct {
	Kind string    `json:"kind"`
	Data ChildData `json:"data"`
}

type ChildData struct {
	Subreddit string  `json:"subreddit"`
	Title     string  `json:"title"`
	Preview   Preview `json:"preview"`
}

type Preview struct {
	Images []Image `json:"images"`
}

type Image struct {
	Source Source `json:"source"`
}

type Source struct {
	URL    string  `json:"url"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Configuration
var (
	subreddit           string
	limit               int
	listing             string
	timeframe           string
	directory           string
	minWidth, minHeight int
)

// Global client
var (
	client *http.Client
)

type image struct {
	url    string
	name   string
	width  float64
	height float64
}

func main() {
	sub := flag.String("sub", "wallpaper", "Subreddit name")
	lim := flag.Int("count", 1, "Amount of posts to load")
	sort := flag.String("sort", "top", "How to sort (controversial, best, hot, new, random, rising, top)")
	tf := flag.String("tf", "all", "Timeframe from which to get the posts (hour, day, week, month, year, all)")
	dir := flag.String("dir", "images", "Specifies the directory where to download the images")
	minX := flag.Int("x", 1920, "minimal width of the image to download")
	minY := flag.Int("y", 1080, "minimal height of the image to download")

	flag.Parse()

	subreddit = *sub
	limit = *lim
	listing = *sort
	timeframe = *tf
	directory = *dir
	minWidth = *minX
	minHeight = *minY

	client = &http.Client{
		Transport: &http.Transport{
			TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
		},
		Timeout: 60 * time.Second,
	}

	fmt.Printf("Using flags:\nSubreddit=%s, Limit=%d, Listing=%s, Timeframe=%s, Directory=%s, Min Width=%v, Min Height=%v\n\n",
		subreddit, limit, listing, timeframe, directory, minWidth, minHeight)

	url := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s",
		subreddit, listing, limit, timeframe)
	resp, err := fetchFromReddit(url)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error, status code: %d\nHeaders: %+v\n", resp.StatusCode, resp.Header)
	}
	defer resp.Body.Close()

	fmt.Println("Decoding json...")
	result, err := decodeJSON(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	images := toImages(result.Data.Children)

	os.Mkdir(directory, os.ModePerm)
	os.Chdir(directory)

	fmt.Println("Downloading images.")
	total := uint64(len(images))

	var finished uint64
	var failed uint64

	go timer(&total, &finished, &failed)

	wg := sync.WaitGroup{}
	for i, v := range images {
		wg.Add(1)
		go func(i int, v image) {
			err := saveImage(i, v)
			if err != nil {
				atomic.AddUint64(&failed, 1)
			} else {
				atomic.AddUint64(&finished, 1)
			}
			wg.Done()
		}(i, v)
	}

	wg.Wait()
	fmt.Println("\nDone!")
}

func fetchFromReddit(url string) (*http.Response, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("User-Agent", "go:getter")

	fmt.Println("Requesting .json from reddit...")
	return client.Do(request)
}

func decodeJSON(r io.ReadCloser) (*RedditResponse, error) {
	var result RedditResponse
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&result)
	return &result, err
}

func toImages(children []Child) []image {
	images := make([]image, 0)

	for _, v := range children {
		for _, v2 := range v.Data.Preview.Images {
			if v2.Source.Width < float64(minWidth) || v2.Source.Height < float64(minHeight) {
				continue
			}
			v2.Source.URL = strings.Replace(v2.Source.URL, "&amp;s", "&s", 1)
			images = append(images, image{
				url:    v2.Source.URL,
				name:   v.Data.Title,
				width:  v2.Source.Width,
				height: v2.Source.Width,
			})
		}
	}
	return images
}

func saveImage(i int, v image) error {
	resp, err := client.Get(v.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("status code is not 200")
	}

	filename := makeFilename(v.name, i)

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

func timer(total *uint64, finished *uint64, failed *uint64) {
	for {
		fmt.Printf("\rTotal: %d, Finished: %d, Failed: %d", *total, *finished, *failed)
		if *total == (*finished + *failed) {
			return
		}
		time.Sleep(time.Millisecond * 100)
	}
}

func makeFilename(name string, idx int) string {
	str, err := filenamify.Filenamify(name, filenamify.Options{
		MaxLength: 250,
	})
	str += ".png"
	if err != nil {
		str = strconv.Itoa(idx+1) + ".png"
	}
	return str
}
