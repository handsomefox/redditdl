package downloader

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
