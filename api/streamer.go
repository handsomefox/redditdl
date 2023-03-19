package api

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

var (
	ErrStreamError error = &StreamError{}
	ErrStreamEOF         = errors.New("worker reached the end of it's stream")
	ErrStreamEnded       = errors.New("stream is ended completely")
)

type StreamResult struct {
	Error     error
	Post      *Post
	Subreddit string
}

type StreamError struct {
	err         error
	explanation string
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
	client *Client
	opts   Options

	workersFinished atomic.Int64

	subreddits []string
}

func NewRedditStreamer(client *Client, opts Options, subreddits ...string) (*RedditStreamer, error) {
	if len(subreddits) == 0 {
		return nil, &StreamError{err: nil, explanation: "empty list of subreddits provided"}
	}

	rs := &RedditStreamer{
		client:          client,
		opts:            opts,
		workersFinished: atomic.Int64{},
		subreddits:      subreddits,
	}

	return rs, nil
}

func (rs *RedditStreamer) Stream(ctx context.Context, stopCh <-chan struct{}, bufferSize int) (<-chan StreamResult, chan<- struct{}, error) {
	// Channels for the consumer of Start()
	resCh := make(chan StreamResult, bufferSize)
	moreCh := make(chan struct{}, bufferSize)

	// Channels for signaling workers and aggregation
	aggregateCh := make(chan StreamResult, bufferSize) // The items from workerLoop are stored here
	workersMoreCh := make(chan struct{}, bufferSize)   // The workers listen to that channel, when they receive, they send item to the aggregateCh

	workerCount := len(rs.subreddits)

	for _, s := range rs.subreddits {
		s := s
		go rs.workerLoop(ctx, s, workersMoreCh, aggregateCh)
	}

	go func() {
		defer close(resCh)
		defer close(moreCh)
		for {
			select {
			case <-stopCh: // If we stop, break out of the loop and let the closing functions run
				return
			case <-moreCh: // If we get asked by the consumer, ask the workers to get an item, then return the result
				workersMoreCh <- struct{}{}
				resCh <- <-aggregateCh
			default:
				// Here, we check if all the workers have exited
				// If they did, report that the stream has ended.
				// Workers will exit after the workersMoreCh is closed.
				if rs.workersFinished.Load() == int64(workerCount) {
					resCh <- StreamResult{Error: ErrStreamEnded}
					return
				}
			}
		}
	}()

	return resCh, moreCh, nil
}

// workerLoop listens on rs.moreCh and appends results to rs.resCh, if needed
func (rs *RedditStreamer) workerLoop(ctx context.Context, subreddit string, moreCh <-chan struct{}, aggregateCh chan<- StreamResult) {
	var (
		after string
		posts []Post
	)

	defer rs.workersFinished.Add(1)

	// We can ignore the error on the first fetch, since we will check len anyway
	fetched, after2, err := rs.fetchPost(ctx, 100, after, subreddit)
	if err != nil {
		fetched = nil
	} else {
		after = after2
	}

	for range moreCh {
		if len(posts) == 0 { // We have to do a fetch
			fetched, after2, err = rs.fetchPost(ctx, 100, after, subreddit)
			if err != nil {
				aggregateCh <- StreamResult{Error: err}
				time.Sleep(500 * time.Millisecond) // Don't spam reddit too much
				continue
			} else {
				after = after2
				posts = fetched
			}
		}
		// Otherwise, there's still posts to give
		aggregateCh <- StreamResult{
			Error:     nil,
			Post:      &posts[0], // Give one
			Subreddit: subreddit,
		}
		posts = posts[1:] // Remove one
	}
}

// fetchItem appends items to results chan, or, if there are no more items ("after is empty"), it returns a StreamError.
func (rs *RedditStreamer) fetchPost(ctx context.Context, count int64, after, subreddit string) ([]Post, string, error) {
	opts := &RequestOptions{
		After:     after,
		Count:     count,
		Sorting:   rs.opts.Sort,
		Timeframe: rs.opts.Timeframe,
		Subreddit: subreddit,
	}

	res, after, err := rs.client.Subreddit.GetPosts(ctx, opts)
	if err != nil {
		return nil, "", err
	}

	if len(res) == 0 {
		return nil, "", ErrStreamEOF
	}

	return res, after, nil
}
