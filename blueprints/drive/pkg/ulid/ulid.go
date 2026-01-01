// Package ulid provides ULID generation.
package ulid

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"
)

const (
	encodingChars = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	ulidLen       = 26
)

var (
	mu      sync.Mutex
	lastMs  int64
	lastRnd uint64
)

// New generates a new ULID string.
func New() string {
	mu.Lock()
	defer mu.Unlock()

	ms := time.Now().UnixMilli()

	var rnd uint64
	if ms == lastMs {
		rnd = lastRnd + 1
	} else {
		var b [8]byte
		rand.Read(b[:])
		rnd = binary.BigEndian.Uint64(b[:])
	}

	lastMs = ms
	lastRnd = rnd

	var buf [ulidLen]byte

	// Encode timestamp (first 10 chars)
	for i := 9; i >= 0; i-- {
		buf[i] = encodingChars[ms&0x1F]
		ms >>= 5
	}

	// Encode randomness (last 16 chars)
	var rndBytes [10]byte
	rand.Read(rndBytes[:])

	for i := 0; i < 16; i++ {
		idx := i / 2
		if i%2 == 0 {
			buf[10+i] = encodingChars[rndBytes[idx]>>3]
		} else {
			buf[10+i] = encodingChars[((rndBytes[idx]&0x07)<<2)|(rndBytes[idx+1]>>6)]
		}
	}

	return string(buf[:])
}
