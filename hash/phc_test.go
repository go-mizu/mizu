package hash

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs"
)

// TestEncode pins the format. A hash written here has to be one that PHP,
// Python, Rust and Node read, and the only way that stays true is to state what
// the bytes are rather than to check that this package reads back what it
// wrote.
func TestEncode(t *testing.T) {
	p := phc{
		memory: 19456,
		passes: 2,
		lanes:  1,
		salt:   []byte("saltsaltsaltsalt"),
		tag:    counting(32),
	}

	const want = "$argon2id$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8"
	if got := p.encode(); got != want {
		t.Errorf("encoded as\n%s\nwant\n%s", got, want)
	}
}

// TestParseVector is a hash this package did not write, at parameters it does
// not use and with a salt and a tag that are not the lengths it writes. It says
// that a row from another stack is read as the numbers it was written with,
// rather than as the ones this package happens to expect.
func TestParseVector(t *testing.T) {
	const in = "$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG"

	p, err := parse(in)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	switch {
	case p.memory != 65536:
		t.Errorf("memory is %d", p.memory)
	case p.passes != 3:
		t.Errorf("passes is %d", p.passes)
	case p.lanes != 4:
		t.Errorf("lanes is %d", p.lanes)
	case string(p.salt) != "somesalt":
		t.Errorf("salt is %q", p.salt)
	case hex.EncodeToString(p.tag) != "45d7ac72e76f242b20b77b9bf9bf9d5915894e669a24e6c6":
		t.Errorf("tag is %x", p.tag)
	}

	// And it goes back out the way it came in, which is what a row that is read
	// and written again has to do.
	if got := p.encode(); got != in {
		t.Errorf("round trip gave\n%s\nwant\n%s", got, in)
	}
}

// TestParseUnsupported covers the hashes that exist in the wild and are not
// checked here. They are told apart from rubbish on purpose: a bcrypt hash in a
// password column is an application waiting to be migrated, and rubbish in one
// is a bug.
func TestParseUnsupported(t *testing.T) {
	cases := map[string]string{
		"bcrypt from PHP":  "$2y$12$IvBOSJhWTgLPfDrLZDNC0.aI9C0DAmMHW7bqZzjhKvvCJHqCJRCEO",
		"bcrypt, modern":   "$2b$12$IvBOSJhWTgLPfDrLZDNC0.aI9C0DAmMHW7bqZzjhKvvCJHqCJRCEO",
		"bcrypt, ancient":  "$2a$10$IvBOSJhWTgLPfDrLZDNC0.aI9C0DAmMHW7bqZzjhKvvCJHqCJRCEO",
		"argon2i":          "$argon2i$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG",
		"argon2d":          "$argon2d$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG",
		"scrypt":           "$scrypt$ln=16,r=8,p=1$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"an older argon2":  "$argon2id$v=16$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"a keyed hash":     "$argon2id$v=19$m=65536,t=3,p=4,keyid=a$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"associated data":  "$argon2id$v=19$m=65536,t=3,p=4,data=a$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"more lanes":       "$argon2id$v=19$m=65536,t=3,p=300$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"pbkdf2 from Rust": "$pbkdf2-sha256$i=10000$c29tZXNhbHQ$RdescudvJCsgt3ub",
	}

	for name, in := range cases {
		_, err := parse(in)
		if errs.KindOf(err) != errs.Unsupported || errs.CodeOf(err) != "hash.unsupported" {
			t.Errorf("%s: %v, want unsupported", name, err)
		}
	}
}

// TestParseMalformed is everything that is not an encoded hash at all. None of
// it may panic, and all of it is invalid rather than unsupported, because there
// is nothing here to migrate.
func TestParseMalformed(t *testing.T) {
	cases := map[string]string{
		"empty":                "",
		"no dollar":            "argon2id",
		"only a dollar":        "$",
		"too few parts":        "$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ",
		"too many parts":       "$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub$extra",
		"no version":           "$argon2id$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub$x",
		"version not a number": "$argon2id$v=nineteen$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"a parameter has no =": "$argon2id$v=19$m=65536,t,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"a parameter is text":  "$argon2id$v=19$m=lots,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"a parameter is huge":  "$argon2id$v=19$m=99999999999999,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"an unknown parameter": "$argon2id$v=19$m=65536,t=3,p=4,x=1$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"no memory":            "$argon2id$v=19$t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"no passes":            "$argon2id$v=19$m=65536,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"no lanes":             "$argon2id$v=19$m=65536,t=3$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"no passes at all":     "$argon2id$v=19$m=65536,t=0,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"no lanes at all":      "$argon2id$v=19$m=65536,t=3,p=0$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"memory below lanes":   "$argon2id$v=19$m=8,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"salt is not base64":   "$argon2id$v=19$m=65536,t=3,p=4$not base64!$RdescudvJCsgt3ub",
		"salt is padded":       "$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ=$RdescudvJCsgt3ub",
		"salt is short":        "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$RdescudvJCsgt3ub",
		"tag is not base64":    "$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$not base64!",
		"tag is short":         "$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$YQ",
		"an unknown format":    "$sodium$v=1$c29tZXNhbHQ",
	}

	for name, in := range cases {
		p, err := parse(in)
		if errs.KindOf(err) != errs.Invalid || errs.CodeOf(err) != "hash.malformed" {
			t.Errorf("%s: %v, want malformed", name, err)
		}
		if p.salt != nil || p.tag != nil {
			t.Errorf("%s: parse returned a hash along with the error", name)
		}
	}
}

// TestParseMessagesSayWhat is what somebody reads at three in the morning when
// a password column has something unexpected in it.
func TestParseMessagesSayWhat(t *testing.T) {
	_, err := parse("$2y$12$IvBOSJhWTgLPfDrLZDNC0.aI9C0DAmMHW7bqZzjhKvvCJHqCJRCEO")
	if !strings.Contains(err.Error(), "bcrypt") {
		t.Errorf("a bcrypt hash does not say bcrypt: %v", err)
	}

	// Nothing out of the column reaches the message, which matters because the
	// message goes into a log. Everything that might have is parsed as a number
	// first, so a value that is not one never gets that far.
	const payload = `"\n2026-01-01 INFO everything is fine"`
	for _, in := range []string{
		"$" + payload + "$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"$argon2id$v=" + payload + "$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"$argon2id$v=19$m=" + payload + ",t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub",
		"$argon2id$v=19$m=65536,t=3,p=4$" + payload + "$RdescudvJCsgt3ub",
	} {
		_, err := parse(in)
		if strings.Contains(err.Error(), "everything is fine") {
			t.Errorf("the message repeats what was stored: %v", err)
		}
	}
}

// TestDecodeRejectsPadding is the part of the format that is a rule rather than
// a preference. PHC strings use base64 with no padding, and accepting padded
// ones would mean writing hashes that some other stack rejects.
func TestDecodeRejectsPadding(t *testing.T) {
	if _, err := decode("c29tZXNhbHQ=", minSalt, "salt"); err == nil {
		t.Error("a padded salt was accepted")
	}
	if _, err := decode("c29tZXNhbHQ", minSalt, "salt"); err != nil {
		t.Errorf("an unpadded salt was rejected: %v", err)
	}
}

// TestEncodeAndParse is the round trip over the shapes a stored hash comes in.
func TestEncodeAndParse(t *testing.T) {
	cases := []phc{
		{8, 1, 1, counting(8), counting(4)},
		{19456, 2, 1, counting(16), counting(32)},
		{65536, 3, 4, counting(24), counting(64)},
		{1 << 20, 10, 255, counting(32), counting(128)},
	}

	for _, want := range cases {
		got, err := parse(want.encode())
		if err != nil {
			t.Fatalf("%s: %v", want.encode(), err)
		}
		if got.memory != want.memory || got.passes != want.passes || got.lanes != want.lanes {
			t.Errorf("%s: parameters came back as m=%d t=%d p=%d", want.encode(), got.memory, got.passes, got.lanes)
		}
		if !bytes.Equal(got.salt, want.salt) || !bytes.Equal(got.tag, want.tag) {
			t.Errorf("%s: salt or tag changed", want.encode())
		}
	}
}
