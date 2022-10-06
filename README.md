# redditdl

Downloads images or videos from a subreddit in a batch.

## Building

```bash
go build --ldflags "-s -w" -o redditdl
```

## Usage

For usage, consult:

```bash
redditdl --help
```

## Example

```bash
redditdl -count 5 -dir example -verbose -progress
```
