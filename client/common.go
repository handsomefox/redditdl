package client

import (
	"fmt"
)

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
