// Package password provides secure password hashing using Argon2id.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Params represents the Argon2id hashing parameters.
type Params struct {
	Memory      uint32 // Memory usage in KiB
	Iterations  uint32 // Number of iterations (time cost)
	Parallelism uint8  // Number of parallel threads
	SaltLength  uint32 // Length of salt in bytes
	KeyLength   uint32 // Length of derived key in bytes
}

// DefaultParams returns secure default parameters for Argon2id.
// These parameters are tuned for a balance of security and performance.
func DefaultParams() *Params {
	return &Params{
		Memory:      64 * 1024, // 64 MiB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

var (
	ErrInvalidHash         = errors.New("invalid hash format")
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
	ErrMismatchedPassword  = errors.New("password does not match")
)

// Hash generates an Argon2id hash of the given password using the provided parameters.
// The returned hash is in PHC format: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
func Hash(password string, params *Params) (string, error) {
	if params == nil {
		params = DefaultParams()
	}

	// Generate a random salt
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	// Hash the password using Argon2id
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Encode the hash in PHC format
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		b64Salt,
		b64Hash,
	)

	return encoded, nil
}

// Verify compares a password against a hash and returns nil if they match.
func Verify(password, encodedHash string) error {
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return err
	}

	// Hash the password with the same parameters
	otherHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Compare the hashes using constant-time comparison
	if subtle.ConstantTimeCompare(hash, otherHash) != 1 {
		return ErrMismatchedPassword
	}

	return nil
}

// decodeHash parses an encoded hash string and extracts the parameters, salt, and hash.
func decodeHash(encodedHash string) (*Params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return nil, nil, nil, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	params := &Params{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Iterations, &params.Parallelism); err != nil {
		return nil, nil, nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	params.SaltLength = uint32(len(salt))

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	params.KeyLength = uint32(len(hash))

	return params, salt, hash, nil
}

// NeedsRehash checks if a hash was created with outdated parameters
// and should be rehashed with the current default parameters.
func NeedsRehash(encodedHash string, params *Params) bool {
	if params == nil {
		params = DefaultParams()
	}

	currentParams, _, _, err := decodeHash(encodedHash)
	if err != nil {
		return true
	}

	return currentParams.Memory != params.Memory ||
		currentParams.Iterations != params.Iterations ||
		currentParams.Parallelism != params.Parallelism ||
		currentParams.KeyLength != params.KeyLength
}
