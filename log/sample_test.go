package log

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// clock is the time a sampler under test runs on, so that an interval can pass
// without a test sleeping through it.
type clock struct{ nanos atomic.Int64 }

func (c *clock) add(d time.Duration) { c.nanos.Add(int64(d)) }

// sampled is a sampler with a clock a test can move.
func sampled(h slog.Handler, o SampleOptions) (slog.Handler, *clock) {
	s := NewSamplingHandler(h, o).(*sampler)
	c := new(clock)
	s.now = c.nanos.Load
	return s, c
}

func TestSampling(t *testing.T) {
	rec := new(recorder)
	h, _ := sampled(rec, SampleOptions{Initial: 3, Every: 2})
	log := slog.New(h)

	for range 10 {
		log.Info("the same thing again")
	}

	// The first three, and then every second one: the fifth, the seventh and
	// the ninth.
	if got := rec.count(); got != 6 {
		t.Errorf("the handler got %d of 10 records, want 6", got)
	}
}

// TestSamplingInterval is the part that makes sampling recoverable. A message
// that was noisy a minute ago gets its full budget again.
func TestSamplingInterval(t *testing.T) {
	rec := new(recorder)
	h, c := sampled(rec, SampleOptions{Initial: 2, Every: 1000, Interval: time.Second})
	log := slog.New(h)

	for range 5 {
		log.Info("noisy")
	}
	if got := rec.count(); got != 2 {
		t.Fatalf("the first interval wrote %d records, want 2", got)
	}

	c.add(time.Second)
	for range 5 {
		log.Info("noisy")
	}
	if got := rec.count(); got != 4 {
		t.Errorf("the second interval brought the total to %d, want 4", got)
	}
}

// TestSamplingCountsPerMessage is what stops one message in a loop from
// silencing everything else in the program.
func TestSamplingCountsPerMessage(t *testing.T) {
	rec := new(recorder)
	h, _ := sampled(rec, SampleOptions{Initial: 1, Every: 1000})
	log := slog.New(h)

	for range 5 {
		log.Info("first")
		log.Info("second")
	}

	if got := rec.messages(); len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Errorf("the handler got %v, want one of each", got)
	}
}

// TestSamplingAttributesDoNotCount is the decision worth writing down: the
// message is the identity, and the attributes are what tell two records with
// the same message apart.
func TestSamplingAttributesDoNotCount(t *testing.T) {
	rec := new(recorder)
	h, _ := sampled(rec, SampleOptions{Initial: 1, Every: 1000})
	log := slog.New(h)

	for i := range 5 {
		log.Info("request", "user_id", i)
	}

	if got := rec.count(); got != 1 {
		t.Errorf("the handler got %d records, want 1", got)
	}
}

// TestSamplingLevel is why a level is part of the count. The same message at
// two levels is two different things happening.
func TestSamplingLevel(t *testing.T) {
	rec := new(recorder)
	h, _ := sampled(rec, SampleOptions{Initial: 1, Every: 1000})
	log := slog.New(h)

	for range 3 {
		log.Info("retrying")
		log.Warn("retrying")
	}

	if got := rec.count(); got != 2 {
		t.Errorf("the handler got %d records, want one of each level", got)
	}
}

// TestSamplingNeverDropsErrors is the default that matters. A failure that
// happens ten thousand times is still ten thousand failures.
func TestSamplingNeverDropsErrors(t *testing.T) {
	rec := new(recorder)
	h, _ := sampled(rec, SampleOptions{Initial: 1, Every: 1000})
	log := slog.New(h)

	for range 50 {
		log.Error("the disk is full")
	}

	if got := rec.count(); got != 50 {
		t.Errorf("the handler got %d of 50 errors", got)
	}
}

func TestSamplingAlways(t *testing.T) {
	rec := new(recorder)
	h, _ := sampled(rec, SampleOptions{Initial: 1, Every: 1000, Always: slog.LevelWarn})
	log := slog.New(h)

	for range 5 {
		log.Warn("slow query")
		log.Info("cache miss")
	}

	if got := rec.count(); got != 6 {
		t.Errorf("the handler got %d records, want 5 warnings and 1 of the rest", got)
	}
}

// TestSamplingDefaults checks the zero value is the documented one, since it is
// what most callers get.
func TestSamplingDefaults(t *testing.T) {
	s := NewSamplingHandler(new(recorder), SampleOptions{}).(*sampler)

	if s.initial != 100 || s.every != 100 {
		t.Errorf("initial %d, every %d, want 100 and 100", s.initial, s.every)
	}
	if s.interval != int64(time.Second) {
		t.Errorf("interval %v, want a second", time.Duration(s.interval))
	}
	if s.always.Level() != slog.LevelError {
		t.Errorf("always %v, want error", s.always)
	}
}

// TestSamplingRealClock is the one test that uses the clock the handler builds
// for itself, since a seam a test never leaves is a seam nothing checks.
func TestSamplingRealClock(t *testing.T) {
	rec := new(recorder)
	log := slog.New(NewSamplingHandler(rec, SampleOptions{Initial: 1, Every: 1000, Interval: time.Millisecond}))

	log.Info("noisy")
	log.Info("noisy")
	time.Sleep(2 * time.Millisecond)
	log.Info("noisy")

	if got := rec.count(); got != 2 {
		t.Errorf("the handler got %d records, want the first of each interval", got)
	}
}

func TestSamplingEnabled(t *testing.T) {
	h, _ := sampled(&recorder{level: slog.LevelWarn}, SampleOptions{})
	ctx := context.Background()

	if h.Enabled(ctx, slog.LevelInfo) {
		t.Error("a level the wrapped handler does not want is enabled")
	}
	if !h.Enabled(ctx, slog.LevelWarn) {
		t.Error("a level the wrapped handler wants is not enabled")
	}
}

// TestSamplingSharesCounts is why the counts live behind a pointer. Two loggers
// derived from the same handler are still one program writing one message.
func TestSamplingSharesCounts(t *testing.T) {
	rec := new(recorder)
	h, _ := sampled(rec, SampleOptions{Initial: 1, Every: 1000})

	slog.New(h).With("service", "api").Info("noisy")
	slog.New(h).WithGroup("db").Info("noisy")

	if got := rec.count(); got != 1 {
		t.Errorf("the handler got %d records, want 1", got)
	}

	if got := h.WithAttrs(nil); got != h {
		t.Error("WithAttrs(nil) made a new handler")
	}
	if got := h.WithGroup(""); got != h {
		t.Error("WithGroup(\"\") made a new handler")
	}
}

// TestSamplingConcurrent is for the race detector, and for the promise that the
// budget holds when the noise comes from everywhere at once.
func TestSamplingConcurrent(t *testing.T) {
	rec := new(recorder)
	h, _ := sampled(rec, SampleOptions{Initial: 10, Every: 1000000, Interval: time.Hour})
	log := slog.New(h)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				log.Info("noisy")
			}
		}()
	}
	wg.Wait()

	if got := rec.count(); got != 10 {
		t.Errorf("800 records from 8 goroutines wrote %d, want the budget of 10", got)
	}
}
