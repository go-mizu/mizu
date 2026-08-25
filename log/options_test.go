package log_test

import (
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/go-mizu/mizu/config"
	"github.com/go-mizu/mizu/log"
)

// TestOptionsDefaults reads every default written on [log.Options] with the
// parsers that will read it, so a default nothing can parse is a failing test
// here rather than a program that will not start.
//
// This is a test importing config rather than the other way around. The tags
// are a promise to whatever loads a configuration, and config is what loads
// one, so the promise is worth checking against the real parsers. A test
// dependency reaches nobody: what a user of log gets is log and the standard
// library.
func TestOptionsDefaults(t *testing.T) {
	walk(t, reflect.TypeOf(log.Options{}))
}

func walk(t *testing.T, st reflect.Type) {
	t.Helper()

	for i := range st.NumField() {
		f := st.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Type != reflect.TypeOf(time.Duration(0)) {
			walk(t, f.Type)
			continue
		}

		text, ok := f.Tag.Lookup("default")
		if !ok {
			continue
		}
		if err := parseDefault(f.Type, fromText(text)); err != nil {
			t.Errorf("%s has default %q, which does not read: %v", f.Name, text, err)
		}
	}
}

// fromText is a value the way every layer except a file has it.
func fromText(s string) config.Value {
	return config.Value{Source: config.Source{From: config.FromEnv, Name: "TEST"}, Text: s}
}

// parseDefault reads a value into a field of this type, the way the generated
// code for a configuration would.
func parseDefault(t reflect.Type, v config.Value) error {
	switch t {
	case reflect.TypeOf(slog.Level(0)):
		var l slog.Level
		return config.Level(&l, v)
	case reflect.TypeOf(time.Duration(0)):
		var d time.Duration
		return config.Duration(&d, v)
	}

	switch t.Kind() {
	case reflect.String:
		var s string
		return config.String(&s, v)
	case reflect.Bool:
		var b bool
		return config.Bool(&b, v)
	case reflect.Int:
		var n int
		return config.Int(&n, v)
	}
	panic("log: TestOptionsDefaults has no parser for " + t.String())
}

// TestOptionsZero is what a program that configures nothing gets, since the
// zero value is what a struct starts as whatever the tags say.
func TestOptionsZero(t *testing.T) {
	var o log.Options

	if o.Level != slog.LevelInfo {
		t.Errorf("the zero level is %v, want info", o.Level)
	}
	if o.Format != "" || o.Output != "" {
		t.Errorf("the zero value asks for format %q output %q", o.Format, o.Output)
	}
	if o.Sampling.Enabled {
		t.Error("the zero value samples")
	}
}
