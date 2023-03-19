package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPosts(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/r/wallpaper/", func(w http.ResponseWriter, r *http.Request) {
		r.Header.Add("content-type", "application/json")

		b, err := os.ReadFile("testdata/sample.json")
		assert.NoError(t, err)

		var ps Posts
		assert.NoError(t, json.Unmarshal(b, &ps))
		assert.NoError(t, json.NewEncoder(w).Encode(ps))
	})
	server := httptest.NewServer(mux)

	urlA, err := url.Parse(server.URL)
	assert.NoError(t, err)

	client := DefaultClient().WithBaseURL(urlA)
	opts := &RequestOptions{
		After:     "",
		Count:     10,
		Sorting:   "best",
		Timeframe: "all",
		Subreddit: "wallpaper",
	}

	posts, _, err := client.Subreddit.GetPosts(context.TODO(), opts)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(posts), "unexpected decoded response")
	assert.Equal(t, "Staring into the woods [3840x2160]", posts[0].Title(), "unexpected decoded title")
}

func TestGetItems(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/r/wallpaper/", func(w http.ResponseWriter, r *http.Request) {
		r.Header.Add("content-type", "application/json")

		b, err := os.ReadFile("testdata/sample.json")
		assert.NoError(t, err)

		var ps Posts
		assert.NoError(t, json.Unmarshal(b, &ps))
		assert.NoError(t, json.NewEncoder(w).Encode(ps))
	})
	server := httptest.NewServer(mux)

	urlA, err := url.Parse(server.URL)
	assert.NoError(t, err)

	client := DefaultClient().WithBaseURL(urlA)
	opts := &RequestOptions{
		After:     "",
		Count:     10,
		Sorting:   "best",
		Timeframe: "all",
		Subreddit: "wallpaper",
	}

	items, _, err := client.Subreddit.GetItems(context.TODO(), opts)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(items), "unexpected length of items")

	item := items[0]

	assert.Equal(t, "05sk8tzriboa1", item.Name, "unexpected name")
	assert.Equal(t, "png", item.Extension, "unexpected extension")
	assert.Equal(t, 3840, item.Height, "unexpected height")
	assert.Equal(t, 6656, item.Width, "unexpected width")
	assert.Equal(t, false, item.IsOver18, "unexpected IsOver18")
	assert.Equal(t, "landscape", item.Orientation, "unexpected orientation")
	assert.Equal(t, "https://i.redd.it/05sk8tzriboa1.png", item.URL, "unexpected url")
	assert.Equal(t, "image", item.Type, "unexpected type")
}
