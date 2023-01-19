package cmd

import (
	"context"

	"github.com/handsomefox/redditdl/cmd/common"
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
	Run: func(cmd *cobra.Command, _ []string) {
		dcfg, ccfg := common.GetSettings(cmd)
		dcfg.ContentType = downloader.ContentVideos
		common.RunCommand(context.Background(), dcfg, ccfg)
	},
}

func init() {
	rootCmd.AddCommand(videoCmd)
	common.SetCommonFlags(videoCmd)
}
