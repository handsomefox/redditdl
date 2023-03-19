package api

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type Streamer interface {
	Stream(ctx context.Context) (<-chan StreamResult, chan<- struct{}, error)
	End()
}

type Item struct {
	Bytes []byte

	Name        string
	Extension   string
	URL         string
	Orientation string
	Type        string

	Width  int
	Height int

	IsOver18 bool
}

type StreamResult struct {
	Post      *Post
	Subreddit string // Specifies from which subreddit the item came from (since multiple can be specified)
	Error     error
}

type StreamError struct {
	err         error
	explanation string
}

// Indicates that stream for the particular subreddit has finished.
type StreamEOF string

func (s StreamEOF) Error() string {
	return string(s)
}

// Indicates that the whole stream has finished
type StreamEnded string

func (s StreamEnded) Error() string {
	return string(s)
}

func (se *StreamError) Error() string {
	if se.err != nil {
		return se.explanation + ": " + se.err.Error()
	}
	return se.explanation
}

type Options struct {
	ContentType string
	Sort        string
	Timeframe   string
	ShowNSFW    bool
}

type RedditStreamer struct {
	client     *Client
	opts       *Options
	subreddits []string

	stream    chan StreamResult
	continue_ chan struct{}

	end atomic.Bool
}

func NewRedditStreamer(client *Client, opts *Options, subreddits ...string) (Streamer, error) {
	if len(subreddits) == 0 {
		return nil, &StreamError{err: nil, explanation: "empty list of subreddits provided"}
	}
	rs := &RedditStreamer{
		client:     client,
		opts:       opts,
		subreddits: subreddits,
		stream:     make(chan StreamResult, len(subreddits)),
		continue_:  make(chan struct{}),
		end:        atomic.Bool{},
	}
	return rs, nil
}

// Stream starts the stream, until the End() is called.
// The streamer listens to the "continue" (chan struct{}) channel.
// If the End() is called, it terminates.
// If the streamer cannot continue, but continue is received, it will send a StreamResult continuing
// StreamEOF error.
func (rs *RedditStreamer) Stream(ctx context.Context) (<-chan StreamResult, chan<- struct{}, error) {
	rs.end.Store(false)

	wg := new(sync.WaitGroup)
	wg.Add(len(rs.subreddits))
	for _, s := range rs.subreddits {
		s := s
		go func() {
			defer wg.Done()
			rs.run(ctx, s)
		}()
	}
	go func() {
		wg.Wait()
		if !rs.end.Load() {
			rs.stream <- StreamResult{
				Error: StreamEnded("all workers have finished, this streamer is now useless"),
			}
		}
	}()

	return rs.stream, rs.continue_, nil
}

// Signals to the streamer to end.
func (rs *RedditStreamer) End() {
	rs.end.Store(true)
	close(rs.stream)
	close(rs.continue_)
}

func (rs *RedditStreamer) run(ctx context.Context, subreddit string) {
	after := ""
	for range rs.continue_ {
		if rs.end.Load() {
			return
		}
		after2, err := rs.fetchPost(ctx, int64(1), after, subreddit)
		if err == nil {
			after = after2
			time.Sleep(500 * time.Millisecond) // Don't spam reddit too much
			continue
		}
		rs.stream <- StreamResult{
			Error: err,
		}
	}
}

// fetchItem appends items to results chan, or, if there are no more items ("after is empty"), it returns a StreamError.
func (rs *RedditStreamer) fetchPost(ctx context.Context, count int64, after, subreddit string) (string, error) {
	opts := &RequestOptions{
		After:     after,
		Count:     count,
		Sorting:   rs.opts.Sort,
		Timeframe: rs.opts.Timeframe,
		Subreddit: subreddit,
	}

	res, after, err := rs.client.Subreddit.GetPosts(ctx, opts)
	if err != nil {
		return "", err
	}

	if len(res) == 0 {
		return "", StreamEOF("no more posts to read")
	}

	for i := 0; i < len(res); i++ {
		if rs.end.Load() {
			break
		}
		rs.stream <- StreamResult{
			Post:      res[i],
			Error:     nil,
			Subreddit: subreddit,
		}
	}

	return after, nil
}
