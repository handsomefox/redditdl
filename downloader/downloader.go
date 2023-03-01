package downloader

import (
	"bufio"
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

// queueItem will be scheduled for saving to disk later using the path specified.
type queueItem struct {
	Content  *media.Content
	Fullpath string
}

type Downloader struct {
	progress struct {
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

	// Figure out absolute path for saving files later.
	if filepath.IsAbs(dl.cliParams.Directory) {
		workdir = dl.cliParams.Directory
	} else {
		workdir = filepath.Join(workdir, dl.cliParams.Directory)
	}

	// Navigate to specified directory.
	err = ChdirOrCreate(workdir, true)
	if err != nil {
		log.Err(err).Msg("failed to navigate to specified directory")
		return Stats{}
	}

	// Create all directories beforehand.
	for _, subreddit := range dl.cliParams.Subreddits {
		dirPath := filepath.Join(workdir, subreddit)
		if err := os.Mkdir(dirPath, os.ModePerm); err != nil {
			if !errors.Is(err, os.ErrExist) {
				log.Fatal().Err(err).Msg("couldn't create a directory")
			}
		}
		log.Debug().Str("dir_path", dirPath).Msg("created directory")
	}

	var (
		queue   = make(chan queueItem)
		fetchWg = new(sync.WaitGroup) // specific to the fetch loop
		wg      = new(sync.WaitGroup)
	)

	// Fetching posts to the queue for further download.
	for _, subreddit := range dl.cliParams.Subreddits {
		subreddit := subreddit

		fetchWg.Add(1)
		go func() {
			defer fetchWg.Done()
			dl.getPostsLoop(ctx, workdir, subreddit, queue)
		}()
	}

	// This functions is responsible for closing the queue after
	// fetching is done (when all goroutines finish in fetchWg).
	wg.Add(1)
	go func() {
		defer close(queue)
		defer wg.Done()
		fetchWg.Wait()
	}()

	// Downloading posts from the content channel and storing data in files channel.
	wg.Add(1)
	go func() {
		defer wg.Done()
		dl.downloadAndSaveLoop(queue)
	}()

	// Wait until everything finishes.
	wg.Wait()

	// Close the progressbar
	if dl.cliParams.ShowProgress {
		exitChan <- struct{}{}
	}

	return Stats{
		Finished: dl.loadFinished(),
		Failed:   dl.loadFailed(),
	}
}

// getPostsLoop is fetching posts using the (Downloader.client).GetPosts() and sends them to contentCh.
func (dl *Downloader) getPostsLoop(ctx context.Context, basePath, subreddit string, queue chan<- queueItem) {
	log.Debug().Str("base_path", basePath).Str("subreddit", subreddit).Msg("started fetching posts")

	contentCh := dl.client.GetPostsContent(ctx, dl.cliParams.MediaCount, subreddit)

	for content := range contentCh {
		if IsFiltered(dl.cliParams, content, dl.filters...) {
			continue
		}

		filename, err := NewFormattedFilename(content.Name, content.Extension)
		if err != nil {
			log.Err(err).Msg("couldn't generate filename")
			continue
		}

		// The path where the file will be saved.
		// Using absolute paths allows us to not care
		// about current working directory.
		fullname := filepath.Join(basePath, subreddit, filename)

		log.Debug().Str("fullname", fullname).Msg("added file to queue")

		dl.addQueued()

		queue <- queueItem{
			Fullpath: fullname,
			Content:  content,
		}
	}
}

func (dl *Downloader) downloadAndSaveLoop(queue <-chan queueItem) {
	log.Debug().Msg("starting download/save loop")

	wg := new(sync.WaitGroup)

	for i := 0; i < dl.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for queued := range queue {
				err := SaveBytesToDisk(queued)
				if err != nil {
					dl.addFailed()
				} else {
					log.Debug().Str("fullpath", queued.Fullpath).Msg("saved to disk")
					dl.addFinished()
				}
				// since the underlying value is a response.Body, we should close it to release resources
				queued.Content.Close()
			}
		}()
	}

	wg.Wait()
}

func SaveBytesToDisk(item queueItem) error {
	file, err := os.Create(item.Fullpath)
	if err != nil {
		return fmt.Errorf("%w: couldn't save file(name=%s)", err, item.Fullpath)
	}
	defer file.Close()

	// Use buffered writes.
	w := bufio.NewWriter(file)
	defer w.Flush()

	// Use io.Copy instead of io.ReadAll
	_, err = io.Copy(file, item.Content)
	if err != nil {
		return fmt.Errorf("%w: couldn't save file(name=%s)", err, item.Content.Name)
	}

	return nil
}

// printProgressLoop prints the current progress of download every two seconds.
func (dl *Downloader) printProgressLoop(exitCh <-chan struct{}) {
	log.Debug().Msg("started the progress loop")

	type LastProgress struct {
		queued, finished, failed int64
	}

	var (
		// Store the progress so we don't print redundant statements
		lastProgress = LastProgress{}
		// Use this to compare progress quickly
		less = func(p1, p2 LastProgress) bool {
			sum1 := p1.queued + p1.failed + p1.finished
			sum2 := p2.queued + p2.failed + p2.finished
			return sum1 < sum2
		}
		// Specified format string for printing
		fStr = "Download status: Queued=%d; Finished=%d; Failed=%d."
		// Function used for printing (by default, zerolog)
		printFunc = func(msg string) {
			log.Info().Msg(msg)
		}
	)

	if !dl.cliParams.VerboseLogging {
		// if no logging will be done, we can take control and print in a single line.
		fStr = "Download status: Queued=%d; Finished=%d; Failed=%d\r"
		// Use package fmt for carriage return working correctly
		printFunc = func(msg string) {
			fmt.Print(msg)
		}
	}

	for {
		select {
		case <-exitCh:
			return
		default:
			p := &dl.progress

			currProgress := LastProgress{
				queued:   p.queued.Load(),
				finished: p.finished.Load(),
				failed:   p.failed.Load(),
			}

			// Only print if there is a difference between the two
			if less(lastProgress, currProgress) {
				printFunc(fmt.Sprintf(fStr, currProgress.queued, currProgress.finished, currProgress.failed))
				// Update the progress
				lastProgress = currProgress
			}
			// No need to update all the time
			time.Sleep(time.Second)
		}
	}
}

func (dl *Downloader) addFinished() {
	dl.progress.finished.Add(1)
}

func (dl *Downloader) addFailed() {
	dl.progress.failed.Add(1)
}

func (dl *Downloader) addQueued() {
	dl.progress.queued.Add(1)
}

func (dl *Downloader) loadFinished() int64 {
	return dl.progress.finished.Load()
}

func (dl *Downloader) loadFailed() int64 {
	return dl.progress.failed.Load()
}

func (dl *Downloader) loadQueued() int64 {
	return dl.progress.queued.Load()
}
