package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	client *api.Client
	args   *AppArguments

	downloadQueue chan *api.Post
	saveQueue     chan SaverItem

	skipped atomic.Int64
	queued  atomic.Int64
	saved   atomic.Int64
	failed  atomic.Int64
}

func NewSaver(args *AppArguments) *Saver {
	return &Saver{
		client:        api.DefaultClient(),
		args:          args,
		downloadQueue: make(chan *api.Post, 16),
		saveQueue:     make(chan SaverItem, 8),
		saved:         atomic.Int64{},
		failed:        atomic.Int64{},
	}
}

func (s *Saver) Run() error {
	if err := ChdirOrCreate(s.args.SaveDirectory, true); err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	subreddits := strings.Split(s.args.SubredditList, ",")
	for i := 0; i < len(subreddits); i++ {
		subreddits[i] = strings.TrimSpace(subreddits[i])
		log.Debug().Str("subreddit", subreddits[i]).Msg("adding subreddit")
		if err := os.Mkdir(filepath.Join(wd, strings.ToLower(subreddits[i])), os.ModePerm); err != nil { // create paths beforehand
			if err == os.ErrExist {
				continue
			}
			return err
		}
	}

	if s.args.ProgressLogging {
		go s.progressLoop()
	}

	opts := &api.Options{
		ContentType: s.args.SubredditContentType,
		Sort:        s.args.SubredditSort,
		Timeframe:   s.args.SubredditTimeframe,
		ShowNSFW:    s.args.SubredditShowNSFW,
	}
	log.Debug().Any("args", opts).Send()

	streamer, err := api.NewRedditStreamer(s.client, opts, subreddits...)
	if err != nil {
		return err
	}

	ctx := context.Background()
	wg := new(sync.WaitGroup)

	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.downloadLoop(ctx, wd)
		}()
	}
	defer close(s.downloadQueue)

	go s.saveLoop()
	defer close(s.saveQueue)

	terminate := make(chan struct{})
	defer close(terminate)

	stream, err := streamer.Stream(ctx, terminate)
	if err != nil {
		return err
	}

	for res := range stream {
		if s.totalWithoutSkipped() >= s.args.MediaCount {
			terminate <- struct{}{}
			break
		}
		if res.Error != nil {
			if v, ok := res.Error.(api.StreamEOF); ok {
				log.Err(v).Msg("end of stream reached")
				terminate <- struct{}{}
				break
			} else {
				log.Err(res.Error)
				continue
			}
		}

		s.downloadQueue <- res.Post
	}

	return nil
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
		print = func(msg string) { log.Info().Msg(msg) }
	)
	if !s.args.VerboseLogging {
		// if no logging will be done, we can take control and print in a single line.
		stringf = "Download status: Queued=%d; Saved=%d; Failed=%d; Skipped=%d\r"
		// Use package fmt for carriage return working correctly
		print = func(msg string) { fmt.Print(msg) }
	}
	for {
		saved := s.saved.Load()
		failed := s.failed.Load()
		queued := s.queued.Load()
		skipped := s.skipped.Load()
		total := s.saved.Load() + failed + queued + skipped
		if lastTotal < total {
			print(fmt.Sprintf(stringf, queued, saved, failed, skipped))
			lastTotal = total
		}
		// No need to update all the time
		time.Sleep(time.Millisecond + 500)
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
	if w < s.args.MediaWidth && h < s.args.MediaHeight {
		log.Debug().Int("width", w).Int("height", h).Msg("unfit dimensions")
		return false
	}

	if p.Data.Over18 && !s.args.SubredditShowNSFW {
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
