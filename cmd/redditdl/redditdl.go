package main

import (
	"flag"
	"os"

	"redditdl/pkg/downloader"
	"redditdl/pkg/logging"
)

func main() {
	// Command-line arguments.
	var (
		verbose     = flag.Bool("verbose", false, "Turns the logging on or off")
		progress    = flag.Bool("progress", false, "Indicates whether the application will show the download progress")
		inclVideo   = flag.Bool("video", false, "Indicates whether the application should download videos as well")
		subreddit   = flag.String("sub", "wallpaper", "Subreddit name")
		sorting     = flag.String("sort", "top", "How to sort (controversial, best, hot, new, random, rising, top)")
		timeframe   = flag.String("timeframe", "all", "Timeframe from which to get the posts (hour, day, week, month, year, all)")
		directory   = flag.String("dir", "media", "Specifies the directory where to download the media")
		amount      = flag.Int("count", 1, "Amount of media to download")
		width       = flag.Int("width", 0, "minimal width of the media to download")
		height      = flag.Int("height", 0, "minimal height of the media to download")
		orientation = flag.String("orientation", "", "image orientation (\"l\" for landscape, \"p\" for portrait, other for any)")
	)

	flag.Parse()

	settings := downloader.Settings{
		Verbose:      *verbose,
		ShowProgress: *progress,
		IncludeVideo: *inclVideo,
		Subreddit:    *subreddit,
		Sorting:      *sorting,
		Timeframe:    *timeframe,
		Directory:    *directory,
		Count:        *amount,
		MinWidth:     *width,
		MinHeight:    *height,
		Orientation:  *orientation,
	}

	log := logging.GetLogger(settings.Verbose)

	if len(os.Args) <= 1 {
		log.Info("No options specified")
		flag.PrintDefaults()
		return
	}

	// Print the configuration
	log.Debugf("Using parameters: %#v", settings)

	// Download the media
	log.Info("Started downloading media")

	count, err := downloader.Download(&settings, downloader.DefaultFilters())
	if err != nil {
		log.Fatal("error downloading media", err)
	}

	log.Infof("Finished downloading %d image(s)/video(s)", count)
}
