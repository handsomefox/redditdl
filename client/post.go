package client

import (
	"errors"
	"strings"
)

var ErrNoImages = errors.New("no images in the post")

// RedditPosts is returned by reddit api, and contains
// a collection of posts (or no posts, if there are no more.).
type RedditPosts struct {
	Data struct {
		After    string       `json:"after"`
		Children []RedditPost `json:"children"`
	} `json:"data"`
}

// RedditPost is returned by reddit api.
// It describes all the data the post contains.
// Most of the information is useless for our use case,
// that's why it's removed from the struct and is not deserialized.
type RedditPost struct {
	Data struct {
		Media struct {
			RedditVideo *struct {
				ScrubberMediaURL string `json:"scrubber_media_url"`
				Height           int    `json:"height"`
				Width            int    `json:"width"`
			} `json:"reddit_video"`
		} `json:"media"`
		Title   string `json:"title"`
		URL     string `json:"url"`
		Preview struct {
			Images []Image `json:"images"`
		}
		Over18  bool `json:"over_18"`
		IsVideo bool `json:"is_video"`
	} `json:"data"`
}

// Width returns either the width of video/image, or 0.
func (p *RedditPost) Width() int {
	switch p.Type() {
	case ContentVideo:
		return p.Data.Media.RedditVideo.Width
	case ContentImage:
		return p.Data.Preview.Images[0].Source.Width
	}
	return 0
}

// Height returns either the height of video/image, or 0.
func (p *RedditPost) Height() int {
	switch p.Type() {
	case ContentVideo:
		return p.Data.Media.RedditVideo.Height
	case ContentImage:
		return p.Data.Preview.Images[0].Source.Height
	}
	return 0
}

// Dimensions calls p.Width() and p.Height(), then return.
func (p *RedditPost) Dimensions() (w, h int) {
	return p.Width(), p.Height()
}

// Orientation calculates the orientation of video/image.
func (p *RedditPost) Orientation() Orientation {
	width, height := p.Dimensions()
	if p.Type() == ContentText {
		return OrientationAny
	}
	if width > height {
		return OrientationLandscape
	}
	if height > width {
		return OrientationPortrait
	}
	return OrientationAny
}

// Title is just the post title.
func (p *RedditPost) Title() string {
	return p.Data.Title
}

// Type determines the type of the post.
func (p *RedditPost) Type() ContentType {
	if p.Data.IsVideo {
		return ContentVideo
	} else if len(p.Data.Preview.Images) > 0 {
		return ContentImage
	}
	return ContentText
}

// URL returns an automatically formatted url of the post.
func (p *RedditPost) URL() string {
	switch p.Type() {
	case ContentImage:
		img, err := p.PreviewImage()
		if err != nil {
			return strings.ReplaceAll(p.Data.URL, "&amp;", "&")
		}
		return strings.ReplaceAll(img.Source.URL, "&amp;", "&")
	case ContentVideo:
		return strings.ReplaceAll(p.Data.Media.RedditVideo.ScrubberMediaURL, "&amp;", "&")
	default:
		return strings.ReplaceAll(p.Data.URL, "&amp;", "&")
	}
}

// PreviewImage returns the first image in the Preview.Images slice
// or an ErrNoImages if there are no images to get.
func (p *RedditPost) PreviewImage() (*Image, error) {
	if len(p.Data.Preview.Images) == 0 {
		return nil, ErrNoImages
	}
	return &p.Data.Preview.Images[0], nil
}

// Image is part of RedditPost struct.
// The reason why it is a separate struct is for usage
// in other functions as a type.
type Image struct {
	Source *struct {
		URL    string `json:"url"`
		Height int    `json:"height"`
		Width  int    `json:"width"`
	} `json:"source"`
}
