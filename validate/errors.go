package validate

import (
	"iter"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu/errs"
)

// Code is what [Errors.OrNil] puts on the error it returns, and what a client
// switches on to know that the request was understood and rejected rather than
// misunderstood.
const Code = "validation.failed"

// A RuleError is one rule that a value did not satisfy.
//
// It carries no copy of the value. A message built from what arrived is a
// message that renders whatever somebody sent, and the rule name and its
// parameters are enough to say what the field takes, which is the part that
// helps. That is the same call D-092 made for binding.
type RuleError struct {
	// Rule is the name of the check, such as required or min. It is what a
	// client switches on, what a message is keyed by, and what ends up in the
	// Code of the [errs.Field].
	Rule string

	// Params are what the rule was configured with, the 3 in min=3, in the
	// order the message writes them.
	Params []any

	// Subject is which of a rule's sentences this failure wants, and is empty
	// for a rule that has one. The names are string, numeric, array, file and
	// duration, matching what a size rule counts. It changes the sentence and
	// not the rule name, since a client that treats min on a string
	// differently from min on a list is a client reading the field it already
	// knows the type of.
	Subject string
}

// Failed is a rule that did not hold, with whatever it was configured with.
//
// It is a function rather than a struct literal because a form produces a lot
// of these and validate.RuleError{Rule: "min", Params: []any{3}} is a line of
// punctuation around two pieces of information.
func Failed(rule string, params ...any) RuleError {
	return RuleError{Rule: rule, Params: params}
}

// Of returns a copy that asks for the sentence written for this subject. See
// [RuleError.Subject] for the names.
func (r RuleError) Of(subject string) RuleError {
	r.Subject = subject
	return r
}

// key is what the message table is looked up by, which is the rule on its own
// for a rule with one sentence and the rule and the subject for a rule with
// several.
func (r RuleError) key() string {
	if r.Subject == "" {
		return r.Rule
	}
	return r.Rule + "." + r.Subject
}

// Errors is what failed, collected as it was found.
//
// The zero value is ready and writes English. Checks add to it and it is
// turned into an error once, at the end, so that a caller hears about every
// field rather than the first one.
//
// It is not safe for concurrent use, which is the same as the struct being
// validated.
type Errors struct {
	// Msgs writes the sentence shown next to a field. Nil means [English].
	//
	// It is a field rather than an argument because the checks in between do
	// not care and should not have to carry it. A generated validator sets it
	// once, from the request's locale, and everything after that line is the
	// same code it would be in a program with one language in it.
	Msgs Messages

	// order is the fields in the order they first failed, so that two runs of
	// the same checks produce the same document. A map alone does not.
	order []string
	rules map[string][]RuleError
}

// Add records that a field did not satisfy a rule.
//
// A field may fail more than once and each failure is kept, because "required"
// and "min" are two different things to tell somebody and picking one for them
// is guessing.
func (e *Errors) Add(field string, r RuleError) {
	if e.rules == nil {
		e.rules = make(map[string][]RuleError)
	}
	if _, seen := e.rules[field]; !seen {
		e.order = append(e.order, field)
	}
	e.rules[field] = append(e.rules[field], r)
}

// Len is how many rules failed, counting a field that failed twice twice. It
// is zero for the zero value, so it is also the question "did anything fail".
func (e *Errors) Len() int {
	var n int
	for _, rs := range e.rules {
		n += len(rs)
	}
	return n
}

// Has is whether a field failed anything.
func (e *Errors) Has(field string) bool { return len(e.rules[field]) > 0 }

// First is the message for the first rule a field failed, or the empty string
// if it passed. It is what a template puts under an input.
func (e *Errors) First(field string) string {
	rs := e.rules[field]
	if len(rs) == 0 {
		return ""
	}
	return e.message(field, rs[0])
}

// All is every message, by field. It is a plain map for a template to range
// over, and it is built fresh each call, so writing to it changes nothing.
func (e *Errors) All() map[string][]string {
	if len(e.order) == 0 {
		return nil
	}
	all := make(map[string][]string, len(e.order))
	for _, field := range e.order {
		rs := e.rules[field]
		msgs := make([]string, len(rs))
		for i, r := range rs {
			msgs[i] = e.message(field, r)
		}
		all[field] = msgs
	}
	return all
}

// Fields iterates the failures themselves, field by field, in the order the
// fields first failed.
//
// This is the machine half: the rule names and what they were configured with,
// without a sentence in the way. A client that renders its own messages reads
// this, and so does a test that wants to say which rule failed rather than
// which words were written about it.
func (e *Errors) Fields() iter.Seq2[string, []RuleError] {
	return func(yield func(string, []RuleError) bool) {
		for _, field := range e.order {
			if !yield(field, e.rules[field]) {
				return
			}
		}
	}
}

// Error makes an Errors usable as an error on its own, for a caller that has
// somewhere to put it that is not an HTTP response.
//
// [Errors.OrNil] is what a validator returns, since that is the value the rest
// of the toolkit knows how to answer for.
func (e *Errors) Error() string {
	if e.Len() == 0 {
		return "validate: nothing failed"
	}

	var b strings.Builder
	b.WriteString("validate: ")
	for i, field := range e.order {
		if i > 0 {
			b.WriteString("; ")
		}
		for j, r := range e.rules[field] {
			if j > 0 {
				b.WriteString(" ")
			}
			b.WriteString(e.message(field, r))
		}
	}
	return b.String()
}

// OrNil is nil when nothing failed, and otherwise the error to return.
//
// The error is an [errs.Error] of kind [errs.Unprocessable] with code
// [Code] and one [errs.Field] per rule that failed, which is the shape binding
// produces and the shape a 422 document is written from. The Errors is the
// cause, so [errors.As] reaches the rules and their parameters again:
//
//	var bad *validate.Errors
//	if errors.As(err, &bad) {
//		for field, rules := range bad.Fields() {
//
// A caller that only wants to know whether the request was rejected asks
// [errs.KindOf], and one that wants the messages asks [errs.Fields], neither
// of which has to know that this package exists.
func (e *Errors) OrNil() error {
	n := e.Len()
	if n == 0 {
		return nil
	}

	fields := make([]errs.Field, 0, n)
	for _, field := range e.order {
		for _, r := range e.rules[field] {
			fields = append(fields, errs.Field{
				Name: field,
				Code: r.Rule,
				Msg:  e.message(field, r),
			})
		}
	}

	// The error keeps a copy rather than e itself, so that a caller's
	//
	//	var bad validate.Errors
	//
	// stays on the stack. Handing e to Wrap would make it outlive the function
	// as far as the compiler can tell, and every validator that passes would pay
	// for an Errors on the heap to say that nothing was wrong. The copy shares
	// the map and the slice, so what errors.As hands back reads the same
	// failures under the same names in the same order.
	kept := *e
	return errs.Wrap(&kept, errs.Unprocessable, Code, detail(n)).WithFields(fields...)
}

// message is the sentence for one failure, from whoever is writing them.
func (e *Errors) message(field string, r RuleError) string {
	if e.Msgs == nil {
		return English.Message(field, r)
	}
	return e.Msgs.Message(field, r)
}

// detail is the sentence about the request as a whole, which is what an RFC
// 9457 document puts in detail and what a bare error prints. The count is the
// part worth saying; which fields is already in the fields.
func detail(n int) string {
	if n == 1 {
		return "One field failed validation."
	}
	return strconv.Itoa(n) + " fields failed validation."
}
