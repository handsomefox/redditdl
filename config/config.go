package config

import "flag"

// Configuration
var (
	Subreddit string
	Sorting   string
	Timeframe string
	Directory string
	Count     int
	MinWidth  int
	MinHeight int
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

	Subreddit = *subFlag
	Sorting = *sortFlag
	Timeframe = *timeframeFlag
	Directory = *directoryFlag
	Count = *countFlag
	MinWidth = *minWidthFlag
	MinHeight = *minHeightFlag
}
