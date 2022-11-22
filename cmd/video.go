package cmd

import (
	"github.com/handsomefox/redditdl/downloader/configuration"
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
		settings.ContentType = configuration.ContentVideos
		RunCommand(&settings)
	},
}

func init() {
	rootCmd.AddCommand(videoCmd)
	SetCommonFlags(videoCmd)
}
