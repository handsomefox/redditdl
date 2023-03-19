package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type SubredditService struct {
	client *Client
}

type Item struct {
	Name        string
	Extension   string
	URL         string
	Orientation string
	Type        string

	Bytes []byte

	Width  int
	Height int

	IsOver18 bool
}

func (s *SubredditService) GetPosts(ctx context.Context, opts *RequestOptions) ([]Post, string, error) {
	res, err := s.client.Do(ctx, opts, http.MethodGet, http.NoBody)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	var ps Posts
	if err := json.NewDecoder(res.Body).Decode(&ps); err != nil {
		return nil, "", err
	}

	return ps.Data.Children, ps.Data.After, nil
}

// GetItems return a slice of items, "after" string (consult reddit api), or an error.
func (s *SubredditService) GetItems(ctx context.Context, opts *RequestOptions) ([]Item, string, error) {
	posts, after, err := s.GetPosts(ctx, opts)
	if err != nil {
		return nil, after, err
	}

	items := make([]Item, 0, len(posts))
	for _, p := range posts {
		p := p
		item, err := s.PostToItem(ctx, &p)
		if err != nil {
			return nil, after, err
		}
		items = append(items, *item)
	}

	return items, after, nil
}

func (s *SubredditService) PostToItem(ctx context.Context, p *Post) (*Item, error) {
	res, err := s.client.GetURL(ctx, p.URL())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	item := Item{
		Bytes:       b,
		Name:        p.Title(),
		Extension:   "",
		URL:         p.URL(),
		Orientation: p.Orientation(),
		Type:        p.Type(),
		Width:       p.Width(),
		Height:      p.Height(),
		IsOver18:    p.Data.Over18,
	}

	switch p.Type() {
	case "video":
		item.Extension = "mp4"
	case "image":
		item.Extension = "jpg"
	case "text":
		item.Extension = "txt"
	default:
		item.Extension = "bin"
	}

	split := strings.Split(res.Request.URL.Path, ".")
	if len(split) == 2 {
		item.Extension = split[1]
		item.Name = split[0][1:] // Skip the forward slash at the start
	}

	return &item, nil
}
