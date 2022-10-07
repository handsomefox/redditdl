package cmd

import (
	"github.com/handsomefox/redditdl/configuration"
	"github.com/handsomefox/redditdl/downloader"
	"github.com/handsomefox/redditdl/filter"
	"github.com/handsomefox/redditdl/logging"
	"github.com/spf13/cobra"
)

func SetCommonFlags(cmd *cobra.Command) {
	cmd.Flags().IntP("width", "x", 0, "Minimal content width")
	cmd.Flags().IntP("height", "y", 0, "Minimal content height")
	cmd.Flags().Int64P("count", "c", 1, "Amount of content to download")
	cmd.Flags().StringP("sub", "r", "wallpaper", "Name of the subreddit")
	cmd.Flags().StringP("sort", "s", "top", "Sort options(controversial, best, hot, new, random, rising, top)")
	cmd.Flags().StringP("timeframe", "t", "all", "Timeframe from which to get the posts (hour, day, week, month, year, all)")
	cmd.Flags().StringP("dir", "d", "media", "Download directory")
	cmd.Flags().StringP("orientation", "o", "", "Content orientation (\"l\" for landscape, \"p\" for portrait, other for any)")
}

func assert(err error) {
	if err != nil {
		panic(err)
	}
}

func GetSettings(cmd *cobra.Command) configuration.Data {
	flags := cmd.Flags()

	directory, err := flags.GetString("dir")
	assert(err)
	subreddit, err := flags.GetString("sub")
	assert(err)
	sorting, err := flags.GetString("sort")
	assert(err)
	timeframe, err := flags.GetString("timeframe")
	assert(err)
	orientation, err := flags.GetString("orientation")
	assert(err)
	count, err := flags.GetInt64("count")
	assert(err)
	width, err := flags.GetInt("width")
	assert(err)
	height, err := flags.GetInt("height")
	assert(err)
	verbose, err := flags.GetBool("verbose")
	assert(err)
	progress, err := flags.GetBool("progress")
	assert(err)

	return configuration.Data{
		Directory:    directory,
		Subreddit:    subreddit,
		Sorting:      sorting,
		Timeframe:    timeframe,
		Orientation:  orientation,
		Count:        count,
		MinWidth:     width,
		MinHeight:    height,
		WorkerCount:  configuration.DefaultWorkerCount,
		SleepTime:    configuration.DefaultSleepTime,
		Verbose:      verbose,
		ShowProgress: progress,
		ContentType:  configuration.MediaAny,
	}
}

func RunCommand(cfg *configuration.Data) {
	log := logging.Get(cfg.Verbose)

	// Print the configuration
	log.Debugf("Using parameters: %#v", cfg)

	// Download the media
	log.Info("Started downloading content")

	client := downloader.New(cfg, filter.Default()...)
	stats := client.Download()

	if stats.HasErrors() {
		log.Info("Encountered errors during download")
		for _, err := range stats.Errors {
			log.Errorf("%v", err)
		}
	}

	fStr := "Finished downloading %d "
	switch cfg.ContentType {
	case configuration.MediaAny:
		fStr += "image(s)/video(s)"
	case configuration.MediaImages:
		fStr += "image(s)"
	case configuration.MediaVideos:
		fStr += "video(s)"
	}

	log.Infof(fStr, stats.Finished.Load())
}
