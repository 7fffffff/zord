package zord

import (
	"io"
	"os"
)

// Writer parses each Write for a JSON object and reorders the top level
// keys according to FirstKeys, before writing to Output. Writer does not
// deduplicate keys.
//
// If the reordering process fails, Writer will write the log event as-is
// without signalling the parsing error.
//
// If compiled with the binary_log build tag, Writer will not inspect or
// reorder the data written to it.
type Writer struct {
	Output    io.Writer // output writer
	FirstKeys []string  // keys to be moved to the beginning of event objects
}

// NewWriter creates a new Writer. The default output writer is
// os.Stderr and the default list of keys is defined by DefaultFirstKeys()
func NewWriter() *Writer {
	w := &Writer{
		Output:    os.Stderr,
		FirstKeys: DefaultFirstKeys(),
	}
	return w
}
