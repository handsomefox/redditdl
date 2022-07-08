package downloader

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
	Title     string      `json:"title"`
	Thumbnail string      `json:"thumbnail"`
	Preview   Preview     `json:"preview"`
	Media     interface{} `json:"media"`
	IsVideo   bool        `json:"is_video"`
}

type ImageData struct {
	URL    string `json:"url"`
	Width  int64  `json:"width"`
	Height int64  `json:"height"`
}

type Preview struct {
	Images []Image `json:"images"`
}

type Image struct {
	Source      ImageData   `json:"source"`
	Resolutions []ImageData `json:"resolutions"`
	ID          string      `json:"id"`
}
