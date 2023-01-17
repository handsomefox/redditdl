package cmd

import (
	"os"

	"github.com/handsomefox/redditdl/pkg/downloader"
	"github.com/handsomefox/redditdl/pkg/downloader/config"
	"github.com/handsomefox/redditdl/pkg/downloader/filters"
	"github.com/handsomefox/redditdl/pkg/logging"
	"github.com/spf13/cobra"
)

func SetCommonFlags(cmd *cobra.Command) {
	cmd.Flags().IntP("width", "x", 0, "Minimal content width")
	cmd.Flags().IntP("height", "y", 0, "Minimal content height")
	cmd.Flags().Int64P("count", "c", 1, "Amount of content to download")
	cmd.Flags().StringP("sub", "r", "wallpaper", "Name of the subreddit")
	cmd.Flags().StringP("sort", "s", "top", "Sort options(controversial, best, hot, new, random, rising, top)")
	cmd.Flags().StringP("timeframe", "t", "all", "Timeframe for the posts (hour, day, week, month, year, all)")
	cmd.Flags().StringP("dir", "d", "media", "Download directory")
	cmd.Flags().StringP("orientation", "o", "", "Content orientation (\"l\"=landscape, \"p\"=portrait, other for any)")
}

func GetSettings(cmd *cobra.Command) config.Config {
	var (
		flags = cmd.Flags()
		err   error
		cfg   config.Config
	)
	cfg.Directory, err = flags.GetString("dir")
	cfg.Subreddit, err = flags.GetString("sub")
	cfg.Sorting, err = flags.GetString("sort")
	cfg.Timeframe, err = flags.GetString("timeframe")
	cfg.Orientation, err = flags.GetString("orientation")
	cfg.Count, err = flags.GetInt64("count")
	cfg.Width, err = flags.GetInt("width")
	cfg.Height, err = flags.GetInt("height")
	cfg.Progress, err = flags.GetBool("progress")

	cfg.ContentType = config.ContentAny
	cfg.WorkerCount = config.DefaultWorkerCount
	cfg.SleepTime = config.DefaultSleepTime

	verbose, err := flags.GetBool("verbose")
	if err != nil {
		panic(err)
	}
	if verbose {
		os.Setenv("ENVIRONMENT", "DEVELOPMENT")
	} else {
		os.Setenv("ENVIRONMENT", "PRODUCTION")
	}

	return cfg
}

func RunCommand(cfg *config.Config) {
	log := logging.Get(cfg.Verbose)

	// Print the configuration
	log.Debugf("Using parameters: %#v", cfg)

	// Download the media
	log.Info("Started downloading content")

	client := downloader.New(cfg, filters.Default()...)
	stats := client.Download()

	if stats.HasErrors() {
		log.Info("Encountered errors during download")
		for _, err := range stats.Errors() {
			log.Errorf("%s", err)
		}
	}

	fStr := "Finished downloading %d "
	switch cfg.ContentType {
	case config.ContentAny:
		fStr += "image(s)/video(s)"
	case config.ContentImages:
		fStr += "image(s)"
	case config.ContentVideos:
		fStr += "video(s)"
	}

	log.Infof(fStr, stats.Finished())
}
