package stream

import (
	"fmt"
	"sync/atomic"

	"github.com/handsomefox/redditdl/api"
)

type Options struct {
	ContentType string
	Sort        string
	Timeframe   string
	Subreddits  []string
	ShowNSFW    bool
}

type Stream struct {
	client *api.Client

	consumerCh chan *api.Post
	continueCh chan struct{}

	workers []Worker

	workersDone atomic.Int32
	completed   atomic.Bool

	terminates []chan struct{}

	opts Options
}

func New(client *api.Client, options Options, bufferSize int) (*Stream, error) {
	if len(options.Subreddits) == 0 {
		return nil, fmt.Errorf("empty subreddits provided")
	}

	s := &Stream{
		client:      client,
		consumerCh:  make(chan *api.Post, bufferSize),
		continueCh:  make(chan struct{}, bufferSize),
		workers:     make([]Worker, 0, len(options.Subreddits)),
		workersDone: atomic.Int32{},
		completed:   atomic.Bool{},
		terminates:  nil,
		opts:        options,
	}

	for i := 0; i < len(s.opts.Subreddits); i++ {
		s.workers = append(s.workers, Worker{
			client:       s.client,
			opts:         &s.opts,
			outCh:        s.consumerCh,
			subreddit:    s.opts.Subreddits[i],
			currentItems: nil,
		})
	}

	return s, nil
}

// Start returns the output channel.
// The value in the output channel may be nil, if the fetch failed.
func (s *Stream) Start() (<-chan *api.Post, error) {
	go s.spinupWorkers()
	return s.consumerCh, nil
}

// End reports to the stream that it has to end.
func (s *Stream) Close() {
	for i := 0; i < len(s.terminates); i++ {
		s.terminates[i] <- struct{}{}
	}
	s.completed.Store(true)
	close(s.continueCh)
	close(s.consumerCh)
}

// Continue reports to the stream that it has to fetch an item again.
// Returns whether the Stream was completely finished.
func (s *Stream) Continue() bool {
	s.continueCh <- struct{}{}
	return s.Done()
}

// Done return whether the Stream was completely finished.
func (s *Stream) Done() bool {
	if s.workersDone.Load() >= int32(len(s.opts.Subreddits)) {
		s.completed.Store(true)
	}
	return s.completed.Load()
}

func (s *Stream) spinupWorkers() {
	for i := 0; i < len(s.workers); i++ {
		i := i
		terminate := make(chan struct{})
		s.terminates = append(s.terminates, terminate)
		// This improves performance if there's multiple subreddits
		s.workers[i].tryPerformInitialFetch()
		go func() {
			_ = s.workers[i].Run(s.continueCh, terminate)
			s.workersDone.Add(1)
		}()
	}
}
