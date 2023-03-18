package api

import (
	"context"
	"runtime"

	"github.com/rs/zerolog/log"
)

type Streamer interface {
	Stream(ctx context.Context, terminate <-chan struct{}) (<-chan StreamResult, error)
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

// Indicates that stream has ended
type StreamEOF string

func (s StreamEOF) Error() string {
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

	completed chan struct{}
}

func NewRedditStreamer(client *Client, opts *Options, subreddits ...string) (Streamer, error) {
	if len(subreddits) == 0 {
		return nil, &StreamError{err: nil, explanation: "empty list of subreddits provided"}
	}
	rs := &RedditStreamer{
		client:     client,
		opts:       opts,
		subreddits: subreddits,
		completed:  make(chan struct{}),
	}

	return rs, nil
}

func (rs *RedditStreamer) Stream(ctx context.Context, terminate <-chan struct{}) (<-chan StreamResult, error) {
	var (
		c       = make(chan StreamResult, len(rs.subreddits)) // Make a buffered channel
		signals = make([]chan struct{}, 0)                    // Each thread has it's own channel for termination, they close it themselves.
	)

	for _, s := range rs.subreddits {
		var (
			s      = s
			signal = make(chan struct{})
		)

		signals = append(signals, signal) // Store it for later use

		go func() { rs.run(ctx, c, signal, s) }()
	}

	go func() {
		defer close(c)
		defer func() {
			for i := 0; i < len(signals); i++ {
				signals[i] <- struct{}{}
			}
		}()
		completed := 0
		for {
			select {
			// This waits until terminnation is received and then reports to other goroutines.
			// It also closes the item channel, the caller only needs to signal termination.
			case <-terminate:
				return
			// This will watch for the runners (if they run out of posts to download)
			// When all of them complete, rs.completed will be equal to len(rs.subreddits)
			case <-rs.completed:
				completed++
				if completed == len(rs.subreddits) {
					// All of the runners finished
					c <- StreamResult{Error: StreamEOF("all of the runners finished")}
					return
				}
				// Yield the CPU for other goroutines
			default:
				runtime.Gosched()
			}
		}
	}()

	return c, nil
}

func (rs *RedditStreamer) run(ctx context.Context, results chan<- StreamResult, terminate chan struct{}, subreddit string) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// We are the ones responsible for closing the channel, after receiving terminate signal.
	defer close(terminate)

	var after string
	var completed bool

	for {
		select {
		case <-terminate:
			return
		default:
			if completed {
				runtime.Gosched()
			}
			after2, err := rs.fetchPost(ctx, after, subreddit, results)
			if err != nil {
				if _, ok := err.(*StreamError); ok {
					log.Info().Str("subreddit", subreddit).Msg("no more posts to fetch")
				} else if _, ok := err.(StreamEOF); ok {
					rs.completed <- struct{}{}
					completed = true
				} else {
					log.Err(err)
				}
			} else {
				after = after2
			}
		}
	}
}

// fetchItem appends items to results chan, or, if there are no more items ("after is empty"), it returns a StreamError.
func (rs *RedditStreamer) fetchPost(ctx context.Context, after, subreddit string, results chan<- StreamResult) (string, error) {
	opts := &RequestOptions{
		After:     after,
		Count:     10,
		Sorting:   rs.opts.Sort,
		Timeframe: rs.opts.Timeframe,
		Subreddit: subreddit,
	}

	res, after, err := rs.client.Subreddit.GetPosts(ctx, opts)
	if err != nil {
		log.Err(err).Msg("error when fetching an item")
		return "", err
	}

	if len(res) == 0 {
		return "", StreamEOF("no more posts to read")
	}

	for i := 0; i < len(res); i++ {
		results <- StreamResult{
			Post:      res[i],
			Error:     nil,
			Subreddit: subreddit,
		}
	}

	return after, nil
}