package downloader

import (
	"time"

	"redditdl/pkg/logging"
	"redditdl/pkg/utils"
)

// Settings is the configuration for the Downloader.
type Settings struct {
	Directory    string
	Subreddit    string
	Sorting      string
	Timeframe    string
	Orientation  string
	Count        int
	MinWidth     int
	MinHeight    int
	Verbose      bool
	ShowProgress bool
	IncludeVideo bool
}

const (
	SleepTime   = 5 * time.Second // sleep period between post fetches
	workerCount = 16              // amount of goroutines for downloading files.
)

// Download downloads the images according to the given configuration.
func Download(settings Settings, filters []Filter) (int64, error) {
	dl := downloader{
		client:  utils.CreateClient(),
		log:     logging.GetLogger(settings.Verbose),
		after:   "",
		counter: counter{queued: 0, finished: 0, failed: 0},
	}

	return dl.download(&settings, filters)
}
