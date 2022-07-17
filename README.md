# redditdl

Downloads images or videos from a subreddit in a batch.

## Building

```bash
go build --ldflags "-s -w" -o redditdl
```

## Usage

```text
Usage of redditdl:
  -cnt
      Amount of media to download (default 1)
  -dir
      Specifies the directory where to download the media (default "media")
  -h
      minimal height of the media to download
  -w
      minimal width of the media to download
  -p
      Indicates whether the application will show the download progress
  -sort
      How to sort (controversial, best, hot, new, random, rising, top) (default "top")
  -sub
      Subreddit name (default "wallpaper")
  -tf
      Timeframe from which to get the posts (hour, day, week, month, year, all) (default "all")
  -v
      Turns the logging on or off
  -video
      Indicates whether the application should download videos as well

```

## Example

```bash
redditdl -cnt 5 -dir example
```
