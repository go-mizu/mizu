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

// New generates a new ULID string.
func New() string {
	entropyLock.Lock()
	defer entropyLock.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

// Parse parses a ULID string and returns the parsed ULID.
func Parse(s string) (ulid.ULID, error) {
	return ulid.Parse(s)
}

// Time extracts the timestamp from a ULID string.
func Time(s string) (time.Time, error) {
	id, err := Parse(s)
	if err != nil {
		return time.Time{}, err
	}
	return ulid.Time(id.Time()), nil
}
