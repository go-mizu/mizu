package log

import (
	"log/slog"
	"time"
)

// Options is where an application's records go and what they look like when
// they get there. It is what [New] takes.
//
// Everything in it is a string, a number or a duration, so a configuration can
// fill it in from files, the environment and the command line. An application
// embeds it:
//
//	type Config struct {
//		Log log.Options
//		DB  struct{ DSN string }
//	}
//
// and the settings arrive as log.level, log.output and the rest, in whatever
// layer the application reads. The tags below are what the generated loading
// code uses when nobody set a value, and they are also the zero value of this
// struct wherever the two can agree, so a program that configures nothing gets
// info level records on standard error.
type Options struct {
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
