package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "redditdl",
	Short: "A tool for downloading images/videos from reddit.com",
	Long: `redditdl is a CLI application written in Go that allows users
to very quickly download images and videos from different
subreddits of reddit.com, filtering them by multiple options.
`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Turns the logging on or off")
	rootCmd.PersistentFlags().BoolP("progress", "p", false, "Indicates whether the application will show the download progress")
}
