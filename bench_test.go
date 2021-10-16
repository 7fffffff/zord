package zord

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

var referenceTime = time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
var errExample = errors.New("oh no!")

func logBenchFn(logger zerolog.Logger) {
	logger.Debug().
		Str("string", "four!").
		Time(zerolog.TimestampFieldName, referenceTime).
		Int("int", 123).
		Float32("float", -2.203230293249593).
		Bool("true", true).
		Err(errExample).
		Str("name", "John Doe").
		Str("email", "john@example.com").
		Msg("the quick brown fox jumped over the lazy dog")
}

func BenchmarkZerologDefault(b *testing.B) {
	logger := zerolog.New(io.Discard)
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		logBenchFn(logger)
	}
}

func BenchmarkZerologConsoleWriter(b *testing.B) {
	writer := zerolog.ConsoleWriter{
		Out:     io.Discard,
		NoColor: true,
	}
	logger := zerolog.New(writer)
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		logBenchFn(logger)
	}
}

func BenchmarkZordSortedWriter(b *testing.B) {
	writer := newSortedWriter()
	writer.Wr = io.Discard
	logger := zerolog.New(writer)
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		logBenchFn(logger)
	}
}

func BenchmarkZordWriter(b *testing.B) {
	writer := NewZordWriter()
	writer.Output = io.Discard
	logger := zerolog.New(writer)
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		logBenchFn(logger)
	}
}
