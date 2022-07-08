package main

import (
	"go.uber.org/zap"
	"redditdl/config"
	"redditdl/pkg/downloader"
	"redditdl/pkg/logging"
)

var logger *zap.SugaredLogger

func main() {
	// Get the configuration from command line
	c := config.GetConfiguration()

	logger = logging.GetLogger(c.Verbose)

	// Print the configuration
	logger.Debugf("Using parameters: %#v", c)

	// Download the images
	logger.Info("Started downloading images")

	count, err := downloader.Download(c)
	if err != nil {
		logger.Fatal("error downloading images", zap.Error(err))
	}
	logger.Infof("Finished downloading %d image(s)", count)
}
