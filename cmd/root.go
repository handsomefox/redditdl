package cmd

import (
	"fmt"
	"os"

	"github.com/handsomefox/redditdl/downloader"
	"github.com/handsomefox/redditdl/internal/logging"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "redditdl",
	Short: "A tool for downloading images/videos from reddit.com",
	Long: `redditdl is a CLI application written in Go that allows users
to very quickly download images and videos from different
subreddits of reddit.com, filtering them by multiple options.
`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Turns the logging on or off")
	rootCmd.PersistentFlags().BoolP("progress", "p", false, "Indicates whether the application will show the download progress")
}

func GetSettings(cmd *cobra.Command) downloader.Settings {
	flags := cmd.Flags()

	directory, err := flags.GetString("dir")
	if err != nil {
		panic(err)
	}
	subreddit, err := flags.GetString("sub")
	if err != nil {
		panic(err)
	}
	sorting, err := flags.GetString("sort")
	if err != nil {
		panic(err)
	}
	timeframe, err := flags.GetString("timeframe")
	if err != nil {
		panic(err)
	}
	orientation, err := flags.GetString("orientation")
	if err != nil {
		panic(err)
	}
	count, err := flags.GetInt64("count")
	if err != nil {
		panic(err)
	}
	width, err := flags.GetInt("width")
	if err != nil {
		panic(err)
	}
	height, err := flags.GetInt("height")
	if err != nil {
		panic(err)
	}
	verbose, err := flags.GetBool("verbose")
	if err != nil {
		panic(err)
	}
	progress, err := flags.GetBool("progress")
	if err != nil {
		panic(err)
	}

	return downloader.Settings{
		Directory:    directory,
		Subreddit:    subreddit,
		Sorting:      sorting,
		Timeframe:    timeframe,
		Orientation:  orientation,
		Count:        count,
		MinWidth:     width,
		MinHeight:    height,
		Verbose:      verbose,
		ShowProgress: progress,
	}
}

func RunCommand(settings *downloader.Settings) {
	log := logging.GetLogger(settings.Verbose)

	// Print the configuration
	log.Debugf("Using parameters: %#v", settings)

	// Download the media
	log.Info("Started downloading content")

	count, err := downloader.Download(settings, downloader.DefaultFilters())
	if err != nil {
		log.Fatal("error downloading content", err)
	}

	finishedStr := "Finished downloading %d "

	switch settings.ContentType {
	case downloader.MediaAny:
		finishedStr += "image(s)/video(s)"
	case downloader.MediaImages:
		finishedStr += "image(s)"
	case downloader.MediaVideos:
		finishedStr += "video(s)"
	}

	log.Infof(finishedStr, count)
}
