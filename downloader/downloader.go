package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/handsomefox/redditdl/client"
	"github.com/handsomefox/redditdl/client/media"
	"github.com/handsomefox/redditdl/cmd/params"
	"go.uber.org/zap"
)

var (
	ErrNoFileExtension = errors.New("failed to pick a file extension")
	ErrFailedSave      = errors.New("downloader cannot navigate to directory, terminating")
)

const (
	DefaultWorkerCount = 16
)

type DownloadStatus byte

const (
	_ DownloadStatus = iota
	StatusStarted
	StatusFinished
	StatusFailed
)

type StatusMessage struct {
	Error  error
	Status DownloadStatus
}

type Downloader struct {
	currProgress struct{ queued, finished, failed atomic.Int64 }
	cliParams    *params.CLIParameters
	log          *zap.SugaredLogger
	client       *client.Client
	filters      []Filter
	workerCount  int
}

func New(cliParams *params.CLIParameters, logger *zap.SugaredLogger, filters ...Filter) (*Downloader, error) {
	if cliParams == nil {
		return nil, fmt.Errorf("no params provided")
	}
	wc := (runtime.NumCPU() * 2) % int(cliParams.MediaCount)
	if wc <= 0 {
		wc = 1
	}
	return &Downloader{
		log:         logger,
		cliParams:   cliParams,
		client:      client.NewClient(cliParams.Sort, cliParams.Timeframe),
		filters:     filters,
		workerCount: wc,
	}, nil
}

// Download return a channel used to communicate download status (started, finished, failed, errors (if any)).
func (dl *Downloader) Download(ctx context.Context) <-chan StatusMessage {
	dl.log.Debug(dl.cliParams)
	statusCh := make(chan StatusMessage, 16)
	go func() {
		dl.run(ctx, statusCh)
		close(statusCh)
	}()
	return statusCh
}

func (dl *Downloader) run(ctx context.Context, statusCh chan<- StatusMessage) {
	exitChan := make(chan struct{})
	defer close(exitChan)
	if dl.cliParams.ShowProgress {
		go dl.progressLoop(exitChan)
	}

	wd, err := os.Getwd()
	if err != nil {
		dl.log.Error("cannot get working directory", err)
		return
	}

	if filepath.IsAbs(dl.cliParams.Directory) {
		wd = dl.cliParams.Directory
	} else {
		wd = filepath.Join(wd, dl.cliParams.Directory)
	}

	for _, subreddit := range dl.cliParams.Subreddits {
		// Navigate to original directory.
		err := NavigateTo(wd, true)
		if err != nil {
			dl.log.Error("failed to navigate to working directory ", err)
			return
		}
		// Change directory to specific subreddit.
		if err := NavigateTo(subreddit, true); err != nil {
			dl.log.Error("failed to navigate to subreddit directory ", err)
			return
		}
		// Start the download
		var (
			contentCh = make(chan *media.Content)
			wg        = new(sync.WaitGroup)
		)
		// Fetching posts to the content channel for further download.
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(contentCh)
			dl.postsLoop(ctx, subreddit, contentCh, statusCh)
		}()
		// Downloading posts from the content channel and storing data in files channel.
		wg.Add(1)
		go func() {
			defer wg.Done()
			dl.downloadAndSaveLoop(ctx, contentCh, statusCh)
		}()
		wg.Wait()
	}
	if dl.cliParams.ShowProgress {
		exitChan <- struct{}{}
	}
}

// postsLoop is fetching posts using the (Downloader.client).GetPosts() and sends them to contentCh.
func (dl *Downloader) postsLoop(ctx context.Context, subreddit string, contentCh chan<- *media.Content, statusCh chan<- StatusMessage) {
	dl.log.Debug("started fetching posts")
	cnts := dl.client.GetPostsContent(ctx, dl.cliParams.MediaCount, subreddit)
	for content := range cnts {
		if IsFiltered(dl.cliParams, *content, dl.filters...) {
			continue
		}
		dl.currProgress.queued.Add(1)
		statusCh <- StatusMessage{Error: nil, Status: StatusStarted}
		contentCh <- content
	}
}

func (dl *Downloader) downloadAndSaveLoop(ctx context.Context, contentCh <-chan *media.Content, statusCh chan<- StatusMessage) {
	dl.log.Debug("starting download/save loop")
	var (
		wg     = new(sync.WaitGroup)
		diskMu = new(sync.Mutex) // locked when the goroutine is saving the file to a disk
	)
	for i := 0; i < dl.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for cnt := range contentCh {
				b, err := io.ReadAll(cnt)
				if err != nil {
					dl.currProgress.failed.Add(1)
					statusCh <- StatusMessage{Error: err, Status: StatusFailed}
					continue
				}
				diskMu.Lock()
				if err := saveContent(b, cnt); err != nil {
					dl.currProgress.failed.Add(1)
					statusCh <- StatusMessage{Error: err, Status: StatusFailed}
				}
				diskMu.Unlock()
				cnt.Close()
				dl.currProgress.finished.Add(1)
				statusCh <- StatusMessage{Error: nil, Status: StatusFinished}
			}
		}()
	}
	wg.Wait()
}

func saveContent(b []byte, content *media.Content) error {
	filename, err := NewFilename(content.Name, content.Extension)
	if err != nil {
		return fmt.Errorf("%w: couldn't save file", err)
	}
	if err := os.WriteFile(filename, b, 0o600); err != nil {
		return fmt.Errorf("%w: couldn't save file(name=%s)", err, filename)
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
