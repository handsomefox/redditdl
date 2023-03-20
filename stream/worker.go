package stream

import (
	"context"
	"errors"

	"github.com/handsomefox/redditdl/api"
)

var ErrWorkerEOF = errors.New("worker reached the end of it's stream")

type Worker struct {
	client *api.Client
	opts   *Options

	outCh     chan *api.Post
	subreddit string
	after     string

	// Store the items here, refetch only if empty
	currentItems []api.Post
}

// Run loops over the provided channel, each receive triggers it to send an item to
// the output channel.
// When the Run() returns, it means that the worker can no longer fetch any items.
func (w *Worker) Run(listenCh <-chan struct{}, terminate <-chan struct{}) struct{} {
	ctx := context.Background()

	for {
		select {
		case <-listenCh:
			if len(w.currentItems) == 0 { // if there are no items
				err := w.fetchItems(ctx) // fetch the items
				if err != nil {
					if !errors.Is(err, ErrWorkerEOF) {
						w.outCh <- nil
						continue
					} else {
						// There are no more items to fetch, report that we're done.
						return struct{}{}
					}
				}
			}
			// We can yield one item to the stream output.
			w.outCh <- &w.currentItems[0]
			w.currentItems = w.currentItems[1:]
		case <-terminate:
			return struct{}{}
		}
	}
}

func (w *Worker) fetchItems(ctx context.Context) error {
	opts := &api.RequestOptions{
		After:     w.after,
		Count:     100,
		Sorting:   w.opts.Sort,
		Timeframe: w.opts.Timeframe,
		Subreddit: w.subreddit,
	}

	res, after, err := w.client.Subreddit.GetPosts(ctx, opts)
	if err != nil {
		return err
	}

	if len(res) == 0 {
		return ErrWorkerEOF
	}

	w.after = after
	w.currentItems = res

	return nil
}

func (w *Worker) tryPerformInitialFetch() {
	ctx := context.Background()
	err := w.fetchItems(ctx)
	if err != nil {
		// Do nothing,
	}
}
