// Package fetch is a package for fetching posts from reddit.com
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

	"github.com/handsomefox/redditdl/pkg/downloader/config"
	"github.com/handsomefox/redditdl/pkg/downloader/models"
	"github.com/handsomefox/redditdl/pkg/files"
)

const (
	// clientTimetout is the timeout used in
	// 	client := NewClient()
	clientTimeout = time.Minute
)

// ErrInvalidStatus is an error returned by package fetch
// in cases when the response status code was not expected.
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

// IsValidURL checks if the URL is valid.
//
// Example:
//
//	fmt.Println(fetch.IsValidURL("www.google.com"))
//	Output: true
//
// Invalid example:
//
//	fmt.Println(fetch.IsValidURL("google.com"))
//	Output: false
func IsValidURL(str string) bool {
	u, err := url.ParseRequestURI(str)
	return err == nil && u.Host != "" && u.Scheme != ""
}

// fStr is the expected format for the request URL to reddit.com.
const fStr = "https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s"

// FormatURL formats the URL using the configuration.
func FormatURL(cfg *config.Config, after string) string {
	URL := fmt.Sprintf(fStr, cfg.Subreddit, cfg.Sorting, cfg.Count, cfg.Timeframe)
	if len(after) > 0 {
		URL = fmt.Sprintf("%s&after=%s&count=%d", URL, after, cfg.Count)
	}
	return URL
}

// File fetches data for a file from reddit's api and returns a *File.
func File(content *models.Content) (*files.File, error) {
	client := NewClient()
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

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, "couldn't read response body")
	}

	return files.New(content.Name, extension, b), nil
}

// Posts fetches a json file from reddit containing information
// about the posts using the given configuration.
func Posts(path string) (*models.Posts, error) {
	client := NewClient()
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

	posts := &models.Posts{}
	if err := json.NewDecoder(response.Body).Decode(posts); err != nil {
		return nil, fmt.Errorf("%w: %s", err, "couldn't decode posts")
	}

	return posts, nil
}
