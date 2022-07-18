package main

import (
	"flag"
	"redditdl/pkg/downloader"
	"redditdl/pkg/logging"
)

// Command-line arguments
var (
	v         = flag.Bool("v", false, "Turns the logging on or off")
	p         = flag.Bool("p", false, "Indicates whether the application will show the download progress")
	inclVideo = flag.Bool("video", false, "Indicates whether the application should download videos as well")
	sub       = flag.String("sub", "wallpaper", "Subreddit name")
	sorting   = flag.String("sort", "top", "How to sort (controversial, best, hot, new, random, rising, top)")
	tf        = flag.String("tf", "all", "Timeframe from which to get the posts (hour, day, week, month, year, all)")
	dir       = flag.String("dir", "media", "Specifies the directory where to download the media")
	cnt       = flag.Int("cnt", 1, "Amount of media to download")
	w         = flag.Int("w", 0, "minimal width of the media to download")
	h         = flag.Int("h", 0, "minimal height of the media to download")
)

func main() {
	flag.Parse()

	c := downloader.DownloaderSettings{
		Verbose:      *v,
		ShowProgress: *p,
		IncludeVideo: *inclVideo,
		Subreddit:    *sub,
		Sorting:      *sorting,
		Timeframe:    *tf,
		Directory:    *dir,
		Count:        *cnt,
		MinWidth:     *w,
		MinHeight:    *h,
	}

	log := logging.GetLogger(c.Verbose)

	// Print the configuration
	log.Debugf("Using parameters: %#v", c)

	// Download the media
	log.Info("Started downloading media")

	dl := downloader.New(c, downloader.Filters)
	count, err := dl.Download()
	if err != nil {
		log.Fatal("error downloading media", err)
	}
	log.Infof("Finished downloading %d image(s)/video(s)", count)
}
