package config

import (
	"errors"
	"os"
	"strings"
)

// Get reads one field into dst.
//
// It finds the value the way [Loader.Lookup] does, resolves any indirection,
// and hands the result to parse. A field that no layer has a value for leaves
// dst alone, so the zero value the struct came with stands.
//
//	config.Get(l, &c.App.Name, config.Field{Path: "app.name", Env: "APP_NAME"}, config.String)
//
// Nothing is returned. A field that will not parse is recorded and the rest
// are read anyway, because someone who has three settings wrong wants to hear
// about three of them and not one at a time. [Loader.Err] is the answer at the
// end.
func Get[T any](l *Loader, dst *T, f Field, parse Parse[T]) {
	v, ok := l.Lookup(f)
	if !ok {
		return
	}
	v, err := l.indirect(f, v)
	if err == nil {
		err = parse(dst, v)
	}
	if err != nil {
		l.errs = append(l.errs, &FieldError{Field: f, Value: v, Err: err})
	}
}

// A FieldError is a value that would not read as what the field is.
type FieldError struct {
	Field Field
	Value Value
	Err   error
}

func (e *FieldError) Error() string {
	name := e.Field.Name
	if name == "" {
		name = e.Field.Path
	}

	var b strings.Builder
	if where := e.where(); where != "" {
		b.WriteString(where)
		b.WriteString(": ")
	}
	if name != "" {
		b.WriteString(name)
		b.WriteString(": ")
	}
	b.WriteString(e.Err.Error())
	return b.String()
}

// where is the place to go and fix it: a file and a line for a file, and the
// name of the layer for everything else. A default is nobody's fault but the
// program's, so it says nothing.
func (e *FieldError) where() string {
	if e.Value.Source.From == FromDefault {
		return ""
	}
	return e.Value.Source.Where()
}

func (e *FieldError) Unwrap() error { return e.Err }

// Err is every field that would not read, joined into one error, or nil when
// they all read. Call it after the last [Get].
func (l *Loader) Err() error {
	if len(l.errs) == 0 {
		return nil
	}
	errs := make([]error, len(l.errs))
	for i, e := range l.errs {
		errs[i] = e
	}
	return errors.Join(errs...)
}

// Errors are the fields that would not read, in the order they were asked
// for. It is what config:check prints as a table, and [Loader.Err] is the same
// thing for a caller that only wants to return it.
func (l *Loader) Errors() []*FieldError { return l.errs[:len(l.errs):len(l.errs)] }

// The prefixes that make a value point somewhere else rather than being the
// value itself.
const (
	filePrefix    = "file:"
	envPrefix     = "env:"
	commandPrefix = "cmd:"
)

// indirect follows file:, env: and cmd: in the value of a secret field.
//
// It only applies to a field marked secret, which is what these are for. A
// setting that is not a secret can hold a file:// URL without it turning into
// a path, and a database DSN that reads its own file is a real thing.
//
// What was written stays in the setting the loader recorded, so config:show
// prints file:/run/secrets/db rather than what was in the file.
func (l *Loader) indirect(f Field, v Value) (Value, error) {
	if !f.Secret {
		return v, nil
	}
	s, err := v.Str()
	if err != nil || s == "" {
		return v, nil // let the parser be the one to complain about the kind
	}

	switch {
	case strings.HasPrefix(s, filePrefix):
		data, err := os.ReadFile(strings.TrimPrefix(s, filePrefix))
		if err != nil {
			return v, err
		}
		// A file written by a person ends with a newline, and a secret with a
		// newline on the end of it is a long afternoon.
		s = strings.TrimRight(string(data), "\r\n")

	case strings.HasPrefix(s, envPrefix):
		name := strings.TrimPrefix(s, envPrefix)
		text, ok := l.environ[name]
		if !ok {
			if e, found := l.dotenv[name]; found {
				text, ok = e.value, true
			}
		}
		if !ok {
			return v, errors.New(name + " is not set, and " + s + " says to read it")
		}
		s = text

	case strings.HasPrefix(s, commandPrefix):
		if l.command == nil {
			return v, errors.New("running a command for a value is turned off here, so " + s + " cannot be read")
		}
		out, err := l.command(strings.TrimPrefix(s, commandPrefix))
		if err != nil {
			return v, err
		}
		s = strings.TrimRight(out, "\r\n")

	default:
		return v, nil
	}

	// The value is now the text that was pointed at, and it still came from
	// where the pointer was written.
	return Value{Source: v.Source, Text: s}, nil
}
