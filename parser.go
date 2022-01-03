package zord

import (
	"errors"
	"fmt"
	"io"

	"github.com/7fffffff/jsonconv"
)

const defaultMaxDepth int = 64

var (
	errEndArray  = errors.New("]")
	errEndObject = errors.New("}")
)

type kv struct {
	keyUnquoted string
	keyBytes    []byte // double-quoted literal
	valueBytes  []byte // literal form (quoted/with brackets/etc)
}

// parser isn't a fully fledged JSON parser. It's only concerned about parsing
// JSON objects, and even then, only about finding the positions of the top
// level key-value pairs within.
type parser struct {
	maxDepth int // maximum nesting depth. If 0, defaultMaxDepth is used
}

func (p *parser) depthLimitReached(depth int) bool {
	maxDepth := p.maxDepth
	if maxDepth == 0 {
		maxDepth = defaultMaxDepth
	}
	return depth >= maxDepth || depth < 0
}

// parse expects buf to contain a valid utf-8 encoded JSON object and
// extracts the top level key-value pairs in the order they appear. parse
// does not deduplicate keys.
//
// parse returns the key-value pairs and the number of bytes read from buf
func (p *parser) parse(buf []byte) (pairs []kv, n int, err error) {
	pairs = make([]kv, 0, 16)
	n = skipWhitespace(buf, 0)
	n, err = p.parseBeginObject(buf, n)
	if err != nil {
		return pairs, n, err
	}
	for {
		pair := kv{}
		n = skipWhitespace(buf, n)
		// allow a trailing comma before the end of the object
		if len(pairs) > 0 {
			n, err = p.parseObjectComma(buf, n)
			if err == errEndObject {
				return pairs, n, nil
			}
			if err != nil {
				return pairs, n, err
			}
			n = skipWhitespace(buf, n)
		}
		if n < len(buf) && buf[n] == '}' {
			return pairs, n + 1, nil
		}
		keyStart := n
		n, err = p.parseString(buf, keyStart)
		if err != nil {
			return pairs, n, err
		}
		pair.keyBytes = buf[keyStart:n]
		if keyString, ok := jsonconv.Unquote(pair.keyBytes); ok {
			pair.keyUnquoted = keyString
		} else {
			return pairs, n, parseErrorAt(keyStart, fmt.Errorf("parse: could not unquote key [%d:%d]", keyStart, n))
		}
		n, err = p.parseColon(buf, skipWhitespace(buf, n))
		if err != nil {
			return pairs, n, err
		}
		valueStart := skipWhitespace(buf, n)
		n, err = p.parseValue(0, buf, valueStart)
		if err != nil {
			return pairs, n, err
		}
		pair.valueBytes = buf[valueStart:n]
		pairs = append(pairs, pair)
	}
}

func (p *parser) parseArray(depth int, buf []byte, initialPos int) (end int, err error) {
	if p.depthLimitReached(depth) {
		return initialPos, parseErrorAt(initialPos, errors.New("array: depth limit reached"))
	}
	i := initialPos
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), io.ErrUnexpectedEOF)
	}
	if b := buf[i]; b != '[' {
		return i + 1, parseErrorAt(i, fmt.Errorf("array: unexpected 0x%X", b))
	}
	i++
	numValues := 0
	for {
		i = skipWhitespace(buf, i)
		// allow a trailing comma before the end of the array
		if numValues > 0 {
			i, err = p.parseArrayComma(buf, i)
			if err == errEndArray {
				return i, nil
			}
			if err != nil {
				return i, err
			}
			i = skipWhitespace(buf, i)
		}
		if i < len(buf) && buf[i] == ']' {
			return i + 1, nil
		}
		valueEnd, err := p.parseValue(depth, buf, i)
		if err != nil {
			return valueEnd, err
		}
		i = valueEnd
		numValues++
	}
}

func (p *parser) parseArrayComma(buf []byte, initialPos int) (end int, err error) {
	i := initialPos
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), io.ErrUnexpectedEOF)
	}
	b := buf[i]
	if b == ']' {
		return i + 1, errEndArray
	}
	if b != ',' {
		return i + 1, parseErrorAt(i, fmt.Errorf("array comma: unexpected 0x%X", b))
	}
	return i + 1, nil
}

func (p *parser) parseBeginObject(buf []byte, initialPos int) (end int, err error) {
	i := initialPos
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), io.ErrUnexpectedEOF)
	}
	b := buf[i]
	if b != '{' {
		return i + 1, parseErrorAt(i, fmt.Errorf("object: unexpected: 0x%X", b))
	}
	return i + 1, nil
}

func (p *parser) parseColon(buf []byte, initialPos int) (end int, err error) {
	i := initialPos
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), io.ErrUnexpectedEOF)
	}
	b := buf[i]
	if b != ':' {
		return i + 1, parseErrorAt(i, fmt.Errorf("colon: unexpected 0x%X", b))
	}
	return i + 1, nil
}

func (p *parser) parseFalse(buf []byte, initialPos int) (end int, err error) {
	raw := [5]byte{'f', 'a', 'l', 's', 'e'}
	for r := 0; r < len(raw); r++ {
		i := initialPos + r
		if i >= len(buf) {
			return i, parseErrorAt(i, io.ErrUnexpectedEOF)
		}
		b := buf[i]
		if b != raw[r] {
			return i + 1, parseErrorAt(initialPos+r, fmt.Errorf("false: unexpected 0x%X", b))
		}
	}
	return initialPos + len(raw), nil
}

func (p *parser) parseNull(buf []byte, initialPos int) (end int, err error) {
	raw := [4]byte{'n', 'u', 'l', 'l'}
	for r := 0; r < len(raw); r++ {
		i := initialPos + r
		if i >= len(buf) {
			return i, parseErrorAt(i, io.ErrUnexpectedEOF)
		}
		b := buf[i]
		if b != raw[r] {
			return i + 1, parseErrorAt(initialPos+r, fmt.Errorf("null: unexpected 0x%X", b))
		}
	}
	return initialPos + len(raw), nil
}

func (p *parser) parseNumber(buf []byte, initialPos int) (end int, err error) {
	i := initialPos
	i, err = p.parseNumberMinus(buf, i)
	if err != nil {
		return i, err
	}
	i, err = p.parseNumberInt(buf, i)
	if err != nil {
		return i, err
	}
	i, err = p.parseNumberFrac(buf, i)
	if err != nil {
		return i, err
	}
	i, err = p.parseNumberExp(buf, i)
	return i, err
}

func (p *parser) parseNumberExp(buf []byte, initialPos int) (end int, err error) {
	i := initialPos
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), fmt.Errorf("number exp: %w", io.ErrUnexpectedEOF))
	}
	if buf[i] != 'e' && buf[i] != 'E' {
		return i, nil
	}
	i++
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), fmt.Errorf("number exp: %w", io.ErrUnexpectedEOF))
	}
	// optional
	if buf[i] == '+' || buf[i] == '-' {
		i++
	}
	digits := 0
	for i < len(buf) && '0' <= buf[i] && buf[i] <= '9' {
		digits++
		i++
	}
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), fmt.Errorf("number exp: %w", io.ErrUnexpectedEOF))
	}
	if digits == 0 {
		return i, parseErrorAt(i, fmt.Errorf("number exp: unexpected 0x%X", buf[i]))
	}
	return i, nil
}

func (p *parser) parseNumberFrac(buf []byte, initialPos int) (end int, err error) {
	i := initialPos
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), fmt.Errorf("number frac: %w", io.ErrUnexpectedEOF))
	}
	if buf[i] != '.' {
		return i, nil
	}
	i++
	digits := 0
	for i < len(buf) && '0' <= buf[i] && buf[i] <= '9' {
		digits++
		i++
	}
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), fmt.Errorf("number frac: %w", io.ErrUnexpectedEOF))
	}
	if digits == 0 {
		return i, parseErrorAt(i, fmt.Errorf("number frac: unexpected 0x%X", buf[i]))
	}
	return i, nil
}

func (p *parser) parseNumberInt(buf []byte, initialPos int) (end int, err error) {
	i := initialPos
	if i < len(buf) && buf[i] == '0' {
		return i + 1, nil
	}
	digits := 0
	for i < len(buf) && '0' <= buf[i] && buf[i] <= '9' {
		digits++
		i++
	}
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), fmt.Errorf("number int: %w", io.ErrUnexpectedEOF))
	}
	if digits == 0 {
		return i, parseErrorAt(i, fmt.Errorf("number int: unexpected 0x%X", buf[i]))
	}
	return i, nil
}

func (p *parser) parseNumberMinus(buf []byte, initialPos int) (end int, err error) {
	i := initialPos
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), fmt.Errorf("number minus: %w", io.ErrUnexpectedEOF))
	}
	if buf[i] != '-' {
		return i, nil
	}
	return i + 1, nil
}

func (p *parser) parseObject(depth int, buf []byte, initialPos int) (end int, err error) {
	if p.depthLimitReached(depth) {
		return initialPos, parseErrorAt(initialPos, errors.New("object: depth limit reached"))
	}
	i := initialPos
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), io.ErrUnexpectedEOF)
	}
	if b := buf[i]; b != '{' {
		return i + 1, parseErrorAt(i, fmt.Errorf("object: unexpected 0x%X", b))
	}
	i++
	numPairs := 0
	for {
		i = skipWhitespace(buf, i)
		// allow a trailing comma before the end of the object
		if numPairs > 0 {
			i, err = p.parseObjectComma(buf, i)
			if err == errEndObject {
				return i, nil
			}
			if err != nil {
				return i, err
			}
			i = skipWhitespace(buf, i)
		}
		if i < len(buf) && buf[i] == '}' {
			return i + 1, nil
		}
		keyEnd, err := p.parseString(buf, i)
		if err != nil {
			return keyEnd, err
		}
		i = skipWhitespace(buf, keyEnd)
		i, err = p.parseColon(buf, i)
		if err != nil {
			return i, err
		}
		i = skipWhitespace(buf, i)
		valueEnd, err := p.parseValue(depth, buf, i)
		if err != nil {
			return valueEnd, err
		}
		i = valueEnd
		numPairs++
	}
}

func (p *parser) parseObjectComma(buf []byte, initialPos int) (end int, err error) {
	i := initialPos
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), io.ErrUnexpectedEOF)
	}
	b := buf[i]
	if b == '}' {
		return i + 1, errEndObject
	}
	if b != ',' {
		return i + 1, parseErrorAt(i, fmt.Errorf("object comma: unexpected 0x%X", b))
	}
	return i + 1, nil
}

func (p *parser) parseString(buf []byte, initialPos int) (end int, err error) {
	i := initialPos
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), io.ErrUnexpectedEOF)
	}
	b := buf[i]
	if b != '"' {
		return i + 1, parseErrorAt(i, fmt.Errorf("string: unexpected 0x%X", b))
	}
	i++
	escapeNext := false
	for i < len(buf) {
		b := buf[i]
		switch b {
		case '\\':
			if escapeNext {
				escapeNext = false
			} else {
				escapeNext = true
			}
		case '"':
			if !escapeNext {
				return i + 1, nil
			}
			escapeNext = false
		default:
			escapeNext = false
		}
		i++
	}
	return len(buf), parseErrorAt(len(buf), io.ErrUnexpectedEOF)
}

func (p *parser) parseTrue(buf []byte, initialPos int) (end int, err error) {
	raw := [4]byte{'t', 'r', 'u', 'e'}
	for r := 0; r < len(raw); r++ {
		i := initialPos + r
		if i >= len(buf) {
			return i, parseErrorAt(i, io.ErrUnexpectedEOF)
		}
		b := buf[i]
		if b != raw[r] {
			return i + 1, parseErrorAt(initialPos+r, fmt.Errorf("true: unexpected 0x%X", b))
		}
	}
	return initialPos + len(raw), nil
}

func (p *parser) parseValue(depth int, buf []byte, initialPos int) (end int, err error) {
	i := initialPos
	if i >= len(buf) {
		return len(buf), parseErrorAt(len(buf), io.ErrUnexpectedEOF)
	}
	b := buf[i]
	switch {
	case b == '"':
		return p.parseString(buf, i)
	case b == '-' || ('0' <= b && b <= '9'):
		return p.parseNumber(buf, i)
	case b == 't':
		return p.parseTrue(buf, i)
	case b == 'f':
		return p.parseFalse(buf, i)
	case b == 'n':
		return p.parseNull(buf, i)
	case b == '[':
		return p.parseArray(depth+1, buf, i)
	case b == '{':
		return p.parseObject(depth+1, buf, i)
	default:
		return i + 1, parseErrorAt(i, fmt.Errorf("value: unexpected: 0x%X", b))
	}
}

// the fastest method
// https://dave.cheney.net/high-performance-json.html
var whitespace = [256]bool{
	' ':  true,
	'\t': true,
	'\n': true,
	'\r': true,
}

func skipWhitespace(buf []byte, initialPos int) (pos int) {
	pos = initialPos
	for pos < len(buf) {
		if c := buf[pos]; whitespace[c] {
			pos++
			continue
		}
		break
	}
	return pos
}
