package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/handsomefox/redditdl/cmd/params"
	"github.com/handsomefox/redditdl/downloader"
	"github.com/handsomefox/redditdl/logging"
	"github.com/spf13/cobra"
)

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
	FilterNSFW      *bool
)

var rootCmd = &cobra.Command{
	Use:   "redditdl",
	Short: "A tool for downloading images/videos from reddit.com",
	Long: `redditdl is a CLI application written in Go that allows users
to very quickly download images and videos from different
subreddits of reddit.com, filtering them by multiple options.
`,
	Run: func(cmd *cobra.Command, args []string) {
		var orientationByte params.RequiredOrientation
		orientation := *OrientationFlag
		switch orientation {
		case "l":
			orientationByte = params.RequiredOrientationLandscape
		case "p":
			orientationByte = params.RequiredOrientationPortrait
		default:
			orientationByte = params.RequiredOrientationAny
		}
		var ct params.RequiredContentType
		switch *ContentTypeFlag {
		case "image":
			ct = params.RequiredContentTypeImages
		case "video":
			ct = params.RequiredContentTypeVideos
		case "any":
			ct = params.RequiredContentTypeAny
		default:
			ct = params.RequiredContentTypeAny
		}
		cliParameters := &params.CLIParameters{
			Sort:             *SortFlag,
			Timeframe:        *TimeframeFlag,
			Directory:        *DirectoryFlag,
			Subreddits:       *SubredditsFlag,
			MediaMinWidth:    *WidthFlag,
			MediaMinHeight:   *HeightFlag,
			MediaCount:       *CountFLag,
			MediaOrientation: orientationByte,
			ContentType:      ct,
			ShowProgress:     *ProgressFlag,
			VerboseLogging:   *VerboseFlag,
		}

		df := downloader.DefaultFilters()
		if *FilterNSFW {
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
	rootCmd.Flags().SortFlags = false
	ContentTypeFlag = rootCmd.Flags().String("ctype", "image", "Describes the type of content to download, possible values: image/video/any")
	WidthFlag = rootCmd.Flags().IntP("width", "x", 0, "Minimal content horizontal resolution")
	HeightFlag = rootCmd.Flags().IntP("height", "y", 0, "Minimal content vertical resolution")
	CountFLag = rootCmd.Flags().Int64P("count", "c", 1, "Amount of files to download (may download less than specified)")
	SortFlag = rootCmd.Flags().StringP("sort", "s", "top", "Possible values: controversial/best/hot/new/random/rising/top)")
	TimeframeFlag = rootCmd.Flags().StringP("timeframe", "t", "all", "Possible values: hour/day/week/month/year/all)")
	DirectoryFlag = rootCmd.Flags().StringP("dir", "d", "media", "Download path")
	OrientationFlag = rootCmd.Flags().StringP("orientation", "o", "", "Content orientation (\"l\"=landscape, \"p\"=portrait, other for any)")
	SubredditsFlag = rootCmd.Flags().StringSlice("subs", []string{}, "Comma-separated list of subreddits to fetch from")
	VerboseFlag = rootCmd.PersistentFlags().BoolP("verbose", "v", false, "If true, more logging is enabled")
	ProgressFlag = rootCmd.PersistentFlags().BoolP("progress", "p", false, "If true, displays the ongoing progress")
	FilterNSFW = rootCmd.Flags().Bool("nsfw", true, "If true, NSFW content will be ignored")
}

func SetGlobalLoggingLevel(verbose bool) {
	if verbose {
		if err := os.Setenv("ENVIRONMENT", "DEVELOPMENT"); err != nil {
			panic(err)
		}
	} else {
		if err := os.Setenv("ENVIRONMENT", "PRODUCTION"); err != nil {
			panic(err)
		}
	}
}

func MustRunCommand(ctx context.Context, p *params.CLIParameters, filters ...downloader.Filter) {
	if p == nil {
		panic("nil parameters provided")
	}

	SetGlobalLoggingLevel(p.VerboseLogging)

	log := logging.Get()

	// Print the configuration
	log.Debugf("Using parameters: %#v", p)

	// Download the media
	log.Info("Started downloading content")

	dl, err := downloader.New(p, log, filters...)
	if err != nil {
		panic(err) // no point in continuing
	}
	statusCh := dl.Download(ctx)

	finished := 0

	for message := range statusCh {
		status, err := message.Status, message.Error
		if err != nil {
			log.Error("error during download=", err.Error())
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

	log.Infof(fStr, finished)
}
