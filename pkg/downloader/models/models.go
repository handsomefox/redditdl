// Package api is a package containing data structures that mimic reddit.com responses.
package models

// Content contains information which is required to filter by resolution,
// Content and store a video or an image.
type Content struct {
	Name          string
	URL           string
	Width, Height int
	IsVideo       bool
}

// Everything below mimics reddit's responses.

type Posts struct {
	Data struct {
		After    string  `json:"after"`
		Children []Child `json:"children"`
	} `json:"data"`
}

type Child struct {
	Data struct {
		Media struct {
			RedditVideo *Video `json:"reddit_video"`
		} `json:"media"`
		Title   string `json:"title"`
		Preview struct {
			Images []struct {
				Source *ImageData `json:"source"`
			} `json:"images"`
		}
		IsVideo bool `json:"is_video"`
	} `json:"data"`
}

type Video struct {
	ScrubberMediaURL string `json:"scrubber_media_url"`
	Height           int    `json:"height"`
	Width            int    `json:"width"`
}

type ImageData struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}
