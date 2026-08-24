package crypt

import (
	"crypto/aes"
	"crypto/cipher"
)

// The algorithm is XAES-256-GCM, specified at https://c2sp.org/XAES-256-GCM.
//
// AES-256-GCM takes a 96 bit nonce, which is small enough that reusing one by
// accident is a real risk: after about 2^32 messages under the same key, two
// random nonces are likely to collide, and a repeated nonce with GCM loses both
// the confidentiality of those two messages and the authentication key. Getting
// that right means counting messages and keeping the counter somewhere, which
// is a thing to get wrong in a process that restarts or runs twice.
//
// XAES-256-GCM takes a 192 bit nonce instead. Random is then the right answer:
// two of them collide after about 2^80 messages, which nothing this framework
// runs is going to reach. The construction derives a key and a 96 bit nonce
// from the key and the long nonce with three AES calls, and then does ordinary
// AES-256-GCM with those, so what protects the message is the standard library
// and the hardware instructions underneath it.
//
// The alternative, XChaCha20-Poly1305, does the same job. It is not here
// because it is not in the standard library, and this module depends on nothing
// else.

const (
	// nonceSize is the 192 bit nonce XAES-256-GCM takes.
	nonceSize = 24

	// tagSize is what GCM appends.
	tagSize = 16
)

// keyed is a key with the part of the derivation that depends on the key alone
// already worked out, since it is the same for every message.
type keyed struct {
	key   Key
	block cipher.Block
	k1    [aes.BlockSize]byte
}

// newKeyed is the state a key needs before it can encrypt anything: the AES-256
// block cipher, and K1, which is the encryption of a block of zeroes doubled in
// the field.
func newKeyed(k Key) *keyed {
	// A key is 32 bytes and a wrong length is the only thing NewCipher rejects,
	// so there is no error here to report to anybody.
	block, _ := aes.NewCipher(k.b[:])

	var l [aes.BlockSize]byte
	block.Encrypt(l[:], l[:])

	return &keyed{key: k, block: block, k1: double(l)}
}

// aead is the AES-256-GCM cipher and the 96 bit nonce that XAES-256-GCM derives
// for a 192 bit one.
//
// The first twelve bytes of the nonce go into the key derivation and the last
// twelve are the nonce GCM sees, so the whole of it is in play.
func (k *keyed) aead(nonce []byte) (cipher.AEAD, []byte) {
	m1 := [aes.BlockSize]byte{1: 0x01, 2: 0x58}
	m2 := [aes.BlockSize]byte{1: 0x02, 2: 0x58}
	copy(m1[4:], nonce[:12])
	copy(m2[4:], nonce[:12])
	for i := range m1 {
		m1[i] ^= k.k1[i]
		m2[i] ^= k.k1[i]
	}

	var kx [KeySize]byte
	k.block.Encrypt(kx[:aes.BlockSize], m1[:])
	k.block.Encrypt(kx[aes.BlockSize:], m2[:])

	// Both of these reject a wrong size and nothing else, and both sizes are
	// fixed here.
	block, _ := aes.NewCipher(kx[:])
	gcm, _ := cipher.NewGCM(block)

	clear(kx[:])
	return gcm, nonce[12:]
}

// double shifts a block left by one bit in GF(2^128), which is the same subkey
// step CMAC uses: the polynomial goes back in when the bit that fell off the
// top was set.
func double(l [aes.BlockSize]byte) [aes.BlockSize]byte {
	var out [aes.BlockSize]byte

	top := l[0] >> 7
	for i := range len(l) - 1 {
		out[i] = l[i]<<1 | l[i+1]>>7
	}

	// Multiplying by the bit rather than branching on it keeps the time the
	// same for every key.
	out[len(l)-1] = l[len(l)-1]<<1 ^ top*0x87
	return out
}
