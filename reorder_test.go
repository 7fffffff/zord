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
		if parseErr, ok := err.(errorAt); ok {
			if parseErr.Pos() == pos {
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
		if parseErr, ok := err.(errorAt); ok {
			if !errors.Is(parseErr, expected) {
				return false
			}
			if parseErr.Pos() == pos {
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
		desc:      "empty object #1",
		obj:       []byte(`{}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{}`),
	},
	{
		desc:      "empty object #2",
		obj:       []byte(`{   }`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{}`),
	},
	{
		desc:        "objects only #1",
		obj:         []byte(`    []`),
		firstKeys:   []string{`aaa`},
		expectedErr: errorAtFunc(4),
	},
	{
		desc:        "objects only #2",
		obj:         []byte(` 123 `),
		firstKeys:   []string{`aaa`},
		expectedErr: errorAtFunc(1),
	},
	{
		desc:      "no changes",
		obj:       []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux"}`),
		firstKeys: []string{},
		expected:  []byte(`{"aaa":"foo", "bbb":"bar", "ccc":"qux"}`),
	},
	{
		desc:      "string values",
		obj:       []byte(`{"aaa":"foo", "bbb":"bar", "ccc"   :   "qux", "":"",  "<":""}`),
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
		obj:       []byte(`{"aaa":0.1, "bbb":0, "ccc":-123.456789, "ddd": 100, "   ": -0}`),
		firstKeys: []string{`ccc`, `   `},
		expected:  []byte(`{"ccc":-123.456789,"   ":-0,"aaa":0.1,"bbb":0,"ddd":100}`),
	},
	{
		desc:      "number values with exponent",
		obj:       []byte(`{"aaa":0e10, "bbb":4.9406564584124654417656879286822137236505980e-324, "ccc":1e-005 , "ddd":1E+005, "eee":0E0}`),
		firstKeys: []string{`ccc`, `ddd`},
		expected:  []byte(`{"ccc":1e-005,"ddd":1E+005,"aaa":0e10,"bbb":4.9406564584124654417656879286822137236505980e-324,"eee":0E0}`),
	},
	{
		desc:      "bool & null values",
		obj:       []byte(`{"aaa":null, "bbb":"", "ccc":false, "ddd": true}`),
		firstKeys: []string{`ccc`},
		expected:  []byte(`{"ccc":false,"aaa":null,"bbb":"","ddd":true}`),
	},
	{
		desc:      "empty array #1",
		obj:       []byte(`{"bbb":"bar", "aaa"  :  [], "ccc":"qux", "aaa":true}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{"aaa":[],"aaa":true,"bbb":"bar","ccc":"qux"}`),
	},
	{
		desc:      "empty array #2",
		obj:       []byte(`{"bbb":"bar", "aaa"  :  [   ], "ccc":"qux", "aaa":true}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{"aaa":[   ],"aaa":true,"bbb":"bar","ccc":"qux"}`),
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
		desc:      "empty nested object #1",
		obj:       []byte(`{"bbb":"bar", "aaa"  :  {}, "ccc":"qux", "aaa":true}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{"aaa":{},"aaa":true,"bbb":"bar","ccc":"qux"}`),
	},
	{
		desc:      "empty nested object #2",
		obj:       []byte(`{"bbb":"bar", "aaa"  :  {   }, "ccc":"qux", "aaa":true}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{"aaa":{   },"aaa":true,"bbb":"bar","ccc":"qux"}`),
	},
	{
		desc:      "nested object with one property",
		obj:       []byte(`{"bbb":["bar"], "aaa"  :  { "qqq":111}, "ccc":"qux", "aaa":true}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{"aaa":{ "qqq":111},"aaa":true,"bbb":["bar"],"ccc":"qux"}`),
	},
	{
		desc:      "nested object with multiple properties",
		obj:       []byte(`{"bbb":["bar"], "aaa"  :  {"qqq":111,"rrr":"sss" ,"www": [{"x":[1, false]}]}, "ccc":"qux", "aaa":true}`),
		firstKeys: []string{`aaa`},
		expected:  []byte(`{"aaa":{"qqq":111,"rrr":"sss" ,"www": [{"x":[1, false]}]},"aaa":true,"bbb":["bar"],"ccc":"qux"}`),
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
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":0xAF}`),
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
		desc:        "invalid number #5",
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":1.e1}`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorAtFunc(35),
	},
	{
		desc:        "invalid number #6",
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":.1}`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorAtFunc(33),
	},
	{
		desc:        "invalid number #7",
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":0E}`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorAtFunc(35),
	},
	{
		desc:        "invalid number #8",
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":0eE2}`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorAtFunc(35),
	},
	{
		desc:        "invalid number #9",
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":-012}`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorAtFunc(35),
	},
	{
		desc:        "invalid number #10",
		obj:         []byte(`{"aaa":"foo", "bbb":"bar", "ccc":-}`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorAtFunc(34),
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
		desc:        "invalid array #3",
		obj:         []byte(`{"bbb":"bar", "aaa"  :  [,], "ccc":"qux", "aaa":true}`),
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
		desc:        "invalid object #3",
		obj:         []byte(`{"bbb":"bar", "aaa"  :  { "qqq" }, "ccc":"qux", "aaa":true}`),
		firstKeys:   []string{`aaa`},
		expectedErr: errorAtFunc(32),
	},
	{
		desc:        "invalid object #4",
		obj:         []byte(`{"bbb":"bar", "aaa"  :  { "qqq"::1}, "ccc":"qux", "aaa":true}`),
		firstKeys:   []string{`aaa`},
		expectedErr: errorAtFunc(32),
	},
	{
		desc:        "invalid object #5",
		obj:         []byte(`{"bbb":"bar", "aaa"  :  { : 1}, ccc:"qux", "aaa":true}`),
		firstKeys:   []string{`aaa`},
		expectedErr: errorAtFunc(26),
	},
	{
		desc:        "invalid object #6",
		obj:         []byte(`{"bbb":"bar", "aaa"  :  { "qqq": 1}, ccc:"qux", "aaa":true}`),
		firstKeys:   []string{`aaa`},
		expectedErr: errorAtFunc(37),
	},
	{
		desc:        "invalid object #7",
		obj:         []byte(`{"bbb":"bar", "aaa"  :  { "qqq": 1}, 'ccc':"qux", "aaa":true}`),
		firstKeys:   []string{`aaa`},
		expectedErr: errorAtFunc(37),
	},
	{
		desc:        "invalid object #8",
		obj:         []byte(`{"bbb":"bar", "aaa"  :  { "qqq": 1}, "ccc"":"qux", "aaa":true}`),
		firstKeys:   []string{`aaa`},
		expectedErr: errorAtFunc(42),
	},
	{
		desc:        "invalid object #9",
		obj:         []byte(`{"bbb":"bar", "aaa"  :  { "qqq": 1}, :"qux", "aaa":true}`),
		firstKeys:   []string{`aaa`},
		expectedErr: errorAtFunc(37),
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
	{
		desc:        "no double trailing comma",
		obj:         []byte(`{"aaa":"foo", "bbb":{"ddd": 0 ,, }, "ccc":"qux"}`),
		firstKeys:   []string{`bbb`},
		expectedErr: errorAtFunc(31),
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
