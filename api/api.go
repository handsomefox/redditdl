// package api contains the code required to make requests, fetch/stream posts and other things from reddit.
// This package is heavily inspired by https://github.com/vartanbeno/go-reddit/, you should check it out.
// But, this package is simpler, smaller and more specialized to the use-case required by me.
package api

import (
	"strings"
)

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
		Title     string `json:"title"`
		URL       string `json:"url"`
		PostHint  string `json:"post_hint"`
		Subreddit string `json:"subreddit"`
		Preview   struct {
			Images []Image `json:"images"`
		}
		Over18  bool `json:"over_18"`
		IsVideo bool `json:"is_video"`
	} `json:"data"`
}

type Video struct {
	ScrubberMediaURL string `json:"scrubber_media_url"`
	Height           int    `json:"height"`
	Width            int    `json:"width"`
}

type Image struct {
	Source *struct {
		URL    string `json:"url"`
		Height int    `json:"height"`
		Width  int    `json:"width"`
	} `json:"source"`
}

// Width returns either the width of video/image, or 0.
func (p *Post) Width() int {
	if p.Data.IsVideo {
		return p.Data.Media.RedditVideo.Width
	}
	if len(p.Data.Preview.Images) != 0 {
		return p.Data.Preview.Images[0].Source.Width
	}
	return 0
}

// Height returns either the height of video/image, or 0.
func (p *Post) Height() int {
	if p.Data.IsVideo {
		return p.Data.Media.RedditVideo.Height
	}
	if len(p.Data.Preview.Images) != 0 {
		return p.Data.Preview.Images[0].Source.Height
	}
	return 0
}

// Dimensions calls p.Width() and p.Height(), then return.
func (p *Post) Dimensions() (w, h int) {
	return p.Width(), p.Height()
}

// Orientation calculates the orientation of video/image.
func (p *Post) Orientation() string {
	width, height := p.Dimensions()
	if width > height {
		return "landscape"
	}
	if height > width {
		return "portrait"
	}
	return "rect"
}

// Title is just the post title.
func (p *Post) Title() string {
	return p.Data.Title
}

// URL returns an automatically formatted url of the post.
func (p *Post) URL() string {
	if p.Data.IsVideo {
		return strings.ReplaceAll(p.Data.Media.RedditVideo.ScrubberMediaURL, "&amp;", "&")
	}
	return strings.ReplaceAll(p.Data.URL, "&amp;", "&")
}

func (p *Post) Type() string {
	return p.Data.PostHint
}
