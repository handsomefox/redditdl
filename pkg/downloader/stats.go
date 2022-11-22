package downloader

import (
	"sync"
	"sync/atomic"
)

// Stats is the struct that describes the statistics
// for the current downloader.
type Stats struct {
	st *collectedStats
}

func (s *Stats) Errors() []error {
	return s.st.errors
}

func (s *Stats) HasErrors() bool {
	s.st.mu.Lock()
	defer s.st.mu.Unlock()
	return len(s.st.errors) != 0
}

func (s *Stats) Finished() int64 {
	return s.st.finished.Load()
}

func (s *Stats) Failed() int64 {
	return s.st.failed.Load()
}

func (s *Stats) Queued() int64 {
	return s.st.queued.Load()
}

// collectedStats is the underlying data for Stats that are returned by the Downloader.
type collectedStats struct {
	mu       sync.Mutex
	queued   atomic.Int64
	finished atomic.Int64
	failed   atomic.Int64
	errors   []error
}

// append is used to append errors to stats.
func (s *collectedStats) append(err error) {
	s.mu.Lock()
	s.errors = append(s.errors, err)
	s.mu.Unlock()
}

// appendIncr appends the error and increments Failed count.
func (s *collectedStats) appendIncr(err error) {
	s.append(err)
	s.failed.Add(1)
}
