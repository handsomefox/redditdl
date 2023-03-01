package media

import (
	"io"
)

// ContentType describes possible media types of the Content struct.
// It can't be anything other than the 3 described options.
// Even if the post only contains a title, it will still be described as Text.
type ContentType byte

const (
	_ ContentType = iota
	ContentText
	ContentImage
	ContentVideo
)

// Orientation describes possible content orientations
// Any is used in cases where width matches the height, meaning it's a square.
// It is also used if the content contains no media and is only text.
type Orientation byte

const (
	_ Orientation = iota
	OrientationLandscape
	OrientationPortrait
	OrientationAny
)

// Content is the collection of useful information from the RedditPost in a more usable format.
// It embeds an io.ReadCloser, because the underlying data usually comes from a URL, meaning that
// the received body must be closed at some point. The easiest way to do it, is make the caller close it.
// Call Close() after usage.
type Content struct {
	io.ReadCloser
	Name          string
	Extension     string
	URL           string
	Width, Height int
	Type          ContentType
	Orientation   Orientation
	NSFW          bool
}
