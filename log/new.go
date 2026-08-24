package log

import (
	"io"
	"log/slog"
	"os"

	"github.com/go-mizu/mizu/config"
	"github.com/go-mizu/mizu/errs"
)

// New builds the logger a [config.Log] asks for: the writer from Output, the
// handler from Format, and sampling and rotation from the rest.
//
//	logger, closer, err := log.New(cfg.Log)
//	if err != nil {
//		return err
//	}
//	defer closer.Close()
//	slog.SetDefault(logger)
//
// The closer is the file when Output is a path, and does nothing when it is
// standard error or standard output, which belong to whatever started the
// program. Closing it waits for any compressing a rotation left running.
//
// An empty Format writes lines when the output is a terminal and JSON when it
// is anything else. That covers both of the cases people mean: a developer
// running the program sees columns, and the same binary in a container writes
// objects for whatever collects them. Set Format to say which rather than
// having it depend on where the output went.
//
// Secrets are masked by the [DefaultRedact] list, since a list of key names is
// not the sort of thing to have to remember to configure.
//
// This is the whole of what a configuration can describe. A program that wants
// something else, a second destination or a filter of its own, builds the
// handlers itself and calls [slog.New], which is all this does.
func New(cfg config.Log) (*slog.Logger, io.Closer, error) {
	// Before opening anything, so a typo in a format does not leave a file open
	// with nothing to close it.
	switch cfg.Format {
	case "", "console", "json":
	default:
		return nil, nil, errs.Invalidf("log: format %q, want console or json", cfg.Format)
	}

	w, closer, err := open(cfg)
	if err != nil {
		return nil, nil, err
	}

	var h slog.Handler
	if formatOf(w, cfg.Format) == "console" {
		h = NewConsoleHandler(w, ConsoleOptions{
			Level:     cfg.Level,
			AddSource: cfg.AddSource,
		})
	} else {
		h = NewJSONHandler(w, JSONOptions{
			Level:     cfg.Level,
			AddSource: cfg.AddSource,
			Stack:     cfg.AddSource,
		})
	}

	if s := cfg.Sampling; s.Enabled {
		h = NewSamplingHandler(h, SampleOptions{
			Initial:  s.Initial,
			Every:    s.Every,
			Interval: s.Interval,
		})
	}
	return slog.New(h), closer, nil
}

// formatOf is the format a configuration asked for, or the one the writer
// suggests when it did not.
func formatOf(w io.Writer, format string) string {
	if format != "" {
		return format
	}
	if isTerminal(w) {
		return "console"
	}
	return "json"
}

// open is the writer an Output names, and the closer for it.
func open(cfg config.Log) (io.Writer, io.Closer, error) {
	switch cfg.Output {
	case "", "stderr":
		return os.Stderr, keepOpen{}, nil
	case "stdout":
		return os.Stdout, keepOpen{}, nil
	}

	f, err := NewFile(cfg.Output, RotateOptions{
		MaxSizeMB: cfg.Rotate.MaxSizeMB,
		MaxAge:    cfg.Rotate.MaxAge,
		MaxFiles:  cfg.Rotate.MaxFiles,
		Compress:  cfg.Rotate.Compress,
	})
	if err != nil {
		return nil, nil, err
	}
	return f, f, nil
}

// keepOpen is the closer for standard error and standard output. This package
// did not open them and closing them would take the rest of the program's
// output with it.
type keepOpen struct{}

func (keepOpen) Close() error { return nil }
