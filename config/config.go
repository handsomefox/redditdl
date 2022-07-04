package config

import "flag"

// Configuration
var (
	// Subreddit name
	Subreddit string
	// How to sort the subreddit
	Sorting string
	// Timeframe of the posts
	Timeframe string
	// Directory to download images to
	Directory string
	// Amount of images to download
	Count int
	// Minimal width of the images
	MinWidth int
	// Minimal height of the images
	MinHeight int
)

func init() {
	subreddit := flag.String("sub", "wallpaper", "Subreddit name")
	sorting := flag.String("sort", "top", "How to sort (controversial, best, hot, new, random, rising, top)")
	timeframe := flag.String("tf", "all", "Timeframe from which to get the posts (hour, day, week, month, year, all)")
	directory := flag.String("dir", "images", "Specifies the directory where to download the images")
	count := flag.Int("count", 1, "Amount of images to download")
	minWidth := flag.Int("width", 0, "minimal width of the image to download")
	minHeight := flag.Int("height", 0, "minimal height of the image to download")

	flag.Parse()

	Subreddit = *subreddit
	Sorting = *sorting
	Timeframe = *timeframe
	Directory = *directory
	Count = *count
	MinWidth = *minWidth
	MinHeight = *minHeight
}
