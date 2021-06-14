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
	var data eventData
	var buf = bytes.NewBuffer(make([]byte, 0, len(event)))
	var dec = json.NewDecoder(bytes.NewReader(event))
	dec.UseNumber()
	err = dec.Decode(&data)
	n = int(dec.InputOffset())
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
		if value, ok := data[key]; ok {
			if pairsWritten > 0 {
				buf.WriteByte(',')
			}
			w.writePair(buf, key, value)
			pairsWritten++
			delete(data, key)
		}
	}
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if pairsWritten > 0 {
			buf.WriteByte(',')
		}
		w.writePair(buf, key, data[key])
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
