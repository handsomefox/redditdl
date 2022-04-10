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
	"time"

	"github.com/flytam/filenamify"
)

type Top struct {
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
	subreddit string
	limit     int
	listing   string
	timeframe string
	directory string
)

func main() {
	sub := flag.String("sub", "wallpaper", "Subreddit name")
	lim := flag.Int("count", 1, "Amount of posts to load")
	sort := flag.String("sort", "top", "How to sort (controversial, best, hot, new, random, rising, top)")
	tf := flag.String("tf", "all", "Timeframe from which to get the posts (hour, day, week, month, year, all)")
	dir := flag.String("dir", "images", "Specifies the directory where to download the images")

	flag.Parse()

	subreddit = *sub
	limit = *lim
	listing = *sort
	timeframe = *tf
	directory = *dir

	fmt.Printf("Using flags:\nSubreddit=%s, Limit=%d, Listing=%s, Timeframe=%s, Directory=%s\n\n", subreddit, limit, listing, timeframe, directory)

	client := &http.Client{
		Transport: &http.Transport{
			TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
		},
		Timeout: 60 * time.Second,
	}

	url := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s", subreddit, listing, limit, timeframe)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Add("User-Agent", "go:getter")

	fmt.Println("Requesting .json from reddit...")
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error, status code: %d\nHeaders: %+v\n", resp.StatusCode, resp.Header)
	}

	fmt.Println("Decoding json...")
	var result Top
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&result)
	if err != nil {
		log.Fatal(err)
	}

	type image struct {
		url    string
		name   string
		width  float64
		height float64
	}

	images := make([]image, 0)

	for _, v := range result.Data.Children {
		for _, v2 := range v.Data.Preview.Images {
			v2.Source.URL = strings.Replace(v2.Source.URL, "&amp;s", "&s", 1)
			images = append(images, image{
				url:    v2.Source.URL,
				name:   v.Data.Title,
				width:  v2.Source.Width,
				height: v2.Source.Width,
			})
		}
	}

	os.Mkdir(directory, os.ModePerm)
	os.Chdir(directory)

	fmt.Println("Downloading images.")
	total := len(images)
	for i, v := range images {
		fmt.Printf("\rCurrent: %d, Total: %d", i+1, total)
		resp, err := client.Get(v.url)
		if err != nil {
			log.Print(err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Print(errors.New("Status code is not 200"))
			continue
		}

		filename := v.name + ".png"

		filename, err = filenamify.Filenamify(filename, filenamify.Options{})
		if err != nil {
			filename = strconv.Itoa(i+1) + ".png"
		}

		file, err := os.Create(filename)
		if err != nil {
			log.Print(err)
			continue
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			log.Print(err)
			os.Remove(filename)
			continue
		}
	}
	fmt.Println("\nDone!")
}
