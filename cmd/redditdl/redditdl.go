package main

import (
	"redditdl/pkg/config"
	"redditdl/pkg/downloader"
	"redditdl/pkg/logging"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func main() {
	// Get the configuration from command line
	c := config.GetConfiguration()

	logger = logging.GetLogger(c.Verbose)

	// Print the configuration
	logger.Debugf("Using parameters: %#v", c)

	// Download the media
	logger.Info("Started downloading media")

	count, err := downloader.Download(c)
	if err != nil {
		logger.Fatal("error downloading media", zap.Error(err))
	}
	logger.Infof("Finished downloading %d image(s)/video(s)", count)
}
