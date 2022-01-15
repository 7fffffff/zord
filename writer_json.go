//go:build !binary_log
// +build !binary_log

package zord

import (
	"fmt"
)

func (z Writer) Write(event []byte) (n int, err error) {
	obj := make([]byte, 0, len(event))
	obj, n, err = tryReorder(obj, event, z.FirstKeys)
	if err != nil {
		// If there's an error in the reordering process, it's more
		// important that the log data get written. So write the event
		// data as-is.
		return z.Output.Write(event)
	}
	if n < len(event) {
		n = skipWhitespace(event, n)
		if n < len(event) {
			// Parsing succeeded but there's unconsumed, non-whitespace
			// bytes after the end of the object. Give up and write the
			// event as-is.
			return z.Output.Write(event)
		}
	}
	_, err = z.Output.Write(obj)
	if err != nil {
		return n, err
	}
	_, err = z.Output.Write([]byte("\n"))
	return n, err
}

func tryReorder(dest, src []byte, firstKeys []string) (extended []byte, n int, err error) {
	defer func() {
		if r := recover(); r != nil {
			if recoveredErr, ok := r.(error); ok {
				err = fmt.Errorf("zord reorder panic: %w", recoveredErr)
			} else {
				err = fmt.Errorf("zord reorder panic: %v", r)
			}
		}
	}()
	return reorder(dest, src, firstKeys)
}
