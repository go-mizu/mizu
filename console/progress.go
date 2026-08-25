package console

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// redraw is how often an animation touches the terminal.
//
// A loop that advances a bar a million times should not spend its afternoon
// writing escape sequences, and an eye cannot read faster than this anyway.
const redraw = 50 * time.Millisecond

// tick is how often a spinner changes frame, and heartbeat is how often the
// same spinner says something when there is no terminal to animate.
//
// Thirty seconds is chosen against a CI log: often enough that a five minute
// step does not look hung, rare enough that an hour of it is a hundred and
// twenty lines rather than a scroll nobody reads.
const (
	tick      = 100 * time.Millisecond
	heartbeat = 30 * time.Second
)

// animated reports whether stderr is something worth drawing on.
//
// Two questions in one. A pipe cannot be redrawn, and a command that was asked
// to be quiet or to speak JSON should not be decorating anything.
func (c *IO) animated() bool { return c.decorated() && isTerminal(c.err) }

// A Bar reports progress through a known number of steps.
//
// On a terminal it is a bar that redraws in place. Anywhere else it is a line
// every ten percent, which is eleven lines for a job of any length: enough to
// see that something is happening, few enough that a CI log stays readable.
//
// It is the one thing in this package that is safe to use from several
// goroutines, because a worker pool that reports its own progress is the
// normal way to end up with one.
type Bar struct {
	c     *IO
	total int
	width int

	mu       sync.Mutex
	current  int
	mark     int // the last tenth already reported, when not animating
	reported int // the value on the last line written, or -1 for none
	painted  time.Time
	done     bool
}

// Progress returns a bar over total steps, drawn on stderr.
//
// Call [Bar.Done] when the work is finished, which is what moves the cursor off
// the line the bar was drawn on.
func (c *IO) Progress(total int) *Bar {
	// A width for a terminal that did not answer. Eighty is the width of every
	// terminal that has ever had to guess.
	width := c.errWidth
	if width <= 0 {
		width = 80
	}
	return &Bar{c: c, total: total, width: width, reported: -1}
}

// Advance moves the bar on by n steps.
func (b *Bar) Advance(n int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.set(b.current + n)
}

// Set moves the bar to n steps, which is what a job that knows how far along it
// is rather than how far it moved wants.
//
// Going past the total or below zero is not an error. Work that turned out to
// be a different size than the count said is not worth ending a command over,
// so the bar clamps and carries on.
func (b *Bar) Set(n int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.set(n)
}

// set is the body of both, with the lock already held, so that two goroutines
// advancing the same bar cannot read the same value and both write it back.
func (b *Bar) set(n int) {
	b.current = min(max(n, 0), b.total)
	if b.done {
		return
	}
	if !b.c.animated() {
		b.report()
		return
	}
	if time.Since(b.painted) < redraw {
		return
	}
	b.paint()
}

// Done finishes the bar and leaves the cursor on a line of its own. It can be
// called more than once, so a deferred one after an early return is fine.
//
// It does not move the bar to the total. A job that stopped at four hundred of
// a thousand says so, since the alternative is a bar that reads as complete
// under an error message.
func (b *Bar) Done() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.done {
		return
	}
	b.done = true
	switch {
	case b.c.animated():
		b.paint()
		fmt.Fprintln(b.c.err)
	case b.c.decorated() && b.reported != b.current:
		// Work that stopped between two tenths has not been reported at all,
		// and where it stopped is the thing somebody reading the log wants.
		b.line()
	}
}

// paint redraws the bar in place. The escape at the end erases whatever the
// last, longer bar left behind, which is cheaper and steadier than padding the
// line out with spaces.
func (b *Bar) paint() {
	fmt.Fprint(b.c.err, "\r", render(b.width, b.current, b.total), "\x1b[K")
	b.painted = time.Now()
}

// report writes a line if the bar has crossed into a new tenth.
func (b *Bar) report() {
	if !b.c.decorated() || b.tenth() == b.mark {
		return
	}
	b.mark = b.tenth()
	b.line()
}

// line writes one progress line, for a stderr that cannot be redrawn.
func (b *Bar) line() {
	fmt.Fprintf(b.c.err, "%d%% (%d/%d)\n", percent(b.current, b.total), b.current, b.total)
	b.reported = b.current
}

func (b *Bar) tenth() int { return percent(b.current, b.total) / 10 }

// percent is how far along, where a total of nothing is finished rather than a
// division by zero.
//
// The multiply is done in 64 bits because int is 32 bits on some of the
// platforms this cross compiles for, and a bar over a hundred million rows is
// a thing somebody will have.
func percent(current, total int) int {
	if total <= 0 {
		return 100
	}
	return int(int64(current) * 100 / int64(total))
}

// The two cells a bar is drawn out of. Both are one column wide in every font
// that has them, which is what keeps the bar from changing length as it fills.
const (
	full  = "█"
	empty = "░"
)

// render draws the bar itself.
//
// The counts are what somebody actually reads, so they come first in the
// budget: a terminal too narrow for the cells gets the numbers and no picture,
// rather than a picture too small to say anything.
func render(width, current, total int) string {
	head := fmt.Sprintf("%3d%%", percent(current, total))
	counts := fmt.Sprintf("%d/%d", current, total)

	// The bar is capped so that a very wide terminal does not turn it into a
	// line of blocks, which reads as decoration rather than as a measurement.
	const (
		most  = 40
		least = 4
	)
	cells := min(width-len(head)-len(counts)-2, most)
	if cells < least {
		return head + " " + counts
	}

	filled := cells
	if total > 0 {
		filled = int(int64(current) * int64(cells) / int64(total))
	}
	return head + " " + strings.Repeat(full, filled) + strings.Repeat(empty, cells-filled) + " " + counts
}

// The frames a spinner turns through. Braille, because the dots move without
// the line jumping about, which is what a rotating slash does.
var frames = [...]string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// A Spinner says that something is happening without saying how far along it
// is, which is the honest answer for work whose size is not known.
//
// On a terminal it turns. Anywhere else it writes a line when it starts, a line
// every thirty seconds while it runs, and a line when it stops.
//
// Nothing else should write to stderr while one is running, because the
// spinner owns the line it is drawn on. [IO.Task] is the shape that gets this
// right without anybody having to remember it.
type Spinner struct {
	c       *IO
	message string
	started time.Time

	once sync.Once
	stop chan struct{}
	gone chan struct{}
}

// Spinner starts one on stderr. Call [Spinner.Stop] when the work is done.
func (c *IO) Spinner(message string) *Spinner {
	s := &Spinner{
		c:       c,
		message: message,
		started: time.Now(),
		stop:    make(chan struct{}),
		gone:    make(chan struct{}),
	}
	if !c.decorated() {
		close(s.gone)
		return s
	}
	if !c.animated() {
		fmt.Fprintf(c.err, "%s...\n", message)
	}
	go s.run()
	return s
}

// run is the animation, and the heartbeat when there is nothing to animate.
func (s *Spinner) run() {
	defer close(s.gone)

	every := heartbeat
	if s.c.animated() {
		every = tick
	}
	t := time.NewTicker(every)
	defer t.Stop()

	for frame := 0; ; frame++ {
		select {
		case <-s.stop:
			return
		case <-t.C:
			if s.c.animated() {
				fmt.Fprint(s.c.err, "\r", styleDim.wrap(frames[frame%len(frames)], s.c.colorErr), " ", s.message, "\x1b[K")
				continue
			}
			fmt.Fprintf(s.c.err, "still %s (%s)\n", s.message, elapsed(s.started))
		}
	}
}

// Stop ends the spinner and leaves the cursor on a line of its own. It can be
// called more than once.
func (s *Spinner) Stop() {
	s.once.Do(func() {
		close(s.stop)
		<-s.gone
		if s.c.animated() {
			// The line the spinner was drawn on is the caller's again, and
			// what it held was never information.
			fmt.Fprint(s.c.err, "\r\x1b[K")
		}
	})
}

// Task runs fn behind a spinner and says how it went.
//
// It returns fn's error unchanged, so it wraps a call without getting in the
// way of what the caller was going to do with it:
//
//	if err := io.Task("running migrations", migrate); err != nil {
//		return err
//	}
func (c *IO) Task(message string, fn func() error) error {
	s := c.Spinner(message)
	err := fn()
	s.Stop()

	if !c.decorated() {
		return err
	}
	if err != nil {
		c.say(styleRed, "", "%s %s (%s)", cross, message, elapsed(s.started))
		return err
	}
	c.say(styleGreen, "", "%s %s (%s)", check, message, elapsed(s.started))
	return nil
}

// The two marks a finished task carries. They are there so the outcome is
// visible in a log that has lost its colour, which is every log that somebody
// pasted into a message.
const (
	check = "✓"
	cross = "✗"
)

// elapsed is how long something took, at the precision somebody watching it
// cares about. A migration that took 1.4 seconds is not 1.42981 seconds.
func elapsed(since time.Time) time.Duration {
	d := time.Since(since)
	if d < time.Second {
		return d.Round(time.Millisecond)
	}
	return d.Round(100 * time.Millisecond)
}
