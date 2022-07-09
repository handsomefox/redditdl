package downloader

import (
	"fmt"
	"redditdl/pkg/config"
	"redditdl/pkg/logging"
	"redditdl/utils"

	"go.uber.org/zap"
)

// downloadable contains information which is required to filter by resolution,
// download and store a video or an image.
type downloadable struct {
	Name      string
	IsVideo   bool
	ImageData *imageData
	VideoData *RedditVideo
}

var client = utils.CreateClient()
var logger *zap.SugaredLogger

// Download downloads the images according to the given configuration
func Download(c config.Configuration) (int, error) {
	logger = logging.GetLogger(c.Verbose)

	media, err := getFilteredMedia(c)
	if err != nil {
		return 0, fmt.Errorf("error getting media from reddit: %v", err)
	}

	count, err := downloadMedia(media, c)
	if err != nil {
		return 0, fmt.Errorf("error downloading the media from reddit: %v", err)
	}

	return count, nil
}
