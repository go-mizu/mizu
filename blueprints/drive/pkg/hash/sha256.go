// Package hash provides file hashing utilities.
package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// SHA256Reader computes SHA256 hash while reading from r.
func SHA256Reader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SHA256Bytes computes SHA256 hash of bytes.
func SHA256Bytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// SHA256String computes SHA256 hash of string.
func SHA256String(s string) string {
	return SHA256Bytes([]byte(s))
}
