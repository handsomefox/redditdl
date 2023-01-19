package cmd

import (
	"context"

	"github.com/handsomefox/redditdl/cmd/common"
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
	Run: func(cmd *cobra.Command, _ []string) {
		dcfg, ccfg := common.GetSettings(cmd)

		dcfg.ContentType = downloader.ContentImages
		common.RunCommand(context.Background(), dcfg, ccfg)
	},
}

func init() {
	rootCmd.AddCommand(imageCmd)
	common.SetCommonFlags(imageCmd)
}
