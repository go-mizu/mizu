package mizutest

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// recorder stands in for *testing.T so this package can check what an assertion
// says when it fails, which is most of what the package is for.
//
// testing.TB cannot be implemented from outside the testing package, so it is
// embedded and the methods that matter are shadowed. The embedded one is nil,
// so anything not shadowed panics rather than quietly reporting a pass.
//
// It deliberately has no Parallel method, which means [NewApp] leaves a fixture
// built on one serial. That is what the tests here want, and it is also the
// only way to test the parallel default without making the whole file parallel.
type recorder struct {
	testing.TB

	mu       sync.Mutex
	name     string
	failures []string
	logs     []string
	cleanups []func()
}

func (r *recorder) Helper()      {}
func (r *recorder) Name() string { return r.name }

func (r *recorder) Log(a ...any)              { r.record(&r.logs, fmt.Sprint(a...)) }
func (r *recorder) Logf(f string, a ...any)   { r.record(&r.logs, fmt.Sprintf(f, a...)) }
func (r *recorder) Error(a ...any)            { r.record(&r.failures, fmt.Sprint(a...)) }
func (r *recorder) Errorf(f string, a ...any) { r.record(&r.failures, fmt.Sprintf(f, a...)) }
func (r *recorder) Fatal(a ...any)            { r.record(&r.failures, fmt.Sprint(a...)) }
func (r *recorder) Fatalf(f string, a ...any) { r.record(&r.failures, fmt.Sprintf(f, a...)) }
func (r *recorder) Cleanup(fn func())         { r.cleanups = append(r.cleanups, fn) }

func (r *recorder) record(into *[]string, msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	*into = append(*into, msg)
}

func (r *recorder) failed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.failures) > 0
}

// first is the message a reader of a real test would have seen. Fatalf stops
// the goroutine and this cannot, so an assertion that gave up part way through
// carries on here and fails again on the consequences of the first failure.
func (r *recorder) first() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.failures) == 0 {
		return ""
	}
	return r.failures[0]
}

// says checks that something failed and that the first failure mentions each of
// want, which is how every test here that is about wording is written.
func (r *recorder) says(t *testing.T, want ...string) {
	t.Helper()

	if !r.failed() {
		t.Fatalf("nothing failed, want a failure saying %q", want)
	}
	for _, w := range want {
		if !strings.Contains(r.first(), w) {
			t.Errorf("the failure does not mention %q. it said:\n%s", w, r.first())
		}
	}
}

// passed checks that nothing failed, and prints what did when something has.
func (r *recorder) passed(t *testing.T) {
	t.Helper()
	if r.failed() {
		t.Fatalf("the assertion failed, want it to pass:\n%s", r.first())
	}
}

// logged reports whether anything printed through Log mentions s.
func (r *recorder) logged(s string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.logs {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
}

// fake is a fixture over a recorder, for the tests that are about what a
// failure says rather than about a handler.
func fake(t *testing.T, opts ...Option) (*App, *recorder) {
	t.Helper()
	r := &recorder{name: t.Name()}
	app := NewApp(r, opts...)
	t.Cleanup(func() {
		for i := len(r.cleanups) - 1; i >= 0; i-- {
			r.cleanups[i]()
		}
	})
	return app, r
}
