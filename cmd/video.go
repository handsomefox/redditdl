package cmd

import (
	"context"

	"github.com/handsomefox/redditdl/cmd/params"
	"github.com/spf13/cobra"
)

var videoCmd = &cobra.Command{
	Use:   "video",
	Short: "Command for downloading videos",
	Long: `This command allows users to download video files from
different subreddits from reddit.com, applying different
filters to the content which will be downloaded.
`,
	Run: func(cmd *cobra.Command, _ []string) {
		params := GetCLIParameters(cmd, params.RequiredContentTypeVideos)
		MustRunCommand(context.Background(), params)
	},
}

func init() {
	rootCmd.AddCommand(videoCmd)
}
