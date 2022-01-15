package zord

import (
	"errors"
	"fmt"
	"io"

	"github.com/7fffffff/jsonconv"
)

var (
	errEndArray  = errors.New("]")
	errEndObject = errors.New("}")
	errMaxDepth  = errors.New("exceeded max depth")
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
	MaxDepth int // maximum nesting depth. If 0, defaultMaxDepth is used
}

func (p *parser) depthLimitReached(depth int) bool {
	maxDepth := p.MaxDepth
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
	if n >= len(buf) {
		return pairs, len(buf), parseErrorAt(len(buf), io.ErrUnexpectedEOF)
	}
	if b := buf[n]; b != '{' {
		return pairs, n + 1, parseErrorAt(n, fmt.Errorf("object: unexpected: 0x%X", b))
	}
	n++
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
		return initialPos, parseErrorAt(initialPos, fmt.Errorf("array: %w", errMaxDepth))
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
	// NumberMinus
	if i < len(buf) && buf[i] == '-' {
		i++
	}
	// NumberInt
	leadingZero := false
	digits := 0
	if i < len(buf) && buf[i] == '0' {
		leadingZero = true
		digits++
		i++
	}
	for i < len(buf) {
		b := buf[i]
		if !leadingZero && '0' <= b && b <= '9' {
			digits++
			i++
			continue
		}
		if digits > 0 {
			switch b {
			case '.':
				i++
				goto NumberFrac
			case 'e', 'E':
				i++
				goto NumberExp
			}
		}
		break
	}
	if i >= len(buf) {
		return len(buf), parseErrorAt(i, fmt.Errorf("number int: %w", io.ErrUnexpectedEOF))
	}
	if digits == 0 {
		return i + 1, parseErrorAt(i, fmt.Errorf("number int: unexpected 0x%X", buf[i]))
	}
	return i, nil
NumberFrac:
	digits = 0
	for i < len(buf) {
		b := buf[i]
		if '0' <= b && b <= '9' {
			digits++
			i++
			continue
		}
		if digits > 0 && (b == 'e' || b == 'E') {
			i++
			goto NumberExp
		}
		break
	}
	if i >= len(buf) {
		return len(buf), parseErrorAt(i, fmt.Errorf("number frac: %w", io.ErrUnexpectedEOF))
	}
	if digits == 0 {
		return i + 1, parseErrorAt(i, fmt.Errorf("number frac: unexpected 0x%X", buf[i]))
	}
	return i, nil
NumberExp:
	// optional leading + or -
	if i < len(buf) && (buf[i] == '+' || buf[i] == '-') {
		i++
	}
	digits = 0
	for i < len(buf) {
		b := buf[i]
		if '0' <= b && b <= '9' {
			digits++
			i++
			continue
		}
		break
	}
	if i >= len(buf) {
		return len(buf), parseErrorAt(i, fmt.Errorf("number exp: %w", io.ErrUnexpectedEOF))
	}
	if digits == 0 {
		return i + 1, parseErrorAt(i, fmt.Errorf("number exp: unexpected 0x%X", buf[i]))
	}
	return i, nil
}

func (p *parser) parseObject(depth int, buf []byte, initialPos int) (end int, err error) {
	if p.depthLimitReached(depth) {
		return initialPos, parseErrorAt(initialPos, fmt.Errorf("object: %w", errMaxDepth))
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
		b = buf[i]
		if escapeNext {
			escapeNext = false
			switch b {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				i++
			case 'u':
				i++
				h := 0
				for i < len(buf) {
					b = buf[i]
					if !hexDigits[b] {
						return i + 1, parseErrorAt(i, fmt.Errorf("string: unexpected 0x%X", b))
					}
					h++
					i++
					if h == 4 {
						break
					}
				}
			default:
				return i + 1, parseErrorAt(i, fmt.Errorf("string: unexpected 0x%X", b))
			}
			continue
		}
		switch b {
		case '"':
			return i + 1, nil
		case '\\':
			escapeNext = true
		default:
			if b < ' ' {
				return i + 1, parseErrorAt(i, fmt.Errorf("string: unexpected 0x%X", b))
			}
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
		if c := buf[pos]; !whitespace[c] {
			break
		}
		pos++
	}
	return pos
}

var hexDigits = [256]bool{
	'0': true,
	'1': true,
	'2': true,
	'3': true,
	'4': true,
	'5': true,
	'6': true,
	'7': true,
	'8': true,
	'9': true,
	'A': true,
	'B': true,
	'C': true,
	'D': true,
	'E': true,
	'F': true,
	'a': true,
	'b': true,
	'c': true,
	'd': true,
	'e': true,
	'f': true,
}
