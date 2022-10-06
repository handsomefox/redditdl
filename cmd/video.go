package cmd

import (
	"github.com/handsomefox/redditdl/downloader"
	"github.com/spf13/cobra"
)

var videoCmd = &cobra.Command{
	Use:   "video",
	Short: "Command for downloading videos",
	Long: `This command allows users to download video files from
different subreddits from reddit.com, applying different
filters to the content which will be downloaded.
`,
	Run: func(cmd *cobra.Command, args []string) {
		settings := GetSettings(cmd)
		settings.ContentType = downloader.MediaVideos
		RunCommand(&settings)
	},
}

func init() {
	rootCmd.AddCommand(videoCmd)

	videoCmd.Flags().IntP("width", "x", 0, "Minimal video width")
	videoCmd.Flags().IntP("height", "y", 0, "Minimal video height")
	videoCmd.Flags().Int64P("count", "c", 1, "Amount of videos to download")
	videoCmd.Flags().StringP("sub", "r", "wallpaper", "Name of the subreddit")
	videoCmd.Flags().StringP("sort", "s", "top", "Sort options(controversial, best, hot, new, random, rising, top)")
	videoCmd.Flags().StringP("timeframe", "t", "all", "Timeframe from which to get the posts (hour, day, week, month, year, all)")
	videoCmd.Flags().StringP("dir", "d", "media", "Download directory")
	videoCmd.Flags().StringP("orientation", "o", "", "Video orientation (\"l\" for landscape, \"p\" for portrait, other for any)")
}
