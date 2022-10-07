// configuration is a package that has data structures
// that describe the settings that can be applied to a downloader.
package configuration

import "time"

// ContentType is the type of media which will be queued for download.
type ContentType uint8

const (
	_ ContentType = iota
	ContentImages
	ContentVideos
	ContentAny
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

	ContentType ContentType
}
