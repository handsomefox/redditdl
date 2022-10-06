package cmd

import (
	"github.com/handsomefox/redditdl/downloader"
	"github.com/spf13/cobra"
)

// anyCmd represents the any command.
var anyCmd = &cobra.Command{
	Use:   "any",
	Short: "Command for downloading any media type",
	Long: `This command allows users to download any media files from
different subreddits from reddit.com, applying different
filters to the content which will be downloaded.
`,
	Run: func(cmd *cobra.Command, args []string) {
		settings := GetSettings(cmd)
		settings.ContentType = downloader.MediaAny
		RunCommand(&settings)
	},
}

func init() {
	rootCmd.AddCommand(anyCmd)

	anyCmd.Flags().IntP("width", "x", 0, "Minimal content width")
	anyCmd.Flags().IntP("height", "y", 0, "Minimal content height")
	anyCmd.Flags().Int64P("count", "c", 1, "Amount of content to download")
	anyCmd.Flags().StringP("sub", "r", "wallpaper", "Name of the subreddit")
	anyCmd.Flags().StringP("sort", "s", "top", "Sort options(controversial, best, hot, new, random, rising, top)")
	anyCmd.Flags().StringP("timeframe", "t", "all", "Timeframe from which to get the posts (hour, day, week, month, year, all)")
	anyCmd.Flags().StringP("dir", "d", "media", "Download directory")
	anyCmd.Flags().StringP("orientation", "o", "", "Content orientation (\"l\" for landscape, \"p\" for portrait, other for any)")
}
