package main

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestDownload(t *testing.T) {
	t.Parallel()
	args := defaultArgs(t.TempDir(), 1)
	saver := NewSaver(args)

	ctx := context.TODO()

	if err := saver.Run(ctx, 1, 1); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkDownload10(b *testing.B) {
	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		dir := b.TempDir()
		args := defaultArgs(dir, 10)
		saver := NewSaver(args)
		if err := saver.Run(ctx, 1, 10); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDownload50(b *testing.B) {
	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		dir := b.TempDir()
		args := defaultArgs(dir, 50)
		saver := NewSaver(args)
		if err := saver.Run(ctx, 1, 50); err != nil {
			b.Fatal(err)
		}
	}
}

func defaultArgs(dir string, count int64) *AppArguments {
	log.Logger = log.Level(zerolog.DebugLevel)
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
