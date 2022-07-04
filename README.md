# redditdl

Downloads images from a subreddit in a batch.

## Building

```bash
go build --ldflags "-s -w" -o redditdl
```

## Usage

```text
Usage of redditdl:

  -count int
        Amount of images to download (default 1)
  -dir string
        Specifies the directory where to download the images (default "images")
  -sort string
        How to sort (controversial, best, hot, new, random, rising, top) (default "top")
  -sub string
        Subreddit name (default "wallpaper")
  -tf string
        Timeframe from which to get the posts (hour, day, week, month, year, all) (default "all")
  -width int
        minimal width of the image to download (default 0)
  -height int
        minimal height of the image to download (default 0)
```

## Example

```bash
redditdl -count 5 -dir example
```
