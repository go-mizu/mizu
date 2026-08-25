package validate

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs"
)

// Nothing failed, so there is nothing to report, and the zero value says so
// without anybody having constructed it.
func TestZeroErrorsIsEmpty(t *testing.T) {
	var e Errors

	if got := e.Len(); got != 0 {
		t.Errorf("Len = %d, want 0", got)
	}
	if e.Has("title") {
		t.Error("Has says a field failed and nothing was added")
	}
	if got := e.First("title"); got != "" {
		t.Errorf("First = %q, want the empty string", got)
	}
	if got := e.All(); got != nil {
		t.Errorf("All = %v, want nil", got)
	}
	if err := e.OrNil(); err != nil {
		t.Errorf("OrNil = %v, want nil", err)
	}
	for field := range e.Fields() {
		t.Errorf("Fields yielded %q and nothing was added", field)
	}
	if got := e.Error(); got != "validate: nothing failed" {
		t.Errorf("Error = %q", got)
	}
}

// A field that fails twice is reported twice, because required and min are two
// different things to tell somebody and choosing one for them is guessing.
func TestAFieldMayFailMoreThanOnce(t *testing.T) {
	var e Errors
	e.Add("password", Failed("min", 12).Of("string"))
	e.Add("password", Failed("nothtml"))

	if got := e.Len(); got != 2 {
		t.Errorf("Len = %d, want 2", got)
	}
	if got := len(e.All()["password"]); got != 2 {
		t.Errorf("All has %d messages for password, want 2", got)
	}
	if got := e.First("password"); !strings.Contains(got, "12 characters") {
		t.Errorf("First = %q, want the first rule added", got)
	}
}

// Two runs of the same checks have to produce the same document, so the order
// is the order the fields first failed and not whatever a map hands back.
func TestFieldsKeepTheOrderTheyFailedIn(t *testing.T) {
	want := []string{"title", "body", "publish_at", "author"}

	for range 20 {
		var e Errors
		e.Add("title", Failed("required"))
		e.Add("body", Failed("required"))
		e.Add("publish_at", Failed("required"))
		e.Add("title", Failed("min", 3).Of("string"))
		e.Add("author", Failed("required"))

		var order []string
		for field := range e.Fields() {
			order = append(order, field)
		}
		if !slices.Equal(order, want) {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}

// collectFields is Fields as a map, for the assertions that do not care about
// the order.
func collectFields(e *Errors) map[string][]RuleError {
	m := make(map[string][]RuleError)
	for field, rules := range e.Fields() {
		m[field] = rules
	}
	return m
}

// A range that stops early stops the iterator, which is the half of an
// iter.Seq2 that gets left out and then never noticed.
func TestFieldsStopsWhenTheRangeDoes(t *testing.T) {
	var e Errors
	e.Add("a", Failed("required"))
	e.Add("b", Failed("required"))
	e.Add("c", Failed("required"))

	var seen []string
	for field := range e.Fields() {
		seen = append(seen, field)
		if len(seen) == 2 {
			break
		}
	}
	if !slices.Equal(seen, []string{"a", "b"}) {
		t.Errorf("seen = %v, want [a b]", seen)
	}
}

// All is built fresh, so a template that sorts it or a handler that deletes
// from it changes nothing that anybody else reads.
func TestAllHandsBackACopy(t *testing.T) {
	var e Errors
	e.Add("title", Failed("required"))

	all := e.All()
	delete(all, "title")
	all["invented"] = []string{"nonsense"}

	if got := e.All(); len(got) != 1 || len(got["title"]) != 1 {
		t.Errorf("All = %v, want the one failure that was added", got)
	}
}

// OrNil is the whole contract with the rest of the toolkit: the kind that maps
// to 422, the code a client switches on, and one field per rule.
func TestOrNilIsAnErrsError(t *testing.T) {
	var e Errors
	e.Add("title", Failed("min", 3).Of("string"))
	e.Add("publish_at", Failed("required"))

	err := e.OrNil()

	if got := errs.KindOf(err); got != errs.Unprocessable {
		t.Errorf("KindOf = %v, want unprocessable", got)
	}
	if got := errs.KindOf(err).Status(); got != 422 {
		t.Errorf("Status = %d, want 422", got)
	}
	if got := errs.CodeOf(err); got != Code {
		t.Errorf("CodeOf = %q, want %q", got, Code)
	}

	want := []errs.Field{
		{Name: "title", Code: "min", Msg: "Title must be at least 3 characters."},
		{Name: "publish_at", Code: "required", Msg: "Publish at is required."},
	}
	if got := errs.Fields(err); !slices.Equal(got, want) {
		t.Errorf("Fields = %v, want %v", got, want)
	}
}

// The Errors stays on the error as the cause, so a caller that wants the rules
// and their parameters back does not have to read a sentence to get them.
func TestOrNilKeepsTheErrorsReachable(t *testing.T) {
	var e Errors
	e.Add("tags", Failed("between", 1, 5).Of("array"))

	var back *Errors
	if !errors.As(e.OrNil(), &back) {
		t.Fatal("errors.As did not find the Errors on the error it returned")
	}

	rules := collectFields(back)["tags"]
	if len(rules) != 1 {
		t.Fatalf("tags has %d rules, want 1", len(rules))
	}
	if got := rules[0]; got.Rule != "between" || got.Subject != "array" || len(got.Params) != 2 {
		t.Errorf("rule = %+v, want between over an array with two parameters", got)
	}
}

// The message about the request as a whole is what an RFC 9457 document puts
// in detail, so it counts rules and not fields, and it says one rather than 1.
func TestDetailCountsTheRules(t *testing.T) {
	cases := []struct {
		add  func(*Errors)
		want string
	}{
		{func(e *Errors) {
			e.Add("a", Failed("required"))
		}, "One field failed validation."},
		{func(e *Errors) {
			e.Add("a", Failed("required"))
			e.Add("b", Failed("required"))
		}, "2 fields failed validation."},
		{func(e *Errors) {
			e.Add("a", Failed("required"))
			e.Add("a", Failed("nothtml"))
		}, "2 fields failed validation."},
	}
	for _, c := range cases {
		var e Errors
		c.add(&e)

		var got *errs.Error
		if !errors.As(e.OrNil(), &got) {
			t.Fatal("OrNil did not return an *errs.Error")
		}
		if got.Msg != c.want {
			t.Errorf("Msg = %q, want %q", got.Msg, c.want)
		}
	}
}

// An Errors is usable as an error on its own, for a caller with somewhere to
// put it that is not an HTTP response.
func TestErrorReadsTheMessages(t *testing.T) {
	var e Errors
	e.Add("title", Failed("required"))
	e.Add("body", Failed("min", 10).Of("string"))
	e.Add("body", Failed("nothtml"))

	want := "validate: Title is required.; Body must be at least 10 characters. Body is not valid."
	if got := e.Error(); got != want {
		t.Errorf("Error = %q,\nwant %q", got, want)
	}
}

// Of returns a copy, so a rule error held somewhere and reused for two
// subjects does not turn into whichever was asked for last.
func TestOfDoesNotChangeTheOriginal(t *testing.T) {
	min := Failed("min", 3)

	text, list := min.Of("string"), min.Of("array")

	if min.Subject != "" {
		t.Errorf("the original has subject %q, want none", min.Subject)
	}
	if text.Subject != "string" || list.Subject != "array" {
		t.Errorf("copies are %q and %q, want string and array", text.Subject, list.Subject)
	}
}

// The lookup key is the rule for a rule with one sentence and the rule with
// its subject for a rule with several, which is the only thing Subject does.
func TestKey(t *testing.T) {
	cases := []struct {
		r    RuleError
		want string
	}{
		{Failed("required"), "required"},
		{Failed("min", 3), "min"},
		{Failed("min", 3).Of("string"), "min.string"},
		{Failed("between", 1, 5).Of("array"), "between.array"},
	}
	for _, c := range cases {
		if got := c.r.key(); got != c.want {
			t.Errorf("key of %+v = %q, want %q", c.r, got, c.want)
		}
	}
}
