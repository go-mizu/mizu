package ulid

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// New generates a new ULID string
func New() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

// Parse parses a ULID string
func Parse(s string) (ulid.ULID, error) {
	return ulid.Parse(s)
}

// IsValid checks if a string is a valid ULID
func IsValid(s string) bool {
	_, err := Parse(s)
	return err == nil
}
