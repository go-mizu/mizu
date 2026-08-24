package hash

import (
	"context"
	"slices"
)

// A Verifier checks passwords against hashes that something else wrote.
//
// It is the half of [Hasher] an application needs while it is moving off
// whatever it used before. [Bcrypt] is the one implementation here, and the
// interface is exported so that an application carrying a format nobody else
// has can add its own and hand it to [Migrating].
type Verifier interface {
	// Reads reports whether a stored value is a hash of the kind this checks.
	// It looks at the format and nothing else, so it is cheap enough to call
	// on every sign in, and it says nothing about whether the hash is valid.
	Reads(encoded string) bool

	// Verify reports whether a password matches an encoded hash. A password
	// that does not match is false and no error. An error means the check did
	// not happen.
	//
	// It must not bound how many hashes run at once. [Migrating] does that for
	// every verifier it holds, and a verifier that did it again would be a
	// second queue behind the first.
	Verify(ctx context.Context, password, encoded string) (bool, error)
}

// A Migration writes one kind of hash and reads several.
//
// It is what an application uses from the day it starts reading somebody else's
// password column until the day the last of those hashes is gone, which is to
// say most of the life of the application.
type Migration struct {
	to   *Argon2id
	from []Verifier
}

// Migration is a [Hasher].
var _ Hasher = (*Migration)(nil)

// Migrating returns a [Hasher] that writes what to writes and reads what to or
// any of from wrote.
//
// This is the whole of moving an application off bcrypt:
//
//	h := hash.Migrating(hash.Default(), hash.Bcrypt{})
//
//	ok, err := h.Verify(ctx, password, user.Password)
//	if err != nil || !ok {
//		return errs.New(errs.Unauthenticated, "auth.bad_password", "that is not the password")
//	}
//	if h.NeedsRehash(user.Password) {
//		user.Password, err = h.Hash(ctx, password)
//	}
//
// Nobody is asked to change their password and nothing is migrated in a batch,
// because the only moment an old hash can be replaced is the one moment the
// password is in hand. [Argon2id.NeedsRehash] says yes for every hash to did
// not write, so the second half of that runs exactly once per account.
//
// A stored value is offered to each of from in order and to to last, so the
// error for something that is not a hash at all is to's, which is the one that
// says what was wrong with it.
func Migrating(to *Argon2id, from ...Verifier) *Migration {
	return &Migration{to: to, from: slices.Clone(from)}
}

// Hash returns the encoded hash of a password, in the format this writes.
func (m *Migration) Hash(ctx context.Context, password string) (string, error) {
	return m.to.Hash(ctx, password)
}

// Verify reports whether a password matches an encoded hash, whichever of the
// formats this reads it is in.
//
// Every hash goes through the same limit on how many run at once, because the
// reason for that limit is the machine and not the algorithm. An old hash is
// still memory and processor time held for the length of a request.
func (m *Migration) Verify(ctx context.Context, password, encoded string) (bool, error) {
	v := m.reader(encoded)
	if v == nil {
		return m.to.Verify(ctx, password, encoded)
	}

	release, err := m.to.gate.enter(ctx)
	if err != nil {
		return false, err
	}
	defer release()

	return v.Verify(ctx, password, encoded)
}

// NeedsRehash reports whether an encoded hash should be replaced. Every hash
// this reads and did not write needs one, which is what makes the migration
// finish.
func (m *Migration) NeedsRehash(encoded string) bool {
	return m.to.NeedsRehash(encoded)
}

// Params is the cost of the hashes this writes.
func (m *Migration) Params() Params { return m.to.Params() }

// reader is the first verifier that recognizes a stored value, or nil for one
// that only to can answer for.
func (m *Migration) reader(encoded string) Verifier {
	for _, v := range m.from {
		if v.Reads(encoded) {
			return v
		}
	}
	return nil
}
