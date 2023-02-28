package cmd

import (
	"context"
	"flag"
	"os"

	"github.com/handsomefox/redditdl/cmd/params"
	"github.com/handsomefox/redditdl/downloader"
	"github.com/handsomefox/redditdl/logging"
	"github.com/spf13/cobra"
)

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

// GetCLIParameters calls flags.Parse(), add additional flags before the call to this function.
func GetCLIParameters(cmd *cobra.Command, ct params.RequiredContentType) *params.CLIParameters {
	wf := cmd.Flags().IntP("width", "x", 0, "Minimal content width")
	hf := cmd.Flags().IntP("height", "y", 0, "Minimal content height")
	cf := cmd.Flags().Int64P("count", "c", 1, "Amount of content to download")
	sf := cmd.Flags().StringP("sort", "s", "top", "Sort options(controversial, best, hot, new, random, rising, top)")
	tf := cmd.Flags().StringP("timeframe", "t", "all", "Timeframe for the posts (hour, day, week, month, year, all)")
	df := cmd.Flags().StringP("dir", "d", "media", "Download directory")
	of := cmd.Flags().StringP("orientation", "o", "", "Content orientation (\"l\"=landscape, \"p\"=portrait, other for any)")
	sbf := cmd.Flags().StringSlice("subs", []string{}, "Comma-separated list of subreddits to fetch from. The specified amount of images will be fetched from each subreddit (If you specify 2 subreddits and set count to 10, 20 total files will be downloaded.)")
	pf := cmd.Flags().BoolP("progress", "p", true, "Specifies whether to show progress during download")
	vf := cmd.Flags().BoolP("verbose", "v", false, "Specifies whether to enable verbose logging")

	flag.Parse()

	var orientationByte params.RequiredOrientation
	orientation := *of

	switch orientation {
	case "l":
		orientationByte = params.RequiredOrientationLandscape
	case "p":
		orientationByte = params.RequiredOrientationPortrait
	default:
		orientationByte = params.RequiredOrientationAny
	}

	cs := &params.CLIParameters{
		Sort:             *sf,
		Timeframe:        *tf,
		Directory:        *df,
		Subreddits:       *sbf,
		MediaMinWidth:    *wf,
		MediaMinHeight:   *hf,
		MediaCount:       *cf,
		MediaOrientation: orientationByte,
		ContentType:      ct,
		ShowProgress:     *pf,
		VerboseLogging:   *vf,
	}

	return cs
}

func MustRunCommand(ctx context.Context, p *params.CLIParameters) {
	if p == nil {
		panic("nil parameters provided")
	}

	SetGlobalLoggingLevel(p.VerboseLogging)

	log := logging.Get()

	// Print the configuration
	log.Debugf("Using parameters: %#v", p)

	// Download the media
	log.Info("Started downloading content")

	dl, err := downloader.New(p, downloader.DefaultFilters()...)
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
