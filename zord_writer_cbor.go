// +build binary_log

package zord

func (z ZordWriter) Write(event []byte) (n int, err error) {
	return z.Output.Write(event)
}
