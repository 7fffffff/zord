//go:build binary_log
// +build binary_log

package zord

func (z Writer) Write(event []byte) (n int, err error) {
	return z.Output.Write(event)
}
