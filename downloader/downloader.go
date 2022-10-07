package downloader

import (
	"sync"
	"sync/atomic"

	"github.com/handsomefox/redditdl/configuration"
	"github.com/handsomefox/redditdl/filter"
	"github.com/handsomefox/redditdl/logging"
)

// Downloader is a single-method interface which takes in configuration and a list of filters
// and returns statistics after it is done downloading all the files.
type Downloader interface {
	Download() *Stats
}

// New returns a new Downloader instance with the specified configuration.
func New(config *configuration.Data, filters ...filter.Filter) Downloader {
	return &downloader{
		Config:  config,
		Logger:  logging.GetLogger(config.Verbose),
		Stats:   &Stats{},
		Filters: filters,
	}
}

// Stats is the struct containing statistics for the download.
// It may or may not be extended with additional data later.
type Stats struct {
	Errors []error

	Queued   atomic.Int64
	Finished atomic.Int64
	Failed   atomic.Int64

	mu sync.Mutex
}

// append is used to append errors to Stats.
func (s *Stats) append(err error) {
	s.mu.Lock()
	s.Errors = append(s.Errors, err)
	s.mu.Unlock()
}

// appendIncr appends the error and increments Failed count.
func (s *Stats) appendIncr(err error) {
	s.append(err)
	s.Failed.Add(1)
}

// HasErrors returns whether the errors slice is non-empty.
func (s *Stats) HasErrors() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.Errors) != 0
}
