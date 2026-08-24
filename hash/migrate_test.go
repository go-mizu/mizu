package hash

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/errs"
)

// migrating is a hasher at test cost that reads bcrypt as well, which is what
// an application moving off Laravel runs.
func migrating(t testing.TB) *Migration {
	t.Helper()
	return Migrating(cheap(t), Bcrypt{MaxCost: 4})
}

// TestMigrationReadsBoth is the point of the type: one call site, two formats,
// and nothing at the call site that knows which is which.
func TestMigrationReadsBoth(t *testing.T) {
	h := migrating(t)

	old := bcryptHashes["cost 4"]
	ok, err := h.Verify(t.Context(), "hunter2", old)
	if err != nil || !ok {
		t.Fatalf("the bcrypt hash did not verify: %v, %v", ok, err)
	}

	fresh, err := h.Hash(t.Context(), "hunter2")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if !strings.HasPrefix(fresh, "$argon2id$") {
		t.Fatalf("it wrote %s", fresh)
	}
	ok, err = h.Verify(t.Context(), "hunter2", fresh)
	if err != nil || !ok {
		t.Fatalf("the argon2id hash did not verify: %v, %v", ok, err)
	}

	// And a wrong password is false and no error in both formats, which is the
	// answer a sign in handler branches on.
	for name, encoded := range map[string]string{"bcrypt": old, "argon2id": fresh} {
		ok, err := h.Verify(t.Context(), "hunter3", encoded)
		if err != nil {
			t.Errorf("%s: a wrong password gave an error: %v", name, err)
		}
		if ok {
			t.Errorf("%s: a wrong password verified", name)
		}
	}
}

// TestRehashOnSignInFromBcrypt walks the migration end to end, because the
// three calls in the documentation are the whole feature and they have to work
// together rather than one at a time.
func TestRehashOnSignInFromBcrypt(t *testing.T) {
	h := migrating(t)
	stored := bcryptHashes["cost 4"]

	// The first sign in. The password is right, the hash is somebody else's,
	// and this is the one moment it can be replaced.
	ok, err := h.Verify(t.Context(), "hunter2", stored)
	if err != nil || !ok {
		t.Fatalf("sign in: %v, %v", ok, err)
	}
	if !h.NeedsRehash(stored) {
		t.Fatal("a bcrypt hash does not need rehashing")
	}
	if stored, err = h.Hash(t.Context(), "hunter2"); err != nil {
		t.Fatalf("rehash: %v", err)
	}

	// The second sign in. The account has moved and nobody was asked to do
	// anything, and it does not move again.
	ok, err = h.Verify(t.Context(), "hunter2", stored)
	if err != nil || !ok {
		t.Fatalf("sign in again: %v, %v", ok, err)
	}
	if h.NeedsRehash(stored) {
		t.Error("it wants to rehash what it wrote a moment ago")
	}
}

// TestMigrationErrorsComeFromTheWriter is the error somebody reads when a
// password column has something in it that is not a hash at all. The value did
// not match any of the old formats, so what answers is the one that writes, and
// its message says what was wrong rather than that bcrypt did not recognize it.
func TestMigrationErrorsComeFromTheWriter(t *testing.T) {
	h := migrating(t)

	ok, err := h.Verify(t.Context(), "hunter2", "not a hash")
	if errs.CodeOf(err) != "hash.malformed" {
		t.Errorf("%v, want hash.malformed", err)
	}
	if ok {
		t.Error("it verified")
	}

	// A format neither of them reads is unsupported and names itself, which is
	// what says which verifier to add next.
	_, err = h.Verify(t.Context(), "hunter2", "$scrypt$ln=16,r=8,p=1$c29tZXNhbHQ$RdescudvJCsgt3ub")
	if errs.CodeOf(err) != "hash.unsupported" {
		t.Errorf("%v, want hash.unsupported", err)
	}
	if !strings.Contains(err.Error(), "scrypt") {
		t.Errorf("the message does not name scrypt: %v", err)
	}
}

// TestMigrationTriesInOrder says that the first verifier that recognizes a
// value is the one that answers, so an application with two readers of the same
// format gets the one it listed first.
func TestMigrationTriesInOrder(t *testing.T) {
	first := &counter{Bcrypt: Bcrypt{MaxCost: 4}}
	second := &counter{Bcrypt: Bcrypt{MaxCost: 4}}
	h := Migrating(cheap(t), first, second)

	if _, err := h.Verify(t.Context(), "hunter2", bcryptHashes["cost 4"]); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if first.calls != 1 || second.calls != 0 {
		t.Errorf("the first was called %d times and the second %d", first.calls, second.calls)
	}

	// And a value neither reads reaches neither of them.
	h.Verify(t.Context(), "hunter2", "not a hash")
	if first.calls != 1 || second.calls != 0 {
		t.Errorf("something that is not bcrypt reached a bcrypt verifier")
	}
}

// counter is a Verifier that says how often it was asked.
type counter struct {
	Bcrypt
	calls int
}

func (c *counter) Verify(ctx context.Context, password, encoded string) (bool, error) {
	c.calls++
	return c.Bcrypt.Verify(ctx, password, encoded)
}

// TestMigrationBoundsEveryFormat is the reason the gate is on the migration
// rather than on the argon2id hasher alone. An old hash costs the machine
// something too, and a login flood does not care which format the column is in.
func TestMigrationBoundsEveryFormat(t *testing.T) {
	to, err := New(Params{Memory: 64, Passes: 1, MaxConcurrent: 1, VerifyTimeout: time.Millisecond})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	h := Migrating(to, Bcrypt{MaxCost: 4})

	held, err := to.gate.enter(t.Context())
	if err != nil {
		t.Fatalf("enter: %v", err)
	}
	defer held()

	ok, err := h.Verify(t.Context(), "hunter2", bcryptHashes["cost 4"])
	if errs.CodeOf(err) != "hash.busy" {
		t.Errorf("%v, want hash.busy", err)
	}
	if ok {
		t.Error("it verified without a slot to run in")
	}
}

// TestMigrationReleasesItsSlot is the other half of that: a bcrypt verify that
// finishes gives the slot back, or the second sign in of the process hangs.
func TestMigrationReleasesItsSlot(t *testing.T) {
	to, err := New(Params{Memory: 64, Passes: 1, MaxConcurrent: 1, VerifyTimeout: time.Second})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	h := Migrating(to, Bcrypt{MaxCost: 4})

	for range 3 {
		if _, err := h.Verify(t.Context(), "hunter2", bcryptHashes["cost 4"]); err != nil {
			t.Fatalf("verify: %v", err)
		}
	}
}

// TestMigrationParams passes the cost through, since what a migration writes is
// what the hasher underneath it writes.
func TestMigrationParams(t *testing.T) {
	to := cheap(t)
	if got := Migrating(to).Params(); got != to.Params() {
		t.Errorf("the parameters came back as %+v", got)
	}
}

// TestMigratingCopiesItsVerifiers says a slice the caller keeps hold of cannot
// change what a hasher reads after the fact.
func TestMigratingCopiesItsVerifiers(t *testing.T) {
	from := []Verifier{Bcrypt{MaxCost: 4}}
	h := Migrating(cheap(t), from...)

	from[0] = Bcrypt{MaxCost: 31}
	if _, err := h.Verify(t.Context(), "hunter2", "$2a$31$CCCCCCCCCCCCCCCCCCCCC.E5YPO9kmyuRGyh0XouQYb4YMJKvyOeW"); errs.CodeOf(err) != "hash.too_costly" {
		t.Errorf("%v, want hash.too_costly", err)
	}
}
