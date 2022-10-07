// configuration is a package that has data structures
// that describe the settings that can be applied to a downloader.
package configuration

import "time"

// MediaType is the type of media which will be queued for download.
type MediaType uint8

const (
	_ MediaType = iota
	MediaImages
	MediaVideos
	MediaAny
)

const (
	DefaultWorkerCount = 16
	DefaultSleepTime   = 200 * time.Millisecond
)

// Config is the configuration data for the Downloader.
type Config struct {
	Directory   string
	Subreddit   string
	Sorting     string
	Timeframe   string
	Orientation string

	Count     int64
	MinWidth  int
	MinHeight int

	WorkerCount int
	SleepTime   time.Duration

	Verbose      bool
	ShowProgress bool

	ContentType MediaType
}
