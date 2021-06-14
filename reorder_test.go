package zord

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type reorderTest struct {
	desc        string
	obj         []byte
	firstKeys   []string
	expected    []byte
	expectedErr func(err error) bool
}

func errorAtFunc(pos int) func(error) bool {
	return func(err error) bool {
		if err, ok := err.(errorAt); ok {
			if err.Pos() == pos {
				return true
			}
		}
		return false
	}
}

func errorIsFunc(expected error) func(error) bool {
	return func(err error) bool {
		if err == nil || expected == nil {
			return false
		}
		return errors.Is(err, expected)
	}
}

func errorIsAtFunc(expected error, pos int) func(error) bool {
	return func(err error) bool {
		if err == nil || expected == nil {
			return false
		}
		if err, ok := err.(errorAt); ok {
			if !errors.Is(err, expected) {
				return false
			}
			if err.Pos() == pos {
				return true
			}
		}
		return false
	}
}

var reorderTests = []reorderTest{
	{
		desc:        "empty input",
		obj:         []byte(``),
		firstKeys:   []string{`aaa`},
		expectedErr: errorIsFunc(io.ErrUnexpectedEOF),
	},
	{
		desc:      "empty object",
		obj:       []byte(`{}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{}`),
	},
	{
		desc:        "objects only",
		obj:         []byte(`    []`),
		firstKeys:   []string{`aaa`},
		expectedErr: errorAtFunc(4),
	},
	{
		desc:      "no changes",
		obj:       []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux"}`),
		firstKeys: []string{},
		expected:  []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux"}`),
	},
	{
		desc:      "string values",
		obj:       []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux", "":"",  "<":""}`),
		firstKeys: []string{`ddd`, `bbb`, ``, `bbb`},
		expected:  []byte(`{"bbb":"bar","":"","aaa":"foo","ccc":"qux","<":""}`),
	},
	{
		desc:      "non ascii",
		obj:       []byte(`{"aaa":"foo","bbb":"😎", "ccc":"qux"}`),
		firstKeys: []string{`bbb`, `bbb`},
		expected:  []byte(`{"bbb":"😎","aaa":"foo","ccc":"qux"}`),
	},
	{
		desc:      "preserve duplicate keys",
		obj:       []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux", "bbb":"BAR"}`),
		firstKeys: []string{`bbb`},
		expected:  []byte(`{"bbb":"bar","bbb":"BAR","aaa":"foo","ccc":"qux"}`),
	},
	{
		desc:      "escaped strings",
		obj:       []byte(`{"aaa":"\f\f\f", "b\\bb":"bar", "ccc":"qu\u005C\"x"}`),
		firstKeys: []string{`b\bb`},
		expected:  []byte(`{"b\\bb":"bar","aaa":"\f\f\f","ccc":"qu\u005C\"x"}`),
	},
	{
		desc:      "number values",
		obj:       []byte(`{"aaa":0.1, "bbb":0, "ccc":-123.333}`),
		firstKeys: []string{`ccc`},
		expected:  []byte(`{"ccc":-123.333,"aaa":0.1,"bbb":0}`),
	},
	{
		desc:      "number values with exponent",
		obj:       []byte(`{"aaa":0e10, "bbb":"bar", "ccc":1e-005, "ddd":1E+005}`),
		firstKeys: []string{`ccc`, `ddd`},
		expected:  []byte(`{"ccc":1e-005,"ddd":1E+005,"aaa":0e10,"bbb":"bar"}`),
	},
	{
		desc:      "bool & null values",
		obj:       []byte(`{"aaa":null, "bbb":"", "ccc":false, "ddd": true}`),
		firstKeys: []string{`ccc`},
		expected:  []byte(`{"ccc":false,"aaa":null,"bbb":"","ddd":true}`),
	},
	{
		desc:      "extra whitespace",
		obj:       []byte(`{ "bbb":"bar", "aaa"  :  "foo", "ccc":"qux" , "aaa":true}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{"aaa":"foo","aaa":true,"bbb":"bar","ccc":"qux"}`),
	},
	{
		desc:      "empty array",
		obj:       []byte(`{"bbb":"bar", "aaa"  :  [ ], "ccc":"qux", "aaa":true}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{"aaa":[ ],"aaa":true,"bbb":"bar","ccc":"qux"}`),
	},
	{
		desc:      "single element array",
		obj:       []byte(`{"bbb":"bar", "aaa"  :  [1], "ccc":"qux", "aaa":true}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{"aaa":[1],"aaa":true,"bbb":"bar","ccc":"qux"}`),
	},
	{
		desc:      "mixed array",
		obj:       []byte(`{"bbb":"bar", "aaa"  :  [1,2,"a" , 0.0,null], "ccc":"qux", "aaa":true}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{"aaa":[1,2,"a" , 0.0,null],"aaa":true,"bbb":"bar","ccc":"qux"}`),
	},
	{
		desc:      "nested arrays",
		obj:       []byte(`{"bbb":"bar", "aaa"  :  [[[1], [2, 3]], 4, [[[]] ]], "ccc":"qux", "aaa":true}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{"aaa":[[[1], [2, 3]], 4, [[[]] ]],"aaa":true,"bbb":"bar","ccc":"qux"}`),
	},
	{
		desc:      "empty nested object",
		obj:       []byte(`{"bbb":"bar", "aaa"  :  {}, "ccc":"qux", "aaa":true}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{"aaa":{},"aaa":true,"bbb":"bar","ccc":"qux"}`),
	},
	{
		desc:      "nested object with one property",
		obj:       []byte(`{"bbb":"bar", "aaa"  :  { "qqq":111}, "ccc":"qux", "aaa":true}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{"aaa":{ "qqq":111},"aaa":true,"bbb":"bar","ccc":"qux"}`),
	},
	{
		desc:      "nested object with multiple properties",
		obj:       []byte(`{"bbb":"bar", "aaa"  :  {"qqq":111,"www": false}, "ccc":"qux", "aaa":true}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{"aaa":{"qqq":111,"www": false},"aaa":true,"bbb":"bar","ccc":"qux"}`),
	},
	{
		desc:        "incomplete literal #1",
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":fal}`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorAtFunc(36),
	},
	{
		desc:        "incomplete literal #2",
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":nul}`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorAtFunc(36),
	},
	{
		desc:        "incomplete literal #3",
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":tru}`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorAtFunc(36),
	},
	{
		desc:        "invalid number #1",
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":11.e}`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorAtFunc(36),
	},
	{
		desc:        "invalid number #2",
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":11.11.11}`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorAtFunc(38),
	},
	{
		desc:        "invalid number #3",
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":1x1}`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorAtFunc(34),
	},
	{
		desc:        "invalid number #4",
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":1e.1}`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorAtFunc(35),
	},
	{
		desc:        "invalid array #1",
		obj:         []byte(`{"bbb":"bar", "aaa"  :  [1:2], "ccc":"qux", "aaa":true}`),
		firstKeys:   []string{`aaa`},
		expectedErr: errorAtFunc(26),
	},
	{
		desc:        "invalid array #2",
		obj:         []byte(`{"bbb":"bar", "aaa"  :  [a], "ccc":"qux", "aaa":true}`),
		firstKeys:   []string{`aaa`},
		expectedErr: errorAtFunc(25),
	},
	{
		desc:        "incomplete object",
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":fals`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorIsAtFunc(io.ErrUnexpectedEOF, 37),
	},
	{
		desc:        "invalid object #1",
		obj:         []byte(`{"bbb":"bar", "aaa"  :  { "qqq":}, "ccc":"qux", "aaa":true}`),
		firstKeys:   []string{`aaa`},
		expectedErr: errorAtFunc(32),
	},
	{
		desc:        "invalid object #2",
		obj:         []byte(`{"bbb":"bar", "aaa"  :  { "qqq",}, "ccc":"qux", "aaa":true}`),
		firstKeys:   []string{`aaa`},
		expectedErr: errorAtFunc(31),
	},
	{
		desc:      "trailing comma #1",
		obj:       []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux",}`),
		firstKeys: []string{`bbb`},
		expected:  []byte(`{"bbb":"bar","aaa":"foo","ccc":"qux"}`),
	},
	{
		desc:      "trailing comma #2",
		obj:       []byte(`{"aaa":"foo", "bbb":[ {"ddd": 1}, ], "ccc":"qux"}`),
		firstKeys: []string{`bbb`},
		expected:  []byte(`{"bbb":[ {"ddd": 1}, ],"aaa":"foo","ccc":"qux"}`),
	},
	{
		desc:      "trailing comma #3",
		obj:       []byte(`{"aaa":"foo", "bbb":{"ddd": 0 , }, "ccc":"qux"}`),
		firstKeys: []string{`bbb`},
		expected:  []byte(`{"bbb":{"ddd": 0 , },"aaa":"foo","ccc":"qux"}`),
	},
}

func TestReorder(t *testing.T) {
	for i, test := range reorderTests {
		result, _, err := reorder(nil, test.obj, test.firstKeys)
		if test.expectedErr != nil && err == nil {
			t.Errorf("test \"%s\" expected an error", test.desc)
			continue
		}
		if err != nil {
			if test.expectedErr != nil && test.expectedErr(err) {
				continue
			}
			if test.desc != "" {
				t.Errorf("test \"%s\" failed: %v", test.desc, err)
			} else {
				t.Errorf("test #%d failed: %v", i, err)
			}
			continue
		}
		if !bytes.Equal(test.expected, result) {
			if test.desc != "" {
				t.Errorf("test \"%s\" unexpected: %s", test.desc, string(result))
			} else {
				t.Errorf("test #%d unexpected: %s", i, string(result))
			}
		}
	}
}
