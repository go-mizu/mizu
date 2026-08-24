package ctxdata

import (
	"context"
	"slices"
	"testing"
)

var (
	tenantID = NewKey[string]("tenant_id", Logged(), Propagated())
	userID   = NewKey[int]("user_id", Logged())
	token    = NewKey[string]("token", Redacted())
	attempt  = NewKey[int]("attempt")
)

func TestWithAndGet(t *testing.T) {
	ctx := With(context.Background(), tenantID, "acme")

	if got, ok := Get(ctx, tenantID); !ok || got != "acme" {
		t.Errorf("Get = %q, %v, want acme, true", got, ok)
	}
	if _, ok := Get(ctx, userID); ok {
		t.Error("a key nobody stored has a value")
	}
	if _, ok := Get(context.Background(), tenantID); ok {
		t.Error("a context nobody stored anything in has a value")
	}
}

// TestWithLeavesTheOldContextAlone is the property that makes a context safe to
// pass around. The one going in does not change.
func TestWithLeavesTheOldContextAlone(t *testing.T) {
	base := With(context.Background(), tenantID, "acme")
	other := With(base, userID, 12)

	if _, ok := Get(base, userID); ok {
		t.Error("storing on a derived context changed the one it came from")
	}
	if got, _ := Get(other, tenantID); got != "acme" {
		t.Errorf("the derived context lost the earlier datum: %q", got)
	}
}

// TestTwoBranchesDoNotShare covers the copy on write. Two contexts derived from
// the same one must not write into the same array.
func TestTwoBranchesDoNotShare(t *testing.T) {
	base := With(context.Background(), tenantID, "acme")
	left := With(base, userID, 1)
	right := With(base, userID, 2)

	if got, _ := Get(left, userID); got != 1 {
		t.Errorf("the left branch has user %d", got)
	}
	if got, _ := Get(right, userID); got != 2 {
		t.Errorf("the right branch has user %d", got)
	}
}

// TestStoringTwiceReplaces is why a chain of middleware that all set the same
// datum does not build a context with fifty copies of it in.
func TestStoringTwiceReplaces(t *testing.T) {
	ctx := With(context.Background(), tenantID, "acme")
	ctx = With(ctx, userID, 12)
	ctx = With(ctx, tenantID, "globex")

	if got, _ := Get(ctx, tenantID); got != "globex" {
		t.Errorf("Get = %q, want globex", got)
	}
	if n := len(bag(ctx)); n != 2 {
		t.Errorf("the context holds %d entries, want 2", n)
	}
	if got, _ := Get(ctx, userID); got != 12 {
		t.Errorf("replacing one datum lost another: %d", got)
	}
}

// TestSameNameIsADifferentKey is the collision that cannot happen. Identity is
// the key variable, not what it is called.
func TestSameNameIsADifferentKey(t *testing.T) {
	mine := NewKey[string]("tenant_id")
	theirs := NewKey[string]("tenant_id")

	ctx := With(context.Background(), mine, "acme")
	if _, ok := Get(ctx, theirs); ok {
		t.Error("another package's key read this one's value")
	}
}

func TestMustGet(t *testing.T) {
	ctx := With(context.Background(), userID, 12)
	if got := MustGet(ctx, userID); got != 12 {
		t.Errorf("MustGet = %d, want 12", got)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("MustGet of a missing datum did not panic")
		}
		if got, want := r.(string), "ctxdata: no value for tenant_id"; got != want {
			t.Errorf("the panic said %q, want %q", got, want)
		}
	}()
	MustGet(ctx, tenantID)
}

func TestName(t *testing.T) {
	if got := tenantID.Name(); got != "tenant_id" {
		t.Errorf("Name = %q", got)
	}
	if got := tenantID.String(); got != "tenant_id" {
		t.Errorf("String = %q", got)
	}
}

func TestKeyNeedsAName(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a key with no name did not panic")
		}
	}()
	NewKey[string]("")
}

// TestZeroKey is the other mistake worth catching loudly. A Key nobody made
// would otherwise read as a datum that is never there.
func TestZeroKey(t *testing.T) {
	for _, c := range []struct {
		name string
		call func()
	}{
		{"With", func() { With(context.Background(), Key[string]{}, "x") }},
		{"Get", func() { Get(context.Background(), Key[string]{}) }},
		{"Name", func() { Key[string]{}.Name() }},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s on the zero Key did not panic", c.name)
				}
			}()
			c.call()
		}()
	}
}

// TestWrongType is what happens when two keys of different types somehow meet
// the same entry. They cannot, since identity is the pointer, so this is here
// to record that the type assertion in Get is not load bearing.
func TestWrongType(t *testing.T) {
	ctx := With(context.Background(), userID, 12)
	if got, ok := Get(ctx, userID); !ok || got != 12 {
		t.Errorf("Get = %d, %v", got, ok)
	}
}

func TestAll(t *testing.T) {
	ctx := context.Background()
	ctx = With(ctx, tenantID, "acme")
	ctx = With(ctx, userID, 12)
	ctx = With(ctx, token, "hunter2")
	ctx = With(ctx, attempt, 3)

	var got []Entry
	for e := range All(ctx) {
		got = append(got, e)
	}

	want := []Entry{
		{Name: "tenant_id", Value: "acme", Logged: true, Propagated: true},
		{Name: "user_id", Value: 12, Logged: true},
		{Name: "token", Value: "hunter2", Logged: true, Redacted: true},
		{Name: "attempt", Value: 3},
	}
	if !slices.Equal(got, want) {
		t.Errorf("All =\n%+v\nwant\n%+v", got, want)
	}
	if got := slices.Collect(All(context.Background())); got != nil {
		t.Errorf("All of an empty context = %+v", got)
	}
}

// TestAllStops checks the iterator honours a break, which a handler that wants
// the first logged datum relies on.
func TestAllStops(t *testing.T) {
	ctx := With(With(context.Background(), tenantID, "acme"), userID, 12)

	n := 0
	for range All(ctx) {
		n++
		break
	}
	if n != 1 {
		t.Errorf("the loop ran %d times after a break", n)
	}
}
