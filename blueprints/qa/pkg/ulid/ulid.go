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

// NewAt generates a new ULID at the given time.
func NewAt(t time.Time) string {
	return ulid.MustNew(ulid.Timestamp(t), rand.Reader).String()
}

// Time extracts the timestamp from a ULID string.
func Time(s string) (time.Time, error) {
	id, err := ulid.Parse(s)
	if err != nil {
		return time.Time{}, err
	}
	return ulid.Time(id.Time()), nil
}

// IsValid checks if a string is a valid ULID.
func IsValid(s string) bool {
	_, err := ulid.Parse(s)
	return err == nil
}
