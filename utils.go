package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flytam/filenamify"
)

// createClient returns a pointer to http.Client configured to work with reddit.
func createClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
		},
		Timeout: 60 * time.Second,
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
	counter := 0
	for fileExists(str) {
		newName := name + "(" + strconv.Itoa(counter) + ")"
		str, err = filenamify.Filenamify(newName, filenamify.Options{
			MaxLength: 240,
		})
		str += ".png"

		if err != nil {
			str = "(" + strconv.Itoa(counter+idx+1) + ")" + ".png"
		}
		counter++
	}

	return str
}

// returns whether the file exists
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// postsToImages converts an array of children that contain images to an array of filteredImage(s)
// it also filters by specified resolution, everything smaller than the specified resolution is not downloaded
func postsToImages(posts *Posts) []filteredImage {
	images := make([]filteredImage, 0)

	for _, v := range posts.Data.Children {
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

// printDownloadStatus is a background printDownloadStatus waiting for all the images to finish loading
func printDownloadStatus(total *uint32, finished *uint32, failed *uint32) {
	for {
		fmt.Printf("\rTotal images: %d, Finished: %d, Failed: %d", *total, *finished, *failed)
		if *total == (*finished + *failed) {
			fmt.Println("\nFinished downloading all the images!")
			return
		}
	}
}
