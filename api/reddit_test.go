package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"
)

func TestBaseURL(t *testing.T) {
	_, err := url.Parse(defaultBaseURL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = url.Parse(defaultBaseImageURL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = url.Parse(defaultBaseVideoURL)
	if err != nil {
		t.Fatal(err)
	}
}

func TestURLFormatting(t *testing.T) {
	const correct = "https://reddit.com/r/example/best.json?after=&limit=10&t=all"
	var (
		opts = &RequestOptions{
			After:     "",
			Count:     10,
			Sorting:   "best",
			Timeframe: "all",
			Subreddit: "example",
		}
		c   = DefaultClient()
		res = c.optsURL(opts)
	)
	if res != correct {
		t.Fatal("incorrect formatting url, want:", correct, "got:", res)
	}
}

func TestGetURL(t *testing.T) {
	var (
		p    = GetSavedPost(t)
		b, _ = GetSavedImage(t)
		c    = DefaultClient()
	)

	res, err := c.GetURL(context.TODO(), p.URL())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	b2, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(b, b2) {
		t.Fatal("couldn't correctly fetch the image data")
	}
}

func TestGetImageByURL(t *testing.T) {
	var (
		p      = GetSavedPost(t)
		b, str = GetSavedImage(t)
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path[1:] != str {
			t.Fatal("unexpected filename, want:", str, "got:", r.URL.Path[1:])
		}
		_, err := w.Write(b)
		if err != nil {
			t.Fatal(err)
		}
	})
	server := httptest.NewServer(mux)
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	imgUrl, err := url.Parse(p.URL())
	if err != nil {
		t.Fatal(err)
	}
	imgUrl.Scheme = "http"

	c := DefaultClient().WithBaseImageURL(u)

	res, err := c.GetImageByURL(context.TODO(), imgUrl.String())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	b2, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(b, b2) {
		t.Fatal("couldn't correctly fetch the image data")
	}
}

func GetSavedPost(t *testing.T) *Post {
	t.Helper()
	b, err := os.ReadFile("testdata/sample.json")
	if err != nil {
		t.Fatal(err)
	}
	var ps Posts
	if err := json.Unmarshal(b, &ps); err != nil {
		t.Fatal(err)
	}
	if len(ps.Data.Children) != 1 {
		t.Fatal("unexpected posts length:", len(ps.Data.Children))
	}

	return ps.Data.Children[0]
}

func GetSavedImage(t *testing.T) ([]byte, string) {
	t.Helper()
	b, err := os.ReadFile("testdata/05sk8tzriboa1.png")
	if err != nil {
		t.Fatal(err)
	}
	return b, "05sk8tzriboa1.png"
}
