package config

import (
	"flag"
)

// Configuration is the parameters for the application
type Configuration struct {
	// Verbose turns the logging on or off
	Verbose bool
	// ShowProgress indicates whether the application will show the download progress
	ShowProgress bool
	// Subreddit name
	Subreddit string
	// Sorting How to sort the subreddit
	Sorting string
	// Timeframe of the posts
	Timeframe string
	// Directory to download images to
	Directory string
	// Count Amount of images to download
	Count int
	// MinWidth Minimal width of the images
	MinWidth int
	// MinHeight Minimal height of the images
	MinHeight int
	// After is a post ID, which is used to fetch posts after that ID
	After string
}

var spec Configuration

func init() {
	verbose := flag.Bool("verbose", false, "Turns the logging on or off")
	showProgress := flag.Bool("progress", false, "Indicates whether the application will show the download progress")
	subreddit := flag.String("sub", "wallpaper", "Subreddit name")
	sorting := flag.String("sort", "top", "How to sort (controversial, best, hot, new, random, rising, top)")
	timeframe := flag.String("tf", "all", "Timeframe from which to get the posts (hour, day, week, month, year, all)")
	directory := flag.String("dir", "images", "Specifies the directory where to download the images")
	count := flag.Int("count", 1, "Amount of images to download")
	minWidth := flag.Int("width", 0, "minimal width of the image to download")
	minHeight := flag.Int("height", 0, "minimal height of the image to download")

	flag.Parse()

	spec.Verbose = *verbose
	spec.ShowProgress = *showProgress
	spec.Subreddit = *subreddit
	spec.Sorting = *sorting
	spec.Timeframe = *timeframe
	spec.Directory = *directory
	spec.Count = *count
	spec.MinWidth = *minWidth
	spec.MinHeight = *minHeight
}

func GetConfiguration() Configuration {
	return spec
}
