# zord

zord provides a writer for https://github.com/rs/zerolog that reorders the log
event objects for better readability.

## Example

```go
package main

import (
	"os"
	"github.com/7fffffff/zord"
	"github.com/rs/zerolog"
)

func main() {
	var logger zerolog.Logger

	// using zerolog by itself
	logger = zerolog.New(os.Stderr).With().Timestamp().Str("service", "greeter").Logger()
	logger.Debug().Int("a", 1).Msg("hello, world!")
	// => {"level":"debug","service":"greeter","a":1,"time":"2006-01-02T15:04:05-07:00","message":"hello, world!"}

	writer := zord.NewZordWriter()
	//writer.Wr = os.Stderr // default
	//writer.FirstKeys = zord.DefaultFirstKeys() // default
	logger = zerolog.New(writer).With().Timestamp().Str("service", "greeter").Logger()
	logger.Debug().Int("a", 1).Msg("hello, world!")
	// => {"time":"2006-01-02T15:04:05-07:00","level":"debug","message":"hello, world!","service":"greeter","a":1}

	// let's make "service" appear after "level"
	writer = zord.NewZordWriter()
	writer.FirstKeys = []string{
		zerolog.TimestampFieldName,
		zerolog.LevelFieldName,
		"service",
		zerolog.CallerFieldName,
		zerolog.MessageFieldName,
		zerolog.ErrorFieldName,
	}
	logger = zerolog.New(writer).With().Timestamp().Str("service", "greeter").Logger()
	logger.Debug().Int("a", 1).Msg("hello, world!")
	// => {"time":"2006-01-02T15:04:05-07:00","level":"debug","service":"greeter","message":"hello, world!","a":1}
}
```

## Why?

In development, log lines are a little easier to read when common fields like
the timestamp are always in the same place. JSON objects are defined as
unordered, so rearranging the key/value pairs should not change how the objects
are interpreted.

In production scenarios where zerolog's output is only consumed programatically, there's not much
point in using ZordWriter.

## Efficiency

ZordWriter parses and reassembles the event object, so there's inevitably
some overhead compared using just zerolog.

```
Logging an event with 10 fields
Using zerolog v1.5.0, the minimum version for this package
BenchmarkZerologDefault-6          952431      1260 ns/op        0 B/op      0 allocs/op
BenchmarkZerologConsoleWriter-6     44083     25790 ns/op     2487 B/op     88 allocs/op
BenchmarkZordWriter-6              130533      9314 ns/op     3080 B/op     28 allocs/op
```

## Duplicate Keys

zerolog doesn't deduplicate keys and neither does ZordWriter.

## License

Round Robin License:
https://roundrobinlicense.com/2.0.0
