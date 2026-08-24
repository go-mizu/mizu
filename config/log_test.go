package config

import (
	"log/slog"
	"reflect"
	"testing"
	"time"
)

// TestLogDefaults reads every default written on [Log] with the parser that
// will read it, so a default nothing can parse is a failing test here rather
// than a program that will not start.
func TestLogDefaults(t *testing.T) {
	walk(t, reflect.TypeOf(Log{}))
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

// parseDefault reads a value into a field of this type, the way the generated
// code for a configuration would.
func parseDefault(t reflect.Type, v Value) error {
	switch t {
	case reflect.TypeOf(slog.Level(0)):
		var l slog.Level
		return Level(&l, v)
	case reflect.TypeOf(time.Duration(0)):
		var d time.Duration
		return Duration(&d, v)
	}

	switch t.Kind() {
	case reflect.String:
		var s string
		return String(&s, v)
	case reflect.Bool:
		var b bool
		return Bool(&b, v)
	case reflect.Int:
		var n int
		return Int(&n, v)
	}
	panic("config: TestLogDefaults has no parser for " + t.String())
}

// TestLogZero is what a program that configures nothing gets, since the zero
// value is what a struct starts as whatever the tags say.
func TestLogZero(t *testing.T) {
	var cfg Log

	if cfg.Level != slog.LevelInfo {
		t.Errorf("the zero level is %v, want info", cfg.Level)
	}
	if cfg.Format != "" || cfg.Output != "" {
		t.Errorf("the zero value asks for format %q output %q", cfg.Format, cfg.Output)
	}
	if cfg.Sampling.Enabled {
		t.Error("the zero value samples")
	}
}
