// +build !binary_log

package zord

func (z ZordWriter) Write(event []byte) (n int, err error) {
	obj := make([]byte, 0, len(event))
	obj, n, err = reorder(obj, event, z.FirstKeys)
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
