package hash

import (
	"encoding/binary"
	"math/bits"
)

// BLAKE2b, specified in RFC 7693, is here because Argon2 is defined in terms of
// it and the standard library does not have it. Nothing outside this package
// uses it, so it is not exported and it carries only what Argon2 asks for: an
// unkeyed hash of any digest length from 1 to 64 bytes.
//
// The implementation is the reference one written out in Go. It is not fast,
// and it does not have to be: the whole of the hashing in Argon2 is the first
// and last few kilobytes, and the pass over memory in between is the
// permutation in argon2.go, not this.

const (
	// blockSize is the 128 bytes BLAKE2b compresses at a time.
	blockSize = 128

	// maxDigest is the largest digest BLAKE2b produces.
	maxDigest = 64
)

// iv is the initialization vector, which is the same as SHA-512's: the
// fractional parts of the square roots of the first eight primes.
var iv = [8]uint64{
	0x6a09e667f3bcc908, 0xbb67ae8584caa73b, 0x3c6ef372fe94f82b, 0xa54ff53a5f1d36f1,
	0x510e527fade682d1, 0x9b05688c2b3e6c1f, 0x1f83d9abfb41bd6b, 0x5be0cd19137e2179,
}

// sigma is the word order each round reads the message block in.
var sigma = [12][16]byte{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
	{14, 10, 4, 8, 9, 15, 13, 6, 1, 12, 0, 2, 11, 7, 5, 3},
	{11, 8, 12, 0, 5, 2, 15, 13, 10, 14, 3, 6, 7, 1, 9, 4},
	{7, 9, 3, 1, 13, 12, 11, 14, 2, 6, 5, 10, 4, 0, 15, 8},
	{9, 0, 5, 7, 2, 4, 10, 15, 14, 1, 11, 12, 6, 8, 3, 13},
	{2, 12, 6, 10, 0, 11, 8, 3, 4, 13, 7, 5, 15, 14, 1, 9},
	{12, 5, 1, 15, 14, 13, 4, 10, 0, 7, 6, 3, 9, 2, 8, 11},
	{13, 11, 7, 14, 12, 1, 3, 9, 5, 0, 15, 4, 8, 6, 2, 10},
	{6, 15, 14, 9, 11, 3, 0, 8, 12, 2, 13, 7, 1, 4, 10, 5},
	{10, 2, 8, 4, 7, 6, 1, 5, 15, 11, 9, 14, 3, 12, 13, 0},

	// Twelve rounds over ten orderings, so the first two come round again.
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
	{14, 10, 4, 8, 9, 15, 13, 6, 1, 12, 0, 2, 11, 7, 5, 3},
}

// blake2b is a hash in progress.
//
// A block is held back until something follows it, because the last block is
// compressed with a flag the others do not have and there is no way to know a
// block is the last one until the next write arrives or the digest is asked
// for.
type blake2b struct {
	h    [8]uint64
	t    uint64
	buf  [blockSize]byte
	n    int
	size int
}

// newBlake2b is a hash producing size bytes, which has to be between 1 and 64.
func newBlake2b(size int) *blake2b {
	d := &blake2b{h: iv, size: size}

	// The parameter block, which for an unkeyed hash of one lane is the digest
	// length, no key, fanout 1 and depth 1.
	d.h[0] ^= 0x01010000 | uint64(size)
	return d
}

// write adds to what is being hashed.
func (d *blake2b) write(p []byte) {
	// Fill the held block first, and only compress it once there is more.
	if d.n > 0 {
		copied := copy(d.buf[d.n:], p)
		d.n += copied
		p = p[copied:]

		if len(p) == 0 {
			return
		}
		d.t += blockSize
		d.compress(d.buf[:], false)
		d.n = 0
	}

	for len(p) > blockSize {
		d.t += blockSize
		d.compress(p[:blockSize], false)
		p = p[blockSize:]
	}

	d.n = copy(d.buf[:], p)
}

// sum appends the digest to dst. The hash is finished afterwards and is not
// written to again.
func (d *blake2b) sum(dst []byte) []byte {
	d.t += uint64(d.n)
	clear(d.buf[d.n:])
	d.compress(d.buf[:], true)

	for _, h := range d.h {
		dst = binary.LittleEndian.AppendUint64(dst, h)
	}
	return dst[:len(dst)-(maxDigest-d.size)]
}

// compress is F from RFC 7693 section 3.2, over one block.
func (d *blake2b) compress(block []byte, last bool) {
	var m [16]uint64
	for i := range m {
		m[i] = binary.LittleEndian.Uint64(block[i*8:])
	}

	var v [16]uint64
	copy(v[:8], d.h[:])
	copy(v[8:], iv[:])

	// The counter is 128 bits and this hashes kilobytes, so the high half is
	// always zero and v[13] is left alone.
	v[12] ^= d.t
	if last {
		v[14] = ^v[14]
	}

	for r := range 12 {
		s := &sigma[r]
		g(&v[0], &v[4], &v[8], &v[12], m[s[0]], m[s[1]])
		g(&v[1], &v[5], &v[9], &v[13], m[s[2]], m[s[3]])
		g(&v[2], &v[6], &v[10], &v[14], m[s[4]], m[s[5]])
		g(&v[3], &v[7], &v[11], &v[15], m[s[6]], m[s[7]])
		g(&v[0], &v[5], &v[10], &v[15], m[s[8]], m[s[9]])
		g(&v[1], &v[6], &v[11], &v[12], m[s[10]], m[s[11]])
		g(&v[2], &v[7], &v[8], &v[13], m[s[12]], m[s[13]])
		g(&v[3], &v[4], &v[9], &v[14], m[s[14]], m[s[15]])
	}

	for i := range d.h {
		d.h[i] ^= v[i] ^ v[i+8]
	}
}

// g is the mixing function, RFC 7693 section 3.1.
func g(a, b, c, d *uint64, x, y uint64) {
	*a += *b + x
	*d = bits.RotateLeft64(*d^*a, -32)
	*c += *d
	*b = bits.RotateLeft64(*b^*c, -24)
	*a += *b + y
	*d = bits.RotateLeft64(*d^*a, -16)
	*c += *d
	*b = bits.RotateLeft64(*b^*c, -63)
}

// blake2bSum is the digest of everything given, in order, at the length asked
// for.
func blake2bSum(size int, parts ...[]byte) []byte {
	d := newBlake2b(size)
	for _, p := range parts {
		d.write(p)
	}
	return d.sum(make([]byte, 0, size))
}

// blake2bLong is H' from RFC 9106 section 3.3, the variable length hash Argon2
// uses where it wants more than the 64 bytes BLAKE2b gives.
//
// Up to 64 bytes it is BLAKE2b over the length and the input. Past that it
// chains 64 byte digests and keeps the first 32 bytes of each, so a 1024 byte
// block is 31 links and a 32 byte remainder.
func blake2bLong(size int, parts ...[]byte) []byte {
	var length [4]byte
	binary.LittleEndian.PutUint32(length[:], uint32(size))

	if size <= maxDigest {
		return blake2bSum(size, append([][]byte{length[:]}, parts...)...)
	}

	out := make([]byte, 0, size)
	v := blake2bSum(maxDigest, append([][]byte{length[:]}, parts...)...)

	// Each link contributes 32 bytes until what is left fits in one digest.
	for size-len(out) > maxDigest {
		out = append(out, v[:32]...)
		v = blake2bSum(maxDigest, v)
	}
	return append(out, v[:size-len(out)]...)
}
