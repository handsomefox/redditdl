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
	Title     string      `json:"title"`
	Thumbnail string      `json:"thumbnail"`
	Preview   preview     `json:"preview"`
	Media     interface{} `json:"media"`
	IsVideo   bool        `json:"is_video"`
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
