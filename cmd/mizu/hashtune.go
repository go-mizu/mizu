package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/console"
	"github.com/go-mizu/mizu/hash"
)

// tuning is what hash:tune prints as JSON.
//
// It is a shape of its own rather than the [hash.Tuning] it came from, because
// the names here are the names in a configuration file and the ones there are
// the names in Go. A field renamed on one side should not silently rename
// itself on the other.
type tuning struct {
	Memory     int    `json:"memory"`
	Passes     int    `json:"passes"`
	Lanes      int    `json:"lanes"`
	Concurrent int    `json:"concurrent"`
	Target     string `json:"target"`
	Elapsed    string `json:"elapsed"`
	Runs       int    `json:"runs"`
	OnTarget   bool   `json:"on_target"`
	AtFloor    bool   `json:"at_floor"`
	AtCeiling  bool   `json:"at_ceiling"`
}

func tuningOf(t hash.Tuning) tuning {
	return tuning{
		Memory:     t.Params.Memory,
		Passes:     t.Params.Passes,
		Lanes:      t.Params.Lanes,
		Concurrent: t.Concurrent,
		Target:     t.Target.String(),
		Elapsed:    t.Elapsed.Round(time.Millisecond).String(),
		Runs:       t.Runs,
		OnTarget:   t.OnTarget(),
		AtFloor:    t.AtFloor,
		AtCeiling:  t.AtCeiling,
	}
}

// HashTune measures argon2id on this machine and prints the cost to configure.
type HashTune struct {
	Target time.Duration
	Passes int
	Lanes  int
}

func (c *HashTune) Spec() console.Spec {
	return console.Spec{
		Name: "hash:tune",
		Desc: "Measure argon2id on this machine",
		Long: tuneLong,
		Flags: []console.Flag{
			{Name: "target", Default: "250ms", Desc: "How long one hash should take", Value: console.Duration(&c.Target)},
			{Name: "passes", Default: "2", Desc: "Argon2id passes, held fixed while the memory is tuned", Value: console.Int(&c.Passes)},
			{Name: "lanes", Default: "1", Desc: "Argon2id lanes, held fixed while the memory is tuned", Value: console.Int(&c.Lanes)},
		},
	}
}

func (c *HashTune) Run(ctx context.Context, io *console.IO) error {
	// The command hashes for several seconds and prints nothing while it does,
	// which reads as a hang. The note goes to stderr, so --json still writes
	// one object to stdout and nothing else.
	io.Info("Measuring argon2id on this machine, aiming for %v a hash. This takes a few seconds.", c.Target)

	t, err := hash.Tune(ctx, hash.Target{Duration: c.Target, Passes: c.Passes, Lanes: c.Lanes})
	if err != nil {
		return err
	}

	if io.JSONMode() {
		return io.JSON(tuningOf(t))
	}
	io.Print("%s", report(t))
	return nil
}

const tuneLong = `It raises the memory until one hash takes about the target, because memory is
what a machine built to guess passwords is short of. The answer is never below
the 19 MiB, two passes and one lane that OWASP recommends.

Run it on the machine that will run the application. A cost measured on a build
server and deployed to a smaller instance is a slow login with no explanation.
Where that is not possible, set GOMEMLIMIT to what the smaller machine has and
run it here, which at least gets the concurrency right.`

// plural counts something in a sentence somebody has to read. The second word
// is the plural where adding an s does not give it.
func plural(n int, word string, plural ...string) string {
	switch {
	case n == 1:
		return fmt.Sprintf("%d %s", n, word)
	case len(plural) > 0:
		return fmt.Sprintf("%d %s", n, plural[0])
	default:
		return fmt.Sprintf("%d %ss", n, word)
	}
}

// report is what somebody reads in a terminal.
func report(t hash.Tuning) string {
	var b strings.Builder

	fmt.Fprintf(&b, "\nargon2id at %v a hash on this machine, measured over %s.\n\n",
		t.Elapsed.Round(time.Millisecond), plural(t.Runs, "run"))
	fmt.Fprintf(&b, "  memory   %d KiB (%d MiB)\n", t.Params.Memory, t.Params.Memory/1024)
	fmt.Fprintf(&b, "  passes   %d\n", t.Params.Passes)
	fmt.Fprintf(&b, "  lanes    %d\n\n", t.Params.Lanes)

	fmt.Fprintf(&b, "At this cost %s at once here, holding %d MiB between them.\n",
		plural(t.Concurrent, "hash runs", "hashes run"), t.Concurrent*t.Params.Memory/1024)

	switch {
	case t.AtFloor:
		fmt.Fprintf(&b, "This machine takes %v at the lowest cost worth storing, which is more than the %v asked for.\n",
			t.Elapsed.Round(time.Millisecond), t.Target)
		b.WriteString("The floor is the answer. A faster login is not worth going below it, so either accept the slower one or find a bigger machine.\n")
	case t.AtCeiling:
		b.WriteString("The memory ran out before the target did, so this is the ceiling rather than the answer.\n")
		b.WriteString("Set GOMEMLIMIT to what this machine has and run this again for a better one.\n")
	case !t.OnTarget():
		fmt.Fprintf(&b, "This is %v against the %v asked for, because the readings kept disagreeing with each other and the search ran out of steps.\n",
			t.Elapsed.Round(time.Millisecond), t.Target)
		b.WriteString("That is what a machine with something else running on it does. The cost above works, it is not the one asked for, and running this again on an idle machine will give a better one.\n")
	}

	b.WriteString("\nPut this in your config:\n\n")
	fmt.Fprintf(&b, "    [hash]\n    memory = %d\n    passes = %d\n    lanes = %d\n",
		t.Params.Memory, t.Params.Passes, t.Params.Lanes)
	return b.String()
}
