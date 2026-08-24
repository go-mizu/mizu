package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/hash"
)

// A target of a millisecond is under what the floor costs on any machine, so
// every test here stops after one measurement. The command is exercised end to
// end and the suite does not spend ten seconds hashing to prove it.
const quick = "-target=1ms"

func TestHashTune(t *testing.T) {
	var out, errOut bytes.Buffer

	if err := run([]string{"hash:tune", quick}, &out, &errOut); err != nil {
		t.Fatalf("hash:tune: %v", err)
	}

	got := out.String()
	for _, want := range []string{"memory", "passes", "lanes", "[hash]", "19456"} {
		if !strings.Contains(got, want) {
			t.Errorf("the output does not mention %q:\n%s", want, got)
		}
	}

	// The note that it is working goes to the error stream, so that a shell
	// redirecting the output gets the answer and nothing else.
	if !strings.Contains(errOut.String(), "Measuring") {
		t.Errorf("nothing said it was working: %q", errOut.String())
	}
	if strings.Contains(got, "Measuring") {
		t.Error("the note that it is working landed in the output")
	}
}

func TestHashTuneJSON(t *testing.T) {
	var out, errOut bytes.Buffer

	if err := run([]string{"hash:tune", quick, "-json"}, &out, &errOut); err != nil {
		t.Fatalf("hash:tune: %v", err)
	}

	var got tuning
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("the output is not JSON: %v\n%s", err, out.String())
	}
	if got.Memory != 19456 || got.Passes != 2 || got.Lanes != 1 {
		t.Errorf("the cost came back as %d KiB, %d passes, %d lanes", got.Memory, got.Passes, got.Lanes)
	}
	if !got.AtFloor {
		t.Error("a target of a millisecond did not land on the floor")
	}
	if got.Concurrent < 1 || got.Runs < 1 {
		t.Errorf("%d at once over %d runs", got.Concurrent, got.Runs)
	}
	if got.Target != "1ms" {
		t.Errorf("the target came back as %q", got.Target)
	}

	// Nothing but the object is on the output, or a shell reading it with jq
	// gets a parse error instead of an answer.
	if n := strings.Count(strings.TrimSpace(out.String()), "\n{"); n != 0 {
		t.Errorf("there is more than one object on the output:\n%s", out.String())
	}
}

func TestHashTuneFlags(t *testing.T) {
	var out, errOut bytes.Buffer

	if err := run([]string{"hash:tune", quick, "-passes=3", "-lanes=2", "-json"}, &out, &errOut); err != nil {
		t.Fatalf("hash:tune: %v", err)
	}

	var got tuning
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("the output is not JSON: %v", err)
	}
	if got.Passes != 3 || got.Lanes != 2 {
		t.Errorf("it came back with %d passes and %d lanes", got.Passes, got.Lanes)
	}
}

func TestHashTuneRejects(t *testing.T) {
	cases := map[string][]string{
		"an argument":                     {"hash:tune", "please"},
		"a flag it has not":               {"hash:tune", "-memory=1024"},
		"a target that is not a duration": {"hash:tune", "-target=soon"},
	}

	for name, args := range cases {
		var out, errOut bytes.Buffer
		if err := run(args, &out, &errOut); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestHashTuneHelp(t *testing.T) {
	var out, errOut bytes.Buffer

	if err := run([]string{"hash:tune", "-h"}, &out, &errOut); err == nil {
		t.Error("asking for help is not an error, and it has to be one to exit 2")
	}
	for _, want := range []string{"Usage: mizu hash:tune", "GOMEMLIMIT", "-target", "-json"} {
		if !strings.Contains(errOut.String(), want) {
			t.Errorf("the help does not mention %q:\n%s", want, errOut.String())
		}
	}
}

// TestReport covers the sentences that only some machines produce, since a
// result that says nothing about being stuck at the floor is one somebody acts
// on without knowing they should not.
func TestReport(t *testing.T) {
	base := hash.Tuning{
		Params:     hash.Params{Memory: 65536, Passes: 2, Lanes: 1},
		Target:     250_000_000,
		Elapsed:    250_000_000,
		Concurrent: 8,
		Runs:       4,
	}

	cases := []struct {
		about string
		in    hash.Tuning
		want  string
	}{
		{"on target says nothing extra", base, "Put this in your config"},
		{"the floor", withFloor(base), "The floor is the answer"},
		{"the ceiling", withCeiling(base), "Set GOMEMLIMIT"},
		{"a busy machine", withNoise(base), "ran out of steps"},
	}

	for _, c := range cases {
		got := report(c.in)
		if !strings.Contains(got, c.want) {
			t.Errorf("%s: the report does not say %q:\n%s", c.about, c.want, got)
		}
	}

	// Being at the floor and off target at once is one sentence and not two,
	// because the floor is the reason and the reader needs the reason.
	if got := report(withFloor(base)); strings.Contains(got, "ran out of steps") {
		t.Errorf("the floor is reported as noise as well:\n%s", got)
	}
}

func withFloor(t hash.Tuning) hash.Tuning {
	t.Params.Memory, t.Elapsed, t.AtFloor = 19456, 400_000_000, true
	return t
}

func withCeiling(t hash.Tuning) hash.Tuning {
	t.Elapsed, t.AtCeiling = 100_000_000, true
	return t
}

func withNoise(t hash.Tuning) hash.Tuning {
	t.Elapsed = 400_000_000
	return t
}

// TestReportCounts is the grammar. A machine that measured once should not read
// as though somebody could not be bothered.
func TestReportCounts(t *testing.T) {
	one := hash.Tuning{
		Params:     hash.Params{Memory: 19456, Passes: 2, Lanes: 1},
		Target:     250_000_000,
		Elapsed:    400_000_000,
		Concurrent: 1,
		Runs:       1,
		AtFloor:    true,
	}

	got := report(one)
	for _, want := range []string{"over 1 run.", "1 hash runs at once"} {
		if !strings.Contains(got, want) {
			t.Errorf("the report does not say %q:\n%s", want, got)
		}
	}

	one.Runs, one.Concurrent = 4, 8
	got = report(one)
	for _, want := range []string{"over 4 runs.", "8 hashes run at once"} {
		if !strings.Contains(got, want) {
			t.Errorf("the report does not say %q:\n%s", want, got)
		}
	}
}
