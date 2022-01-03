package zord

import (
	"strconv"
)

type errorAt interface {
	Error() string
	Pos() int
}

type errAt struct {
	err error
	pos int
}

func (e *errAt) Error() string {
	if e.err == nil {
		return "error at position " + strconv.Itoa(e.pos)
	}
	return "error at position " + strconv.Itoa(e.pos) + ": " + e.err.Error()
}

func (e *errAt) Pos() int {
	return e.pos
}

func (e *errAt) Unwrap() error {
	return e.err
}

func parseErrorAt(pos int, err error) errorAt {
	if err == nil {
		return nil
	}
	if errp, ok := err.(errorAt); ok {
		if errp.Pos() == pos {
			return errp
		}
	}
	return &errAt{
		err: err,
		pos: pos,
	}
}
