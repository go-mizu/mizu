// Package password provides password hashing using Argon2id.
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

// Config holds Argon2id parameters.
type Config struct {
	Time    uint32 // Number of iterations
	Memory  uint32 // Memory in KB
	Threads uint8  // Parallelism
	KeyLen  uint32 // Length of derived key
	SaltLen uint32 // Length of salt
}

// DefaultConfig returns sensible defaults for Argon2id.
func DefaultConfig() Config {
	return Config{
		Time:    1,
		Memory:  64 * 1024, // 64 MB
		Threads: 4,
		KeyLen:  32,
		SaltLen: 16,
	}
}

// Hash generates an Argon2id hash of the password.
func Hash(password string) (string, error) {
	return HashWithConfig(password, DefaultConfig())
}

// HashWithConfig generates an Argon2id hash with custom config.
func HashWithConfig(password string, cfg Config) (string, error) {
	salt := make([]byte, cfg.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, cfg.Time, cfg.Memory, cfg.Threads, cfg.KeyLen)

	// Encode as: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, cfg.Memory, cfg.Time, cfg.Threads, b64Salt, b64Hash), nil
}

// Verify checks if a password matches a hash.
func Verify(password, encodedHash string) (bool, error) {
	cfg, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	otherHash := argon2.IDKey([]byte(password), salt, cfg.Time, cfg.Memory, cfg.Threads, cfg.KeyLen)

	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}
	return false, nil
}

func decodeHash(encodedHash string) (Config, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return Config{}, nil, nil, errors.New("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return Config{}, nil, nil, errors.New("unsupported algorithm")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return Config{}, nil, nil, err
	}
	if version != argon2.Version {
		return Config{}, nil, nil, errors.New("incompatible version")
	}

	var cfg Config
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &cfg.Memory, &cfg.Time, &cfg.Threads); err != nil {
		return Config{}, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Config{}, nil, nil, err
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Config{}, nil, nil, err
	}
	cfg.KeyLen = uint32(len(hash))

	return cfg, salt, hash, nil
}
