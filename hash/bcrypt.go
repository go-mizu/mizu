package hash

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-mizu/mizu/errs"
)

// Bcrypt checks bcrypt hashes. It does not write one.
//
// A password column full of bcrypt is what an application arrives with, because
// bcrypt is what PHP, Ruby and Node wrote for twenty years and what Laravel
// still writes today. The hashes are there, the passwords behind them are not,
// and asking everybody to reset theirs is the way to lose the accounts that
// only sign in twice a year. So this reads them, [Migrating] moves each account
// to argon2id the next time its owner signs in, and after a while what is left
// belongs to people who have not been back.
//
// Writing one is not offered. bcrypt costs an attacker time and nothing else,
// which is the whole reason argon2id exists, and it silently ignores everything
// after the 72nd byte of a password, which is not a property to hand to
// somebody starting today. There is no case where a new hash should be bcrypt:
// an application still sharing its users table with the system it is migrating
// away from can write argon2id, because PHP has read argon2id since 7.2.
//
// The zero value reads hashes at any cost up to [Bcrypt.MaxCost]'s default.
type Bcrypt struct {
	// MaxCost is the highest cost this checks, and 0 means 15.
	//
	// A bcrypt hash names its own cost and each step up doubles the work, so
	// cost 31 is years of processor time for one sign in attempt and a row
	// carrying it is a way to take the logins down. This is the same bound
	// [maxStoredCost] puts on argon2id, for the same reason.
	//
	// The default is five steps above the cost anything recommends, so it is
	// far above every hash that exists and far below the ones that are only a
	// weapon. Laravel writes 12, and 15 is about four seconds here.
	MaxCost int
}

// Bcrypt is a [Verifier].
var _ Verifier = Bcrypt{}

// defaultMaxCost is [Bcrypt.MaxCost] when it is left at zero.
const defaultMaxCost = 15

// bcryptLimit is where bcrypt stops reading a password.
//
// The limit is in the algorithm rather than in any one implementation of it.
// Blowfish sets up its P array from exactly 72 bytes of key, and the S boxes
// after that come from the P array and the salt, so a 73rd byte has nowhere to
// go. Every bcrypt agrees on this because none of them had a choice, which is
// why a hash written by PHP still reads here for a password longer than that.
const bcryptLimit = 72

// Reads reports whether a stored value is a bcrypt hash.
//
// It is the prefix and nothing else, which is a version letter rather than a
// name: 2a in hashes from before 2011, 2y in the ones PHP and Laravel write,
// and 2b in the ones everything writes now.
func (b Bcrypt) Reads(encoded string) bool {
	rest, ok := strings.CutPrefix(encoded, "$")
	if !ok {
		return false
	}
	name, _, ok := strings.Cut(rest, "$")
	return ok && algorithm(name) == "bcrypt"
}

// Verify reports whether a password matches a bcrypt hash.
//
// The context is in the signature because [Verifier] has it. Nothing here reads
// it, because a bcrypt hash in progress cannot be stopped: it is one loop of
// key expansions with no place to check anything. What bounds the damage is the
// number of them running at once, which is [Migrating]'s job, and the cost
// ceiling above, which is this one's.
//
// Everything after the 72nd byte of a password is ignored, so two passwords
// that agree that far both verify. That is [bcryptLimit] and not a decision
// made here, it is what the system that wrote the hash did as well, and it is
// one of the reasons nothing new is written this way.
func (b Bcrypt) Verify(ctx context.Context, password, encoded string) (bool, error) {
	if !b.Reads(encoded) {
		return false, errs.New(errs.Unsupported, "hash.unsupported",
			"hash: this is not a bcrypt hash, and bcrypt is all this checks")
	}

	// Cost parses the hash without spending it, so a hash that is not one, or
	// is one nobody should run, is answered before any work happens.
	cost, err := bcrypt.Cost([]byte(encoded))
	if err != nil {
		return false, malformed("it starts like bcrypt and is not shaped like it")
	}
	if max := b.maxCost(); cost > max {
		return false, errs.Newf(errs.Unsupported, "hash.too_costly",
			"hash: this bcrypt hash is cost %d, and this checks up to %d", cost, max)
	}

	switch err := bcrypt.CompareHashAndPassword([]byte(encoded), []byte(password)); {
	case err == nil:
		return true, nil
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return false, nil
	default:
		// Cost reads the version and the cost and stops, so a salt that is not
		// in bcrypt's alphabet gets this far and fails here. It is a hash that
		// cannot be checked rather than a password that did not match, and the
		// two must not come back the same way.
		return false, malformed("it is not a bcrypt hash this reads")
	}
}

// maxCost is [Bcrypt.MaxCost] with its default filled in.
func (b Bcrypt) maxCost() int {
	if b.MaxCost == 0 {
		return defaultMaxCost
	}
	return b.MaxCost
}
