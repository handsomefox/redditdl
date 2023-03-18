package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	clientTimeout       = time.Minute
	defaultBaseURL      = "https://reddit.com"
	defaultBaseVideoURL = "https://v.redd.it"
	defaultBaseImageURL = "https://i.redd.it"
)

// This is the client used to make requests in RedditStreamer.Stream()
type Client struct {
	Subreddit *SubredditService

	client *http.Client

	base    *url.URL
	imgbase *url.URL
	vidbase *url.URL
}

func (c *Client) WithTimeout(timeout time.Duration) *Client {
	c.client.Timeout = timeout
	return c
}

func (c *Client) WithBaseURL(u *url.URL) *Client {
	c.base = u
	return c
}

func (c *Client) WithBaseVideoURL(u *url.URL) *Client {
	c.vidbase = u
	return c
}

func (c *Client) WithBaseImageURL(u *url.URL) *Client {
	c.imgbase = u
	return c
}

type RequestOptions struct {
	After     string
	Count     int64
	Sorting   string
	Timeframe string
	Subreddit string
}

func (c *Client) Do(ctx context.Context, opts *RequestOptions, method string, body io.Reader) (*http.Response, error) {
	if opts == nil {
		return nil, fmt.Errorf("empty options")
	}
	url := c.optsURL(opts)

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "go:getter")

	return c.client.Do(req)
}

func (c *Client) GetURL(ctx context.Context, surl string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, surl, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "go:getter")

	return c.client.Do(req)
}

func (c *Client) GetImageByURL(ctx context.Context, surl string) (*http.Response, error) {
	u, err := url.Parse(surl)
	if err != nil {
		return nil, err
	}

	if c.imgbase.Host != defaultBaseImageURL {
		u.Host = c.imgbase.Host
	}

	return c.GetURL(ctx, u.String())
}

func (c *Client) GetVideoByURL(ctx context.Context, surl string) (*http.Response, error) {
	u, err := url.Parse(surl)
	if err != nil {
		return nil, err
	}

	if c.vidbase.Host != defaultBaseVideoURL {
		u.Host = c.vidbase.Host
	}

	return c.GetURL(ctx, u.String())
}

func (c *Client) optsURL(opts *RequestOptions) string {
	u := c.base.
		JoinPath("r").
		JoinPath(opts.Subreddit).
		JoinPath(opts.Sorting + ".json")

	values := u.Query()
	values.Add("after", opts.After)
	values.Add("limit", fmt.Sprint(opts.Count))
	values.Add("t", opts.Timeframe)

	u.RawQuery = values.Encode()

	return u.String()
}

func (c *Client) BaseURL() *url.URL {
	return c.base
}

func (c *Client) BaseImageURL() *url.URL {
	return c.imgbase
}

func (c *Client) BaseVideoURL() *url.URL {
	return c.vidbase
}

func DefaultClient() *Client {
	baseURL, _ := url.Parse(defaultBaseURL)
	basevidURL, _ := url.Parse(defaultBaseVideoURL)
	baseimgURL, _ := url.Parse(defaultBaseImageURL)
	c := &Client{
		client: &http.Client{
			Transport: &http.Transport{
				TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
			},
			Timeout: clientTimeout,
		},
		base:    baseURL,
		imgbase: baseimgURL,
		vidbase: basevidURL,
	}
	c.Subreddit = &SubredditService{
		client: c,
	}
	return c
}
