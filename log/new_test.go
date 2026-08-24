package log

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/config"
	"github.com/go-mizu/mizu/errs"
)

// intoFile is the configuration a test can read back afterwards: a file in a
// directory of its own.
func intoFile(t *testing.T, cfg config.Log) (config.Log, string) {
	t.Helper()
	cfg.Output = filepath.Join(t.TempDir(), "app.log")
	return cfg, cfg.Output
}

func TestNew(t *testing.T) {
	cfg, path := intoFile(t, config.Log{Level: slog.LevelInfo, Format: "json"})

	logger, closer, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debug("not this one")
	logger.Info("started", "port", 8080)
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}

	got := read(t, path)
	if strings.Count(got, "\n") != 1 {
		t.Fatalf("the file holds:\n%s", got)
	}
	if !strings.Contains(got, `"msg":"started"`) || !strings.Contains(got, `"port":8080`) {
		t.Errorf("the file holds %s", got)
	}
}

// TestNewFormat covers the choice a configuration makes and the one it leaves
// to the output. A file is not a terminal, so an empty format writes JSON.
func TestNewFormat(t *testing.T) {
	cases := map[string]any{
		"console": &console{},
		"json":    &jsonHandler{},
		"":        &jsonHandler{},
	}
	for format, want := range cases {
		cfg, _ := intoFile(t, config.Log{Format: format})
		logger, closer, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}
		closer.Close()

		if got := logger.Handler(); reflectKind(got) != reflectKind(want) {
			t.Errorf("format %q built %T, want %T", format, got, want)
		}
	}
}

// reflectKind is the type of a handler as a string, which is all these tests
// compare. Naming the types keeps this from importing reflect for one line.
func reflectKind(h any) string {
	switch h.(type) {
	case *console:
		return "console"
	case *jsonHandler:
		return "json"
	case *sampler:
		return "sampler"
	}
	return "unknown"
}

func TestNewOutput(t *testing.T) {
	cases := map[string]io.Writer{
		"":       os.Stderr,
		"stderr": os.Stderr,
		"stdout": os.Stdout,
	}
	for output, want := range cases {
		logger, closer, err := New(config.Log{Output: output, Format: "json"})
		if err != nil {
			t.Fatal(err)
		}

		h, ok := logger.Handler().(*jsonHandler)
		if !ok {
			t.Fatalf("output %q built %T", output, logger.Handler())
		}
		if h.w != want {
			t.Errorf("output %q writes to %v, want %v", output, h.w, want)
		}

		// The closer for a file this package did not open does nothing, which
		// is what keeps a deferred Close from taking standard error away from
		// the rest of the program.
		if _, isFile := closer.(*File); isFile {
			t.Errorf("output %q returned a file to close", output)
		}
		if err := closer.Close(); err != nil {
			t.Errorf("closing output %q: %v", output, err)
		}
	}
}

// TestNewRotate checks the rotation settings arrive, since they are read once
// at startup and nothing says so afterwards.
func TestNewRotate(t *testing.T) {
	cfg, _ := intoFile(t, config.Log{Format: "json"})
	cfg.Rotate.MaxSizeMB = 5
	cfg.Rotate.MaxAge = time.Hour
	cfg.Rotate.MaxFiles = 3
	cfg.Rotate.Compress = true

	_, closer, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer closer.Close()

	f, ok := closer.(*File)
	if !ok {
		t.Fatalf("a path output returned %T, want a file", closer)
	}
	if f.max != 5<<20 || f.age != time.Hour || f.keep != 3 || !f.compress {
		t.Errorf("the file rotates at %d bytes, %v, %d files, compress %v", f.max, f.age, f.keep, f.compress)
	}
}

func TestNewSampling(t *testing.T) {
	cfg, _ := intoFile(t, config.Log{Format: "json"})
	cfg.Sampling.Enabled = true
	cfg.Sampling.Initial = 2
	cfg.Sampling.Every = 5
	cfg.Sampling.Interval = time.Minute

	logger, closer, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer closer.Close()

	s, ok := logger.Handler().(*sampler)
	if !ok {
		t.Fatalf("sampling built %T, want a sampler", logger.Handler())
	}
	if s.initial != 2 || s.every != 5 || s.interval != int64(time.Minute) {
		t.Errorf("the sampler keeps %d then one in %d every %v", s.initial, s.every, time.Duration(s.interval))
	}

	cfg.Sampling.Enabled = false
	logger, closer, err = New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer closer.Close()
	if _, ok := logger.Handler().(*sampler); ok {
		t.Error("sampling was turned off and a sampler was built anyway")
	}
}

func TestNewAddSource(t *testing.T) {
	cfg, path := intoFile(t, config.Log{Format: "json", AddSource: true})

	logger, closer, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	logger.Error("job failed", "err", errs.Internalf("nothing works"))
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}

	got := read(t, path)
	if !strings.Contains(got, `"source":"`) || !strings.Contains(got, "new_test.go:") {
		t.Errorf("the record has no source:\n%s", got)
	}
	if !strings.Contains(got, `"stack":`) {
		t.Errorf("the record has no stack:\n%s", got)
	}
}

// TestNewLevel is the setting people change most often, and the one they change
// while something is on fire.
func TestNewLevel(t *testing.T) {
	cfg, path := intoFile(t, config.Log{Level: slog.LevelWarn, Format: "console"})

	logger, closer, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("quiet")
	logger.Warn("loud")
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}

	got := read(t, path)
	if strings.Contains(got, "quiet") || !strings.Contains(got, "loud") {
		t.Errorf("the file holds:\n%s", got)
	}
}

func TestNewFailures(t *testing.T) {
	_, _, err := New(config.Log{Format: "logfmt"})
	if err == nil {
		t.Fatal("a format nothing writes was accepted")
	}
	if errs.KindOf(err) != errs.Invalid {
		t.Errorf("the error is %v, want invalid", errs.KindOf(err))
	}
	if !strings.Contains(err.Error(), "logfmt") {
		t.Errorf("the error is %q, and does not say which format", err)
	}

	if _, _, err := New(config.Log{Output: t.TempDir()}); err == nil {
		t.Error("an output that is a directory was opened")
	}
}

// TestIsTerminal is what decides the format when a configuration does not. The
// case that matters is the one a server runs in, where the output is a file or
// a pipe and the answer is no.
func TestIsTerminal(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if isTerminal(f) {
		t.Error("a file is a terminal")
	}
	if isTerminal(io.Discard) {
		t.Error("something that is not a file at all is a terminal")
	}
}

// TestFormatOf is the other side of it, including the terminal a test does not
// have. The null device is a character device on the systems that have one,
// which is the same answer a terminal gives.
func TestFormatOf(t *testing.T) {
	if got := formatOf(io.Discard, "console"); got != "console" {
		t.Errorf("a format that was asked for came back as %q", got)
	}
	if got := formatOf(io.Discard, ""); got != "json" {
		t.Errorf("an output that is not a terminal chose %q, want json", got)
	}

	dev, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("no null device here: %v", err)
	}
	defer dev.Close()
	if !isTerminal(dev) {
		t.Skip("the null device is not a character device here")
	}
	if got := formatOf(dev, ""); got != "console" {
		t.Errorf("an output that is a terminal chose %q, want console", got)
	}
}
