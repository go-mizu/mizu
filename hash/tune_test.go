package hash

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/go-mizu/mizu/errs"
)

// machine is a made up computer that hashes at a fixed rate, so the search can
// be tested without waiting for anything. Real argon2id is close to linear in
// the memory cost, which is the assumption the search is built on, so a machine
// that is exactly linear is the search working and a machine that is not is the
// search coping.
type machine struct {
	per   time.Duration // How long a kibibyte takes.
	runs  int
	sizes []int
	err   error
}

func (m *machine) measure(ctx context.Context, p Params) (time.Duration, error) {
	m.runs++
	m.sizes = append(m.sizes, p.Memory)
	if m.err != nil {
		return 0, m.err
	}
	return time.Duration(p.Memory) * m.per, nil
}

// TestTuneFindsTheTarget is the ordinary case: a machine faster than the floor,
// a target it can reach, and an answer that hits it.
func TestTuneFindsTheTarget(t *testing.T) {
	cases := []struct {
		about  string
		per    time.Duration
		target time.Duration
		want   int
	}{
		{"a fast machine", time.Microsecond, 250 * time.Millisecond, 244 * 1024},
		{"a slow one", 10 * time.Microsecond, 250 * time.Millisecond, 24 * 1024},
		{"a long target", time.Microsecond, time.Second, 977 * 1024},
		{"a short one", time.Microsecond, 30 * time.Millisecond, 29 * 1024},
	}

	for _, c := range cases {
		m := &machine{per: c.per}
		got, err := tune(t.Context(), Target{Duration: c.target}, m.measure)
		if err != nil {
			t.Errorf("%s: %v", c.about, err)
			continue
		}
		if got.Params.Memory != c.want {
			t.Errorf("%s: %d KiB, want %d", c.about, got.Params.Memory, c.want)
		}
		if got.AtFloor || got.AtCeiling {
			t.Errorf("%s: floor %v, ceiling %v, and it should be neither", c.about, got.AtFloor, got.AtCeiling)
		}
		if !near(got.Elapsed, c.target) {
			t.Errorf("%s: %v against a target of %v", c.about, got.Elapsed, c.target)
		}
	}
}

// TestTuneReportsWhatItMeasured is the invariant that keeps the result honest:
// Elapsed is the reading for the cost that is printed next to it, never for a
// cost the search moved on from.
func TestTuneReportsWhatItMeasured(t *testing.T) {
	// A machine whose time is nothing like linear in the memory, so the search
	// runs out of steps rather than converging.
	awkward := func(ctx context.Context, p Params) (time.Duration, error) {
		return time.Duration(math.Sqrt(float64(p.Memory))) * time.Millisecond, nil
	}

	got, err := tune(t.Context(), Target{Duration: 400 * time.Millisecond}, awkward)
	if err != nil {
		t.Fatalf("tune: %v", err)
	}

	want, err := awkward(t.Context(), got.Params)
	if err != nil {
		t.Fatal(err)
	}
	if got.Elapsed != want {
		t.Errorf("it reported %v for %d KiB, which measures %v", got.Elapsed, got.Params.Memory, want)
	}
	if got.Runs > tuneSteps {
		t.Errorf("it measured %d times, and there are %d steps", got.Runs, tuneSteps)
	}
}

// TestTuneStopsAtTheFloor is a machine too slow for the target. The answer is
// the OWASP floor, because a login on a small instance being slower than asked
// for is better than a password hash that is not worth storing.
func TestTuneStopsAtTheFloor(t *testing.T) {
	m := &machine{per: time.Millisecond}

	got, err := tune(t.Context(), Target{Duration: 250 * time.Millisecond}, m.measure)
	if err != nil {
		t.Fatalf("tune: %v", err)
	}
	if got.Params.Memory != floorMemory {
		t.Errorf("%d KiB, want the floor of %d", got.Params.Memory, floorMemory)
	}
	if !got.AtFloor {
		t.Error("it did not say it was at the floor")
	}
	if m.runs != 1 {
		t.Errorf("it measured %d times to find out the floor was too slow", m.runs)
	}
	if got.Elapsed <= got.Target {
		t.Errorf("it reported %v, which is inside a target of %v", got.Elapsed, got.Target)
	}
}

// TestTuneStopsAtTheCeiling is the other end: a machine so fast, or a target so
// long, that the search runs into MaxMemory. The answer says so, because the
// number to change then is GOMEMLIMIT and not the target.
func TestTuneStopsAtTheCeiling(t *testing.T) {
	m := &machine{per: time.Nanosecond}

	got, err := tune(t.Context(), Target{Duration: time.Second, MaxMemory: 64 * 1024}, m.measure)
	if err != nil {
		t.Fatalf("tune: %v", err)
	}
	if got.Params.Memory != 64*1024 {
		t.Errorf("%d KiB, want the ceiling of %d", got.Params.Memory, 64*1024)
	}
	if !got.AtCeiling {
		t.Error("it did not say it was at the ceiling")
	}
	if got.AtFloor {
		t.Error("it said it was at the floor as well")
	}
}

// TestTuneHoldsPassesAndLanes says the search moves one number. The other two
// come out the way they went in, because a result that quietly changed them
// would not be the cost anybody asked to measure.
func TestTuneHoldsPassesAndLanes(t *testing.T) {
	m := &machine{per: time.Microsecond}

	got, err := tune(t.Context(), Target{Duration: 250 * time.Millisecond, Passes: 4, Lanes: 3}, m.measure)
	if err != nil {
		t.Fatalf("tune: %v", err)
	}
	if got.Params.Passes != 4 || got.Params.Lanes != 3 {
		t.Errorf("it came back with %d passes and %d lanes", got.Params.Passes, got.Params.Lanes)
	}

	// And the cost it answers with is one New accepts, which is the only thing
	// a caller does with it.
	if _, err := New(got.Params); err != nil {
		t.Errorf("the answer is not a valid cost: %v", err)
	}
}

// TestTuneDefaults covers the zero Target, which is what the command runs.
func TestTuneDefaults(t *testing.T) {
	m := &machine{per: time.Microsecond}

	got, err := tune(t.Context(), Target{}, m.measure)
	if err != nil {
		t.Fatalf("tune: %v", err)
	}
	if got.Target != 250*time.Millisecond {
		t.Errorf("the target defaulted to %v", got.Target)
	}
	if got.Params.Passes != 2 || got.Params.Lanes != 1 {
		t.Errorf("the cost defaulted to %d passes and %d lanes", got.Params.Passes, got.Params.Lanes)
	}
	if got.Concurrent < 1 {
		t.Errorf("it says %d hashes run at once here", got.Concurrent)
	}
	if m.sizes[0] != floorMemory {
		t.Errorf("it started at %d KiB rather than the floor", m.sizes[0])
	}
}

// TestTuneAnswersInWholeMebibytes is for whoever has to read the number. A
// memory cost goes into a configuration file that people look at.
func TestTuneAnswersInWholeMebibytes(t *testing.T) {
	for _, per := range []time.Duration{113, 271, 587, 1471} {
		m := &machine{per: per}

		got, err := tune(t.Context(), Target{Duration: 250 * time.Millisecond}, m.measure)
		if err != nil {
			t.Fatalf("tune: %v", err)
		}
		if got.Params.Memory%1024 != 0 {
			t.Errorf("%d KiB is not a whole number of mebibytes", got.Params.Memory)
		}
	}
}

func TestTuneRejects(t *testing.T) {
	m := &machine{per: time.Microsecond}

	cases := map[string]Target{
		"a negative target":            {Duration: -time.Second},
		"negative passes":              {Passes: -1},
		"negative lanes":               {Lanes: -1},
		"negative memory":              {MaxMemory: -1},
		"a ceiling too low":            {MaxMemory: 1024},
		"a ceiling of 19 MiB less one": {MaxMemory: floorMemory - 1},
	}

	for name, target := range cases {
		if _, err := tune(t.Context(), target, m.measure); errs.CodeOf(err) != "hash.target" {
			t.Errorf("%s: %v, want hash.target", name, err)
		}
	}
	if m.runs != 0 {
		t.Errorf("it hashed %d times before finding the target was wrong", m.runs)
	}
}

// TestTuneRejectsAnUncountableCeiling is the same bound New checks, and it is
// its own test for the same reason: the constant does not fit in an int where
// one is 32 bits, so there is nothing to check there.
func TestTuneRejectsAnUncountableCeiling(t *testing.T) {
	if math.MaxInt <= math.MaxUint32 {
		t.Skip("an int does not reach that far here")
	}

	m := &machine{per: time.Microsecond}
	if _, err := tune(t.Context(), Target{MaxMemory: math.MaxUint32 + 1}, m.measure); errs.CodeOf(err) != "hash.target" {
		t.Errorf("%v, want hash.target", err)
	}
	if m.runs != 0 {
		t.Errorf("it hashed %d times before finding the ceiling was wrong", m.runs)
	}
}

// TestTunePassesOnAFailedHash says a measurement that goes wrong stops the
// search rather than being counted as a fast one.
func TestTunePassesOnAFailedHash(t *testing.T) {
	want := errors.New("the machine went away")
	m := &machine{per: time.Microsecond, err: want}

	if _, err := tune(t.Context(), Target{}, m.measure); !errors.Is(err, want) {
		t.Errorf("%v, want the measurement error", err)
	}
}

// TestTuneStopsWhenTheCallerDoes matters because this runs from a terminal and
// takes seconds, so somebody will press control C part way through.
func TestTuneStopsWhenTheCallerDoes(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	m := &machine{per: time.Microsecond}
	_, err := tune(ctx, Target{}, m.measure)
	if errs.CodeOf(err) != "hash.canceled" {
		t.Errorf("%v, want hash.canceled", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Error("the context error did not survive")
	}
	if m.runs != 0 {
		t.Errorf("it hashed %d times after being told to stop", m.runs)
	}
}

// TestOnTarget is the one bit of the result the command changes what it prints
// on, so the difference between an answer to act on and one to measure again
// comes down to this.
func TestOnTarget(t *testing.T) {
	const target = 250 * time.Millisecond

	cases := []struct {
		about   string
		elapsed time.Duration
		want    bool
	}{
		{"exactly there", target, true},
		{"a little under", 245 * time.Millisecond, true},
		{"a little over", 255 * time.Millisecond, true},
		{"half the target", 125 * time.Millisecond, false},
		{"twice it", 500 * time.Millisecond, false},
		{"nothing measured", 0, false},
	}

	for _, c := range cases {
		got := Tuning{Target: target, Elapsed: c.elapsed}.OnTarget()
		if got != c.want {
			t.Errorf("%s: %v, want %v", c.about, got, c.want)
		}
	}
}

// TestMeasureRejectsACostItCannotBuild says a reading that did not happen comes
// back as an error and not as a fast hash. A cost nobody timed is the one thing
// the search above must never build on.
//
// The search only ever hands over costs it clamped itself, so this is what
// stands between a bug there and an answer that looks reasonable.
func TestMeasureRejectsACostItCannotBuild(t *testing.T) {
	if _, err := measure(t.Context(), Params{Memory: 1}); errs.CodeOf(err) != "hash.params" {
		t.Errorf("a cost too small to hash with: %v, want hash.params", err)
	}
}

// TestCeiling covers the default MaxMemory, which is the one number here that
// depends on what the machine was told about itself.
func TestCeiling(t *testing.T) {
	const gib = 1 << 20

	cases := []struct {
		about string
		limit int64
		want  int
	}{
		{"a machine that was told nothing", math.MaxInt64, gib},
		{"2 GiB, so a quarter of it", 2 << 30, 512 * 1024},
		{"64 GiB, capped at a gibibyte", 64 << 30, gib},
		{"64 MiB, held at the floor", 64 << 20, floorMemory},
	}

	for _, c := range cases {
		if got := ceilingOf(c.limit); got != c.want {
			t.Errorf("%s: %d KiB, want %d", c.about, got, c.want)
		}
	}

	// And whatever this machine says about itself, the answer is one the search
	// can work between.
	if got := ceiling(); got < floorMemory || got > gib {
		t.Errorf("the ceiling here is %d KiB, outside the floor of %d and the 1 GiB cap", got, floorMemory)
	}
}

func TestScale(t *testing.T) {
	cases := []struct {
		about     string
		memory    int
		want, got time.Duration
		expect    int
	}{
		{"twice as long", 1024, 200 * time.Millisecond, 100 * time.Millisecond, 2048},
		{"half as long", 4096, 50 * time.Millisecond, 100 * time.Millisecond, 2048},
		{"already there", 2048, 100 * time.Millisecond, 100 * time.Millisecond, 2048},
		{"a reading of nothing", 2048, time.Second, 0, 2048},
		{"a jump up is bounded", 1024, time.Hour, time.Millisecond, tuneJump * 1024},
		{"a jump down is too", 4096, time.Millisecond, time.Hour, 4096 / tuneJump},
		{"a reading so fast it overflows", 1 << 30, time.Hour, 1, math.MaxInt32},
	}

	for _, c := range cases {
		if got := scale(c.memory, c.want, c.got); got != c.expect {
			t.Errorf("%s: %d, want %d", c.about, got, c.expect)
		}
	}
}

// TestTuneOnThisMachine is the one that hashes for real. It asks for less than
// the floor costs, so it stops after one measurement and the suite stays fast,
// and what it proves is that the measurement and the search fit together.
func TestTuneOnThisMachine(t *testing.T) {
	got, err := Tune(t.Context(), Target{Duration: time.Millisecond})
	if err != nil {
		t.Fatalf("Tune: %v", err)
	}
	if !got.AtFloor {
		t.Errorf("a target of 1ms did not land on the floor: %d KiB in %v", got.Params.Memory, got.Elapsed)
	}
	if got.Elapsed <= 0 {
		t.Errorf("it measured %v", got.Elapsed)
	}
	if _, err := New(got.Params); err != nil {
		t.Errorf("the answer is not a valid cost: %v", err)
	}
}
