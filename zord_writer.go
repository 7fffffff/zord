package zord

import (
	"io"
	"os"
)

// ZordWriter parses each Write for a JSON object and reorders the top level
// keys according to FirstKeys, before writing to Output. ZordWriter does not
// deduplicate keys.
//
// If the reordering process fails, ZordWriter will write the log event as-is
// without signalling the parsing error.
//
// If compiled with the binary_log build tag, ZordWriter will not inspect or
// reorder the data written to it.
type ZordWriter struct {
	Output    io.Writer // output writer
	FirstKeys []string  // keys to be moved to the beginning of event objects
}

// NewZordWriter creates a new ZordWriter. The default output writer is
// os.Stderr and the default list of keys is defined by DefaultFirstKeys()
func NewZordWriter() *ZordWriter {
	w := &ZordWriter{
		Output:    os.Stderr,
		FirstKeys: DefaultFirstKeys(),
	}
	return w
}
