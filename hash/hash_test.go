package hash

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/errs"
)

// cheap is a hasher at parameters nothing should store passwords with, so that
// a test that hashes a hundred times finishes. Every test here uses it except
// the ones about the defaults.
func cheap(t testing.TB) *Argon2id {
	t.Helper()

	h, err := New(Params{Memory: 64, Passes: 1, Lanes: 1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return h
}

func TestHashAndVerify(t *testing.T) {
	h := cheap(t)
	ctx := t.Context()

	encoded, err := h.Hash(ctx, "correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	ok, err := h.Verify(ctx, "correct horse battery staple", encoded)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Error("the password it was given does not verify")
	}

	// A wrong password is an answer and not an error, which is the distinction
	// a sign in handler depends on.
	ok, err = h.Verify(ctx, "correct horse battery stapl", encoded)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if ok {
		t.Error("the wrong password verified")
	}
}

// TestHashIsSalted is the property that makes a password column worth storing.
// The same password twice is two different rows, so a table nobody can read
// still cannot be read for who shares a password with whom.
func TestHashIsSalted(t *testing.T) {
	h := cheap(t)
	ctx := t.Context()

	seen := map[string]bool{}
	for range 8 {
		encoded, err := h.Hash(ctx, "hunter2")
		if err != nil {
			t.Fatalf("Hash: %v", err)
		}
		if seen[encoded] {
			t.Fatal("the same password hashed the same way twice")
		}
		seen[encoded] = true

		ok, _ := h.Verify(ctx, "hunter2", encoded)
		if !ok {
			t.Error("a hash does not verify against the password it came from")
		}
	}
}

// TestHashFormat pins what goes in the column: the algorithm, the version, the
// parameters the hasher was built with, a 16 byte salt and a 32 byte tag.
func TestHashFormat(t *testing.T) {
	h, err := New(Params{Memory: 128, Passes: 3, Lanes: 2})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	encoded, err := h.Hash(t.Context(), "hunter2")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	const prefix = "$argon2id$v=19$m=128,t=3,p=2$"
	if !strings.HasPrefix(encoded, prefix) {
		t.Errorf("hashed as %s, want it to start with %s", encoded, prefix)
	}

	p, err := parse(encoded)
	if err != nil {
		t.Fatalf("what Hash wrote does not parse: %v", err)
	}
	if len(p.salt) != saltSize {
		t.Errorf("the salt is %d bytes, want %d", len(p.salt), saltSize)
	}
	if len(p.tag) != keySize {
		t.Errorf("the tag is %d bytes, want %d", len(p.tag), keySize)
	}
}

// TestVerifyUsesTheStoredCost is what lets parameters change without a
// migration. The hash carries its own cost, so one written when the numbers
// were lower still verifies after they are raised.
func TestVerifyUsesTheStoredCost(t *testing.T) {
	weak, err := New(Params{Memory: 32, Passes: 1, Lanes: 1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	strong, err := New(Params{Memory: 256, Passes: 2, Lanes: 1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := t.Context()

	old, err := weak.Hash(ctx, "hunter2")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	ok, err := strong.Verify(ctx, "hunter2", old)
	if err != nil || !ok {
		t.Errorf("an old hash does not verify against a stronger hasher: %v %v", ok, err)
	}

	// And the other way round, since a rollback happens too.
	fresh, err := strong.Hash(ctx, "hunter2")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if ok, err := weak.Verify(ctx, "hunter2", fresh); err != nil || !ok {
		t.Errorf("a newer hash does not verify against a weaker hasher: %v %v", ok, err)
	}
}

// TestVerifyErrors are the stored values that cannot be checked at all, which
// a caller has to tell apart from a wrong password. Treating one of these as a
// failed sign in is the bug this distinction exists to stop.
func TestVerifyErrors(t *testing.T) {
	h := cheap(t)

	cases := map[string]string{
		"nothing at all": "",
		"not a hash":     "hunter2",
		"a bcrypt hash":  "$2y$12$IvBOSJhWTgLPfDrLZDNC0.aI9C0DAmMHW7bqZzjhKvvCJHqCJRCEO",
		"a cut off hash": "$argon2id$v=19$m=64,t=1,p=1$c29tZXNhbHQ",
	}

	for name, encoded := range cases {
		ok, err := h.Verify(t.Context(), "hunter2", encoded)
		if err == nil {
			t.Errorf("%s: verified with no error", name)
		}
		if ok {
			t.Errorf("%s: verified", name)
		}
	}
}

// TestVerifyRefusesAnUnaffordableHash is the ceiling on what a stored value may
// ask for. A hash names its own cost and checking it spends that cost, so a
// single row in the password column decides how much memory a request holds and
// for how long.
//
// The two inputs below came out of the fuzzer within seconds of each other. The
// first fills 4 GiB and the second runs for the better part of an hour, and
// either one is a way to take the logins down from one row.
func TestVerifyRefusesAnUnaffordableHash(t *testing.T) {
	h := cheap(t)
	tag := "$c2FsdHNhbHRzYWx0c2FsdA$" + b64.EncodeToString(counting(32))

	cases := map[string]string{
		"far too much memory": "$argon2id$v=19$m=4194304,t=1,p=1" + tag,
		"far too many passes": "$argon2id$v=19$m=72,t=500000,p=1" + tag,
	}

	for name, encoded := range cases {
		done := make(chan error, 1)
		go func() {
			_, err := h.Verify(t.Context(), "hunter2", encoded)
			done <- err
		}()

		select {
		case err := <-done:
			if errs.CodeOf(err) != "hash.too_costly" {
				t.Errorf("%s: %v, want hash.too_costly", name, err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: Verify started hashing it", name)
		}
	}

	// The ceiling is generous, so a hash from a configuration that was more
	// expensive than this one still verifies. That is the case it must not
	// break: a stronger old hash is the normal state of a password column
	// during a change of parameters.
	stronger, err := New(Params{Memory: 64 * maxStoredCost, Passes: 1, Lanes: 1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	encoded, err := stronger.Hash(t.Context(), "hunter2")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if ok, err := h.Verify(t.Context(), "hunter2", encoded); err != nil || !ok {
		t.Errorf("a hash at the ceiling does not verify: %v %v", ok, err)
	}
}

func TestNeedsRehash(t *testing.T) {
	h, err := New(Params{Memory: 128, Passes: 2, Lanes: 1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	fresh, err := h.Hash(t.Context(), "hunter2")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if h.NeedsRehash(fresh) {
		t.Error("a hash this hasher wrote wants rehashing")
	}

	yes := map[string]string{
		"less memory":    "$argon2id$v=19$m=64,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$" + b64.EncodeToString(counting(32)),
		"fewer passes":   "$argon2id$v=19$m=128,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$" + b64.EncodeToString(counting(32)),
		"other lanes":    "$argon2id$v=19$m=128,t=2,p=2$c2FsdHNhbHRzYWx0c2FsdA$" + b64.EncodeToString(counting(32)),
		"a shorter salt": "$argon2id$v=19$m=128,t=2,p=1$c29tZXNhbHQ$" + b64.EncodeToString(counting(32)),
		"a shorter tag":  "$argon2id$v=19$m=128,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$" + b64.EncodeToString(counting(16)),
		"bcrypt":         "$2y$12$IvBOSJhWTgLPfDrLZDNC0.aI9C0DAmMHW7bqZzjhKvvCJHqCJRCEO",
		"nothing at all": "",
		"rubbish":        "hunter2",
	}
	for name, encoded := range yes {
		if !h.NeedsRehash(encoded) {
			t.Errorf("%s: does not want rehashing", name)
		}
	}

	// A hash that cost more than this hasher does is left alone. Replacing it
	// would be a downgrade, which is not what anybody asking this wants.
	no := map[string]string{
		"more memory": "$argon2id$v=19$m=65536,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$" + b64.EncodeToString(counting(32)),
		"more passes": "$argon2id$v=19$m=128,t=10,p=1$c2FsdHNhbHRzYWx0c2FsdA$" + b64.EncodeToString(counting(32)),
		"a longer salt and tag": "$argon2id$v=19$m=128,t=2,p=1$" +
			b64.EncodeToString(counting(32)) + "$" + b64.EncodeToString(counting(64)),
	}
	for name, encoded := range no {
		if h.NeedsRehash(encoded) {
			t.Errorf("%s: wants rehashing", name)
		}
	}
}

// TestRehashOnSignIn is the whole of moving an application to stronger
// parameters, written out as the handler would.
func TestRehashOnSignIn(t *testing.T) {
	before, err := New(Params{Memory: 64, Passes: 1, Lanes: 1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	after, err := New(Params{Memory: 128, Passes: 2, Lanes: 1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := t.Context()

	stored, err := before.Hash(ctx, "hunter2")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	ok, err := after.Verify(ctx, "hunter2", stored)
	if err != nil || !ok {
		t.Fatalf("the stored hash does not verify: %v %v", ok, err)
	}
	if !after.NeedsRehash(stored) {
		t.Fatal("the stored hash does not want rehashing")
	}

	stored, err = after.Hash(ctx, "hunter2")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if after.NeedsRehash(stored) {
		t.Error("the hash written by the upgrade still wants rehashing")
	}
	if ok, err := after.Verify(ctx, "hunter2", stored); err != nil || !ok {
		t.Errorf("the rehashed password does not verify: %v %v", ok, err)
	}
}

// TestDefaults pins the parameters an application gets for asking for nothing.
// They are the OWASP recommendation, and changing them is a decision rather
// than an edit.
func TestDefaults(t *testing.T) {
	h := Default()
	p := h.Params()

	if p.Memory != 19456 || p.Passes != 2 || p.Lanes != 1 {
		t.Errorf("the defaults are m=%d t=%d p=%d, want m=19456 t=2 p=1", p.Memory, p.Passes, p.Lanes)
	}
	if p.VerifyTimeout != 5*time.Second {
		t.Errorf("the default timeout is %s", p.VerifyTimeout)
	}
	if p.MaxConcurrent < 1 {
		t.Errorf("the default allows %d hashes at once", p.MaxConcurrent)
	}

	// A field left alone keeps its default while the others change.
	partial, err := New(Params{Passes: 5})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := partial.Params(); got.Passes != 5 || got.Memory != 19456 || got.Lanes != 1 {
		t.Errorf("m=%d t=%d p=%d, want the defaults around t=5", got.Memory, got.Passes, got.Lanes)
	}
}

func TestNewRejects(t *testing.T) {
	cases := map[string]Params{
		"negative memory":  {Memory: -1},
		"negative passes":  {Passes: -1},
		"negative lanes":   {Lanes: -1},
		"negative limit":   {MaxConcurrent: -1},
		"negative timeout": {VerifyTimeout: -time.Second},
		"too many lanes":   {Lanes: 256},
		"memory below lanes": {
			Memory: 16,
			Lanes:  4,
		},
	}

	for name, p := range cases {
		h, err := New(p)
		if errs.CodeOf(err) != "hash.params" || errs.KindOf(err) != errs.Invalid {
			t.Errorf("%s: %v, want hash.params", name, err)
		}
		if h != nil {
			t.Errorf("%s: New returned a hasher along with the error", name)
		}
	}
}

// TestNewRejectsUncountableCost is the bound where the cost stops fitting in
// what argon2 counts in. It is a 64 bit machine only, since on a 32 bit one an
// int cannot get that far in the first place.
func TestNewRejectsUncountableCost(t *testing.T) {
	if math.MaxInt <= math.MaxUint32 {
		t.Skip("an int does not reach that far here")
	}
	tooMuch := math.MaxInt

	if _, err := New(Params{Memory: tooMuch}); errs.CodeOf(err) != "hash.params" {
		t.Errorf("memory: %v, want hash.params", err)
	}
	if _, err := New(Params{Passes: tooMuch}); errs.CodeOf(err) != "hash.params" {
		t.Errorf("passes: %v, want hash.params", err)
	}
}

// TestHasherIsConcurrent is the shape a hasher is used in: one of them, shared
// by every request handler.
func TestHasherIsConcurrent(t *testing.T) {
	h := cheap(t)
	ctx := t.Context()

	encoded, err := h.Hash(ctx, "hunter2")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	done := make(chan bool, 16)
	for i := range 16 {
		go func() {
			if i%2 == 0 {
				_, err := h.Hash(ctx, "hunter2")
				done <- err == nil
				return
			}
			ok, err := h.Verify(ctx, "hunter2", encoded)
			done <- ok && err == nil
		}()
	}
	for range 16 {
		if !<-done {
			t.Error("a hash or a verify failed while others were running")
		}
	}
}

// TestLongPassword is the input a password field with no maximum length
// receives sooner or later. Argon2 hashes it the same as any other.
func TestLongPassword(t *testing.T) {
	h := cheap(t)
	ctx := t.Context()

	long := strings.Repeat("a very long passphrase ", 1000)
	encoded, err := h.Hash(ctx, long)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	if ok, err := h.Verify(ctx, long, encoded); err != nil || !ok {
		t.Errorf("a long password does not verify: %v %v", ok, err)
	}

	// bcrypt stops reading at 72 bytes, which is how two different long
	// passwords come to match the same hash. This does not.
	if ok, _ := h.Verify(ctx, long+"x", encoded); ok {
		t.Error("a longer password verified against the shorter one")
	}
}

// TestEmptyPassword is the first keystroke on a sign in form, and it hashes
// like anything else rather than failing somewhere further down.
func TestEmptyPassword(t *testing.T) {
	h := cheap(t)
	ctx := t.Context()

	encoded, err := h.Hash(ctx, "")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if ok, err := h.Verify(ctx, "", encoded); err != nil || !ok {
		t.Errorf("an empty password does not verify: %v %v", ok, err)
	}
	if ok, _ := h.Verify(ctx, "x", encoded); ok {
		t.Error("a password verified against the hash of an empty one")
	}
}

// FuzzVerify is the property that matters for a value out of a database: no
// input makes this panic, and none of them verify against a password.
func FuzzVerify(f *testing.F) {
	h := cheap(f)

	encoded, err := h.Hash(f.Context(), "hunter2")
	if err != nil {
		f.Fatalf("Hash: %v", err)
	}
	f.Add(encoded)
	f.Add("$argon2id$v=19$m=64,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8")
	f.Add("$2y$12$IvBOSJhWTgLPfDrLZDNC0.aI9C0DAmMHW7bqZzjhKvvCJHqCJRCEO")
	f.Add("")

	// The fuzzer finds m=4194304 and t=500000 within seconds, and without the
	// ceiling on what a stored hash may ask for it would sit there filling
	// memory or running for an hour. That it does not is the property, so
	// nothing is skipped here.
	ctx := context.Background()
	f.Fuzz(func(t *testing.T, encoded string) {
		ok, err := h.Verify(ctx, "no", encoded)
		if ok && err != nil {
			t.Errorf("verified and returned an error: %v", err)
		}
		if ok {
			t.Errorf("%q verified against a password that did not make it", encoded)
		}
		h.NeedsRehash(encoded)
	})
}
