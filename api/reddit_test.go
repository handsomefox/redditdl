package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseURL(t *testing.T) {
	t.Parallel()
	_, err := url.Parse(defaultBaseURL)
	assert.NoError(t, err)
	_, err = url.Parse(defaultBaseImageURL)
	assert.NoError(t, err)
	_, err = url.Parse(defaultBaseVideoURL)
	assert.NoError(t, err)
}

func TestURLFormatting(t *testing.T) {
	t.Parallel()
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
	assert.Equal(t, correct, res, "incorrect url format")
}

func TestGetURL(t *testing.T) {
	t.Parallel()
	var (
		p    = GetSavedPost(t)
		b, _ = GetSavedImage(t)
		c    = DefaultClient()
	)

	res, err := c.GetURL(context.TODO(), p.URL())
	assert.NoError(t, err)
	defer res.Body.Close()

	b2, err := io.ReadAll(res.Body)
	assert.NoError(t, err)

	assert.Equal(t, b, b2, "couldn't correctly fetch the image data")
}

func TestGetImageByURL(t *testing.T) {
	t.Parallel()
	var (
		p      = GetSavedPost(t)
		b, str = GetSavedImage(t)
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, str, r.URL.Path[1:], "unexpected filename")
		_, err := w.Write(b)
		assert.NoError(t, err)
	})
	server := httptest.NewServer(mux)
	u, err := url.Parse(server.URL)
	assert.NoError(t, err)

	imgURL, err := url.Parse(p.URL())
	assert.NoError(t, err)
	imgURL.Scheme = "http"

	c := DefaultClient().WithBaseImageURL(u)

	res, err := c.GetImageByURL(context.TODO(), imgURL.String())
	assert.NoError(t, err)
	defer res.Body.Close()

	b2, err := io.ReadAll(res.Body)
	assert.NoError(t, err)

	assert.Equal(t, b, b2, "couldn't correctly fetch the image data")
}

func GetSavedPost(t *testing.T) Post {
	t.Helper()
	b, err := os.ReadFile("testdata/sample.json")
	assert.NoError(t, err)

	var ps Posts
	err = json.Unmarshal(b, &ps)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(ps.Data.Children))

	return ps.Data.Children[0]
}

func GetSavedImage(t *testing.T) (b []byte, name string) {
	t.Helper()
	b, err := os.ReadFile("testdata/05sk8tzriboa1.png")
	assert.NoError(t, err)
	return b, "05sk8tzriboa1.png"
}
