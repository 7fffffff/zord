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

	// let's move some common fields to the front
	writer := zord.NewWriter()
	//writer.Wr = os.Stderr // default
	//writer.FirstKeys = zord.DefaultFirstKeys() // default
	logger = zerolog.New(writer).With().Timestamp().Str("service", "greeter").Logger()
	logger.Debug().Int("a", 1).Msg("hello, world!")
	// => {"time":"2006-01-02T15:04:05-07:00","level":"debug","message":"hello, world!","service":"greeter","a":1}

	// let's make "service" appear after "level"
	writer = zord.NewWriter()
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
unordered sets of name/value pairs, so rearranging the pairs should not change
how objects are interpreted by a downstream system.

In production scenarios where log output is only ever read by other programs,
there's not much point in using zord.Writer.

## Duplicate Keys

zerolog doesn't deduplicate keys and neither does zord.Writer. Duplicate keys
will maintain their ordering relative to each other.

## Binary Logs (CBOR)

If compiled with the binary_log build tag, zord.Writer won't inspect or modify
the bytes it's given.

## Efficiency

zord.Writer parses and reassembles the event object, so there's inevitably some
overhead compared using just zerolog.

```
Logging an event with 10 fields
Using zerolog v1.5.0, the minimum version for this package
BenchmarkZerologDefault-6          952431      1260 ns/op        0 B/op      0 allocs/op
BenchmarkZerologConsoleWriter-6     44083     25790 ns/op     2487 B/op     88 allocs/op
BenchmarkZordWriter-6              130533      9314 ns/op     3080 B/op     28 allocs/op
```

## License

Round Robin License:
https://roundrobinlicense.com/2.0.0
