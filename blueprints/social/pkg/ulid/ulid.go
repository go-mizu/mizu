// Package ulid provides ULID generation utilities.
package ulid

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// New generates a new ULID.
func New() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

// NewAt generates a new ULID at a specific time.
func NewAt(t time.Time) string {
	return ulid.MustNew(ulid.Timestamp(t), rand.Reader).String()
}

// Parse parses a ULID string.
func Parse(s string) (ulid.ULID, error) {
	return ulid.Parse(s)
}

// Time extracts the time from a ULID string.
func Time(s string) (time.Time, error) {
	id, err := ulid.Parse(s)
	if err != nil {
		return time.Time{}, err
	}
	return ulid.Time(id.Time()), nil
}
