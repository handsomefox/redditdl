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
	"github.com/rs/zerolog/log"
)

// Stats reports the download results.
type Stats struct {
	Finished, Failed int64
}

type Downloader struct {
	progressTracker struct {
		queued   atomic.Int64
		finished atomic.Int64
		failed   atomic.Int64
	}

	cliParams   *params.CLIParameters
	client      *client.Client
	filters     []Filter
	workerCount int
}

func New(cliParams *params.CLIParameters, filters ...Filter) (*Downloader, error) {
	if cliParams == nil {
		return nil, ErrNoParams
	}

	workerCount := (2 * runtime.NumCPU()) % int(cliParams.MediaCount)
	if workerCount <= 0 {
		workerCount = 1
	}

	log.Debug().Int("worker_count", workerCount).Send()

	return &Downloader{
		cliParams:   cliParams,
		client:      client.NewClient(cliParams.Sort, cliParams.Timeframe),
		filters:     filters,
		workerCount: workerCount,
	}, nil
}

// Download return a channel used to communicate download status (started, finished, failed, errors (if any)).
func (dl *Downloader) Download(ctx context.Context) Stats {
	log.Debug().Any("parameters", dl.cliParams).Send()

	return dl.run(ctx)
}

func (dl *Downloader) run(ctx context.Context) Stats {
	exitChan := make(chan struct{})

	defer close(exitChan)

	if dl.cliParams.ShowProgress {
		go dl.printProgressLoop(exitChan)
	}

	workdir, err := os.Getwd()
	if err != nil {
		log.Err(err).Msg("cannot get working directory")
		return Stats{}
	}

	if filepath.IsAbs(dl.cliParams.Directory) {
		workdir = dl.cliParams.Directory
	} else {
		workdir = filepath.Join(workdir, dl.cliParams.Directory)
	}

	for _, subreddit := range dl.cliParams.Subreddits {
		// Navigate to original directory.
		err := ChdirOrCreate(workdir, true)
		if err != nil {
			log.Err(err).Msg("failed to navigate to working directory")
			return Stats{}
		}

		currentDir := filepath.Join(workdir, subreddit)

		log.Debug().Str("current_dir", currentDir).Msg("currently saving to this directory")

		// Change directory to specific subreddit.
		if err := ChdirOrCreate(subreddit, true); err != nil {
			log.Err(err).Msg("failed to navigate to subreddit directory")
			return Stats{}
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
			dl.getPostsLoop(ctx, subreddit, contentCh)
		}()

		// Downloading posts from the content channel and storing data in files channel.
		wg.Add(1)
		go func() {
			defer wg.Done()
			dl.downloadAndSaveLoop(currentDir, contentCh)
		}()
		wg.Wait()
	}

	if dl.cliParams.ShowProgress {
		exitChan <- struct{}{}
	}

	return Stats{
		Finished: dl.loadFinished(),
		Failed:   dl.loadFailed(),
	}
}

// getPostsLoop is fetching posts using the (Downloader.client).GetPosts() and sends them to contentCh.
func (dl *Downloader) getPostsLoop(ctx context.Context, subreddit string, contentOutCh chan<- *media.Content) {
	log.Debug().Msg("started fetching posts")

	contentCh := dl.client.GetPostsContent(ctx, dl.cliParams.MediaCount, subreddit)

	for content := range contentCh {
		if IsFiltered(dl.cliParams, content, dl.filters...) {
			continue
		}

		dl.addQueued()

		contentOutCh <- content
	}
}

func (dl *Downloader) downloadAndSaveLoop(basePath string, contentCh <-chan *media.Content) {
	log.Debug().Msg("starting download/save loop")

	type Saving struct {
		Name      string
		Extension string
		Bytes     []byte
	}

	const BufferSize = 8

	saveCh := make(chan *Saving, BufferSize)

	defer close(saveCh)

	wg := new(sync.WaitGroup)

	for i := 0; i < dl.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for content := range contentCh {
				b, err := io.ReadAll(content)
				if err != nil {
					dl.addFailed()
					continue
				}

				content.Close()

				saveCh <- &Saving{
					Bytes:     b,
					Name:      content.Name,
					Extension: content.Extension,
				}
			}
		}()
	}

	go func() {
		for saving := range saveCh {
			err := SaveBytesToDisk(saving.Bytes, basePath, saving.Name, saving.Extension)
			if err != nil {
				dl.addFailed()
			} else {
				dl.addFinished()
			}
		}
	}()

	wg.Wait()
}

// printProgressLoop prints the current progress of download every two seconds.
func (dl *Downloader) printProgressLoop(exitCh <-chan struct{}) {
	log.Debug().Msg("started the progress loop")

	const fStr = "Current progress: queued=%d, finished=%d, failed=%d"

	for {
		select {
		case <-exitCh:
			return
		default:
			p := &dl.progressTracker
			log.Info().Msg(fmt.Sprintf(fStr, p.queued.Load(), p.finished.Load(), p.failed.Load()))
			time.Sleep(time.Second)
		}
	}
}

func (dl *Downloader) addFinished() {
	dl.progressTracker.finished.Add(1)
}

func (dl *Downloader) addFailed() {
	dl.progressTracker.failed.Add(1)
}

func (dl *Downloader) addQueued() {
	dl.progressTracker.queued.Add(1)
}

func (dl *Downloader) loadFinished() int64 {
	return dl.progressTracker.finished.Load()
}

func (dl *Downloader) loadFailed() int64 {
	return dl.progressTracker.failed.Load()
}

func (dl *Downloader) loadQueued() int64 {
	return dl.progressTracker.queued.Load()
}
