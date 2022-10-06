package cmd

import (
	"github.com/handsomefox/redditdl/downloader"
	"github.com/spf13/cobra"
)

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Command for downloading images",
	Long: `This command allows users to download video files from
different subreddits from reddit.com, applying different
filters to the content which will be downloaded.
`,
	Run: func(cmd *cobra.Command, args []string) {
		settings := GetSettings(cmd)
		settings.ContentType = downloader.MediaImages
		RunCommand(&settings)
	},
}

func init() {
	rootCmd.AddCommand(imageCmd)

	imageCmd.Flags().IntP("width", "x", 0, "Minimal image width")
	imageCmd.Flags().IntP("height", "y", 0, "Minimal image height")
	imageCmd.Flags().Int64P("count", "c", 1, "Amount of images to download")
	imageCmd.Flags().StringP("sub", "r", "wallpaper", "Name of the subreddit")
	imageCmd.Flags().StringP("sort", "s", "top", "Sort options(controversial, best, hot, new, random, rising, top)")
	imageCmd.Flags().StringP("timeframe", "t", "all", "Timeframe from which to get the posts (hour, day, week, month, year, all)")
	imageCmd.Flags().StringP("dir", "d", "media", "Download directory")
	imageCmd.Flags().StringP("orientation", "o", "", "Image orientation (\"l\" for landscape, \"p\" for portrait, other for any)")
}
