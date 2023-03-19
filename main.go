package main

import (
	"context"
	"os"
	"runtime"

	"github.com/alexflint/go-arg"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type AppArguments struct {
	SubredditContentType string `arg:"-t,--type" help:"values: image,video,both" default:"image"`
	SubredditSort        string `arg:"-s,--sort" help:"values: controversial/best/hot/new/random/rising/top" default:"top"`
	SubredditTimeframe   string `arg:"-f,--timeframe" help:"values: hour/day/week/month/year/all" default:"all"`
	SubredditList        string `arg:"-r,--subreddits" help:"a comma-separated list of subreddits to download from"`
	SaveDirectory        string `arg:"-d,--dir" help:"output path"`

	MediaOrientation   string `arg:"-o, --orientation" help:"values: landspace/portrait/rect/all" default:"all"`
	MediaCount         int64  `arg:"-c, --count" help:"amount of media to download"`
	MediaMinimalWidth  int    `arg:"-x, --width" help:"minimal content width"`
	MediaMinimalHeight int    `arg:"-y, --height" help:"minimal content height"`

	ShowNSFW        bool `arg:"-n, --nsfw" help:"enable if you want to show NSFW content"`
	VerboseLogging  bool `arg:"-v, --verbose" help:"enable debug logging"`
	ProgressLogging bool `arg:"-p, --progress" help:"enable current progress logging"`
}

func main() {
	var args AppArguments
	parser := arg.MustParse(&args)

	if args.SaveDirectory == "" {
		parser.Fail("you must provide a valid output path using -d or --dir")
	}

	if args.SubredditList == "" {
		parser.Fail("you must provide a list of comma-separated subreddits using -r or --subreddits")
	}

	if args.MediaCount == 0 {
		log.Info().Msg("no media requested to download, ending")
		os.Exit(0)
	}

	if args.VerboseLogging {
		log.Logger = log.Level(zerolog.DebugLevel)
	} else {
		log.Logger = log.Level(zerolog.InfoLevel)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Debug().Any("app_arguments", args).Send()

	ctx := context.Background()
	if err := run(ctx, &args); err != nil {
		log.Fatal().Err(err).Msg("error running the app")
	}
}

func run(ctx context.Context, args *AppArguments) error {
	return NewSaver(args, runtime.NumCPU(), runtime.NumCPU()*2).Run(ctx)
}
