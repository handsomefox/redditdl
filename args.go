package main

type AppArguments struct {
	SubredditContentType string   `arg:"-t, --type" help:"content type, values: image,video,both" default:"image"`
	SubredditSort        string   `arg:"-s, --sort" help:"sorting, values: controversial/best/hot/new/random/rising/top" default:"top"`
	SubredditTimeframe   string   `arg:"-f, --timeframe" help:"timeframe, values: hour/day/week/month/year/all" default:"all"`
	SubredditList        []string `arg:"-r, --subreddits" help:"a comma-separated list of subreddits to download from"`
	SubredditShowNSFW    bool     `arg:"-n, --nsfw" help:"enable if you want to show NSFW content"`

	MediaCount       int64  `arg:"-c, --count" help:"amount of media to download"`
	MediaOrientation string `arg:"-o, --orientation" help:"values: landspace/portrait/both" default:"both"`
	MediaWidth       int    `arg:"-w, --width" help:"minimal content width"`
	MediaHeight      int    `arg:"-h, --height" help:"minimal content height"`

	SaveDirectory string `arg:"-d, --dir" help:"output path"`

	VerboseLogging  bool `arg:"-v, --verbose" help:"enable debug logging"`
	ProgressLogging bool `arg:"-p, --progress" help:"enable current progress logging"`
}