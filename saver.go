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
	"runtime"
	"strings"
	"sync"
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

	downloadQueue chan *api.Post
	saveQueue     chan SaverItem
}

func NewSaver(args *AppArguments) *Saver {
	return &Saver{
		skipped:       atomic.Int64{},
		queued:        atomic.Int64{},
		saved:         atomic.Int64{},
		failed:        atomic.Int64{},
		client:        api.DefaultClient(),
		args:          args,
		downloadQueue: make(chan *api.Post, 16),
		saveQueue:     make(chan SaverItem, 8),
	}
}

func (s *Saver) Run(ctx context.Context) error {
	if err := ChdirOrCreate(s.args.SaveDirectory, true); err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	wg := new(sync.WaitGroup)
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.downloadLoop(ctx, wd)
		}()
	}
	defer close(s.downloadQueue)

	go s.saveLoop()
	defer close(s.saveQueue)

	opts := s.argsAsOpts()
	log.Debug().Any("args", opts).Send()

	subreddits, err := s.formatSubreddits()
	if err != nil {
		return err
	}

	streamer, err := api.NewRedditStreamer(s.client, opts, subreddits...)
	if err != nil {
		return err
	}

	stream, continue_, err := streamer.Stream(ctx)
	if err != nil {
		return err
	}
	defer streamer.End()

	var exit bool // If exit is true, this goroutine stop asking for more to fetched.
	go func() {
		for s.totalWithoutSkipped() != s.args.MediaCount && !exit {
			continue_ <- struct{}{}
		}
	}()

	if s.args.ProgressLogging {
		go s.progressLoop()
	}

	for res := range stream {
		if s.totalWithoutSkipped() >= s.args.MediaCount && s.queued.Load() == 0 {
			log.Info().Int64("total", s.saved.Load()).Msg("Finished downloading")
			exit = true
			return nil
		}

		if err := res.Error; err != nil {

			if errors.Is(err, api.ErrStreamEOF) {
				log.Debug().Msg("worker finished")
			}

			if errors.Is(err, api.ErrStreamEnded) {
				exit = true
				streamer.End()
				return nil
			}

			log.Err(err).Send()
		}

		s.downloadQueue <- res.Post
	}

	return nil
}

func (s *Saver) formatSubreddits() ([]string, error) {
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

func (s *Saver) argsAsOpts() *api.Options {
	return &api.Options{
		ContentType: s.args.SubredditContentType,
		Sort:        s.args.SubredditSort,
		Timeframe:   s.args.SubredditTimeframe,
		ShowNSFW:    s.args.ShowNSFW,
	}
}

func (s *Saver) downloadLoop(ctx context.Context, wd string) {
	log.Debug().Msg("started the save loop")

	for post := range s.downloadQueue {
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

		if s.totalWithoutSkipped() < s.args.MediaCount {
			s.saveQueue <- SaverItem{
				Data: item,
				Path: filepath.Join(wd, strings.ToLower(post.Data.Subreddit), filename),
			}
		} else {
			return
		}

		s.queued.Add(1)
	}
}

func (s *Saver) saveLoop() {
	for item := range s.saveQueue {
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
	// printProgressLoop prints the current progress of download every two seconds.
	log.Debug().Msg("started the progress loop")

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

	for s.totalWithoutSkipped() < s.args.MediaCount {
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
		time.Sleep(time.Millisecond + 500)
	}
	if !s.args.VerboseLogging {
		fmt.Println()
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
	return s.saved.Load() + s.failed.Load()
}
