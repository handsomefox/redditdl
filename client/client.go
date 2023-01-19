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
)

var (
	ErrCreateRequest     = errors.New("error creating a request")
	ErrInvalidStatusCode = errors.New("invalid status code")
)

type Client struct {
	impl *http.Client
}

func NewClient() *Client {
	const clientTimeout = time.Minute
	return &Client{
		impl: &http.Client{
			Transport: &http.Transport{
				TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
			},
			Timeout: clientTimeout,
		},
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

// GetPosts returns a channel to which the posts will be sent to during fetching.
// The Channel will be closed after required count is reached, or if there is no more posts we can fetch.
func (c *Client) GetPosts(ctx context.Context, cfg *Config) <-chan Post {
	ch := make(chan Post, 8)
	go func() {
		c.postsLoop(ctx, cfg, ch)
		close(ch)
	}()
	return ch
}

// GetPostsSync does the same as GetPosts, but instead of returning a channel, returns a slice where
// all the results are stored.
func (c *Client) GetPostsSync(ctx context.Context, cfg *Config) []Post {
	posts := make([]Post, 0, cfg.Count)
	postCh := c.GetPosts(ctx, cfg)
	for post := range postCh {
		posts = append(posts, post)
	}
	return posts
}

// GetFile returns the file data and extension (if found).
func (c *Client) GetFile(ctx context.Context, url string) (b []byte, extension *string, err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, nil, ErrCreateRequest
	}

	response, err := c.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("%w: %s", ErrInvalidStatusCode, http.StatusText(response.StatusCode))
	}

	// the URL path is usually equal to something like "randomid.extension",
	// this way we can get the actual file extension
	split := strings.Split(response.Request.URL.Path, ".")
	if len(split) == 2 {
		extension = &split[1]
	}

	b, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s", err, "couldn't read response body")
	}

	return
}

func (c *Client) postsLoop(ctx context.Context, cfg *Config, ch chan<- Post) {
	const sleepTime = 200 * time.Millisecond // this is enough to not get ratelimited
	var after string
	for count := int64(0); count < cfg.Count; {
		posts, err := c.getPosts(ctx, formatURL(cfg, after))
		if err != nil {
			time.Sleep(sleepTime)
			continue
		}
		for _, post := range posts.Data.Children {
			ch <- post
			count++
			if count == cfg.Count {
				break
			}
		}
	}
}

// getPosts fetches a json file from reddit containing information
// about the posts using the given configuration.
func (c *Client) getPosts(ctx context.Context, url string) (*Posts, error) {
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

	posts := &Posts{}
	if err := json.NewDecoder(response.Body).Decode(posts); err != nil {
		return nil, fmt.Errorf("%w: %s", err, "couldn't decode posts")
	}

	return posts, nil
}

// formatURL formats the URL using the configuration.
func formatURL(cfg *Config, after string) string {
	// fStr is the expected format for the request URL to reddit.com.
	const fStr = "https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s"
	URL := fmt.Sprintf(fStr, cfg.Subreddit, cfg.Sorting, cfg.Count, cfg.Timeframe)
	if len(after) > 0 {
		URL = fmt.Sprintf("%s&after=%s&count=%d", URL, after, cfg.Count)
	}
	return URL
}
