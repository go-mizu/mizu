// Package hash stores passwords.
//
// There is one thing to do here and one way to do it. [Default] is argon2id at
// the parameters OWASP recommends, and the encoded hash it writes is the
// standard PHC string, which every other stack already reads.
//
//	h := hash.Default()
//
//	encoded, err := h.Hash(ctx, password)
//	// $argon2id$v=19$m=19456,t=2,p=1$xSGZFyNbLVIfLLGvvxNaLQ$Cn2TbC...
//
//	ok, err := h.Verify(ctx, password, user.Password)
//
// A password that does not match is false and no error. An error means the
// check did not happen, which is not the same thing and must not be treated as
// a failed sign in.
//
// # Cost
//
// A password hash is the one thing in a request that is meant to be slow. What
// it costs the server to check a password is what it costs an attacker to guess
// one, so the cost is the security, and argon2id spends memory as well as time
// because memory is what a machine built to guess passwords does not have much
// of per guess.
//
//	h, err := hash.New(hash.Params{Memory: 65536, Passes: 3})
//
// The defaults are a floor. mizu hash:tune measures the machine and prints
// parameters for it, and [Argon2id.Params] gives back what a hasher settled on,
// which is worth logging at startup: the same configuration on a smaller
// instance is a slower login, and a log line says so where a latency graph only
// hints at it.
//
// # Concurrency
//
// Every hash in flight holds its memory for as long as it runs, so a hundred
// logins at once at the default parameters is 1.9 GiB that arrives in under a
// second. Left alone that is an out of memory kill, which takes down every
// request on the process and not only the logins.
//
// So the number of hashes running at once is bounded, and the rest wait. The
// bound comes from GOMEMLIMIT and the configured memory cost, or from
// Params.MaxConcurrent where a deployment knows better. A caller that waits
// longer than Params.VerifyTimeout gets an error of kind [errs.Unavailable],
// which is a 503, and 503 under a login flood is the correct answer.
//
// # Rehashing
//
// [Argon2id.NeedsRehash] says whether a stored hash is behind what is
// configured now. Called after a successful sign in, when the password is in
// hand, it moves an account to stronger parameters without anybody being asked
// to do anything.
//
//	if h.NeedsRehash(user.Password) {
//		user.Password, err = h.Hash(ctx, password)
//	}
//
// It says yes for a hash written by something else, which is how an application
// moves off bcrypt: everybody who signs in is moved, and what is left belongs to
// accounts nobody has used since.
//
// # What is not here
//
// There is no algorithm to select and no cipher suite to negotiate. Reading a
// bcrypt hash from an application being migrated is supported because those
// hashes exist and the passwords behind them do not. Writing one is not.
//
// The argon2id itself is golang.org/x/crypto/argon2, which is written and
// reviewed by the Go team under the same process as the standard library and
// carries assembly for the machines most servers run on. What is here is
// everything around it: the parameters, the encoded form, the limit on how many
// run at once, and the answer to whether a stored hash is out of date.
//
// This package does not export a hash function. An application that wants one
// wants crypto/sha256.
package hash
