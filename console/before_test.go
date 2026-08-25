package console

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// key is a context key, for checking that what Before puts in a context is what
// the command is handed.
type key struct{}

// notes records what happened in what order, since the point of Before is that
// it runs before and its cleanup runs after.
type notes struct{ steps []string }

func (n *notes) add(step string) { n.steps = append(n.steps, step) }

// watched is a command that writes down when it ran and what its context held.
type watched struct {
	notes *notes
	saw   any
	err   error
}

func (c *watched) Spec() Spec { return Spec{Name: "watched"} }

func (c *watched) Run(ctx context.Context, io *IO) error {
	c.notes.add("command")
	c.saw = ctx.Value(key{})
	return c.err
}

func TestBeforeRunsAroundTheCommand(t *testing.T) {
	n := &notes{}
	cmd := &watched{notes: n}
	a, c, _, _ := app(cmd)
	a.Before = func(ctx context.Context, c *IO) (context.Context, func(), error) {
		n.add("before")
		return context.WithValue(ctx, key{}, "the deps"), func() { n.add("after") }, nil
	}

	if err := run(t, a, c, "watched"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(n.steps, " "); got != "before command after" {
		t.Errorf("the order was %q", got)
	}
	if cmd.saw != "the deps" {
		t.Errorf("the command was handed %v, want what Before put in the context", cmd.saw)
	}
}

// What Before opened has to be closed whatever the command did with it, which
// is the whole reason it hands back a function rather than closing at the end
// of itself.
func TestBeforeCleansUpAfterACommandThatFailed(t *testing.T) {
	n := &notes{}
	want := errors.New("the database is not there")
	a, c, _, _ := app(&watched{notes: n, err: want})
	a.Before = func(ctx context.Context, c *IO) (context.Context, func(), error) {
		return nil, func() { n.add("after") }, nil
	}

	if err := run(t, a, c, "watched"); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
	if got := strings.Join(n.steps, " "); got != "command after" {
		t.Errorf("the order was %q", got)
	}
}

func TestBeforeCanReturnNothingToDo(t *testing.T) {
	cmd := &watched{notes: &notes{}}
	a, c, _, _ := app(cmd)
	a.Before = func(ctx context.Context, c *IO) (context.Context, func(), error) {
		return nil, nil, nil
	}

	if err := run(t, a, c, "watched"); err != nil {
		t.Fatal(err)
	}
	if cmd.saw != nil {
		t.Errorf("the command was handed %v, want the context it would have had", cmd.saw)
	}
}

func TestBeforeThatFailsStopsTheCommand(t *testing.T) {
	n := &notes{}
	cmd := &watched{notes: n}
	want := errors.New("the config file is not there")
	a, c, _, _ := app(cmd)
	a.Before = func(ctx context.Context, c *IO) (context.Context, func(), error) {
		return nil, func() { n.add("after") }, want
	}

	if err := run(t, a, c, "watched"); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
	if len(n.steps) != 0 {
		t.Errorf("%v happened after Before failed", n.steps)
	}
}

// Asking what a command takes should not wait for whatever Before opens, and a
// name nobody registered should not start it either.
func TestBeforeDoesNotRunWithoutACommandToRun(t *testing.T) {
	tests := []struct {
		name string
		argv []string
	}{
		{"help", []string{"--help"}},
		{"help for a command", []string{"help", "watched"}},
		{"nothing at all", nil},
		{"an unknown command", []string{"watchd"}},
		{"a command line that does not parse", []string{"watched", "--nope"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ran := false
			a, c, _, _ := app(&watched{notes: &notes{}})
			a.Before = func(ctx context.Context, c *IO) (context.Context, func(), error) {
				ran = true
				return nil, nil, nil
			}

			run(t, a, c, tt.argv...)
			if ran {
				t.Error("Before ran")
			}
		})
	}
}
