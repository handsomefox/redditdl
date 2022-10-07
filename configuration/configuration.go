package configuration

import "time"

// MediaType is the type of media which will be queued for downoad.
type MediaType uint8

const (
	_ MediaType = iota
	MediaImages
	MediaVideos
	MediaAny
)

const (
	DefaultWorkerCount = 16
	DefaultSleepTime   = 5 * time.Second
)

// Data is the configuration data for the Downloader.
type Data struct {
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
