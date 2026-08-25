package config

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu/errs/diag"
	"github.com/go-mizu/mizu/toml"
)

// An Error is something wrong with a configuration source, at the place it
// went wrong.
//
// A configuration problem is nearly always one line of one file, and an error
// that names the line is the difference between fixing it and looking for it.
// Errors from the TOML parser arrive as this type too, so a caller has one
// thing to match on.
type Error struct {
	File string
	Line int
	Col  int // zero when the source has no columns, as .env files do not
	Msg  string
	Err  error // the underlying error, when there is one
}

func (e *Error) Error() string {
	var b strings.Builder
	if e.File != "" {
		b.WriteString(e.File)
		if e.Line > 0 {
			b.WriteByte(':')
			b.WriteString(strconv.Itoa(e.Line))
			if e.Col > 0 {
				b.WriteByte(':')
				b.WriteString(strconv.Itoa(e.Col))
			}
		}
		b.WriteString(": ")
	}
	b.WriteString(e.Msg)
	return b.String()
}

func (e *Error) Unwrap() error { return e.Err }

// wrap turns a TOML error into a config error, so that reading configuration
// does not hand callers an error type from a package they never named.
func wrap(err error) error {
	var terr *toml.Error
	if errors.As(err, &terr) {
		return &Error{
			File: terr.Pos.File,
			Line: terr.Pos.Line,
			Col:  terr.Pos.Col,
			Msg:  terr.Msg,
			Err:  err,
		}
	}
	return err
}

// An Unknown is a setting that was written down but that nothing asked for.
//
// It is almost always a typo, and the whole point of reporting it is that a
// misspelled setting otherwise does nothing at all and says nothing about it.
type Unknown struct {
	Path string   // the dotted path, as written
	From Source   // the file and line, or the flag
	Near []string // the settings that do exist and are close, closest first
}

func (u Unknown) Error() string {
	var b strings.Builder
	b.WriteString(u.From.Where())
	b.WriteString(": unknown setting ")
	b.WriteString(strconv.Quote(u.Path))
	if did := diag.Did(u.Near, strconv.Quote); did != "" {
		b.WriteString(", ")
		b.WriteString(did)
	}
	return b.String()
}
