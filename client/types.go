package client

import (
	"strings"
)

type ContentType byte

const (
	_ ContentType = iota
	ContentText
	ContentImage
	ContentVideo
)

// Content contains information which is required to filter by resolution,
// Content and store a video or an image.
type Content struct {
	Name          string
	URL           string
	Width, Height int
	Type          ContentType
}

func NewContent(p Post) *Content {
	if !p.Data.IsVideo {
		if len(p.Data.Preview.Images) == 0 {
			return &Content{Type: ContentText}
		}

		img := &p.Data.Preview.Images[0]
		return &Content{
			Name:   p.Data.Title,
			URL:    strings.ReplaceAll(img.Source.URL, "&amp;s", "&s"),
			Width:  img.Source.Width,
			Height: img.Source.Height,
			Type:   ContentImage,
		}
	} else {
		return &Content{
			Name:   p.Data.Title,
			URL:    strings.ReplaceAll(p.Data.Media.RedditVideo.ScrubberMediaURL, "&amp;s", "&s"),
			Width:  p.Data.Media.RedditVideo.Width,
			Height: p.Data.Media.RedditVideo.Height,
			Type:   ContentVideo,
		}
	}
}

type Posts struct {
	Data struct {
		After    string `json:"after"`
		Children []Post `json:"children"`
	} `json:"data"`
}

type Post struct {
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
