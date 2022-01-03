package zord

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sort"

	"github.com/7fffffff/jsonconv"
)

type eventData map[string]json.RawMessage

// sortedWriter was the first attempt. It is included for comparison
// purposes only.
type sortedWriter struct {
	Wr        io.Writer // output writer
	FirstKeys []string  // keys to be moved to the beginning of event objects
}

func newSortedWriter() *sortedWriter {
	w := &sortedWriter{
		Wr:        os.Stderr,
		FirstKeys: DefaultFirstKeys(),
	}
	return w
}

func (w sortedWriter) Write(event []byte) (n int, err error) {
	if len(w.FirstKeys) == 0 {
		return w.Wr.Write(event)
	}
	var pairs eventData
	var buf = bytes.NewBuffer(make([]byte, 0, len(event)))
	err = json.Unmarshal(event, &pairs)
	n = len(event)
	if err != nil {
		// If there's an error in the reordering process, it's more
		// important that the log data get written. So write the event
		// as-is.
		return w.Wr.Write(event)
		//return n, fmt.Errorf("zord: cannot decode event: %w", err)
	}
	pairsWritten := 0
	buf.WriteByte('{')
	for _, key := range w.FirstKeys {
		if value, ok := pairs[key]; ok {
			if pairsWritten > 0 {
				buf.WriteByte(',')
			}
			w.writePair(buf, key, value)
			pairsWritten++
			delete(pairs, key)
		}
	}
	keys := make([]string, 0, len(pairs))
	for key := range pairs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if pairsWritten > 0 {
			buf.WriteByte(',')
		}
		w.writePair(buf, key, pairs[key])
		pairsWritten++
	}
	buf.WriteByte('}')
	buf.WriteByte('\n')
	_, err = buf.WriteTo(w.Wr)
	return n, err
}

func (w sortedWriter) writePair(buf *bytes.Buffer, key string, value json.RawMessage) {
	buf.Write(jsonconv.Quote(key))
	buf.WriteByte(':')
	buf.Write(value)
}
