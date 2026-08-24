package hash

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs"
)

// bcryptVectors are from the crypt_blowfish test suite, which is the reference
// every other bcrypt is checked against. They are here for the same reason the
// argon2id vectors are: a password hash that agrees with itself and with
// nothing else is a password column nobody can leave.
//
// The last one is 98 bytes of password against a hash of its first 72, and it
// is the whole of what [bcryptLimit] means.
var bcryptVectors = []struct{ hash, password string }{
	{"$2a$05$CCCCCCCCCCCCCCCCCCCCC.E5YPO9kmyuRGyh0XouQYb4YMJKvyOeW", "U*U"},
	{"$2a$05$CCCCCCCCCCCCCCCCCCCCC.VGOzA784oUp/Z0DY336zx7pLYAy0lwK", "U*U*"},
	{"$2a$05$XXXXXXXXXXXXXXXXXXXXXOAcXxm9kjPGEMsLznoKqmqw7tc8WCx4a", "U*U*U"},
	{
		"$2a$05$abcdefghijklmnopqrstuu5s2v8.iXieOjg/.AySBTTZIIVFJeBui",
		"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789chars after 72 are ignored",
	},
}

func TestBcryptVectors(t *testing.T) {
	var b Bcrypt

	for _, v := range bcryptVectors {
		ok, err := b.Verify(t.Context(), v.password, v.hash)
		if err != nil {
			t.Errorf("%s: %v", v.hash, err)
			continue
		}
		if !ok {
			t.Errorf("%s: the password did not verify", v.hash)
		}

		ok, err = b.Verify(t.Context(), v.password+" ", v.hash)
		if err != nil {
			t.Errorf("%s: %v", v.hash, err)
			continue
		}
		if ok && len(v.password) < bcryptLimit {
			t.Errorf("%s: a different password verified", v.hash)
		}
	}
}

// bcryptHashes are hashes of "hunter2" at the costs that turn up in a real
// column, one per version letter. They are written down rather than generated
// because a test that hashes and then checks its own hash proves only that the
// package agrees with itself.
var bcryptHashes = map[string]string{
	"cost 4":         "$2a$04$eW942w2Bny37FnmXsXTiqOIDWf0zPQRohk7YTt.UeuSgXuHq5Rptu",
	"cost 10":        "$2a$10$Zsopuw13D35obwuvLBw/zeNfPG/kKqfLhKyhhAyjtWOMPCVKsELLa",
	"cost 12":        "$2a$12$CBNpAu9qNU6R5JU5I.zqwOgGoK6OFSUFJSTW9yNfAO7repC7y2UZG",
	"2y, as Laravel": "$2y$04$eW942w2Bny37FnmXsXTiqOIDWf0zPQRohk7YTt.UeuSgXuHq5Rptu",
	"2b, as modern":  "$2b$04$eW942w2Bny37FnmXsXTiqOIDWf0zPQRohk7YTt.UeuSgXuHq5Rptu",
}

func TestBcryptVerify(t *testing.T) {
	var b Bcrypt

	for name, encoded := range bcryptHashes {
		ok, err := b.Verify(t.Context(), "hunter2", encoded)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !ok {
			t.Errorf("%s: the password did not verify", name)
		}

		// A wrong password is false and no error, which is the one thing every
		// caller of this depends on.
		ok, err = b.Verify(t.Context(), "hunter3", encoded)
		if err != nil {
			t.Errorf("%s: a wrong password gave an error: %v", name, err)
		}
		if ok {
			t.Errorf("%s: a wrong password verified", name)
		}
	}
}

func TestBcryptReads(t *testing.T) {
	var b Bcrypt

	for name, encoded := range bcryptHashes {
		if !b.Reads(encoded) {
			t.Errorf("%s: not read as bcrypt", name)
		}
	}

	notBcrypt := map[string]string{
		"empty":            "",
		"no dollar":        "2a$04$eW942w2Bny37FnmXsXTiqOIDWf0zPQRohk7YTt.UeuSgXuHq5Rptu",
		"one dollar":       "$2a",
		"argon2id":         "$argon2id$v=19$m=19456,t=2,p=1$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"scrypt":           "$scrypt$ln=16,r=8,p=1$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"a plaintext":      "hunter2",
		"an md5 crypt":     "$1$salt$qJH7.N4xYta3aEG/dfqo/0",
		"a version we lie": "$3a$04$eW942w2Bny37FnmXsXTiqOIDWf0zPQRohk7YTt.UeuSgXuHq5Rptu",
	}
	for name, encoded := range notBcrypt {
		if b.Reads(encoded) {
			t.Errorf("%s: read as bcrypt", name)
		}
	}
}

// TestBcryptIgnoresPast72 is bcrypt's oldest sharp edge, written down because
// it is the reason this package reads bcrypt and does not write it.
func TestBcryptIgnoresPast72(t *testing.T) {
	var b Bcrypt

	const same = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if len(same) != bcryptLimit {
		t.Fatalf("the prefix is %d bytes, not %d", len(same), bcryptLimit)
	}
	const encoded = "$2a$05$abcdefghijklmnopqrstuu5s2v8.iXieOjg/.AySBTTZIIVFJeBui"

	for _, tail := range []string{"", "chars after 72 are ignored", strings.Repeat("z", 500)} {
		ok, err := b.Verify(t.Context(), same+tail, encoded)
		if err != nil {
			t.Errorf("%d bytes of tail: %v", len(tail), err)
			continue
		}
		if !ok {
			t.Errorf("%d bytes of tail: the password did not verify", len(tail))
		}
	}

	// Inside the limit every byte counts, which is what says the check above
	// is bcrypt's behaviour and not this package failing to compare anything.
	ok, err := b.Verify(t.Context(), same[:71]+"x", encoded)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ok {
		t.Error("a password that differs inside the limit verified")
	}
}

// TestBcryptRefusesAnUnaffordableHash is the same hole [maxStoredCost] closes
// for argon2id. A bcrypt cost is a power of two, so cost 31 is 2^31 rounds of
// key expansion, which is days rather than milliseconds, and the row that says
// so is in a column an application often lets people write to.
func TestBcryptRefusesAnUnaffordableHash(t *testing.T) {
	const salt = "CCCCCCCCCCCCCCCCCCCCC.E5YPO9kmyuRGyh0XouQYb4YMJKvyOeW"

	cases := map[string]struct {
		hasher  Bcrypt
		encoded string
	}{
		"cost 31 at the default":   {Bcrypt{}, "$2a$31$" + salt},
		"cost 16 at the default":   {Bcrypt{}, "$2a$16$" + salt},
		"cost 12 at a low ceiling": {Bcrypt{MaxCost: 10}, "$2a$12$" + salt},
	}

	for name, c := range cases {
		ok, err := c.hasher.Verify(t.Context(), "hunter2", c.encoded)
		if errs.CodeOf(err) != "hash.too_costly" {
			t.Errorf("%s: %v, want hash.too_costly", name, err)
		}
		if errs.KindOf(err) != errs.Unsupported {
			t.Errorf("%s: the kind is %s, want unsupported", name, errs.KindOf(err))
		}
		if ok {
			t.Errorf("%s: it verified", name)
		}
	}

	// The ceiling is a ceiling and not a floor. A hash at exactly the highest
	// cost this checks still checks.
	if ok, err := (Bcrypt{MaxCost: 4}).Verify(t.Context(), "hunter2", bcryptHashes["cost 4"]); !ok || err != nil {
		t.Errorf("a hash at exactly the ceiling: %v, %v", ok, err)
	}
}

func TestBcryptErrors(t *testing.T) {
	var b Bcrypt

	bad := map[string]string{
		"nothing after the version": "$2a$",
		"the cost is not a number":  "$2a$xx$CCCCCCCCCCCCCCCCCCCCC.E5YPO9kmyuRGyh0XouQYb4YMJKvyOeW",
		"the cost is below 4":       "$2a$03$CCCCCCCCCCCCCCCCCCCCC.E5YPO9kmyuRGyh0XouQYb4YMJKvyOeW",
		"the cost is above 31":      "$2a$32$CCCCCCCCCCCCCCCCCCCCC.E5YPO9kmyuRGyh0XouQYb4YMJKvyOeW",
		"it is too short":           "$2a$05$CCCC",

		// The version and the cost read, and then the salt is not in bcrypt's
		// alphabet. Nothing catches this until the hash is run, which is the
		// one place a broken hash could be mistaken for a wrong password.
		"the salt is not base64": "$2a$05$!!!!!!!!!!!!!!!!!!!!!!E5YPO9kmyuRGyh0XouQYb4YMJKvyOeW",
	}
	for name, encoded := range bad {
		ok, err := b.Verify(t.Context(), "hunter2", encoded)
		if errs.CodeOf(err) != "hash.malformed" || errs.KindOf(err) != errs.Invalid {
			t.Errorf("%s: %v, want hash.malformed", name, err)
		}
		if ok {
			t.Errorf("%s: it verified", name)
		}
	}

	// Something that is not bcrypt at all is unsupported rather than malformed,
	// because it is a hash and this is not the thing that reads it.
	ok, err := b.Verify(t.Context(), "hunter2", "$argon2id$v=19$m=19456,t=2,p=1$c29tZXNhbHQ$RdescudvJCsgt3ub")
	if errs.CodeOf(err) != "hash.unsupported" || errs.KindOf(err) != errs.Unsupported {
		t.Errorf("an argon2id hash: %v, want hash.unsupported", err)
	}
	if ok {
		t.Error("an argon2id hash verified as bcrypt")
	}
}

func TestBcryptMaxCostDefault(t *testing.T) {
	if got := (Bcrypt{}).maxCost(); got != defaultMaxCost {
		t.Errorf("the default ceiling is %d, want %d", got, defaultMaxCost)
	}
	if got := (Bcrypt{MaxCost: 7}).maxCost(); got != 7 {
		t.Errorf("a set ceiling reads back as %d", got)
	}
}

// FuzzBcrypt says that nothing in a password column panics and nothing in one
// takes an unbounded amount of time to reject.
func FuzzBcrypt(f *testing.F) {
	for _, v := range bcryptVectors {
		f.Add(v.password, v.hash)
	}
	for _, encoded := range bcryptHashes {
		f.Add("hunter2", encoded)
	}
	f.Add("", "")
	f.Add("hunter2", "$2a$31$CCCCCCCCCCCCCCCCCCCCC.E5YPO9kmyuRGyh0XouQYb4YMJKvyOeW")

	// Cost 4 is the cheapest bcrypt defines, so a hash the fuzzer builds that
	// happens to parse costs a millisecond and not a minute.
	b := Bcrypt{MaxCost: 4}

	f.Fuzz(func(t *testing.T, password, encoded string) {
		ok, err := b.Verify(t.Context(), password, encoded)
		if ok && err != nil {
			t.Errorf("%q verified and gave an error: %v", encoded, err)
		}
	})
}
