package config_test

import (
	"errors"
	"log/slog"
	"net/netip"
	"testing"
	"time"

	"github.com/go-mizu/mizu/config"
	"github.com/go-mizu/mizu/errs/diag/diagtest"
)

// TestDiagnostics runs the golden message corpus for this package.
//
// Each directory under testdata/diag holds an app.toml that is wrong in one
// way, and a want.txt, which is what somebody starting a program with that file
// sees. The file goes through the whole of loading: it is parsed, the fields
// below are read out of it, and the settings nothing asked for are reported.
//
// A configuration mistake is nearly always one line of one file, so these are
// the entries where the file and the line in the message are the point. The
// path to the case directory is taken back out of the report, which is why the
// golden files say app.toml rather than testdata/diag/whatever/app.toml.
//
// Run it with -update to rewrite the want.txt files, then read the diff. That
// diff is user-facing text and the five rules in doc 36 section 2.1 are the
// review checklist for it.
func TestDiagnostics(t *testing.T) {
	diagtest.Run(t, "testdata/diag", func(tb testing.TB, c diagtest.Case) error {
		return load(c.Path("app.toml"))
	})
}

// settings is what the program the corpus stands in for reads.
//
// A real one has fifty of these and generated code to fill them in. Five is
// enough for a message about the sixth, which is the whole subject here.
type settings struct {
	Name     string
	Debug    bool
	Level    slog.Level
	Listen   netip.AddrPort
	Timeout  time.Duration
	MaxConns int
	Secret   string
}

// load reads one file the way an application would.
//
// Parsing comes first and stops everything if it fails, because a file that is
// not TOML has no settings to disagree about. Then every field is read, so that
// somebody with three wrong values hears about three of them, and then the
// settings nothing asked for are reported, which is where a typo turns up.
func load(path string) error {
	l, err := config.Open(config.Sources{Env: "production", Files: []string{path}})
	if err != nil {
		return err
	}

	var s settings
	config.Get(l, &s.Name, config.Field{Name: "App.Name", Path: "app.name", Env: "APP_NAME"}, config.String)
	config.Get(l, &s.Debug, config.Field{Name: "App.Debug", Path: "app.debug", Env: "APP_DEBUG"}, config.Bool)
	config.Get(l, &s.Level, config.Field{Name: "Log.Level", Path: "log.level", Env: "LOG_LEVEL"}, config.Level)
	config.Get(l, &s.Listen, config.Field{Name: "HTTP.Listen", Path: "http.listen", Env: "HTTP_LISTEN"}, config.AddrPort)
	config.Get(l, &s.Timeout, config.Field{Name: "HTTP.Timeout", Path: "http.timeout", Env: "HTTP_TIMEOUT"}, config.Duration)
	config.Get(l, &s.MaxConns, config.Field{Name: "DB.MaxOpenConns", Path: "database.max_open_conns", Env: "DB_MAX_OPEN_CONNS"}, config.Int)
	config.Get(l, &s.Secret, config.Field{Name: "App.Key", Path: "app.key", Env: "APP_KEY", Secret: true}, config.String)

	return errors.Join(l.Err(), l.Check())
}
