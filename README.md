# redditdl

Downloads images or videos from a subreddit in a batch.

## Building

```bash
go build --ldflags "-s -w" -o redditdl
```

## Usage

```text
Usage of redditdl:
  -orientation string
        Specifies the image orientation ("l" for landscape, "p" for portrait, empty for any) (default "")
  -count
      Amount of media to download (default 1)
  -dir
      Specifies the directory where to download the media (default "media")
  -height
      minimal height of the media to download
  -width
      minimal width of the media to download
  -progress
      Indicates whether the application will show the download progress
  -sort
      How to sort (controversial, best, hot, new, random, rising, top) (default "top")
  -sub
      Subreddit name (default "wallpaper")
  -timeframe
      Timeframe from which to get the posts (hour, day, week, month, year, all) (default "all")
  -verbose
      Turns the logging on or off
  -video
      Indicates whether the application should download videos as well

```

## Example

```bash
redditdl -count 5 -dir example -verbose -progress
```
