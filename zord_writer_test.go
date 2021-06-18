package zord

import (
	"bytes"
	"testing"
)

type zordWriterTest struct {
	desc      string
	obj       []byte
	firstKeys []string
	expected  []byte
}

var zordWriterTests = []zordWriterTest{
	{
		desc:      "empty object",
		obj:       []byte(`{}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{}`),
	},
	{
		desc:     "no changes",
		obj:      []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux"}`),
		expected: []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux"}`),
	},
	{
		desc:      "as-is",
		obj:       []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux"`),
		firstKeys: []string{`bbb`},
		expected:  []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux"`),
	},
	{
		desc:      "string values",
		obj:       []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux", "ddd":"baz"}`),
		firstKeys: []string{`bbb`, `ddd`},
		expected:  []byte(`{"bbb":"bar","ddd":"baz","aaa":"foo","ccc":"qux"}`),
	},
	{
		desc:      "preserve order",
		obj:       []byte(`{"bbb":0, "aaa":"foo", "ddd": 222, "ccc":-123.333}`),
		firstKeys: []string{`ccc`},
		expected:  []byte(`{"ccc":-123.333,"bbb":0,"aaa":"foo","ddd":222}`),
	},
	{
		desc:      "duplicate keys allowed",
		obj:       []byte(`{"bbb":true, "aaa":"foo", "ccc":null, "aaa": false}`),
		firstKeys: []string{`ccc`},
		expected:  []byte(`{"ccc":null,"bbb":true,"aaa":"foo","aaa":false}`),
	},
	{
		desc:      "array",
		obj:       []byte(`{"aaa":"foo", "ddd":[1, 2, 3], "bbb":"bar", "ccc":"qux"}`),
		firstKeys: []string{`ccc`},
		expected:  []byte(`{"ccc":"qux","aaa":"foo","ddd":[1, 2, 3],"bbb":"bar"}`),
	},
	{
		desc:      "object",
		obj:       []byte(`{"aaa":"foo", "ddd":{"xxx":1,"yyy":2,"zzz":3}, "bbb":"bar", "ccc":"qux"}`),
		firstKeys: []string{`ccc`},
		expected:  []byte(`{"ccc":"qux","aaa":"foo","ddd":{"xxx":1,"yyy":2,"zzz":3},"bbb":"bar"}`),
	},
}

func TestZordWriter(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	writer := NewZordWriter()
	writer.Output = buf
	for i, test := range zordWriterTests {
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
