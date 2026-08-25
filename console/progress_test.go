package console

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

// TestRender is the bar itself, which is the part a terminal is not needed to
// check. Widths are chosen around the edges: room for the cap, room for less
// than that, and not enough room for a picture at all.
func TestRender(t *testing.T) {
	tests := []struct {
		width, current, total int
		want                  string
	}{
		{80, 50, 100, " 50% ████████████████████░░░░░░░░░░░░░░░░░░░░ 50/100"},
		{80, 0, 100, "  0% ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 0/100"},
		{80, 100, 100, "100% ████████████████████████████████████████ 100/100"},
		{20, 5, 10, " 50% █████░░░░░ 5/10"},
		{12, 5, 10, " 50% 5/10"},
		{80, 1, 0, "100% ████████████████████████████████████████ 1/0"},
	}
	for _, tt := range tests {
		if got := render(tt.width, tt.current, tt.total); got != tt.want {
			t.Errorf("render(%d, %d, %d) =\n%q\nwant\n%q", tt.width, tt.current, tt.total, got, tt.want)
		}
	}
}

func TestPercent(t *testing.T) {
	tests := []struct {
		current, total, want int
	}{
		{0, 100, 0},
		{1, 3, 33},
		{100, 100, 100},
		{0, 0, 100}, // nothing to do is finished, not a division by zero
	}
	for _, tt := range tests {
		if got := percent(tt.current, tt.total); got != tt.want {
			t.Errorf("percent(%d, %d) = %d, want %d", tt.current, tt.total, got, tt.want)
		}
	}
}

// TestBarWithNoTerminal is what a CI log sees. A line per tenth, and no
// escape sequences, because nothing there can act on them.
func TestBarWithNoTerminal(t *testing.T) {
	s := newStreams(t, Options{})

	bar := s.io.Progress(10)
	for range 10 {
		bar.Advance(1)
	}
	bar.Done()

	want := strings.Join([]string{
		"10% (1/10)",
		"20% (2/10)",
		"30% (3/10)",
		"40% (4/10)",
		"50% (5/10)",
		"60% (6/10)",
		"70% (7/10)",
		"80% (8/10)",
		"90% (9/10)",
		"100% (10/10)",
		"",
	}, "\n")
	if got := s.err.String(); got != want {
		t.Errorf("the bar wrote\n%s\nwant\n%s", got, want)
	}
	if got := s.out.String(); got != "" {
		t.Errorf("stdout has %q, and progress is not data", got)
	}
}

// TestBarIsBounded is the reason the lines go by tenth rather than by step. A
// job of a million steps is still ten lines.
func TestBarIsBounded(t *testing.T) {
	s := newStreams(t, Options{})

	bar := s.io.Progress(100_000)
	for range 100_000 {
		bar.Advance(1)
	}
	bar.Done()

	if got := strings.Count(s.err.String(), "\n"); got != 10 {
		t.Errorf("a hundred thousand steps wrote %d lines", got)
	}
}

// TestBarSaysWhereItStopped covers the job that ended early. The bar does not
// jump to the total, because a bar that reads as complete under an error
// message is a lie somebody has to untangle later.
func TestBarSaysWhereItStopped(t *testing.T) {
	s := newStreams(t, Options{})

	bar := s.io.Progress(10)
	bar.Set(4)
	bar.Advance(1)
	bar.Done()

	want := "40% (4/10)\n50% (5/10)\n"
	if got := s.err.String(); got != want {
		t.Errorf("the bar wrote %q, want %q", got, want)
	}
}

// TestBarDoneReportsTheLastState is the other half of it: work that stopped
// between two tenths still gets a line saying so.
func TestBarDoneReportsTheLastState(t *testing.T) {
	s := newStreams(t, Options{})

	bar := s.io.Progress(100)
	bar.Set(4)
	bar.Done()
	bar.Done() // and again, which a deferred one after an early return does

	if got := s.err.String(); got != "4% (4/100)\n" {
		t.Errorf("the bar wrote %q", got)
	}
}

func TestBarClamps(t *testing.T) {
	s := newStreams(t, Options{})

	bar := s.io.Progress(10)
	bar.Set(-5)
	bar.Advance(400)

	if got := s.err.String(); got != "100% (10/10)\n" {
		t.Errorf("the bar wrote %q", got)
	}
}

func TestBarIsSilentWhenNobodyIsReading(t *testing.T) {
	for name, opts := range map[string]Options{
		"quiet": {Verbosity: Quiet},
		"json":  {JSON: true},
	} {
		s := newStreams(t, opts)

		bar := s.io.Progress(10)
		bar.Advance(5)
		bar.Done()

		if got := s.err.String(); got != "" {
			t.Errorf("%s wrote %q", name, got)
		}
	}
}

// TestBarIsSafeForConcurrentUse is the promise the mutex is there for, since a
// worker pool reporting its own progress is how most bars get advanced. Run
// under -race, which CI does.
func TestBarIsSafeForConcurrentUse(t *testing.T) {
	s := newStreams(t, Options{})
	bar := s.io.Progress(100)

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bar.Advance(1)
		}()
	}
	wg.Wait()
	bar.Done()

	if bar.current != 100 {
		t.Errorf("a hundred goroutines advanced the bar to %d", bar.current)
	}
}

// TestBarPaintsInPlace reaches past the terminal check, since a test does not
// have a terminal and this is the half of the bar that only runs on one. What
// it checks is the framing: back to the start of the line, and erase whatever
// a longer bar left behind.
func TestBarPaintsInPlace(t *testing.T) {
	s := newStreams(t, Options{})

	bar := s.io.Progress(10)
	bar.current = 3
	bar.paint()

	got := s.err.String()
	if !strings.HasPrefix(got, "\r") || !strings.HasSuffix(got, "\x1b[K") {
		t.Errorf("the bar painted %q", got)
	}
	if !strings.Contains(got, "30% ") {
		t.Errorf("the bar painted %q, and does not say how far along it is", got)
	}
}

// TestBarWidth covers the override, which is what a test has instead of a
// terminal. The width is measured on stderr rather than on stdout, because
// that is where the bar is drawn: a command whose output is piped still has a
// person watching the other stream.
func TestBarWidth(t *testing.T) {
	s := newStreams(t, Options{Width: 30})

	if got := s.io.Progress(10).width; got != 30 {
		t.Errorf("the bar is %d columns wide, want 30", got)
	}
}

func TestSpinnerWithNoTerminal(t *testing.T) {
	s := newStreams(t, Options{})

	spin := s.io.Spinner("running migrations")
	spin.Stop()
	spin.Stop() // and again, which a deferred one does

	if got := s.err.String(); got != "running migrations...\n" {
		t.Errorf("the spinner wrote %q", got)
	}
}

func TestSpinnerIsSilentWhenNobodyIsReading(t *testing.T) {
	s := newStreams(t, Options{Verbosity: Quiet})

	s.io.Spinner("running migrations").Stop()

	if got := s.err.String(); got != "" {
		t.Errorf("the spinner wrote %q", got)
	}
}

func TestTask(t *testing.T) {
	s := newStreams(t, Options{})

	err := s.io.Task("running migrations", func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}

	got := s.err.String()
	if !strings.HasPrefix(got, "running migrations...\n") {
		t.Errorf("the task wrote %q", got)
	}
	if !strings.Contains(got, "✓ running migrations (") {
		t.Errorf("the task wrote %q, and does not say it finished", got)
	}
}

// TestTaskReturnsTheError is what keeps Task out of the way of the caller. It
// reports, it does not decide.
func TestTaskReturnsTheError(t *testing.T) {
	s := newStreams(t, Options{})
	want := errors.New("relation users does not exist")

	err := s.io.Task("running migrations", func() error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("Task returned %v, want the error the work returned", err)
	}
	if got := s.err.String(); !strings.Contains(got, "✗ running migrations (") {
		t.Errorf("the task wrote %q, and does not say it failed", got)
	}
}

func TestTaskRunsTheWorkWhenNobodyIsReading(t *testing.T) {
	s := newStreams(t, Options{JSON: true})

	ran := false
	if err := s.io.Task("running migrations", func() error { ran = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Error("--json skipped the work rather than the report about it")
	}
	if got := s.err.String(); got != "" {
		t.Errorf("the task wrote %q", got)
	}
}

func TestAnimated(t *testing.T) {
	s := newStreams(t, Options{})
	if s.io.animated() {
		t.Error("a buffer was taken for something worth drawing on")
	}
}
