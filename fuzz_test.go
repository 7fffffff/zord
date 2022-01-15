//go:build go1.18
// +build go1.18

package zord

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func FuzzParser(f *testing.F) {
	for _, test := range reorderTests {
		f.Add(test.obj)
	}
	for _, test := range zordWriterTests {
		f.Add(test.obj)
	}
	f.Fuzz(func(t *testing.T, obj []byte) {
		// the parser only works with objects
		if bytes.IndexByte(obj, '{') < 0 {
			t.Skip()
		}
		var fields eventData
		stdlibErr := json.Unmarshal(obj, &fields)
		p := parser{}
		kv1, n, err := p.parse(obj)
		if err == nil {
			n = skipWhitespace(obj, n)
			if n < len(obj) {
				err = errors.New("unconsumed input")
			}
		}
		if err != nil {
			if stdlibErr != nil {
				// if the stdlib parser also didn't like it, skip it
				t.Skip()
			}
			// as of go 1.15 the max nesting depth of the stdlib parser is 10000:
			// https://github.com/golang/go/commit/84afaa9e9491d76ea43d7125b336030a0a2a902d
			// so it will accept objects that the zord parser won't. if obj
			// causes a depth limit error, ignore it
			if errors.Is(err, errMaxDepth) {
				t.Skip()
			} else {
				t.Fatal(err)
			}
		}
		if stdlibErr != nil {
			t.Fatal(errors.New("accepted invalid JSON"))
		}
		missingKeys := map[string]struct{}{}
		for key, _ := range fields {
			missingKeys[key] = struct{}{}
		}
		for _, pair := range kv1 {
			if _, ok := fields[pair.keyUnquoted]; !ok {
				t.Fatal(errors.New("unexpected key"))
			}
			delete(missingKeys, pair.keyUnquoted)
		}
		if len(missingKeys) > 0 {
			t.Fatal(fmt.Errorf("missing keys: %d", len(missingKeys)))
		}
		pairsWritten := 0
		buf := make([]byte, 0, len(obj))
		buf = append(buf, '{')
		for _, pair := range kv1 {
			if pairsWritten > 0 {
				buf = append(buf, ',')
			}
			buf = append(buf, pair.keyBytes...)
			buf = append(buf, ':')
			buf = append(buf, pair.valueBytes...)
			pairsWritten++
		}
		buf = append(buf, '}')
		kv2, _, err := p.parse(buf)
		if err != nil {
			t.Fatal(err)
		}
		if !samePairs(kv1, kv2) {
			t.Fatal(errors.New("parser roundtrip error"))
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
