package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

func TestGetPosts(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/r/wallpaper/", func(w http.ResponseWriter, r *http.Request) {
		r.Header.Add("content-type", "application/json")
		b, err := os.ReadFile("testdata/sample.json")
		if err != nil {
			t.Fatal(err)
		}
		var ps Posts
		if err := json.Unmarshal(b, &ps); err != nil {
			t.Fatal(err)
		}
		if err := json.NewEncoder(w).Encode(ps); err != nil {
			t.Fatal(err)
		}
	})
	server := httptest.NewServer(mux)

	urlA, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := DefaultClient().WithBaseURL(urlA)
	opts := &RequestOptions{
		After:     "",
		Count:     10,
		Sorting:   "best",
		Timeframe: "all",
		Subreddit: "wallpaper",
	}

	posts, _, err := client.Subreddit.GetPosts(context.TODO(), opts)
	if err != nil {
		t.Fatal(err)
	}

	if len(posts) != 1 {
		t.Fatal("unexpected decoded response")
	}

	if posts[0].Title() != "Staring into the woods [3840x2160]" {
		t.Fatal("undexpected decoded title")
	}
}

func TestGetItems(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/r/wallpaper/", func(w http.ResponseWriter, r *http.Request) {
		r.Header.Add("content-type", "application/json")
		b, err := os.ReadFile("testdata/sample.json")
		if err != nil {
			t.Fatal(err)
		}
		var ps Posts
		if err := json.Unmarshal(b, &ps); err != nil {
			t.Fatal(err)
		}
		if err := json.NewEncoder(w).Encode(ps); err != nil {
			t.Fatal(err)
		}
	})
	server := httptest.NewServer(mux)

	urlA, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := DefaultClient().WithBaseURL(urlA)
	opts := &RequestOptions{
		After:     "",
		Count:     10,
		Sorting:   "best",
		Timeframe: "all",
		Subreddit: "wallpaper",
	}

	items, _, err := client.Subreddit.GetItems(context.TODO(), opts)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatal("unexpected length of items:", len(items), items)
	}
	item := items[0]
	if item.Name != "05sk8tzriboa1" {
		t.Fatal("unexpected name:", item.Name)
	}
	if item.Extension != "png" {
		t.Fatal("unexpected extension:", item.Extension)
	}
	if item.Height != 3840 {
		t.Fatal("unexpected height:", item.Height)
	}
	if item.Width != 6656 {
		t.Fatal("unexpected width:", item.Width)
	}
	if item.IsOver18 != false {
		t.Fatal("unexpected IsOver18:", item.IsOver18)
	}
	if item.Orientation != "landscape" {
		t.Fatal("unexpected orientation:", item.Orientation)
	}
	if item.URL != "https://i.redd.it/05sk8tzriboa1.png" {
		t.Fatal("unexpected url:", item.URL)
	}
	if item.Type != "image" {
		t.Fatal("unexpected type:", item.Type)
	}
	_ = items
}
