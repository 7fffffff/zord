// +build gofuzzbeta

package zord

import (
	"bytes"
	"testing"
)

func FuzzParser(f *testing.F) {
	for _, test := range zordWriterTests {
		f.Add(test.obj)
	}
	f.Fuzz(func(t *testing.T, obj []byte) {
		parser := &parser{}
		pairs1, _, err := parser.parse(obj)
		if err != nil {
			t.Skip()
		}
		pairsWritten := 0
		dest := make([]byte, 0, len(obj))
		dest = append(dest, '{')
		for _, pair := range pairs1 {
			if pairsWritten > 0 {
				dest = append(dest, ',')
			}
			dest = append(dest, pair.keyBytes...)
			dest = append(dest, ':')
			dest = append(dest, pair.valueBytes...)
			pairsWritten++
		}
		dest = append(dest, '}')
		pairs2, _, err := parser.parse(dest)
		if err != nil {
			t.Fatal(err)
		}
		if !samePairs(pairs1, pairs2) {
			t.Fatalf("roundtrip error")
		}
	})
}

func samePairs(pairs1, pairs2 []kv) bool {
	if len(pairs1) != len(pairs2) {
		return false
	}
	for i, _ := range pairs1 {
		if pairs1[i].keyUnquoted != pairs2[i].keyUnquoted {
			return false
		}
		if !bytes.Equal(pairs1[i].valueBytes, pairs2[i].valueBytes) {
			return false
		}
	}
	return true
}
