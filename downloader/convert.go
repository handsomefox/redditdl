package downloader

import (
	"strings"

	"github.com/handsomefox/redditdl/configuration"
	"github.com/handsomefox/redditdl/structs"
)

// Converts posts to content depending on the configuration, leaving only the required types of media in.
func postsToContent(typ configuration.MediaType, children []structs.Child) []structs.Content {
	data := make([]structs.Content, 0, len(children))
	for i := 0; i < len(children); i++ {
		value := &children[i].Data

		if !value.IsVideo && typ == configuration.MediaAny || typ == configuration.MediaImages {
			for _, img := range value.Preview.Images {
				data = append(data, structs.Content{
					Name:    value.Title,
					URL:     strings.ReplaceAll(img.Source.URL, "&amp;s", "&s"),
					Width:   img.Source.Width,
					Height:  img.Source.Height,
					IsVideo: false,
				})
			}
		} else if value.IsVideo && typ == configuration.MediaAny || typ == configuration.MediaVideos {
			data = append(data, structs.Content{
				Name:    value.Title,
				URL:     strings.ReplaceAll(value.Media.RedditVideo.ScrubberMediaURL, "&amp;s", "&s"),
				Width:   value.Media.RedditVideo.Width,
				Height:  value.Media.RedditVideo.Height,
				IsVideo: true,
			})
		}
	}

	return data
}
