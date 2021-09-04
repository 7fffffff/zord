//go:build !binary_log
// +build !binary_log

package zord

import (
	"fmt"
)

func (z ZordWriter) Write(event []byte) (n int, err error) {
	obj := make([]byte, 0, len(event))
	obj, n, err = tryReorder(obj, event, z.FirstKeys)
	if err != nil {
		// If there's an error in the reordering process, it's more
		// important that the log data get written. So write the event
		// data as-is.
		return z.Output.Write(event)
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
