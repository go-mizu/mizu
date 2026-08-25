package validate

import "context"

// A Validator checks itself and says what was wrong.
//
// It is the one interface the rest of the toolkit looks for. A generated
// validator is a method with this signature, a hand-written one is the same
// method written by somebody, and neither has to be registered anywhere: the
// method is on the type, so a value that can check itself is one that does.
//
// The context is there for the checks that need something outside the value,
// a database row or a locale, and is ignored by the ones that do not. It is a
// parameter rather than a field so that a request struct stays a request
// struct, with nothing on it that has to be filled in before it works.
//
// The error is what [Errors.OrNil] returned, so a caller reads it with
// [errs.KindOf] and [errs.Fields] rather than with a type assertion.
type Validator interface {
	Validate(ctx context.Context) error
}
