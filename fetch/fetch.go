package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/handsomefox/redditdl/configuration"
	"github.com/handsomefox/redditdl/structs"
	"github.com/handsomefox/redditdl/utils"
)

// FormatURL formats the URL using the configration.
func FormatURL(cfg *configuration.Data, after string) string {
	URL := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d&t=%s",
		cfg.Subreddit, cfg.Sorting, cfg.Count, cfg.Timeframe)

	if len(after) > 0 {
		URL = fmt.Sprintf("%s&after=%s&count=%d", URL, after, cfg.Count)
	}

	return URL
}

func File(content *structs.Content) (*structs.File, error) {
	client := utils.CreateClient()

	request, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, content.URL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("error creating a request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error making a request: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code in response: %v", http.StatusText(response.StatusCode))
	}

	var extension string
	if content.IsVideo {
		extension = "mp4" // if we didn't manage to figure out the video extension, assume mp4
	} else {
		extension = "jpg" // if we didn't manage to figure out the image extension, assume jpg
	}

	// the URL path is usually equal to something like "randomid.extension",
	// this way we can get the actual file extension
	split := strings.Split(response.Request.URL.Path, ".")
	if len(split) == 2 {
		extension = split[1]
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	response.Body.Close()

	return &structs.File{
		Data:      b,
		Name:      content.Name,
		Extension: extension,
	}, nil
}

// Posts fetches a json file from reddit containing information
// about the posts using the given configuration.
func Posts(url string) (*structs.Posts, error) {
	client := utils.CreateClient()

	request, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("error creating a request: %w", err)
	}
	request.Header.Add("User-Agent", "go:getter")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetching from reddit failed, error: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %v", "unexpected status in response", http.StatusText(response.StatusCode))
	}

	posts := &structs.Posts{}
	if err := json.NewDecoder(response.Body).Decode(posts); err != nil {
		return nil, fmt.Errorf("error decoding posts: %w", err)
	}

	return posts, nil
}
