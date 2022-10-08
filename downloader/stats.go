package downloader

import (
	"sync"
	"sync/atomic"
)

var _ Stats = &stats{}

// Stats is the interface that describes the statistics
// for the current downloader
type Stats interface {
	Errors() []error
	HasErrors() bool
	Finished() int64
	Failed() int64
	Queued() int64
}

// stats is the struct containing statistics for the download.
// implements Stats interface
// It may or may not be extended with additional data later.
type stats struct {
	errors []error
	mu     sync.Mutex

	queued   atomic.Int64
	finished atomic.Int64
	failed   atomic.Int64
}

func (s *stats) Errors() []error {
	return s.errors
}

func (s *stats) HasErrors() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.errors) != 0
}

func (s *stats) Finished() int64 {
	return s.finished.Load()
}

func (s *stats) Failed() int64 {
	return s.failed.Load()
}

func (s *stats) Queued() int64 {
	return s.queued.Load()
}

// append is used to append errors to Stats.
func (s *stats) append(err error) {
	s.mu.Lock()
	s.errors = append(s.errors, err)
	s.mu.Unlock()
}

// appendIncr appends the error and increments Failed count.
func (s *stats) appendIncr(err error) {
	s.append(err)
	s.failed.Add(1)
}
