package hash_test

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-mizu/mizu/hash"
)

// cheap is argon2id at parameters no application should store a password with.
//
// An example runs on every go test, and [hash.Default] is deliberately slow,
// which is the one thing a password hash is for. What is shown here is the
// same in either case.
func cheap() *hash.Argon2id {
	h, err := hash.New(hash.Params{Memory: 64, Passes: 1, Lanes: 1})
	if err != nil {
		log.Fatal(err)
	}
	return h
}

// There is one thing to do here and one way to do it.
func Example() {
	ctx := context.Background()
	h := cheap() // hash.Default() in an application

	encoded, err := h.Hash(ctx, "correct horse battery staple")
	if err != nil {
		log.Fatal(err)
	}

	// The encoded hash is the standard PHC string, and it carries the salt and
	// the parameters, so a column of them is all the state there is.
	fmt.Println(strings.HasPrefix(encoded, "$argon2id$v=19$"))

	fmt.Println(h.Verify(ctx, "correct horse battery staple", encoded))
	fmt.Println(h.Verify(ctx, "hunter2", encoded))
	// Output:
	// true
	// true <nil>
	// false <nil>
}

// A password that does not match is false and no error. An error means the
// check did not happen, which is not the same thing and must not be treated as
// a failed sign in.
func ExampleArgon2id_Verify() {
	ctx := context.Background()
	h := cheap()

	ok, err := h.Verify(ctx, "hunter2", "not a hash at all")

	fmt.Println(ok)
	fmt.Println(err)
	// Output:
	// false
	// hash: this is not an encoded password hash, it does not start with a dollar sign
}

// Raising the parameters is a decision about new hashes. The ones already
// stored say what they were made with, so each one moves the next time its
// owner signs in and nobody is asked to reset anything.
func ExampleArgon2id_NeedsRehash() {
	ctx := context.Background()
	old := cheap()

	encoded, err := old.Hash(ctx, "correct horse battery staple")
	if err != nil {
		log.Fatal(err)
	}

	now, err := hash.New(hash.Params{Memory: 128, Passes: 1, Lanes: 1})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(old.NeedsRehash(encoded), now.NeedsRehash(encoded))
	// Output: false true
}

// An application that arrives with a password column full of bcrypt reads it
// and writes argon2id, which is what most of the life of an application looks
// like.
func ExampleMigrating() {
	ctx := context.Background()
	h := hash.Migrating(cheap(), &hash.Bcrypt{})

	// From the crypt_blowfish test suite, which is the reference every other
	// bcrypt is checked against.
	const stored = "$2a$05$CCCCCCCCCCCCCCCCCCCCC.E5YPO9kmyuRGyh0XouQYb4YMJKvyOeW"

	ok, err := h.Verify(ctx, "U*U", stored)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ok, h.NeedsRehash(stored))

	// So the sign in that worked is where the account moves.
	if ok && h.NeedsRehash(stored) {
		encoded, err := h.Hash(ctx, "U*U")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(strings.HasPrefix(encoded, "$argon2id$"))
	}
	// Output:
	// true true
	// true
}
