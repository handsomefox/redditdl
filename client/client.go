// client is a package that wraps net/http to provide access for some reddit api features
package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/handsomefox/redditdl/client/media"
)

var (
	ErrCreateRequest     = errors.New("error creating a request")
	ErrInvalidStatusCode = errors.New("invalid status code")
)

type Client struct {
	impl      *http.Client
	sorting   string
	timeframe string
}

func NewClient(sorting, timeframe string) *Client {
	const clientTimeout = time.Minute
	return &Client{
		impl: &http.Client{
			Transport: &http.Transport{
				TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
			},
			Timeout: clientTimeout,
		},
		sorting:   sorting,
		timeframe: timeframe,
	}
}

// Do wraps the (*http.Client).Do(), settings required headers before the request is done.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", "go:getter")
	resp, err := c.impl.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching from reddit: %w", err)
	}
	return resp, nil
}

// NewContent just gets the information from posts and returns it in a *Content struct.
func (c *Client) NewContent(ctx context.Context, p media.RedditPost) (*media.Content, error) {
	data, extension, err := c.GetFileDataAndExtension(ctx, p.URL())
	if err != nil {
		data.Close() // close if cannot create.
		return nil, err
	}
	cnt := &media.Content{
		ReadCloser:  data,
		Name:        p.Title(),
		URL:         p.URL(),
		Width:       p.Width(),
		Height:      p.Height(),
		Type:        p.Type(),
		Orientation: p.Orientation(),
		NSFW:        p.Data.Over18,
	}
	if extension == nil {
		switch cnt.Type {
		case media.ContentVideo:
			cnt.Extension = "mp4"
		case media.ContentImage:
			cnt.Extension = "jpg"
		default:
			cnt.Extension = ""
		}
	} else {
		cnt.Extension = *extension
	}
	return cnt, nil
}

// GetPostsContent returns a channel to which the posts will be sent to during fetching.
// The Channel will be closed after required count is reached, or if there is no more posts we can fetch.
func (c *Client) GetPostsContent(ctx context.Context, count int64, subreddit string) <-chan *media.Content {
	ch := make(chan *media.Content, 8)
	go func() {
		c.postsLoop(ctx, count, subreddit, ch)
		close(ch)
	}()
	return ch
}

// GetPostsContentSync does the same as GetPostsContent, but instead of returning a channel, returns a slice where
// all the results are stored.
func (c *Client) GetPostsContentSync(ctx context.Context, count int64, subreddit string) []*media.Content {
	s := make([]*media.Content, 0, count)
	contentCh := c.GetPostsContent(ctx, count, subreddit)
	for content := range contentCh {
		s = append(s, content)
	}
	return s
}

// GetFileDataAndExtension returns the file data and extension (if found).
func (c *Client) GetFileDataAndExtension(ctx context.Context, url string) (io.ReadCloser, *string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, nil, ErrCreateRequest
	}
	response, err := c.Do(request)
	if err != nil {
		return nil, nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("%w: %s", ErrInvalidStatusCode, http.StatusText(response.StatusCode))
	}
	// the URL path is usually equal to something like "randomid.extension",
	// this way we can get the actual file extension
	var extension *string
	split := strings.Split(response.Request.URL.Path, ".")
	if len(split) == 2 {
		extension = &split[1]
	}
	return response.Body, extension, nil
}

func (c *Client) postsLoop(ctx context.Context, count int64, subreddit string, ch chan<- *media.Content) {
	const sleepTime = 200 * time.Millisecond // this is enough to not get ratelimited
	var after string
	for i := int64(0); i < count; {
		posts, err := c.getPosts(ctx, formatURL(count, c.sorting, c.timeframe, subreddit, after))
		if err != nil {
			time.Sleep(sleepTime)
			continue
		}
		if len(posts.Data.Children) == 0 {
			return
		}
		for _, post := range posts.Data.Children {
			cnt, err := c.NewContent(ctx, post)
			if err != nil {
				continue
			}
			ch <- cnt
			i++
			if i == count {
				break
			}
		}
	}
}

// getPosts fetches a json file from reddit containing information
// about the posts using the given configuration.
func (c *Client) getPosts(ctx context.Context, url string) (*media.RedditPosts, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, ErrCreateRequest
	}

	response, err := c.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStatusCode, http.StatusText(response.StatusCode))
	}

	posts := &media.RedditPosts{}
	if err := json.NewDecoder(response.Body).Decode(posts); err != nil {
		return nil, fmt.Errorf("%w: %s", err, "couldn't decode posts")
	}

	return posts, nil
}

// formatURL formats the URL using the configuration.
func formatURL(count int64, sorting, timeframe, subreddit, after string) string {
	// fStr is the expected format for the request URL to reddit.com.
	const fStr = "https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s"
	URL := fmt.Sprintf(fStr, subreddit, sorting, count, timeframe)
	if len(after) > 0 {
		URL = fmt.Sprintf("%s&after=%s&count=%d", URL, after, count)
	}
	return URL
}
