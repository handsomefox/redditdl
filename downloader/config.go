package downloader

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
)

// Config is the configuration data for the Downloader.
type Config struct {
	Directory    string
	WorkerCount  int
	ShowProgress bool
	ContentType  ContentType
}
