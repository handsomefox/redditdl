package cmd

import (
	"github.com/handsomefox/redditdl/pkg/downloader/config"
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
		settings.ContentType = config.ContentImages
		RunCommand(&settings)
	},
}

func init() {
	rootCmd.AddCommand(imageCmd)
	SetCommonFlags(imageCmd)
}
