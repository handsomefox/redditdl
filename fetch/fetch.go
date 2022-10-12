// fetch is a package for fetching posts from reddit.com
// and some utility-like functions.
package fetch

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/handsomefox/redditdl/configuration"
	"github.com/handsomefox/redditdl/fetch/api"
	"github.com/handsomefox/redditdl/files"
)

const (
	clientTimeout = time.Minute
)

var client = NewClient()

var ErrInvalidStatus = errors.New("unexpected status code in response")

// NewClient returns a pointer to http.Client configured to work with reddit.
func NewClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
		},
		Timeout: clientTimeout,
	}
}

// IsURL checks if the URL is valid.
func IsURL(str string) bool {
	u, err := url.ParseRequestURI(str)
	return err == nil && u.Host != "" && u.Scheme != ""
}

const fStr = "https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s"

// FormatURL formats the URL using the configuration.
func FormatURL(cfg *configuration.Config, after string) string {
	URL := fmt.Sprintf(fStr, cfg.Subreddit, cfg.Sorting, cfg.Count, cfg.Timeframe)
	if len(after) > 0 {
		URL = fmt.Sprintf("%s&after=%s&count=%d", URL, after, cfg.Count)
	}
	return URL
}

// File fetches data for a file from reddit's api and returns a *File.
func File(content *api.Content) (*files.File, error) {
	request, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, content.URL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, "couldn't create the request")
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, "couldn't perform the request")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStatus, http.StatusText(response.StatusCode))
	}

	extension := "jpg" // if we didn't manage to figure out the image extension, assume jpg
	if content.IsVideo {
		extension = "mp4" // if we didn't manage to figure out the video extension, assume mp4
	}

	// the URL path is usually equal to something like "randomid.extension",
	// this way we can get the actual file extension
	split := strings.Split(response.Request.URL.Path, ".")
	if len(split) == 2 {
		extension = split[1]
	}

	f := &files.File{
		Name:      content.Name,
		Extension: extension,
		Data:      nil,
	}

	_, err = io.Copy(f, response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, "couldn't copy response body")
	}

	return f, nil
}

// Posts fetches a json file from reddit containing information
// about the posts using the given configuration.
func Posts(path string) (*api.Posts, error) {
	request, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, path, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, "couldn't create the request")
	}
	request.Header.Add("User-Agent", "go:getter")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, "couldn't fetch posts from reddit")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStatus, http.StatusText(response.StatusCode))
	}

	posts := &api.Posts{}
	if err := json.NewDecoder(response.Body).Decode(posts); err != nil {
		return nil, fmt.Errorf("%w: %s", err, "couldn't decode posts")
	}

	return posts, nil
}
