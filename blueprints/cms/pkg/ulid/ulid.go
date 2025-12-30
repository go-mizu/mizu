// Package ulid provides ULID generation utilities.
package ulid

import (
	"crypto/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	entropy     = ulid.Monotonic(rand.Reader, 0)
	entropyLock sync.Mutex
)

// New generates a new ULID.
func New() string {
	entropyLock.Lock()
	defer entropyLock.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

// Parse parses a ULID string.
func Parse(s string) (ulid.ULID, error) {
	return ulid.Parse(s)
}

// IsValid checks if a string is a valid ULID.
func IsValid(s string) bool {
	_, err := ulid.Parse(s)
	return err == nil
}

// Time extracts the timestamp from a ULID string.
func Time(s string) (time.Time, error) {
	id, err := ulid.Parse(s)
	if err != nil {
		return time.Time{}, err
	}
	return ulid.Time(id.Time()), nil
}
