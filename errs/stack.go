package errs

import (
	"log/slog"
	"runtime"
	"strconv"
)

// depth is how many frames are kept. Thirty two is past the handler, past the
// middleware and into the server, which is further than anyone reads.
const depth = 32

// skip is the frames between runtime.Callers and whoever called the
// constructor: Callers itself, capture, and the constructor.
const skip = 3

// A Frame is one line of a stack trace.
type Frame struct {
	// Func is the fully qualified function name, such as
	// blog/post.(*Store).ByID.
	Func string

	// File is the absolute path of the source file, as the compiler recorded
	// it, and Line is the line in it.
	File string
	Line int
}

// String is the frame as a compiler writes one, file and line first, so an
// editor and a terminal both make it clickable.
func (f Frame) String() string {
	return f.File + ":" + strconv.Itoa(f.Line) + " " + f.Func
}

// capture records where a failure happened, if it is worth recording.
//
// Nothing is captured for a kind that does not log at error level, since a
// stack that is never printed is a kilobyte of allocation per request for
// nothing, and nothing is captured when something below already has one, so
// the deepest capture is the one that survives wrapping.
func capture(k Kind, cause error, extra int) []uintptr {
	if k.Level() < slog.LevelError || hasStack(cause) {
		return nil
	}
	pc := make([]uintptr, depth)
	return pc[:runtime.Callers(skip+extra, pc)]
}

// hasStack is whether anything in the chain already knows where it happened.
func hasStack(err error) bool {
	return walk(err, func(e error) bool {
		x, ok := e.(*Error)
		return ok && x.stack != nil
	})
}

// StackTrace is where this error was built, innermost frame first, or nil for
// an error that did not capture one.
//
// It resolves the program counters on the way out rather than at capture time,
// because most errors are counted, logged as one line and dropped, and the
// resolution is most of the cost.
func (e *Error) StackTrace() []Frame {
	if len(e.stack) == 0 {
		return nil
	}
	out := make([]Frame, 0, len(e.stack))
	frames := runtime.CallersFrames(e.stack)
	for {
		f, more := frames.Next()
		if f.Function != "" || f.File != "" {
			out = append(out, Frame{Func: f.Function, File: f.File, Line: f.Line})
		}
		if !more {
			return out
		}
	}
}

// Stack is the deepest stack trace in the chain, which is the one that says
// where the failure started rather than where it was described.
func Stack(err error) []Frame {
	var out []Frame
	walk(err, func(e error) bool {
		if x, ok := e.(*Error); ok && x.stack != nil {
			out = x.StackTrace()
			return true
		}
		return false
	})
	return out
}
