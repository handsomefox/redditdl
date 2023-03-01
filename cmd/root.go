package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/handsomefox/redditdl/cmd/params"
	"github.com/handsomefox/redditdl/downloader"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// Flags for the command-line.
var (
	ContentTypeFlag *string
	WidthFlag       *int
	HeightFlag      *int
	CountFLag       *int64
	SortFlag        *string
	TimeframeFlag   *string
	DirectoryFlag   *string
	OrientationFlag *string
	SubredditsFlag  *[]string
	VerboseFlag     *bool
	ProgressFlag    *bool
	FilterNSFWFlag  *bool
)

// The only command we need.
var rootCmd = &cobra.Command{
	Use:   "redditdl",
	Short: "A tool for downloading images/videos from reddit.com",
	Run: func(_ *cobra.Command, _ []string) {
		cliParameters := &params.CLIParameters{
			Sort:             *SortFlag,
			Timeframe:        *TimeframeFlag,
			Directory:        *DirectoryFlag,
			Subreddits:       *SubredditsFlag,
			MediaMinWidth:    *WidthFlag,
			MediaMinHeight:   *HeightFlag,
			MediaCount:       *CountFLag,
			MediaOrientation: params.OrientationFromString(*OrientationFlag),
			ContentType:      params.RequiredContentTypeFromString(*ContentTypeFlag),
			ShowProgress:     *ProgressFlag,
			VerboseLogging:   *VerboseFlag,
		}

		df := downloader.DefaultFilters()
		if *FilterNSFWFlag {
			df = append(df, downloader.FilterNSFW())
		}

		MustRunCommand(context.Background(), cliParameters, df...)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Don't sort, as it is confusing to see width/height be in different parts of help message and etc.

	rootCmd.Flags().SortFlags = false

	// Set the flags
	ContentTypeFlag = rootCmd.Flags().String("ctype",
		"image", "Describes the type of content to download, possible values: image/video/any")
	WidthFlag = rootCmd.Flags().IntP("width", "x", 0,
		"Minimal content horizontal resolution")
	HeightFlag = rootCmd.Flags().IntP("height", "y", 0,
		"Minimal content vertical resolution")
	CountFLag = rootCmd.Flags().Int64P("count", "c", 1,
		"Amount of files to download (may download less than specified)")
	SortFlag = rootCmd.Flags().StringP("sort", "s", "top",
		"Possible values: controversial/best/hot/new/random/rising/top)")
	TimeframeFlag = rootCmd.Flags().StringP("timeframe", "t", "all",
		"Possible values: hour/day/week/month/year/all)")
	DirectoryFlag = rootCmd.Flags().StringP("dir", "d", "media",
		"Download path")
	OrientationFlag = rootCmd.Flags().StringP("orientation", "o", "",
		"Content orientation (\"l\"=landscape, \"p\"=portrait, other for any)")
	SubredditsFlag = rootCmd.Flags().StringSlice("subs", []string{},
		"Comma-separated list of subreddits to fetch from")
	VerboseFlag = rootCmd.PersistentFlags().BoolP("verbose", "v", false,
		"If true, more logging is enabled")
	ProgressFlag = rootCmd.PersistentFlags().BoolP("progress", "p", false,
		"If true, displays the ongoing progress")
	FilterNSFWFlag = rootCmd.
		Flags().Bool("nsfw", true, "If true, NSFW content will be ignored")
}

// SetGlobalLoggingLevel changes the zerolog.Log level.
func SetGlobalLoggingLevel(verbose bool) {
	if verbose {
		log.Logger = log.Logger.Level(zerolog.DebugLevel)
	} else {
		log.Logger = log.Logger.Level(zerolog.InfoLevel)
	}
}

func MustRunCommand(ctx context.Context, p *params.CLIParameters, filters ...downloader.Filter) {
	if p == nil {
		panic("nil parameters provided")
	}

	SetGlobalLoggingLevel(p.VerboseLogging)

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Print the configuration
	log.Debug().Any("configuration", *p).Send()
	// Download the media
	log.Info().Msg("started downloading content")

	dl, err := downloader.New(p, filters...)
	if err != nil {
		panic(err) // no point in continuing
	}

	var (
		statusCh = dl.Download(ctx)
		finished = 0
	)

	for message := range statusCh {
		status, err := message.Status, message.Error
		if err != nil {
			log.Err(err).Msg("error during download")
		}
		if status == downloader.StatusFinished {
			finished++
		}
	}

	fStr := "Finished downloading %d "
	switch p.ContentType {
	case params.RequiredContentTypeAny:
		fStr += "image(s)/video(s)"
	case params.RequiredContentTypeImages:
		fStr += "image(s)"
	case params.RequiredContentTypeVideos:
		fStr += "video(s)"
	}

	log.Info().Msg(fmt.Sprintf(fStr, finished))
}
