package zord

import (
	"bytes"
	"testing"
)

type sortedWriterTest struct {
	desc      string
	obj       []byte
	firstKeys []string
	expected  []byte
}

var sortedWriterTests = []sortedWriterTest{
	{
		desc:      "empty object",
		obj:       []byte(`{  }`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{}`),
	},
	{
		desc:     "no changes",
		obj:      []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux"}`),
		expected: []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux"}`),
	},
	{
		desc:      "string values",
		obj:       []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux", "ddd":"baz"}`),
		firstKeys: []string{`bbb`, `ddd`},
		expected:  []byte(`{"bbb":"bar","ddd":"baz","aaa":"foo","ccc":"qux"}`),
	},
	{
		desc:      "sorted output",
		obj:       []byte(`{"bbb":0, "aaa":"foo", "ddd": 222, "ccc":-123.333}`),
		firstKeys: []string{`ccc`},
		expected:  []byte(`{"ccc":-123.333,"aaa":"foo","bbb":0,"ddd":222}`),
	},
	{
		desc:      "no duplicates",
		obj:       []byte(`{"bbb":true, "aaa":"foo", "ccc":null, "aaa": false}`),
		firstKeys: []string{`ccc`},
		expected:  []byte(`{"ccc":null,"aaa":false,"bbb":true}`),
	},
	{
		desc:      "array",
		obj:       []byte(`{"aaa":"foo", "ddd":[1, 2, 3], "bbb":"bar", "ccc":"qux"}`),
		firstKeys: []string{`ccc`},
		expected:  []byte(`{"ccc":"qux","aaa":"foo","bbb":"bar","ddd":[1, 2, 3]}`),
	},
	{
		desc:      "object",
		obj:       []byte(`{"aaa":"foo", "":{"xxx":1,"yyy":2,"zzz":3}, "bbb":["bar", null], "ccc":"qux"}`),
		firstKeys: []string{`ccc`},
		expected:  []byte(`{"ccc":"qux","":{"xxx":1,"yyy":2,"zzz":3},"aaa":"foo","bbb":["bar", null]}`),
	},
	{
		desc:      "as-is #1",
		obj:       []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux"`),
		firstKeys: []string{`bbb`},
		expected:  []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux"`),
	},
	{
		desc:      "as-is #2",
		obj:       []byte(`{"aaa":"foo", "bbb":"bar", "ccc":{"ddd": 3}}, "eee": 4}`),
		firstKeys: []string{`bbb`},
		expected:  []byte(`{"aaa":"foo", "bbb":"bar", "ccc":{"ddd": 3}}, "eee": 4}`),
	},
}

func TestSortedWriter(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	writer := newSortedWriter()
	writer.Wr = buf
	for i, test := range sortedWriterTests {
		buf.Reset()
		writer.FirstKeys = test.firstKeys
		_, err := writer.Write(test.obj)
		if err != nil {
			if test.desc != "" {
				t.Errorf("test \"%s\" failed: %v", test.desc, err)
			} else {
				t.Errorf("test #%d failed: %v", i, err)
			}
			continue
		}
		result := buf.Bytes()
		result = bytes.TrimRight(result, " \r\n")
		if !bytes.Equal(test.expected, result) {
			if test.desc != "" {
				t.Errorf("test \"%s\" unexpected: %s", test.desc, string(result))
			} else {
				t.Errorf("test #%d unexpected: %s", i, string(result))
			}
		}
	}
}
