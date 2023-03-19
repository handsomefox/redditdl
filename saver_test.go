package main

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestDownload(t *testing.T) {
	t.Parallel()
	args := defaultArgs(t.TempDir(), 1)
	assert.NoError(t, NewSaver(args, 1, 1).Run(context.TODO()))
}

func BenchmarkDownload10(b *testing.B) {
	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		dir := b.TempDir()
		args := defaultArgs(dir, 10)
		if err := NewSaver(args, 1, 10).Run(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDownload50(b *testing.B) {
	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		dir := b.TempDir()
		args := defaultArgs(dir, 50)
		if err := NewSaver(args, 1, 50).Run(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func defaultArgs(dir string, count int64) *AppArguments {
	log.Logger = log.Level(zerolog.FatalLevel)
	return &AppArguments{
		SubredditContentType: "image",
		SubredditSort:        "best",
		SubredditTimeframe:   "all",
		SubredditList:        "wallpaper",
		ShowNSFW:             false,
		MediaCount:           count,
		MediaOrientation:     "all",
		MediaMinimalWidth:    0,
		MediaMinimalHeight:   0,
		SaveDirectory:        dir,
		VerboseLogging:       false,
		ProgressLogging:      false,
	}
}
