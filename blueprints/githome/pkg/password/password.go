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

var (
	ErrInvalidHash         = errors.New("invalid password hash format")
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
)

// Config for Argon2id hashing
type Config struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultConfig returns sensible defaults for password hashing
var DefaultConfig = Config{
	Memory:      64 * 1024, // 64MB
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// Hash generates an Argon2id hash of the password
func Hash(password string) (string, error) {
	return HashWithConfig(password, DefaultConfig)
}

// HashWithConfig generates an Argon2id hash with custom config
func HashWithConfig(password string, c Config) (string, error) {
	salt := make([]byte, c.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, c.Iterations, c.Memory, c.Parallelism, c.KeyLength)

	// Encode as: $argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, c.Memory, c.Iterations, c.Parallelism, b64Salt, b64Hash)

	return encoded, nil
}

// Verify checks if a password matches the hash
func Verify(password, encodedHash string) (bool, error) {
	c, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	otherHash := argon2.IDKey([]byte(password), salt, c.Iterations, c.Memory, c.Parallelism, c.KeyLength)

	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}
	return false, nil
}

func decodeHash(encodedHash string) (c Config, salt, hash []byte, err error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return c, nil, nil, ErrInvalidHash
	}

	var version int
	_, err = fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return c, nil, nil, err
	}
	if version != argon2.Version {
		return c, nil, nil, ErrIncompatibleVersion
	}

	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &c.Memory, &c.Iterations, &c.Parallelism)
	if err != nil {
		return c, nil, nil, err
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return c, nil, nil, err
	}
	c.SaltLength = uint32(len(salt))

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return c, nil, nil, err
	}
	c.KeyLength = uint32(len(hash))

	return c, salt, hash, nil
}
