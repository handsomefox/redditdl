package client

type Config struct {
	Subreddit   string
	Sorting     string
	Timeframe   string
	Orientation string
	Count       int64
	MinWidth    int
	MinHeight   int
}
