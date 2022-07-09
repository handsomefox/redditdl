# redditdl

Downloads images or videos from a subreddit in a batch.

## Building

```bash
go build --ldflags "-s -w" -o redditdl
```

## Usage

```text
Usage of redditdl:

  -count int
        Amount of images (and videos if specified) to download (default 1)
  -dir string
        Specifies the directory where to download the media (default "media")
  -height int
        minimal height of the media to download
  -progress
        Indicates whether the application will show the download progress (default false)
  -sort string
        How to sort (controversial, best, hot, new, random, rising, top) (default "top")
  -sub string
        Subreddit name (default "wallpaper")
  -tf string
        Timeframe from which to get the posts (hour, day, week, month, year, all) (default "all")
  -verbose
        Turns the logging on or off (default false)
  -video
        Indicates wheter the application should download videos as well (default false)
  -width int
        minimal width of the media to download
```

## Example

```bash
redditdl -count 5 -dir example
```
