package structs

// Files is the structure that is saved to disk later.
type File struct {
	Name, Extension string
	Data            []byte
}

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
	Kind string    `json:"kind"`
	Data PostsData `json:"data"`
}

type PostsData struct {
	After    string  `json:"after"`
	Children []Child `json:"children"`
}

type Child struct {
	Kind string    `json:"kind"`
	Data ChildData `json:"data"`
}

type ChildData struct {
	Title     string  `json:"title"`
	Thumbnail string  `json:"thumbnail"`
	Preview   Preview `json:"preview"`
	Media     struct {
		RedditVideo Video `json:"reddit_video"`
	} `json:"media"`
	IsVideo bool `json:"is_video"`
}

type Video struct {
	FallbackURL       string `json:"fallback_url"`
	TranscodingStatus string `json:"transcoding_status"`
	ScrubberMediaURL  string `json:"scrubber_media_url"`
	DashURL           string `json:"dash_url"`
	HlsURL            string `json:"hls_url"`
	BitrateKbps       int    `json:"bitrate_kbps"`
	Height            int    `json:"height"`
	Duration          int    `json:"duration"`
	Width             int    `json:"width"`
	IsGif             bool   `json:"is_gif"`
}

type ImageData struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Preview struct {
	Images []Image `json:"images"`
}

type Image struct {
	ID          string      `json:"id"`
	Resolutions []ImageData `json:"resolutions"`
	Source      ImageData   `json:"source"`
}
