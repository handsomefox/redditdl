package cmd

import (
	"context"

	"github.com/handsomefox/redditdl/cmd/params"
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
	Run: func(cmd *cobra.Command, _ []string) {
		params := GetCLIParameters(cmd, params.RequiredContentTypeAny)
		MustRunCommand(context.Background(), params)
	},
}

func init() {
	rootCmd.AddCommand(anyCmd)
}
