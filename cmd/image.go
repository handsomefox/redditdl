package cmd

import (
	"context"

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
		dcfg, ccfg := GetSettings(cmd)

		dcfg.ContentType = downloader.ContentImages
		RunCommand(context.Background(), dcfg, ccfg)
	},
}

func init() {
	rootCmd.AddCommand(imageCmd)
	SetCommonFlags(imageCmd)
}
