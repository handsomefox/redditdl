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
		RedditVideo struct {
			BitrateKbps       int    `json:"bitrate_kbps"`
			FallbackURL       string `json:"fallback_url"`
			Height            int    `json:"height"`
			Width             int    `json:"width"`
			ScrubberMediaURL  string `json:"scrubber_media_url"`
			DashURL           string `json:"dash_url"`
			Duration          int    `json:"duration"`
			HlsURL            string `json:"hls_url"`
			IsGif             bool   `json:"is_gif"`
			TranscodingStatus string `json:"transcoding_status"`
		} `json:"reddit_video"`
	} `json:"media"`
	IsVideo bool `json:"is_video"`
}

type imageData struct {
	URL    string `json:"url"`
	Width  int64  `json:"width"`
	Height int64  `json:"height"`
}

type preview struct {
	Images []image `json:"images"`
}

type image struct {
	Source      imageData   `json:"source"`
	Resolutions []imageData `json:"resolutions"`
	ID          string      `json:"id"`
}
