package config

import (
	"log/slog"
	"time"
)

// Log is where an application's records go and what they look like when they
// get there.
//
// It lives in this package rather than in the log package because both ends
// have to name it. Generated loading code fills it in from files, the
// environment and the command line, and log.New turns the result into a
// [log/slog.Logger]. An application embeds it:
//
//	type Config struct {
//		Log config.Log
//		DB  struct{ DSN string }
//	}
//
// and the settings arrive as log.level, log.output and the rest, in whatever
// layer the application reads.
type Log struct {
	// Level is the lowest level that gets written. Everything below it costs
	// the check and nothing else.
	Level slog.Level `default:"info"`

	// Format is console or json. Empty picks console when the output is a
	// terminal and json when it is anything else, which is a pipe, a file or a
	// container, so a developer gets lines to read and a server gets objects to
	// collect without either of them saying which.
	Format string

	// Output is stderr, stdout, or the path of a file. A path is created along
	// with the directories above it, and rotated according to Rotate.
	Output string `default:"stderr"`

	// AddSource adds the file and line each record was made at. In the json
	// format it also adds the stack of an error, since both answer the same
	// question and both cost the same kind of work.
	AddSource bool `default:"false"`

	// Sampling drops records that repeat, so that one message in a loop cannot
	// fill a disk or a bill. Errors are written whatever it says.
	Sampling struct {
		Enabled bool `default:"false"`

		// Initial is how many records of the same level and message are written
		// in each interval before sampling starts.
		Initial int `default:"100"`

		// Every is the one in Every that is written after that.
		Every int `default:"100"`

		// Interval is how long a count lasts before it starts again.
		Interval time.Duration `default:"1s"`
	}

	// Rotate is what happens to a file Output when it gets big or old. It is
	// read only when Output is a path.
	Rotate struct {
		// MaxSizeMB is how large the file gets before it is renamed out of the
		// way and a new one is started.
		MaxSizeMB int `default:"100"`

		// MaxAge is how long a renamed file is kept.
		MaxAge time.Duration `default:"336h"`

		// MaxFiles is how many renamed files are kept, whatever their age.
		MaxFiles int `default:"10"`

		// Compress gzips a renamed file, which is where most of the disk goes
		// back, since a log compresses to about a tenth of itself.
		Compress bool `default:"true"`
	}
}
