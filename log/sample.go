package log

import (
	"context"
	"hash/maphash"
	"log/slog"
	"sync/atomic"
	"time"
)

// SampleOptions configures [NewSamplingHandler]. The zero value writes the
// first hundred of a message each second, then one in a hundred, and never
// samples error.
type SampleOptions struct {
	// Initial is how many records of the same message are written in each
	// interval before sampling starts, and defaults to 100.
	Initial int

	// Every is the one in Every that is written after that, and defaults to
	// 100. One writes them all, which turns the tail back on without turning
	// off the counting.
	Every int

	// Interval is how long a count lasts before it starts again, and defaults
	// to a second.
	Interval time.Duration

	// Always is the level that is never sampled, and defaults to
	// [slog.LevelError]. A failure that happens ten thousand times is still ten
	// thousand failures, and the one that gets dropped is the one somebody
	// needed.
	Always slog.Leveler
}

// NewSamplingHandler drops records that repeat, so that one message in a loop
// cannot fill a disk or a bill.
//
// It writes the first Initial records of a message in each interval, then one
// in Every after that, and starts counting again when the interval is over.
// Records at or above [SampleOptions.Always] are written whatever the count
// says.
//
//	h := log.NewSamplingHandler(h, log.SampleOptions{})
//
// The count is per level and message, not per record, so a message that is
// written once for each of a thousand users is one message that repeats. The
// attributes are what tell those thousand records apart, and sampling that kept
// them all would not be sampling.
//
// Counts live in a fixed table of counters, which is what makes this cost an
// atomic add rather than a lock. Two messages can land on the same counter and
// share a budget. That happens to a handful of pairs out of thousands, and the
// alternative is a map with a lock in front of it on the path every record
// takes.
//
// A logger derived with [slog.Logger.With] shares the counts with the one it
// came from, since the message is the same message however the logger was
// built.
func NewSamplingHandler(h slog.Handler, o SampleOptions) slog.Handler {
	s := &sampler{
		h:        h,
		initial:  uint64(o.Initial),
		every:    uint64(o.Every),
		interval: int64(o.Interval),
		always:   o.Always,
		seed:     maphash.MakeSeed(),
		counts:   new([buckets]counter),
	}
	if o.Initial <= 0 {
		s.initial = 100
	}
	if o.Every <= 0 {
		s.every = 100
	}
	if o.Interval <= 0 {
		s.interval = int64(time.Second)
	}
	if s.always == nil {
		s.always = slog.LevelError
	}

	// The clock is monotonic and starts at zero here, so a counter that was
	// never touched is in an interval that ended long ago, and the wall clock
	// moving does not stop the sampler for an hour.
	start := time.Now()
	s.now = func() int64 { return int64(time.Since(start)) }
	return s
}

// buckets is how many counters there are. Two hundred and fifty six of them is
// sixteen kilobytes, which is small enough to sit in a program that logs one
// message and wide enough that the messages of a real one rarely collide.
const buckets = 256

// counter is one bucket: when the interval it is counting started, and how many
// records have landed in it since. The padding puts each one on its own cache
// line, so two goroutines logging two different messages do not fight over the
// same line.
type counter struct {
	start atomic.Int64
	n     atomic.Uint64
	_     [48]byte
}

type sampler struct {
	h        slog.Handler
	initial  uint64
	every    uint64
	interval int64
	always   slog.Leveler
	seed     maphash.Seed
	counts   *[buckets]counter
	now      func() int64
}

// key is what a record is counted by.
type key struct {
	level slog.Level
	msg   string
}

func (s *sampler) Enabled(ctx context.Context, l slog.Level) bool {
	return s.h.Enabled(ctx, l)
}

func (s *sampler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < s.always.Level() && !s.keep(r) {
		return nil
	}
	return s.h.Handle(ctx, r)
}

// keep counts this record and says whether it is one of the ones written.
func (s *sampler) keep(r slog.Record) bool {
	c := &s.counts[maphash.Comparable(s.seed, key{r.Level, r.Message})%buckets]

	now := s.now()
	if start := c.start.Load(); now-start >= s.interval {
		// Whoever wins the swap clears the count. Whoever loses is already in
		// the new interval, and its record is counted there.
		if c.start.CompareAndSwap(start, now) {
			c.n.Store(0)
		}
	}

	n := c.n.Add(1)
	return n <= s.initial || (n-s.initial)%s.every == 0
}

func (s *sampler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return s
	}
	next := *s
	next.h = s.h.WithAttrs(attrs)
	return &next
}

func (s *sampler) WithGroup(name string) slog.Handler {
	if name == "" {
		return s
	}
	next := *s
	next.h = s.h.WithGroup(name)
	return &next
}
