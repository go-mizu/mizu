package errs_test

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-mizu/mizu/errs"
)

// A repository says what went wrong once, and every transport reads the answer
// off the kind rather than deciding again.
func Example() {
	err := errs.NotFoundf("no post with id %d", 7).WithCode("post.not_found")

	fmt.Println(err)
	fmt.Println(errs.KindOf(err), errs.CodeOf(err))
	fmt.Println(errs.KindOf(err).Status())
	// Output:
	// no post with id 7
	// not_found post.not_found
	// 404
}

// A kind is an error, so matching one needs no sentinel variable per failure.
func ExampleKind() {
	err := errs.Invalidf("page must be a number")

	fmt.Println(errors.Is(err, errs.Invalid))
	fmt.Println(errors.Is(err, errs.NotFound))
	// Output:
	// true
	// false
}

// Wrap keeps the cause reachable, so a handler classifies a failure without
// hiding what a log needs.
func ExampleWrap() {
	err := errs.Wrap(sql.ErrNoRows, errs.NotFound, "post.not_found", "that post is gone")

	fmt.Println(err)
	fmt.Println(errors.Is(err, sql.ErrNoRows))
	// Output:
	// that post is gone: sql: no rows in result set
	// true
}

// Fields are the individual things wrong with a request, which is what
// validation produces and what a form redisplays.
func ExampleError_WithField() {
	err := errs.Invalidf("that form has two problems").
		WithCode("user.invalid").
		WithField("email", "email.taken", "that address is already registered").
		WithField("age", "number.min", "you have to be 18")

	for _, f := range errs.Fields(err) {
		fmt.Printf("%s: %s (%s)\n", f.Name, f.Msg, f.Code)
	}
	// Output:
	// email: that address is already registered (email.taken)
	// age: you have to be 18 (number.min)
}

// Whether trying again is worth anything is part of what the kind decides, and
// an error that knows how long to wait says so.
func ExampleRetryable() {
	busy := errs.New(errs.Unavailable, "search.down", "search is unavailable").
		WithRetry(30 * time.Second)
	wrong := errs.Invalidf("page must be a number")

	fmt.Println(errs.Retryable(busy), errs.Retryable(wrong))

	after, ok := errs.RetryAfter(busy)
	fmt.Println(after, ok)
	// Output:
	// true false
	// 30s true
}

// One place turns an error into a response, and it asks the error rather than
// the handler that produced it.
func ExampleKind_Status() {
	respond := func(err error) (int, string) {
		k := errs.KindOf(err)
		if !k.Safe() {
			return k.Status(), http.StatusText(k.Status())
		}
		return k.Status(), err.Error()
	}

	fmt.Println(respond(errs.NotFoundf("no post with id 7")))
	fmt.Println(respond(errs.Internalf("connection pool exhausted")))
	// Output:
	// 404 no post with id 7
	// 500 Internal Server Error
}
