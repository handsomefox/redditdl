package downloader

import (
	"fmt"
	"redditdl/config"
	"redditdl/pkg/logging"
	"redditdl/utils"

	"go.uber.org/zap"
)

// FinalImage represents the image information which is required to filter by resolution, download and store it.
type FinalImage struct {
	Name string
	Data ImageData
}

var client = utils.CreateClient()
var logger *zap.SugaredLogger

// Download downloads the images according to the given configuration
func Download(c config.Configuration) (int, error) {
	logger = logging.GetLogger(c.Verbose)

	// Get the images
	images, err := getFilteredImages(c)
	if err != nil {
		return 0, fmt.Errorf("error getting images from reddit: %v", err)
	}

	// Download the filtered images
	count, err := downloadImages(images, c)
	if err != nil {
		return 0, fmt.Errorf("error downloading the images from reddit: %v", err)
	}

	return count, nil
}
