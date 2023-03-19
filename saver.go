package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/handsomefox/redditdl/api"
	"github.com/rs/zerolog/log"
)

type SaverItem struct {
	Data *api.Item
	Path string
}

type Saver struct {
	skipped atomic.Int64
	queued  atomic.Int64
	saved   atomic.Int64
	failed  atomic.Int64

	client *api.Client
	args   *AppArguments
}

func NewSaver(args *AppArguments) *Saver {
	return &Saver{
		skipped: atomic.Int64{},
		queued:  atomic.Int64{},
		saved:   atomic.Int64{},
		failed:  atomic.Int64{},
		client:  api.DefaultClient(),
		args:    args,
	}
}

func (s *Saver) Run(ctx context.Context, workerCount int, bufferSize int) error {
	if err := ChdirOrCreate(s.args.SaveDirectory, true); err != nil {
		return err
	}

	if bufferSize == 0 {
		log.Debug().Msg("using unbuffered channels")
	}

	if workerCount == 0 {
		log.Debug().Msg("no worker count provided, falling back on 1")
		workerCount = 1
	}

	subreddits, err := s.prepareSubreddits()
	if err != nil {
		return err
	}

	saveQueue := make(chan SaverItem, bufferSize)
	defer close(saveQueue)
	go s.saveLoop(saveQueue)

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	downloadQueue := make(chan *api.Post, bufferSize)
	defer close(downloadQueue)
	for i := 0; i < workerCount; i++ {
		go s.downloadLoop(ctx, downloadQueue, saveQueue, wd)
	}

	streamer, err := api.NewRedditStreamer(s.client, s.argsAsOpts(), subreddits...)
	if err != nil {
		return err
	}

	stopCh := make(chan struct{})
	defer close(stopCh)
	stream, moreCh, err := streamer.Stream(ctx, stopCh, bufferSize)
	if err != nil {
		return err
	}

	if s.args.ProgressLogging {
		go s.progressLoop()
	}

	moreCh <- struct{}{}
	for i := int64(0); i < s.args.MediaCount; i = s.saved.Load() {
		result, ok := <-stream
		if !ok {
			log.Debug().Msg("stream finished")
			break
		}
		if err := result.Error; err != nil {
			if errors.Is(err, api.ErrStreamEOF) {
				log.Debug().Msg("worker finished")
			}
			if errors.Is(err, api.ErrStreamEnded) {
				log.Debug().Msg("stream finished")
				break
			}
			log.Err(err).Send()
		} else {
			downloadQueue <- result.Post
			s.queued.Add(1)
		}
		moreCh <- struct{}{}
	}

	if !s.args.VerboseLogging {
		fmt.Println()
	}
	log.Info().Int64("total", s.saved.Load()).Msg("Finished downloading")

	return nil
}

func (s *Saver) prepareSubreddits() ([]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	subreddits := strings.Split(s.args.SubredditList, ",")
	for i := 0; i < len(subreddits); i++ {
		subreddits[i] = strings.TrimSpace(subreddits[i])
		log.Debug().Str("subreddit", subreddits[i]).Msg("adding subreddit")
		dir := filepath.Join(wd, strings.ToLower(subreddits[i]))
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			if !errors.Is(err, os.ErrExist) {
				log.Err(err).Send()
			} else {
				continue
			}
		}
	}

	return subreddits, nil
}

func (s *Saver) argsAsOpts() api.Options {
	return api.Options{
		ContentType: s.args.SubredditContentType,
		Sort:        s.args.SubredditSort,
		Timeframe:   s.args.SubredditTimeframe,
		ShowNSFW:    s.args.ShowNSFW,
	}
}

func (s *Saver) downloadLoop(ctx context.Context, downloadQueue <-chan *api.Post, saverQueue chan<- SaverItem, wd string) {
	for post := range downloadQueue {
		if !s.isEligibleForSaving(post) {
			log.Debug().Msg("skipped an item")
			s.skipped.Add(1)
			continue
		}

		item, err := s.client.Subreddit.PostToItem(ctx, post)
		if err != nil {
			log.Err(err).Msg("failed to convert a post to an item")
			s.failed.Add(1)
			continue
		}

		// item path is:
		// {working_directory}/{subreddit}/{item_name}.{item_extension}
		filename, err := NewFormattedFilename(item.Name, item.Extension)
		if err != nil {
			log.Err(err).Str("item_name", item.Name).Msg("failed to save item")
			s.failed.Add(1)
			continue
		}

		saverQueue <- SaverItem{
			Data: item,
			Path: filepath.Join(wd, strings.ToLower(post.Data.Subreddit), filename),
		}
	}
}

func (s *Saver) saveLoop(saveQueue <-chan SaverItem) {
	for item := range saveQueue {
		if err := s.WriteFile(item.Path, item.Data.Bytes); err != nil {
			s.failed.Add(1)
			log.Err(err).Msg("failed to write file to disk")
		} else {
			s.saved.Add(1)
		}
		s.queued.Store(s.queued.Load() - 1)
	}
}

func (s *Saver) progressLoop() {
	var (
		lastTotal = int64(0)
		// Specified format string for printing
		stringf = "Download status: Queued=%d; Saved=%d; Failed=%d; Skipped=%d"
		// Function used for printing (by default, zerolog)
		progprint = func(msg string) { log.Info().Msg(msg) }
	)

	if !s.args.VerboseLogging {
		// if no logging will be done, we can take control and print in a single line.
		stringf = "Download status: Queued=%d; Saved=%d; Failed=%d; Skipped=%d\r"
		// Use package fmt for carriage return working correctly
		progprint = func(msg string) { fmt.Print(msg) }
	}

	for s.saved.Load()+s.failed.Load() < s.args.MediaCount {
		saved := s.saved.Load()
		failed := s.failed.Load()
		queued := s.queued.Load()
		skipped := s.skipped.Load()
		total := s.saved.Load() + failed + queued + skipped

		if lastTotal < total {
			progprint(fmt.Sprintf(stringf, queued, saved, failed, skipped))
			lastTotal = total
		}
		// No need to update all the time
		time.Sleep(time.Second * 1)
	}
}

func (s *Saver) WriteFile(path string, b []byte) error {
	file, err := os.Create(path)
	if err != nil {
		log.Debug().Msg(path)
		return err
	}
	defer file.Close()

	fw := bufio.NewWriter(file)
	defer fw.Flush()

	br := bytes.NewBuffer(b)

	n, err := io.Copy(fw, br)
	if err != nil {
		return err
	}

	log.Debug().Int64("written_bytes", n).Str("path", path).Msg("wrote to disk")

	return nil
}

// isEligibleForSaving checks if the post goes through all the specified parameters by the user.
func (s *Saver) isEligibleForSaving(p *api.Post) bool {
	if p == nil {
		return false
	}
	if s.args.SubredditContentType != "both" {
		if s.args.SubredditContentType != p.Type() {
			log.Debug().
				Str("want_content_type", s.args.SubredditContentType).
				Str("got_content_type", p.Type()).
				Msg("unexpected content_type")
			return false
		}
	}

	if s.args.SubredditContentType == "link" || s.args.SubredditContentType == "text" {
		log.Debug().Str("content_type", s.args.SubredditContentType).Msg("unexpected content type")
		return false
	}

	w, h := p.Dimensions()
	if w < s.args.MediaMinimalWidth && h < s.args.MediaMinimalHeight {
		log.Debug().Int("width", w).Int("height", h).Msg("unfit dimensions")
		return false
	}

	if p.Data.Over18 && !s.args.ShowNSFW {
		log.Debug().Msg("filtered out NSFW")
		return false
	}

	if s.args.MediaOrientation != "all" {
		if s.args.MediaOrientation != p.Orientation() {
			log.Debug().Msg("filtered out by orientation")
			return false
		}
	}

	return true
}

func (s *Saver) totalWithoutSkipped() int64 {
	return s.saved.Load() + s.failed.Load() + s.queued.Load()
}
