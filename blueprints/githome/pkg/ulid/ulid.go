package ulid

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"
)

const (
	encodedLen = 26
	encoding   = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
)

var (
	mu      sync.Mutex
	lastGen time.Time
	entropy []byte
)

// New generates a new ULID string.
func New() string {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now().UTC()

	// Generate random entropy
	if entropy == nil || now.Sub(lastGen) > time.Millisecond {
		entropy = make([]byte, 10)
		rand.Read(entropy)
		lastGen = now
	} else {
		// Increment entropy for same-millisecond generation
		for i := len(entropy) - 1; i >= 0; i-- {
			entropy[i]++
			if entropy[i] != 0 {
				break
			}
		}
	}

	// Encode timestamp (48 bits = 6 bytes)
	ts := uint64(now.UnixMilli())
	var buf [16]byte
	buf[0] = byte(ts >> 40)
	buf[1] = byte(ts >> 32)
	buf[2] = byte(ts >> 24)
	buf[3] = byte(ts >> 16)
	buf[4] = byte(ts >> 8)
	buf[5] = byte(ts)

	// Copy entropy (80 bits = 10 bytes)
	copy(buf[6:], entropy)

	return encode(buf[:])
}

// FromTime generates a ULID with a specific timestamp (useful for testing).
func FromTime(t time.Time) string {
	ts := uint64(t.UnixMilli())
	var buf [16]byte
	buf[0] = byte(ts >> 40)
	buf[1] = byte(ts >> 32)
	buf[2] = byte(ts >> 24)
	buf[3] = byte(ts >> 16)
	buf[4] = byte(ts >> 8)
	buf[5] = byte(ts)

	entropy := make([]byte, 10)
	rand.Read(entropy)
	copy(buf[6:], entropy)

	return encode(buf[:])
}

// MustParse parses a ULID string or panics.
func MustParse(s string) [16]byte {
	if len(s) != encodedLen {
		panic("invalid ULID length")
	}
	return decode(s)
}

// Time extracts the timestamp from a ULID string.
func Time(s string) time.Time {
	if len(s) != encodedLen {
		return time.Time{}
	}
	buf := decode(s)
	ts := binary.BigEndian.Uint64(append([]byte{0, 0}, buf[:6]...))
	return time.UnixMilli(int64(ts))
}

func encode(src []byte) string {
	dst := make([]byte, encodedLen)

	// Encode the first 6 bytes (timestamp) into 10 characters
	dst[0] = encoding[(src[0]&224)>>5]
	dst[1] = encoding[src[0]&31]
	dst[2] = encoding[(src[1]&248)>>3]
	dst[3] = encoding[((src[1]&7)<<2)|((src[2]&192)>>6)]
	dst[4] = encoding[(src[2]&62)>>1]
	dst[5] = encoding[((src[2]&1)<<4)|((src[3]&240)>>4)]
	dst[6] = encoding[((src[3]&15)<<1)|((src[4]&128)>>7)]
	dst[7] = encoding[(src[4]&124)>>2]
	dst[8] = encoding[((src[4]&3)<<3)|((src[5]&224)>>5)]
	dst[9] = encoding[src[5]&31]

	// Encode the remaining 10 bytes (entropy) into 16 characters
	dst[10] = encoding[(src[6]&248)>>3]
	dst[11] = encoding[((src[6]&7)<<2)|((src[7]&192)>>6)]
	dst[12] = encoding[(src[7]&62)>>1]
	dst[13] = encoding[((src[7]&1)<<4)|((src[8]&240)>>4)]
	dst[14] = encoding[((src[8]&15)<<1)|((src[9]&128)>>7)]
	dst[15] = encoding[(src[9]&124)>>2]
	dst[16] = encoding[((src[9]&3)<<3)|((src[10]&224)>>5)]
	dst[17] = encoding[src[10]&31]
	dst[18] = encoding[(src[11]&248)>>3]
	dst[19] = encoding[((src[11]&7)<<2)|((src[12]&192)>>6)]
	dst[20] = encoding[(src[12]&62)>>1]
	dst[21] = encoding[((src[12]&1)<<4)|((src[13]&240)>>4)]
	dst[22] = encoding[((src[13]&15)<<1)|((src[14]&128)>>7)]
	dst[23] = encoding[(src[14]&124)>>2]
	dst[24] = encoding[((src[14]&3)<<3)|((src[15]&224)>>5)]
	dst[25] = encoding[src[15]&31]

	return string(dst)
}

func decode(s string) [16]byte {
	var buf [16]byte
	// Reverse the encoding process
	dec := make([]byte, 26)
	for i := 0; i < 26; i++ {
		for j := 0; j < 32; j++ {
			if encoding[j] == s[i] {
				dec[i] = byte(j)
				break
			}
		}
	}

	buf[0] = (dec[0] << 5) | dec[1]
	buf[1] = (dec[2] << 3) | (dec[3] >> 2)
	buf[2] = (dec[3] << 6) | (dec[4] << 1) | (dec[5] >> 4)
	buf[3] = (dec[5] << 4) | (dec[6] >> 1)
	buf[4] = (dec[6] << 7) | (dec[7] << 2) | (dec[8] >> 3)
	buf[5] = (dec[8] << 5) | dec[9]
	buf[6] = (dec[10] << 3) | (dec[11] >> 2)
	buf[7] = (dec[11] << 6) | (dec[12] << 1) | (dec[13] >> 4)
	buf[8] = (dec[13] << 4) | (dec[14] >> 1)
	buf[9] = (dec[14] << 7) | (dec[15] << 2) | (dec[16] >> 3)
	buf[10] = (dec[16] << 5) | dec[17]
	buf[11] = (dec[18] << 3) | (dec[19] >> 2)
	buf[12] = (dec[19] << 6) | (dec[20] << 1) | (dec[21] >> 4)
	buf[13] = (dec[21] << 4) | (dec[22] >> 1)
	buf[14] = (dec[22] << 7) | (dec[23] << 2) | (dec[24] >> 3)
	buf[15] = (dec[24] << 5) | dec[25]

	return buf
}
