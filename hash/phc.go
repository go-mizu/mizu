package hash

import (
	"encoding/base64"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/go-mizu/mizu/errs"
)

// The encoded form is the PHC string format, which is what the Argon2
// reference implementation writes and what every other stack reads:
//
//	$argon2id$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$3f7cN...
//
// A hash written here verifies in PHP, Python, Rust or Node, and one written
// there verifies here. That is the whole reason not to invent a format: a
// password column is the hardest thing in an application to migrate, and it
// stops being a migration at all when the format is already shared.
//
// The parts are the algorithm, the version, the parameters, the salt and the
// tag, each after a dollar sign, with the salt and the tag in base64 with the
// standard alphabet and no padding.

const (
	// saltSize is the 16 bytes RFC 9106 recommends, which is what this writes.
	// A hash from somewhere else is read at whatever salt it carries.
	saltSize = 16

	// keySize is the length of the tag this writes, again from RFC 9106.
	keySize = 32

	// minSalt and minTag are the smallest RFC 9106 defines, and a hash below
	// them is not one this can check.
	minSalt = 8
	minTag  = 4
)

// b64 is the encoding PHC strings use: the standard alphabet, no padding.
var b64 = base64.RawStdEncoding

// phc is an encoded hash taken apart.
type phc struct {
	memory uint32
	passes uint32
	lanes  uint8
	salt   []byte
	tag    []byte
}

// encode is the PHC string for a hash.
func (p phc) encode() string {
	var b strings.Builder
	b.WriteString("$argon2id$v=")
	b.WriteString(strconv.Itoa(argon2.Version))
	b.WriteString("$m=")
	b.WriteString(strconv.FormatUint(uint64(p.memory), 10))
	b.WriteString(",t=")
	b.WriteString(strconv.FormatUint(uint64(p.passes), 10))
	b.WriteString(",p=")
	b.WriteString(strconv.FormatUint(uint64(p.lanes), 10))
	b.WriteByte('$')
	b.WriteString(b64.EncodeToString(p.salt))
	b.WriteByte('$')
	b.WriteString(b64.EncodeToString(p.tag))
	return b.String()
}

// parse reads an encoded hash.
//
// What arrives here came out of a database and may be anything at all, so the
// two failures are kept apart. Invalid means this is not an encoded hash.
// Unsupported means it is one and something else wrote it, which is the answer
// for a bcrypt hash from the application being migrated away from, and it is
// the difference between a bug to fix and a password to leave alone.
func parse(s string) (phc, error) {
	rest, ok := strings.CutPrefix(s, "$")
	if !ok {
		return phc{}, malformed("it does not start with a dollar sign")
	}

	parts := strings.Split(rest, "$")
	if parts[0] != "argon2id" {
		if other := algorithm(parts[0]); other != "" {
			return phc{}, errs.Newf(errs.Unsupported, "hash.unsupported",
				"hash: this hash was written by %s, and only argon2id is checked here", other)
		}
		return phc{}, malformed("the algorithm is not one this reads")
	}
	if len(parts) != 5 {
		return phc{}, malformed("it does not have five parts")
	}

	field, ok := strings.CutPrefix(parts[1], "v=")
	if !ok {
		return phc{}, malformed("the version is missing")
	}
	version, err := strconv.ParseUint(field, 10, 32)
	if err != nil {
		return phc{}, malformed("the version is not a number")
	}
	if version != argon2.Version {
		return phc{}, errs.Newf(errs.Unsupported, "hash.unsupported",
			"hash: this hash is argon2 version %d, and only version %d is checked here", version, argon2.Version)
	}

	p, err := params(parts[2])
	if err != nil {
		return phc{}, err
	}

	if p.salt, err = decode(parts[3], minSalt, "salt"); err != nil {
		return phc{}, err
	}
	if p.tag, err = decode(parts[4], minTag, "tag"); err != nil {
		return phc{}, err
	}
	return p, nil
}

// params reads the m, t and p of an encoded hash, which arrive together and in
// that order.
func params(s string) (phc, error) {
	var p phc
	var seen int

	for field := range strings.SplitSeq(s, ",") {
		name, value, ok := strings.Cut(field, "=")
		if !ok {
			return phc{}, malformed("a parameter has no value")
		}

		// keyid and data are the two parameters RFC 9106 has that this does not
		// pass, so a hash carrying either needs something this does not hold.
		if name == "keyid" || name == "data" {
			return phc{}, errs.Newf(errs.Unsupported, "hash.unsupported",
				"hash: this hash was made with %s, which is not something this package holds", name)
		}

		n, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return phc{}, malformed("a parameter is not a number")
		}

		switch name {
		case "m":
			p.memory, seen = uint32(n), seen|1
		case "t":
			p.passes, seen = uint32(n), seen|2
		case "p":
			if n > 255 {
				return phc{}, errs.Newf(errs.Unsupported, "hash.unsupported",
					"hash: this hash has %d lanes, and this checks up to 255", n)
			}
			p.lanes, seen = uint8(n), seen|4
		default:
			return phc{}, malformed("a parameter is not one this reads")
		}
	}

	if seen != 1|2|4 {
		return phc{}, malformed("the memory, passes and lanes are not all there")
	}
	if p.passes < 1 || p.lanes < 1 || p.memory < 8*uint32(p.lanes) {
		return phc{}, malformed("the parameters are outside what argon2 defines")
	}
	return p, nil
}

// decode reads one base64 part, which has to be there and long enough to be
// what it claims to be.
func decode(s string, min int, what string) ([]byte, error) {
	b, err := b64.DecodeString(s)
	if err != nil {
		return nil, malformed("the " + what + " is not base64")
	}
	if len(b) < min {
		return nil, malformed("the " + what + " is too short")
	}
	return b, nil
}

// algorithm names what wrote a hash this cannot check, for the error message,
// and returns the empty string for anything it does not recognize.
//
// bcrypt does not name itself: its prefix is a version letter, which is 2a in
// hashes from before 2011, 2y in the ones PHP and Laravel write, and 2b in the
// ones everything writes now.
func algorithm(prefix string) string {
	switch prefix {
	case "2", "2a", "2b", "2x", "2y":
		return "bcrypt"
	case "argon2i", "argon2d":
		return prefix
	case "scrypt", "pbkdf2-sha256", "pbkdf2-sha512", "sha256", "sha512", "md5":
		return prefix
	}
	return ""
}

// malformed is the answer for a string that is not an encoded hash. The reason
// is in the message and not in the code, because there is nothing a caller does
// differently for one of these than for another.
//
// Nothing that came out of the database reaches the message. The parts that
// might have gone in are a version and three numbers, and they are parsed as
// numbers before anything is said about them, so a password column with a log
// injection payload in it stays a parse error.
func malformed(why string) error {
	return errs.Newf(errs.Invalid, "hash.malformed", "hash: this is not an encoded password hash, %s", why)
}
