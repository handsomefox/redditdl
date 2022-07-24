package downloader

type posts struct {
	Kind string    `json:"kind"`
	Data postsData `json:"data"`
}

type postsData struct {
	After    string  `json:"after"`
	Children []child `json:"children"`
}

type child struct {
	Kind string    `json:"kind"`
	Data childData `json:"data"`
}

type childData struct {
	Title     string  `json:"title"`
	Thumbnail string  `json:"thumbnail"`
	Preview   preview `json:"preview"`
	Media     struct {
		RedditVideo RedditVideo `json:"reddit_video"`
	} `json:"media"`
	IsVideo bool `json:"is_video"`
}

type RedditVideo struct {
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

type imageData struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type preview struct {
	Images []image `json:"images"`
}

type image struct {
	ID          string      `json:"id"`
	Resolutions []imageData `json:"resolutions"`
	Source      imageData   `json:"source"`
}
