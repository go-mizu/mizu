package main

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"runtime/trace"

	"github.com/go-mizu/mizu/console"
)

// globals are the flags every mizu command takes on top of the ones in
// [console.Globals].
//
// The ones in console are about what a command says. These are about the run
// itself, which is why they are here and why they are answered in one place
// rather than by each command remembering to.
type globals struct {
	Profile string
	Trace   string
}

func (g *globals) flags() []console.Flag {
	return []console.Flag{
		{Name: "profile", Desc: "Write a CPU profile of this run to a file", Value: console.String(&g.Profile)},
		{Name: "trace", Desc: "Write an execution trace of this run to a file", Value: console.String(&g.Trace)},
	}
}

// before starts what the flags asked for and returns what stops it again.
//
// It is [console.App.Before], so it runs once a command line has been
// understood and its result runs when the command has finished. That is the
// window worth measuring: not the argument parsing, and not the time spent
// printing an error afterwards.
//
// A profile that could not be started is an error rather than a warning.
// Somebody who passed the flag is going to read the file, and a run that
// quietly did not write one is a run they have to do again.
func (g *globals) before(ctx context.Context, c *console.IO) (context.Context, func(), error) {
	var stops []func()
	done := func() {
		for i := len(stops) - 1; i >= 0; i-- {
			stops[i]()
		}
	}

	for _, start := range []struct {
		name string
		file string
		fn   func(*console.IO, string) (func(), error)
	}{
		{"profile", g.Profile, startProfile},
		{"trace", g.Trace, startTrace},
	} {
		if start.file == "" {
			continue
		}
		stop, err := start.fn(c, start.file)
		if err != nil {
			// Whatever is already running has to be stopped, or a --profile
			// that worked leaves a file nothing ever closed.
			done()
			return nil, nil, fmt.Errorf("--%s: %w", start.name, err)
		}
		stops = append(stops, stop)
	}

	if len(stops) == 0 {
		return nil, nil, nil
	}
	return nil, done, nil
}

// startProfile writes a CPU profile of everything that happens until the
// returned function is called.
func startProfile(c *console.IO, name string) (func(), error) {
	f, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		pprof.StopCPUProfile()
		write(c, f, "CPU profile", "go tool pprof "+name)
	}, nil
}

// startTrace writes an execution trace of everything that happens until the
// returned function is called.
func startTrace(c *console.IO, name string) (func(), error) {
	f, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	if err := trace.Start(f); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		trace.Stop()
		write(c, f, "execution trace", "go tool trace "+name)
	}, nil
}

// write closes the file the profiler was writing to and says where it went.
//
// The note names the command to read it with, because the file is useless
// without one and the tool is not the same for the two of them. It goes to
// stderr, where it cannot land in the middle of what the command printed.
//
// A close that fails is a warning rather than an error: the command has
// already finished and taking its exit code away now would say the work did
// not happen when it did.
func write(c *console.IO, f *os.File, what, how string) {
	if err := f.Close(); err != nil {
		c.Warn("%s: %v", what, err)
		return
	}
	c.Info("%s written to %s, read it with %s", what, f.Name(), how)
}
