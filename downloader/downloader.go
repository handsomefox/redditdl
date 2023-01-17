package downloader

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/handsomefox/redditdl/client"
	"github.com/handsomefox/redditdl/logging"
	"github.com/handsomefox/redditdl/util"
	"go.uber.org/zap"
)

type Downloader struct {
	currProgress struct{ queued, finished, failed atomic.Int64 }
	cfg          *Config

	log *zap.SugaredLogger

	client       *client.Client
	clientConfig *client.Config

	statusCh chan StatusMessage
	filters  []Filter
}

func New(cfg *Config, clientCfg *client.Config, filters ...Filter) *Downloader {
	cfg.WorkerCount %= int(clientCfg.Count)
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 1
	}
	return &Downloader{
		log:          logging.Get(),
		cfg:          cfg,
		client:       client.NewClient(),
		clientConfig: clientCfg,
		filters:      filters,
		statusCh:     nil,
	}
}

// Download return a channel used to communicate download status (started, finished, failed, errors (if any)).
func (dl *Downloader) Download(ctx context.Context) <-chan StatusMessage {
	dl.log.Debug(dl.cfg, dl.clientConfig)
	dl.statusCh = make(chan StatusMessage, 16)
	go func() {
		dl.run(ctx)
		close(dl.statusCh)
	}()
	return dl.statusCh
}

func (dl *Downloader) run(ctx context.Context) {
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
		dl.postsLoop(ctx, contentCh)
	}()

	// Downloading posts from the content channel and storing data in files channel.
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(fileCh)
		dl.downloadLoop(ctx, fileCh, contentCh)
	}()

	// Saving data from files channel to disk.
	wg.Add(1)
	go func() {
		defer wg.Done()
		// If we cannot save files, gracefully stop downloading
		if err := dl.saveLoop(fileCh); err != nil {
			close(contentCh)
			close(fileCh)
			dl.log.Info("cannot save files", err)
			return
		}
	}()

	exitChan := make(chan struct{})
	defer close(exitChan)
	if dl.cfg.ShowProgress {
		go dl.progressLoop(exitChan)
	}

	wg.Wait()

	if dl.cfg.ShowProgress {
		exitChan <- struct{}{}
	}
}

// postsLoop is fetching posts using the (Downloader.client).GetPosts() and sends them to contentCh.
func (dl *Downloader) postsLoop(ctx context.Context, contentCh chan<- *client.Content) {
	dl.log.Debug("started fetching posts")
	posts := dl.client.GetPosts(ctx, dl.clientConfig)
	for post := range posts {
		dl.statusCh <- StatusMessage{Error: nil, Status: StatusStarted}
		dl.currProgress.queued.Add(1)
		contentCh <- client.NewContent(post)
	}
}

// downloadLoop is downloading the files from content chan to files chan using multiple goroutines.
func (dl *Downloader) downloadLoop(ctx context.Context, fileCh chan<- *util.File, contentCh <-chan *client.Content) {
	dl.log.Debug("started the download loop")
	wg := new(sync.WaitGroup)
	for i := 0; i < dl.cfg.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dl.fileLoop(ctx, fileCh, contentCh)
		}()
	}
	wg.Wait()
}

// fileLoop gets files from the inChan, fetches their data and stores it in outChan.
func (dl *Downloader) fileLoop(ctx context.Context, fileCh chan<- *util.File, contentCh <-chan *client.Content) {
	dl.log.Debug("started the file loop")
	for content := range contentCh {
		file, ext, err := dl.client.GetFile(ctx, content.URL)
		if err != nil {
			dl.statusCh <- StatusMessage{Error: err, Status: StatusFailed}
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
				dl.statusCh <- StatusMessage{Error: ErrNoFileExtension, Status: StatusFailed}
				dl.currProgress.failed.Add(1)
				continue
			}
		} else {
			extension = *ext
		}

		fileCh <- &util.File{
			Name:      content.Name,
			Extension: extension,
			Data:      file,
		}
	}
}

// saveLoop gets data from filesChan and stores it on disk.
func (dl *Downloader) saveLoop(fileCh <-chan *util.File) error {
	dl.log.Debug("started the save loop")
	if err := util.NavigateTo(dl.cfg.Directory, true); err != nil {
		dl.currProgress.failed.Store(dl.currProgress.queued.Load())
		return err
	}
	for file := range fileCh {
		filename, err := util.NewFilename(file.Name, file.Extension)
		if err != nil {
			dl.log.Debugf("couldn't generate a filename: %s", err.Error())
			dl.statusCh <- StatusMessage{Error: err, Status: StatusFailed}
			dl.currProgress.failed.Add(1)
			continue
		}
		if err := util.Save(filename, file.Data); err != nil {
			dl.log.Debugf("couldn't save a file: %s", err.Error())
			dl.statusCh <- StatusMessage{Error: err, Status: StatusFailed}
			dl.currProgress.failed.Add(1)
			continue
		}
		dl.statusCh <- StatusMessage{Error: nil, Status: StatusFinished}
		dl.currProgress.finished.Add(1)
		dl.log.Debugf("saved file: %s", file.Name)
	}

	return nil
}

// progressLoop prints the current progress of download every two seconds.
func (dl *Downloader) progressLoop(exitCh <-chan struct{}) {
	dl.log.Debug("started the progress loop")
	const fStr = "Current progress: queued=%d, finished=%d, failed=%d"
	for {
		select {
		case <-exitCh:
			return
		default:
			p := &dl.currProgress
			dl.log.Infof(fStr, p.queued.Load(), p.finished.Load(), p.failed.Load())
			time.Sleep(time.Second)
		}
	}
}
