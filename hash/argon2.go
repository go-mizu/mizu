package hash

import (
	"encoding/binary"
	"math/bits"
)

// Argon2id, specified in RFC 9106, is the password hash this package writes.
// It is here rather than imported because the core of mizu depends on nothing
// outside the standard library, and the standard library does not have it.
//
// What a password hash is for is being slow in a way that cannot be skipped.
// Argon2 spends memory as well as time, so the machine an attacker would build
// to guess passwords costs the same per guess as the server does, which is the
// part that SHA-256 in a loop does not do.
//
// The id variant is the one to use, and the only one here. It reads its own
// memory in a data independent order for the first half of the first pass,
// which resists an attacker who can watch memory access patterns, and in a data
// dependent order afterwards, which resists one who trades memory for time.

const (
	// argonVersion is 1.3, the version RFC 9106 specifies and the only one
	// anything writes today. It goes in the PHC string as v=19.
	argonVersion = 0x13

	// argonType is the type byte for Argon2id, which goes into the first hash.
	argonType = 2

	// argonBlockSize is the size of a block in bytes, and argonWords is the
	// same thing in 64 bit words.
	argonBlockSize = 1024
	argonWords     = argonBlockSize / 8

	// syncPoints is how many slices a pass is divided into. It is where the
	// lanes of a parallel run would wait for each other, and it is 4 because
	// the specification says so, not because anything here can be tuned.
	syncPoints = 4
)

// block is one 1024 byte block of the memory Argon2 fills.
type block [argonWords]uint64

// argon2id returns the tag for a password.
//
// time is the number of passes over memory, memory is in kibibytes, lanes is
// the degree of parallelism, and tagLen is the length of the result. The caller
// has already checked them: memory has to be at least 8 lanes, time and lanes
// at least 1, and tagLen at least 4.
//
// secret and data are the two inputs RFC 9106 has that the PHC string format
// has nowhere to put: a key held by the application rather than the database,
// and data to bind the hash to. Nothing here passes them, and they are here so
// that this can be run against the vector in the specification, which uses
// both.
//
// Lanes are filled one after another rather than in parallel. The answer is the
// same either way, since a lane only reads blocks that are already finished,
// and a login is one hash at a time on a server that is doing other things.
func argon2id(password, salt, secret, data []byte, time, memory uint32, lanes uint8, tagLen uint32) []byte {
	h0 := initialHash(password, salt, secret, data, time, memory, lanes, tagLen)

	// Memory is rounded down to a multiple of four blocks per lane, so that
	// every slice of every lane is the same size.
	p := uint32(lanes)
	blocks := memory / (syncPoints * p) * (syncPoints * p)
	cols := blocks / p
	segment := cols / syncPoints

	mem := make([]block, blocks)
	fillFirst(mem, h0, cols, lanes)

	for pass := range time {
		for slice := range uint32(syncPoints) {
			for lane := range p {
				fillSegment(mem, pass, slice, lane, time, blocks, cols, segment, lanes)
			}
		}
	}

	// The tag comes from the last block of every lane, xored together.
	final := mem[cols-1]
	for lane := uint32(1); lane < p; lane++ {
		last := &mem[lane*cols+cols-1]
		for i := range final {
			final[i] ^= last[i]
		}
	}

	tag := blake2bLong(int(tagLen), final.bytes())
	clear(mem)
	return tag
}

// initialHash is H0, the 64 byte hash of the parameters and the input that
// every block is derived from.
func initialHash(password, salt, secret, data []byte, time, memory uint32, lanes uint8, tagLen uint32) []byte {
	d := newBlake2b(maxDigest)

	var buf [4]byte
	put := func(v uint32) {
		binary.LittleEndian.PutUint32(buf[:], v)
		d.write(buf[:])
	}
	putBytes := func(b []byte) {
		put(uint32(len(b)))
		d.write(b)
	}

	put(uint32(lanes))
	put(tagLen)
	put(memory)
	put(time)
	put(argonVersion)
	put(argonType)
	putBytes(password)
	putBytes(salt)
	putBytes(secret)
	putBytes(data)

	return d.sum(make([]byte, 0, maxDigest))
}

// fillFirst writes the two blocks at the start of every lane, which are the
// only ones that come from H0 rather than from other blocks.
func fillFirst(mem []block, h0 []byte, cols uint32, lanes uint8) {
	buf := make([]byte, maxDigest+8)
	copy(buf, h0)

	for lane := range uint32(lanes) {
		for col := range uint32(2) {
			binary.LittleEndian.PutUint32(buf[maxDigest:], col)
			binary.LittleEndian.PutUint32(buf[maxDigest+4:], lane)
			mem[lane*cols+col].read(blake2bLong(argonBlockSize, buf))
		}
	}
}

// fillSegment fills one lane's share of one slice, which is the unit the lanes
// of a parallel run would synchronize on.
func fillSegment(mem []block, pass, slice, lane, time, blocks, cols, segment uint32, lanes uint8) {
	// Argon2id reads its memory in an order that does not depend on the
	// password for the first half of the first pass, and in one that does for
	// the rest. The first half is what an attacker watching memory accesses
	// would learn from, and the rest is what makes trading memory for time
	// expensive.
	independent := pass == 0 && slice < syncPoints/2

	var addr, input, zero block
	if independent {
		input[0] = uint64(pass)
		input[1] = uint64(lane)
		input[2] = uint64(slice)
		input[3] = uint64(blocks)
		input[4] = uint64(time)
		input[5] = argonType
	}

	start := uint32(0)
	if pass == 0 && slice == 0 {
		// The first two blocks of the lane are already there, and the addresses
		// for them have been drawn and thrown away.
		start = 2
		if independent {
			nextAddresses(&addr, &input, &zero)
		}
	}

	for i := start; i < segment; i++ {
		if independent && i%argonWords == 0 {
			nextAddresses(&addr, &input, &zero)
		}

		col := slice*segment + i
		prev := col - 1
		if col == 0 {
			prev = cols - 1
		}

		// rand picks the block to mix in. Data independent addressing takes it
		// from a block of addresses generated from the parameters, and data
		// dependent addressing takes it from the previous block, which is where
		// the password gets into the order.
		rand := mem[lane*cols+prev][0]
		if independent {
			rand = addr[i%argonWords]
		}

		refLane := uint32(rand>>32) % uint32(lanes)
		if pass == 0 && slice == 0 {
			// Nothing in another lane exists yet.
			refLane = lane
		}
		refCol := refIndex(uint32(rand), pass, slice, i, segment, cols, refLane == lane)

		fillBlock(&mem[lane*cols+prev], &mem[refLane*cols+refCol], &mem[lane*cols+col], pass > 0)
	}
}

// refIndex is which block of the reference lane to mix in, RFC 9106 section
// 3.4.1.2.
//
// The area to choose from is everything already finished, and the choice inside
// it is weighted towards the recent end, so that the blocks written last are
// the ones most likely to be read again.
func refIndex(rand, pass, slice, index, segment, cols uint32, sameLane bool) uint32 {
	var area uint32
	switch {
	case pass == 0 && slice == 0:
		area = index - 1
	case pass == 0 && sameLane:
		area = slice*segment + index - 1
	case pass == 0:
		area = slice * segment
		if index == 0 {
			area--
		}
	case sameLane:
		area = cols - segment + index - 1
	default:
		area = cols - segment
		if index == 0 {
			area--
		}
	}

	// Two multiplications that keep the high half, which is a square weighted
	// towards the end of the area without dividing by anything.
	pos := uint64(rand)
	pos = pos * pos >> 32
	pos = uint64(area) - 1 - (uint64(area)*pos)>>32

	// Everything before the current slice, wrapping round the lane, which for
	// a later pass means starting after the slice being overwritten.
	var from uint32
	if pass != 0 && slice != syncPoints-1 {
		from = (slice + 1) * segment
	}
	return uint32((uint64(from) + pos) % uint64(cols))
}

// nextAddresses is the block of addresses that data independent addressing
// draws from, which is the parameters run through the compression function
// twice.
func nextAddresses(addr, input, zero *block) {
	input[6]++
	fillBlock(zero, input, addr, false)
	fillBlock(zero, addr, addr, false)
}

// fillBlock is the compression function G, RFC 9106 section 3.5. It xors two
// blocks, runs the permutation over the rows and then over the columns, and
// xors the result back with what went in.
//
// With xor set, the answer goes into next by xor rather than by assignment,
// which is what a pass after the first does to the block it is overwriting.
func fillBlock(prev, ref, next *block, xor bool) {
	var t block
	for i := range t {
		t[i] = prev[i] ^ ref[i]
	}

	// The block is an eight by eight grid of pairs of words. The permutation
	// runs over the rows and then over the columns, which is how a change to
	// any word reaches every other word.
	for i := 0; i < argonWords; i += 16 {
		permute((*[16]uint64)(t[i : i+16]))
	}
	for i := 0; i < 16; i += 2 {
		column(&t, i)
	}

	// What comes out is the permuted block xored with what went in, which is
	// what stops the whole thing from being invertible.
	for i := range next {
		if xor {
			next[i] ^= prev[i] ^ ref[i] ^ t[i]
		} else {
			next[i] = prev[i] ^ ref[i] ^ t[i]
		}
	}
}

// permute is P, the BLAKE2b round over sixteen words in place.
func permute(v *[16]uint64) {
	gb(&v[0], &v[4], &v[8], &v[12])
	gb(&v[1], &v[5], &v[9], &v[13])
	gb(&v[2], &v[6], &v[10], &v[14])
	gb(&v[3], &v[7], &v[11], &v[15])
	gb(&v[0], &v[5], &v[10], &v[15])
	gb(&v[1], &v[6], &v[11], &v[12])
	gb(&v[2], &v[7], &v[8], &v[13])
	gb(&v[3], &v[4], &v[9], &v[14])
}

// column is P over one column of the grid, which is the pair of words at
// offset i in each of the eight rows.
//
// The words are spread through the block rather than next to each other, so
// this is the same eight calls as [permute] with the addresses worked out
// rather than a copy taken and put back.
func column(t *block, i int) {
	v := [16]*uint64{
		&t[i], &t[i+1], &t[16+i], &t[16+i+1], &t[32+i], &t[32+i+1], &t[48+i], &t[48+i+1],
		&t[64+i], &t[64+i+1], &t[80+i], &t[80+i+1], &t[96+i], &t[96+i+1], &t[112+i], &t[112+i+1],
	}
	gb(v[0], v[4], v[8], v[12])
	gb(v[1], v[5], v[9], v[13])
	gb(v[2], v[6], v[10], v[14])
	gb(v[3], v[7], v[11], v[15])
	gb(v[0], v[5], v[10], v[15])
	gb(v[1], v[6], v[11], v[12])
	gb(v[2], v[7], v[8], v[13])
	gb(v[3], v[4], v[9], v[14])
}

// gb is BLAKE2b's mixing function with the multiplication Argon2 adds. The
// multiply is the reason a graphics card does not run this much faster than a
// processor does.
func gb(a, b, c, d *uint64) {
	*a += *b + 2*uint64(uint32(*a))*uint64(uint32(*b))
	*d = bits.RotateLeft64(*d^*a, -32)
	*c += *d + 2*uint64(uint32(*c))*uint64(uint32(*d))
	*b = bits.RotateLeft64(*b^*c, -24)
	*a += *b + 2*uint64(uint32(*a))*uint64(uint32(*b))
	*d = bits.RotateLeft64(*d^*a, -16)
	*c += *d + 2*uint64(uint32(*c))*uint64(uint32(*d))
	*b = bits.RotateLeft64(*b^*c, -63)
}

// read fills a block from its little endian bytes.
func (b *block) read(p []byte) {
	for i := range b {
		b[i] = binary.LittleEndian.Uint64(p[i*8:])
	}
}

// bytes is the block as the little endian bytes it hashes as.
func (b *block) bytes() []byte {
	out := make([]byte, 0, argonBlockSize)
	for _, v := range b {
		out = binary.LittleEndian.AppendUint64(out, v)
	}
	return out
}
