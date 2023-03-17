package main

import (
	"os"

	"github.com/alexflint/go-arg"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	var args AppArguments
	p := arg.MustParse(&args)

	if args.SaveDirectory == "" {
		p.Fail("you must provide a valid output path using -d or --dir")
	}
	if len(args.SubredditList) == 0 {
		p.Fail("you must provide a list of comma-separated subreddits using -r or --subreddits")
	}
	if args.MediaCount == 0 {
		log.Info().Msg("no media requested to download, ending")
		os.Exit(0)
	}
	if args.VerboseLogging {
		log.Logger = log.Level(zerolog.InfoLevel)
	} else {
		log.Logger = log.Level(zerolog.DebugLevel)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Debug().Any("app_arguments", args).Send()

	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("error running the app")
	}
}

func run() error {
	return nil
}
