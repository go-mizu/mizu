package hash

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"math"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/go-mizu/mizu/errs"
)

// A Hasher turns a password into something safe to store and checks a password
// against one.
//
// It is an interface for the two things an application does with it: swap in a
// cheap one in tests, where real parameters make a suite that creates users
// unusably slow, and read a hash written by whatever the application used
// before this one.
type Hasher interface {
	// Hash returns the encoded hash of a password.
	Hash(ctx context.Context, password string) (string, error)

	// Verify reports whether a password matches an encoded hash. A password
	// that does not match is false and no error. An error means the check did
	// not happen: the stored value is not one this reads, or there was no room
	// to run.
	Verify(ctx context.Context, password, encoded string) (bool, error)

	// NeedsRehash reports whether an encoded hash should be replaced, which is
	// how an application moves to stronger parameters without asking anybody to
	// change their password.
	NeedsRehash(encoded string) bool
}

// Params are the cost of a hash.
//
// The three that matter are the ones RFC 9106 names, and they go into the
// encoded hash as m, t and p, so a value read back carries the cost it was
// written with and old hashes keep working after the numbers change.
//
// Every field has a default, so the zero value is the OWASP recommendation and
// [New] with it is the right thing to do until a measurement says otherwise.
// [Argon2id.Params] gives back what a hasher settled on, which is worth logging
// at startup: parameters tuned on a build machine and deployed to a smaller one
// are a slow login that nobody can explain.
type Params struct {
	// Memory is how much a single hash holds while it runs, in kibibytes. It is
	// the parameter that costs an attacker the most, since it is what stops a
	// graphics card from running thousands of guesses at once.
	//
	// The default is 19456, which is the 19 MiB OWASP recommends.
	Memory int

	// Passes is how many times the hash goes over that memory. Raising it costs
	// time without costing memory, which is the parameter to raise when memory
	// is already as high as the machine allows.
	//
	// The default is 2.
	Passes int

	// Lanes is how many lanes the memory is divided into, up to 255.
	//
	// The lanes run at the same time, so raising this makes one hash finish
	// sooner without making it any cheaper for somebody guessing passwords on
	// the same number of cores. It also means one hash occupies more than one
	// processor, which is worth reading next to MaxConcurrent: four lanes and
	// eight concurrent hashes is thirty two cores' worth of work.
	//
	// The default is 1.
	Lanes int

	// MaxConcurrent is how many hashes may run at once. The rest wait.
	//
	// The default, 0, works it out from GOMEMLIMIT and Memory, which is the
	// number that keeps a burst of logins from becoming an out of memory kill.
	MaxConcurrent int

	// VerifyTimeout is how long a hash waits for its turn before giving up with
	// an error of kind [errs.Unavailable], which is a 503 rather than a request
	// that hangs until the client gives up.
	//
	// The default is 5 seconds.
	VerifyTimeout time.Duration
}

// Argon2id hashes passwords with argon2id, which is what RFC 9106 recommends
// and what a new application should use.
//
// It is safe for concurrent use, and it is meant to be made once and kept: the
// limit on how many hashes run at once lives in it, so two of them do not know
// about each other.
type Argon2id struct {
	p    Params
	gate *gate
}

// Argon2id is a [Hasher].
var _ Hasher = (*Argon2id)(nil)

// Default is argon2id at the parameters OWASP recommends: 19 MiB of memory,
// two passes and one lane.
//
// They are the floor rather than a target. A machine that can afford more
// should be measured, and mizu hash:tune is what measures it.
func Default() *Argon2id {
	// The defaults are the only input, and they are inside every bound New
	// checks, so the error is one that cannot happen.
	h, _ := New(Params{})
	return h
}

// New returns a hasher at the given cost. A field left at zero takes its
// default, so New(Params{}) is [Default].
func New(p Params) (*Argon2id, error) {
	if p.Memory == 0 {
		p.Memory = 19456
	}
	if p.Passes == 0 {
		p.Passes = 2
	}
	if p.Lanes == 0 {
		p.Lanes = 1
	}
	if p.VerifyTimeout == 0 {
		p.VerifyTimeout = 5 * time.Second
	}

	switch {
	case p.Passes < 0 || p.Lanes < 0 || p.Memory < 0 || p.MaxConcurrent < 0 || p.VerifyTimeout < 0:
		return nil, errs.New(errs.Invalid, "hash.params", "hash: a cost cannot be negative")
	case p.Lanes > 255:
		return nil, errs.Newf(errs.Invalid, "hash.params", "hash: %d lanes is more than the 255 argon2id runs here", p.Lanes)
	case uint64(p.Memory) > math.MaxUint32 || uint64(p.Passes) > math.MaxUint32:
		return nil, errs.New(errs.Invalid, "hash.params", "hash: that is a larger cost than argon2id counts in")
	case p.Memory < 8*p.Lanes:
		return nil, errs.Newf(errs.Invalid, "hash.params",
			"hash: %d KiB is below the %d KiB that %d lanes need", p.Memory, 8*p.Lanes, p.Lanes)
	}

	if p.MaxConcurrent == 0 {
		p.MaxConcurrent = concurrency(p.Memory, memoryLimit(), cpus())
	}
	return &Argon2id{p: p, gate: newGate(p.MaxConcurrent, p.VerifyTimeout)}, nil
}

// Params is the cost this hasher settled on, with every default filled in.
func (h *Argon2id) Params() Params { return h.p }

// Hash returns the encoded hash of a password, with a fresh 16 byte salt.
//
// The same password hashes differently every time, which is the salt doing its
// job: two people who chose the same password do not look alike in the table,
// and a precomputed answer for a common password is worth nothing.
func (h *Argon2id) Hash(ctx context.Context, password string) (string, error) {
	release, err := h.gate.enter(ctx)
	if err != nil {
		return "", err
	}
	defer release()

	salt := make([]byte, saltSize)
	rand.Read(salt)

	return phc{
		memory: uint32(h.p.Memory),
		passes: uint32(h.p.Passes),
		lanes:  uint8(h.p.Lanes),
		salt:   salt,
		tag:    argon2.IDKey([]byte(password), salt, uint32(h.p.Passes), uint32(h.p.Memory), uint8(h.p.Lanes), keySize),
	}.encode(), nil
}

// maxStoredCost is how much more expensive a stored hash may be than this
// hasher is configured to be, before it is refused rather than run.
//
// A hash names its own cost, and what it names is what checking it spends. That
// is what lets parameters change without invalidating anything written before,
// and it means a row in the password column decides how much memory a request
// holds and for how long. A row asking for 4 TiB is an out of memory kill, and
// one asking for half a million passes holds a slot until the process restarts,
// so anybody who can write one row can take the logins down.
//
// Sixteen times is far above any real hash and far below anything dangerous. At
// the defaults it accepts up to 304 MiB and 32 passes, which covers every set
// of parameters anything has recommended, including the ones from before this
// application was written.
//
// The bound is a multiple of the configured cost rather than a share of the
// memory limit, so that a row verifies or does not verify for reasons that are
// in the configuration, and not because one machine has more memory than
// another.
const maxStoredCost = 16

// Verify reports whether a password matches an encoded hash.
//
// The cost comes from the stored hash and not from this hasher, which is what
// lets parameters change without invalidating anything written before. A hash
// far more expensive than this hasher is configured to be is refused rather
// than run, for the reasons on [maxStoredCost].
func (h *Argon2id) Verify(ctx context.Context, password, encoded string) (bool, error) {
	p, err := parse(encoded)
	if err != nil {
		return false, err
	}
	if err := h.affordable(p); err != nil {
		return false, err
	}

	release, err := h.gate.enter(ctx)
	if err != nil {
		return false, err
	}
	defer release()

	tag := argon2.IDKey([]byte(password), p.salt, p.passes, p.memory, p.lanes, uint32(len(p.tag)))
	return subtle.ConstantTimeCompare(tag, p.tag) == 1, nil
}

// affordable reports whether a stored hash asks for an amount of work this
// hasher is willing to spend checking it.
func (h *Argon2id) affordable(p phc) error {
	if int64(p.memory) > maxStoredCost*int64(h.p.Memory) {
		return errs.Newf(errs.Unsupported, "hash.too_costly",
			"hash: this hash asks for %d KiB, more than %d times the %d KiB this hasher is configured for",
			p.memory, maxStoredCost, h.p.Memory)
	}
	if int64(p.passes) > maxStoredCost*int64(h.p.Passes) {
		return errs.Newf(errs.Unsupported, "hash.too_costly",
			"hash: this hash asks for %d passes, more than %d times the %d this hasher is configured for",
			p.passes, maxStoredCost, h.p.Passes)
	}
	return nil
}

// NeedsRehash reports whether an encoded hash should be replaced.
//
// It says yes when the stored hash cost less than this hasher does, when
// something else wrote it, and when it is not a hash at all. The place to call
// it is right after a successful sign in, where the password is in hand and can
// be hashed again:
//
//	ok, err := h.Verify(ctx, password, user.Password)
//	if err != nil || !ok {
//		return errs.New(errs.Unauthenticated, "auth.bad_password", "that is not the password")
//	}
//	if h.NeedsRehash(user.Password) {
//		user.Password, err = h.Hash(ctx, password)
//	}
//
// That is the whole of moving an application off bcrypt. Everybody who signs in
// is moved, nobody is asked to do anything, and the old hashes that are left
// belong to accounts that have not been used since.
func (h *Argon2id) NeedsRehash(encoded string) bool {
	p, err := parse(encoded)
	if err != nil {
		return true
	}

	// A stored hash that cost more than this one is left alone. Rehashing it
	// would be replacing it with something weaker, which is not what anybody
	// calling this wants.
	return p.memory < uint32(h.p.Memory) ||
		p.passes < uint32(h.p.Passes) ||
		p.lanes != uint8(h.p.Lanes) ||
		len(p.salt) < saltSize ||
		len(p.tag) < keySize
}
