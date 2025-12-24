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

// NewAt generates a new ULID at a specific time.
func NewAt(t time.Time) string {
	entropyLock.Lock()
	defer entropyLock.Unlock()
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}

// Time extracts the timestamp from a ULID.
func Time(id string) time.Time {
	u, err := ulid.Parse(id)
	if err != nil {
		return time.Time{}
	}
	return ulid.Time(u.Time())
}

// IsValid checks if a string is a valid ULID.
func IsValid(id string) bool {
	_, err := ulid.Parse(id)
	return err == nil
}
