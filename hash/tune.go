package hash

import (
	"context"
	"math"
	"slices"
	"time"

	"github.com/go-mizu/mizu/errs"
)

// A Target is what [Tune] aims for.
//
// The zero value is what mizu hash:tune runs with, and it is the one to use
// unless something about the deployment says otherwise.
type Target struct {
	// Duration is how long one hash should take on this machine. The default
	// is 250 milliseconds.
	//
	// It is a budget rather than a preference. Whatever is spent here is spent
	// on every sign in, and a login that takes a second is one people notice,
	// so the number comes from how slow a sign in may be and not from how safe
	// somebody would like to feel.
	Duration time.Duration

	// Passes and Lanes are held where they are while the memory cost is moved
	// to meet Duration. The defaults are the 2 and 1 in [Params].
	//
	// Memory is the lever because memory is what the machines built to guess
	// passwords are short of. A graphics card has thousands of cores and a few
	// gigabytes between them, so doubling the passes halves what it manages
	// and doubling the memory can stop it running at all. Raise Passes here
	// only when the memory has already gone as high as the machine allows.
	Passes int
	Lanes  int

	// MaxMemory is the most memory one hash may be given, in kibibytes.
	//
	// The default comes from GOMEMLIMIT, a quarter of it, or 1 GiB where there
	// is no limit set. It is a quarter because the answer has to leave room
	// for more than one hash and for the rest of the process, and 1 GiB
	// because a machine that has not been told what it has should not be
	// guessed at.
	MaxMemory int
}

// A Tuning is what [Tune] found.
type Tuning struct {
	// Params is the answer: the cost to configure on a machine like this one.
	Params Params

	// Target is the duration that was asked for and Elapsed is what Params
	// measured. They are close but not equal, because the search stops when it
	// is near enough rather than going on for a number that the next run would
	// not reproduce anyway.
	Target  time.Duration
	Elapsed time.Duration

	// Concurrent is how many hashes at this cost run at once here, which is
	// what [Params.MaxConcurrent] left at zero works out to. It is the second
	// half of the answer: the cost decides what one login spends, and this
	// decides what a thousand at the same moment spend.
	Concurrent int

	// Runs is how many hashes were timed to get here.
	Runs int

	// AtFloor is a machine that is already slower than Target at the lowest
	// cost worth using, so the answer is that floor and sign ins on it will
	// take longer than was asked for.
	AtFloor bool

	// AtCeiling is a search that ran into MaxMemory before it reached Target.
	// The answer is the ceiling, and a machine with more memory, or one told
	// what it has through GOMEMLIMIT, would give a better one.
	AtCeiling bool
}

// OnTarget reports whether Elapsed landed near Target.
//
// It is false when the search ran out of steps without settling, which is what
// a machine with something else running on it does: each reading disagrees with
// the last, so every step corrects for noise rather than for cost. The result
// is still a cost that works, it is not the cost that was asked for, and the
// answer is to run this again on a machine that is doing nothing else.
//
// It is also false at the floor and at the ceiling, where the search stopped
// because it was not allowed to go further rather than because it arrived.
func (t Tuning) OnTarget() bool { return near(t.Elapsed, t.Target) }

// floorMemory is the lowest memory cost [Tune] will answer with, which is the
// 19 MiB OWASP recommends. A machine too slow to reach the target at the floor
// gets the floor, because the floor is what makes the hash worth storing and a
// faster login is not worth going below it.
const floorMemory = 19456

// tuneSteps is how many times the search adjusts before it takes what it has.
// Time is close to linear in memory, so the first adjustment lands in the right
// neighbourhood and the second usually arrives. The rest are there for the
// machine that is busy.
const tuneSteps = 8

// measureRuns is how many hashes go into one reading.
const measureRuns = 3

// tuneJump is the most one step may multiply or divide the memory cost by.
//
// Without it a single reading that came out four times too fast, which is what
// a hash that landed while something else was finishing looks like, moves the
// search somewhere it then needs every remaining step to come back from. With
// it a bad reading costs one step.
const tuneJump = 4

// tuneTolerance is how near the target the search has to get to stop. Two runs
// of this on the same idle machine differ by more than this, so measuring more
// finely would be reporting noise as a result.
const tuneTolerance = 0.05

// Tune returns the argon2id parameters that take about [Target.Duration] on
// this machine.
//
// It works by hashing, so it takes a few seconds and it measures the machine it
// runs on. A number tuned on a build server and deployed to a smaller instance
// is a slow login nobody can account for, which is why mizu hash:tune is a
// command somebody runs and reads rather than something the application does at
// boot: parameters are configuration, and configuration that changes by itself
// on every restart is not configuration.
//
// The result is never below the OWASP floor of 19 MiB, two passes and one lane.
// [Tuning.AtFloor] says when the machine could not reach the target even there.
func Tune(ctx context.Context, t Target) (Tuning, error) {
	return tune(ctx, t, measure)
}

// tune is [Tune] with the measurement handed in, so the search can be tested
// against a machine that does not exist.
func tune(ctx context.Context, t Target, measure func(context.Context, Params) (time.Duration, error)) (Tuning, error) {
	if t.Duration == 0 {
		t.Duration = 250 * time.Millisecond
	}
	if t.Passes == 0 {
		t.Passes = 2
	}
	if t.Lanes == 0 {
		t.Lanes = 1
	}
	if t.MaxMemory == 0 {
		t.MaxMemory = ceiling()
	}

	switch {
	case t.Duration < 0 || t.Passes < 0 || t.Lanes < 0 || t.MaxMemory < 0:
		return Tuning{}, errs.New(errs.Invalid, "hash.target", "hash: a target cannot be negative")
	case t.MaxMemory < floorMemory:
		return Tuning{}, errs.Newf(errs.Invalid, "hash.target",
			"hash: %d KiB is below the %d KiB floor, so there is nothing to tune", t.MaxMemory, floorMemory)
	case uint64(t.MaxMemory) > math.MaxUint32:
		return Tuning{}, errs.New(errs.Invalid, "hash.target",
			"hash: that is a larger memory cost than argon2id counts in")
	}

	result := Tuning{Target: t.Duration}
	p := Params{Memory: floorMemory, Passes: t.Passes, Lanes: t.Lanes}

	// Every pass measures the cost it is holding, so whatever the loop leaves
	// behind, Elapsed is the reading for the Memory that is reported with it.
	// The last pass adjusts nothing, since a cost nobody timed is a guess.
	for step := range tuneSteps {
		if err := ctx.Err(); err != nil {
			return Tuning{}, errs.Wrap(err, errs.KindOf(err), "hash.canceled", "hash: gave up part way through tuning")
		}

		elapsed, err := measure(ctx, p)
		if err != nil {
			return Tuning{}, err
		}
		result.Elapsed, result.Runs = elapsed, result.Runs+1

		// The floor is the floor. A machine slower than the target at 19 MiB
		// stays at 19 MiB and signs people in slowly.
		if p.Memory == floorMemory && elapsed >= t.Duration {
			result.AtFloor = true
			break
		}

		next := clamp(scale(p.Memory, t.Duration, elapsed), t.MaxMemory)
		if step == tuneSteps-1 || next == p.Memory || near(elapsed, t.Duration) {
			break
		}
		p.Memory = next
	}

	result.Params = p
	result.AtCeiling = p.Memory == t.MaxMemory && result.Elapsed < t.Duration
	result.Concurrent = concurrency(p.Memory, memoryLimit(), cpus())
	return result, nil
}

// measure is how long a hash takes at a given cost.
//
// It runs several and takes the fastest, which is the reading that says the
// most about the cost and the least about what else the machine was doing. The
// first hash on a fresh hasher pays for memory the operating system has not
// handed over yet, and any of them can land next to a compiler, so a reading
// above the fastest is the cost plus something that is not the cost.
//
// The fastest is also the reading that reproduces. The search below moves in
// proportion to how far off the last reading was, so a reading that is 20
// percent high sends the next step 20 percent wrong, and averaging noise in
// rather than dropping it is what makes a search wander instead of settle.
func measure(ctx context.Context, p Params) (time.Duration, error) {
	h, err := New(p)
	if err != nil {
		return 0, err
	}

	var runs [measureRuns]time.Duration
	for i := range runs {
		start := time.Now()
		if _, err := h.Hash(ctx, "the password this is measured with"); err != nil {
			return 0, err
		}
		runs[i] = time.Since(start)
	}
	return slices.Min(runs[:]), nil
}

// scale is the memory that should take want, given that memory took got.
//
// Argon2 fills its memory once per pass and reads it in an order that does not
// change with the size of it, so the time is close enough to linear in the
// memory that one multiplication lands within a few percent. The loop that
// calls this is what covers the rest.
func scale(memory int, want, got time.Duration) int {
	if got <= 0 {
		return memory
	}
	next := float64(memory) * float64(want) / float64(got)
	next = min(max(next, float64(memory)/tuneJump), float64(memory)*tuneJump)
	if next > math.MaxInt32 {
		return math.MaxInt32
	}
	return round(int(next))
}

// round takes a memory cost to a whole number of mebibytes, because 164 MiB is
// a number somebody can read in a configuration file and 167,935 KiB is not.
func round(kib int) int {
	const mib = 1024
	return (kib + mib/2) / mib * mib
}

// clamp holds a memory cost between the floor and the ceiling.
func clamp(kib, top int) int {
	return min(max(kib, floorMemory), top)
}

// near reports whether a measurement is close enough to the target to stop.
func near(got, want time.Duration) bool {
	return math.Abs(float64(got-want)) < tuneTolerance*float64(want)
}

// ceiling is the most memory [Tune] gives one hash when a [Target] does not
// say.
func ceiling() int { return ceilingOf(memoryLimit()) }

// ceilingOf is ceiling with the limit handed in, the way [concurrency] takes
// one, so the arithmetic can be tested without moving the running program's
// memory limit around underneath the rest of the suite.
func ceilingOf(limit int64) int {
	const gib = 1 << 20 // In kibibytes, which is what a memory cost is in.

	if limit == math.MaxInt64 {
		return gib
	}
	return clamp(int(limit/4/1024), gib)
}
