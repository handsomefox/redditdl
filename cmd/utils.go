package cmd

import (
	"context"
	"os"

	"github.com/handsomefox/redditdl/client"
	"github.com/handsomefox/redditdl/downloader"
	"github.com/handsomefox/redditdl/logging"

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

func MustNotError(err error) {
	if err != nil {
		panic(err)
	}
}

func GetSettings(cmd *cobra.Command) (*downloader.Config, *client.Config) {
	var (
		flags = cmd.Flags()
		err   error
		dcfg  downloader.Config
		ccfg  client.Config
	)

	dcfg.Directory, err = flags.GetString("dir")
	MustNotError(err)
	dcfg.WorkerCount = downloader.DefaultWorkerCount
	MustNotError(err)
	dcfg.ShowProgress, err = flags.GetBool("progress")
	MustNotError(err)
	dcfg.ContentType = downloader.ContentAny
	MustNotError(err)
	ccfg.Subreddit, err = flags.GetString("sub")
	MustNotError(err)
	ccfg.Sorting, err = flags.GetString("sort")
	MustNotError(err)
	ccfg.Timeframe, err = flags.GetString("timeframe")
	MustNotError(err)
	ccfg.Orientation, err = flags.GetString("orientation")
	MustNotError(err)
	ccfg.Count, err = flags.GetInt64("count")
	MustNotError(err)
	ccfg.MinWidth, err = flags.GetInt("width")
	MustNotError(err)
	ccfg.MinHeight, err = flags.GetInt("height")
	MustNotError(err)
	verbose, err := flags.GetBool("verbose")
	MustNotError(err)

	if verbose {
		if err := os.Setenv("ENVIRONMENT", "DEVELOPMENT"); err != nil {
			panic(err)
		}
	} else {
		if err := os.Setenv("ENVIRONMENT", "PRODUCTION"); err != nil {
			panic(err)
		}
	}

	return &dcfg, &ccfg
}

func RunCommand(ctx context.Context, dcfg *downloader.Config, ccfg *client.Config) {
	log := logging.Get()

	// Print the configuration
	log.Debugf("Using parameters: %#v", dcfg)
	log.Debugf("Using parameters: %#v", ccfg)

	// Download the media
	log.Info("Started downloading content")

	dl := downloader.New(dcfg, ccfg, downloader.DefaultFilters()...)
	statusCh := dl.Download(ctx)

	finished := 0

	for message := range statusCh {
		status, err := message.Status, message.Error
		if err != nil {
			log.Error("error during download=", err.Error())
		}

		if status == downloader.StatusFinished {
			finished++
		}
	}

	fStr := "Finished downloading %d "
	switch dcfg.ContentType {
	case downloader.ContentAny:
		fStr += "image(s)/video(s)"
	case downloader.ContentImages:
		fStr += "image(s)"
	case downloader.ContentVideos:
		fStr += "video(s)"
	}

	log.Infof(fStr, finished)
}
