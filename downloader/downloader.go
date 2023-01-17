package downloader

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/handsomefox/redditdl/client"
	"github.com/handsomefox/redditdl/logging"
	"github.com/handsomefox/redditdl/util"
)

var log = logging.Get()

type DownloadStatus byte

const (
	_ DownloadStatus = iota
	StatusStarted
	StatusFinished
	StatusFailed
)

type Downloader struct {
	currProgress struct{ queued, finished, failed atomic.Int64 }
	cfg          *Config

	client       *client.Client
	clientConfig *client.Config

	statusCh chan DownloadStatus
	filters  []Filter
}

func NewDownloader(cfg Config, clientCfg client.Config, filters ...Filter) *Downloader {
	return &Downloader{
		cfg:          &cfg,
		client:       client.NewClient(),
		clientConfig: &clientCfg,
		filters:      filters,
		statusCh:     nil,
	}
}

// Download return a channel used to communicate download status (started, finished, failed).
func (dl *Downloader) Download() <-chan DownloadStatus {
	dl.statusCh = make(chan DownloadStatus, 16)
	go func() {
		dl.loop()
		close(dl.statusCh)
	}()
	return dl.statusCh
}

func (dl *Downloader) loop() {
	var (
		contentCh = make(chan *client.Content)
		fileCh    = make(chan *util.File)
	)

	// Fetching posts to the content channel for further download.
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(contentCh)
		dl.fetchPosts(context.Background(), contentCh)
	}()

	// Downloading posts from the content channel and storing data in files channel.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(fileCh)
		dl.downloadRoutine(context.Background(), fileCh, contentCh)
	}()

	// Saving data from files channel to disk.
	wg.Add(1)
	go func() {
		defer wg.Done()
		dl.saveFiles(fileCh)
	}()

	exitChan := make(chan struct{})
	if dl.cfg.ShowProgress {
		go dl.showProgress(exitChan)
	}

	wg.Wait()

	if dl.cfg.ShowProgress {
		exitChan <- struct{}{}
	}

	close(exitChan)
}

func (dl *Downloader) fetchPosts(ctx context.Context, ch chan<- *client.Content) {
}

// downloadRoutine is downloading the files from content chan to files chan using multiple goroutines.
func (dl *Downloader) downloadRoutine(ctx context.Context, fileChan chan<- *util.File, contentChan <-chan *client.Content) {
	wg := new(sync.WaitGroup)
	for i := 0; i < dl.cfg.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dl.downloadFiles(ctx, fileChan, contentChan)
		}()
	}
	wg.Wait()
}

// downloadFiles gets files from the inChan, fetches their data and stores it in outChan.
func (dl *Downloader) downloadFiles(ctx context.Context, fileChan chan<- *util.File, contentChan <-chan *client.Content) {
	for content := range contentChan {
		file, ext, err := dl.client.GetFile(ctx, content.URL)
		if err != nil {
			dl.currProgress.failed.Add(1)
			continue
		}
		var extension string
		if ext == nil {
			switch content.Type {
			case client.ContentImage:
				extension = ".jpg"
			case client.ContentVideo:
				extension = ".mp4"
			default:
				dl.currProgress.failed.Add(1)
				continue
			}
		} else {
			extension = *ext
		}

		fileChan <- &util.File{
			Name:      content.Name,
			Extension: extension,
			Data:      file,
		}
	}
}

// saveFiles gets data from filesChan and stores it on disk.
func (dl *Downloader) saveFiles(filesChan <-chan *util.File) {
	if err := util.NavigateTo(dl.cfg.Directory, true); err != nil {
		dl.currProgress.failed.Store(dl.currProgress.queued.Load())
		return
	}
	for file := range filesChan {
		filename, err := util.NewFilename(file.Name, file.Extension)
		if err != nil {
			log.Debugf("%s: failed to save file", err)
			continue
		}
		if err := util.Save(filename, file.Data); err != nil {
			continue
		}
		dl.currProgress.finished.Add(1)
		log.Debugf("saved file: %s", file.Name)
	}
}

// showProgress prints the current progress of download every two seconds.
func (dl *Downloader) showProgress(exit <-chan struct{}) {
	fStr := "Current progress: queued=%d, finished=%d, failed=%d"
	for {
		select {
		case <-exit:
			return
		default:
			p := &dl.currProgress
			log.Infof(fStr, p.queued.Load(), p.finished.Load(), p.failed.Load())
			time.Sleep(time.Second)
		}
	}
}
