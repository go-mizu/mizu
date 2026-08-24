package hash

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// TestArgon2idVector is the one in RFC 9106 section 5.3, with every input the
// specification names, including the secret and the associated data that
// nothing else here passes.
//
// It is the test that says this is Argon2id and not something that merely
// verifies against itself.
func TestArgon2idVector(t *testing.T) {
	const want = "0d640df58d78766c08c037a34a8b53c9d01ef0452d75b65eb52520e96b01e659"

	got := argon2id(
		bytes.Repeat([]byte{0x01}, 32),
		bytes.Repeat([]byte{0x02}, 16),
		bytes.Repeat([]byte{0x03}, 8),
		bytes.Repeat([]byte{0x04}, 12),
		3, 32, 4, 32,
	)
	if hex.EncodeToString(got) != want {
		t.Errorf("the tag is %x, want %s", got, want)
	}
}

// TestArgon2id covers the parameter space around it: one lane and several, one
// pass and several, memory that divides evenly and memory that does not, and
// tags that are shorter and longer than a BLAKE2b digest.
//
// The answers come from the reference implementation, by way of
// golang.org/x/crypto/argon2, which this package does not depend on.
func TestArgon2id(t *testing.T) {
	password := []byte("password")
	salt := []byte("somesalt12345678")

	cases := []struct {
		time, memory uint32
		lanes        uint8
		tagLen       uint32
		want         string
	}{
		{1, 64, 1, 32, "2a8bf2c64a9623c36221b9d54ec1c91ad59d6b4edac54aa825522e3e8b6a2027"},
		{2, 65536, 1, 32, "1e6938f511f9d7a88f1c6a4a49d446685ce2e3f58ecf335e07950920a0201dbb"},
		{3, 32, 4, 32, "cd342db86d5658c8e3267ba1c32c14f66bd95c49ca2c842dbff0e298d37817c9"},
		{1, 8, 1, 16, "dcbbca10a2cd4bad977cc61b6d33aed5"},
		{4, 1024, 2, 64, "b198116b0948cf3759cf213eae70cb11930a2700cccd4d10655a69af63a5c11e7efceea0f11605d1243859f5a7fc510f01d9bb9d797696af4067fe36dae7a43f"},
	}

	for _, c := range cases {
		got := argon2id(password, salt, nil, nil, c.time, c.memory, c.lanes, c.tagLen)
		if hex.EncodeToString(got) != c.want {
			t.Errorf("t=%d m=%d p=%d n=%d: %x, want %s", c.time, c.memory, c.lanes, c.tagLen, got, c.want)
		}
	}
}

// TestArgon2idInputsMatter is the property behind the vectors: change anything
// that goes in, and what comes out is different.
func TestArgon2idInputsMatter(t *testing.T) {
	password := []byte("password")
	salt := []byte("somesalt12345678")
	base := argon2id(password, salt, nil, nil, 2, 64, 1, 32)

	others := map[string][]byte{
		"another password": argon2id([]byte("passwore"), salt, nil, nil, 2, 64, 1, 32),
		"another salt":     argon2id(password, []byte("somesalt12345679"), nil, nil, 2, 64, 1, 32),
		"a secret":         argon2id(password, salt, []byte("pepper"), nil, 2, 64, 1, 32),
		"associated data":  argon2id(password, salt, nil, []byte("user:42"), 2, 64, 1, 32),
		"another pass":     argon2id(password, salt, nil, nil, 3, 64, 1, 32),
		"more memory":      argon2id(password, salt, nil, nil, 2, 128, 1, 32),
		"another lane":     argon2id(password, salt, nil, nil, 2, 64, 2, 32),
	}
	for name, got := range others {
		if bytes.Equal(got, base) {
			t.Errorf("%s gives the same tag", name)
		}
	}

	// The same inputs give the same tag, which is the whole point of a password
	// hash being checkable at all.
	if !bytes.Equal(argon2id(password, salt, nil, nil, 2, 64, 1, 32), base) {
		t.Error("the same inputs gave two different tags")
	}
}

// TestArgon2idMemoryRounding is memory that is not a multiple of four blocks
// per lane. The number of blocks is rounded down, so 24 through 31 with two
// lanes all fill the same 24 blocks, but the figure that was asked for is
// hashed in as it was asked for, so none of them agree on the tag.
//
// That is what the specification says, and it is worth a test because the
// tempting reading is the other one.
func TestArgon2idMemoryRounding(t *testing.T) {
	password := []byte("password")
	salt := []byte("somesalt12345678")

	seen := map[string]uint32{}
	for _, memory := range []uint32{24, 25, 30, 31, 32} {
		got := string(argon2id(password, salt, nil, nil, 1, memory, 2, 32))
		if before, dup := seen[got]; dup {
			t.Errorf("m=%d and m=%d give the same tag", before, memory)
		}
		seen[got] = memory
	}

	// The smallest memory the specification allows is eight blocks per lane,
	// and it has to work rather than divide by zero on the way.
	if got := argon2id(password, salt, nil, nil, 1, 16, 2, 32); len(got) != 32 {
		t.Errorf("the smallest run gave %d bytes", len(got))
	}
}

// TestArgon2idTagLengths walks the sizes where H' changes shape: under a
// digest, exactly a digest, and into the chaining above it.
func TestArgon2idTagLengths(t *testing.T) {
	password := []byte("password")
	salt := []byte("somesalt12345678")

	for _, n := range []uint32{4, 16, 31, 32, 63, 64, 65, 128, 512} {
		if got := argon2id(password, salt, nil, nil, 1, 32, 1, n); len(got) != int(n) {
			t.Errorf("asked for %d bytes and got %d", n, len(got))
		}
	}
}

// TestArgon2idEmptyInputs is the boundary a login form reaches on the first
// keystroke, and a salt is not required to be there for the hash to be defined.
func TestArgon2idEmptyInputs(t *testing.T) {
	if got := argon2id(nil, nil, nil, nil, 1, 8, 1, 32); len(got) != 32 {
		t.Errorf("an empty password and salt gave %d bytes", len(got))
	}
	if bytes.Equal(argon2id(nil, nil, nil, nil, 1, 8, 1, 32), argon2id([]byte("x"), nil, nil, nil, 1, 8, 1, 32)) {
		t.Error("an empty password hashes the same as a password")
	}
}
