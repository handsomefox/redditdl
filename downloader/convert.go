package downloader

import (
	"strings"

	"github.com/handsomefox/redditdl/configuration"
	"github.com/handsomefox/redditdl/structs"
)

// Converts posts to content depending on the configuration, leaving only the required types of media in.
func postsToContent(terminate chan uint8, out chan structs.Content, typ configuration.MediaType, c []structs.Child) {
	for i := 0; i < len(c); i++ {
		select {
		case <-terminate:
			return
		default:
			makeContent(out, &c[i], typ)
		}
	}
}

// makeContent is a helper for postsToContent function.
func makeContent(outChan chan structs.Content, v *structs.Child, typ configuration.MediaType) {
	switch typ {
	case configuration.MediaAny:
		switch v.Data.IsVideo {
		case true:
			makeVideo(outChan, &v.Data)
		case false:
			makeImages(outChan, &v.Data)
		}
	case configuration.MediaImages:
		if !v.Data.IsVideo {
			makeImages(outChan, &v.Data)
		}
	case configuration.MediaVideos:
		if v.Data.IsVideo {
			makeVideo(outChan, &v.Data)
		}
	}
}

// makeImages is a helper for postsToContent function.
func makeImages(outChan chan structs.Content, data *structs.ChildData) {
	for _, img := range data.Preview.Images {
		outChan <- structs.Content{
			Name:    data.Title,
			URL:     strings.ReplaceAll(img.Source.URL, "&amp;s", "&s"),
			Width:   img.Source.Width,
			Height:  img.Source.Height,
			IsVideo: false,
		}
	}
}

// makeVideo is a helper for postsToContent function.
func makeVideo(outChan chan structs.Content, data *structs.ChildData) {
	outChan <- structs.Content{
		Name:    data.Title,
		URL:     strings.ReplaceAll(data.Media.RedditVideo.ScrubberMediaURL, "&amp;s", "&s"),
		Width:   data.Media.RedditVideo.Width,
		Height:  data.Media.RedditVideo.Height,
		IsVideo: true,
	}
}
