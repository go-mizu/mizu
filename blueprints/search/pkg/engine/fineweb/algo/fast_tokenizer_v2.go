package algo

import (
	"unsafe"
)

// FixedTokenizeV2 is an optimized tokenizer that reads 8 bytes at a time
// for faster delimiter scanning.
func FixedTokenizeV2(text string, table *FixedHashTable) int {
	if len(text) == 0 {
		return 0
	}

	const fnvOffset = 14695981039346656037
	const fnvPrime = 1099511628211

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	table.Reset()

	for i < n {
		// Skip delimiters - read 8 bytes at a time when possible
		for i+8 <= n {
			// Read 8 bytes
			chunk := *(*uint64)(unsafe.Pointer(&data[i]))

			// Check each byte using lookup table
			// If any byte is alphanumeric, break
			if megaToLower[byte(chunk)] != 0 {
				break
			}
			if megaToLower[byte(chunk>>8)] != 0 {
				i++
				break
			}
			if megaToLower[byte(chunk>>16)] != 0 {
				i += 2
				break
			}
			if megaToLower[byte(chunk>>24)] != 0 {
				i += 3
				break
			}
			if megaToLower[byte(chunk>>32)] != 0 {
				i += 4
				break
			}
			if megaToLower[byte(chunk>>40)] != 0 {
				i += 5
				break
			}
			if megaToLower[byte(chunk>>48)] != 0 {
				i += 6
				break
			}
			if megaToLower[byte(chunk>>56)] != 0 {
				i += 7
				break
			}
			i += 8
		}
		// Handle remaining bytes
		for i < n && megaToLower[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		// Found start of token
		start := i
		hash := uint64(fnvOffset)

		// Hash the token - process multiple bytes when possible
		for i+4 <= n {
			c0 := megaToLower[data[i]]
			if c0 == 0 {
				goto done
			}
			hash = (hash ^ uint64(c0)) * fnvPrime

			c1 := megaToLower[data[i+1]]
			if c1 == 0 {
				i++
				goto done
			}
			hash = (hash ^ uint64(c1)) * fnvPrime

			c2 := megaToLower[data[i+2]]
			if c2 == 0 {
				i += 2
				goto done
			}
			hash = (hash ^ uint64(c2)) * fnvPrime

			c3 := megaToLower[data[i+3]]
			if c3 == 0 {
				i += 3
				goto done
			}
			hash = (hash ^ uint64(c3)) * fnvPrime
			i += 4
		}

		// Handle remaining bytes
		for i < n {
			c := megaToLower[data[i]]
			if c == 0 {
				break
			}
			hash = (hash ^ uint64(c)) * fnvPrime
			i++
		}

	done:
		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			table.Insert(hash)
			tokenCount++
		}
	}

	return tokenCount
}

// Pre-computed bitmask for alphanumeric detection
// bit[i] = 1 if byte i is alphanumeric (a-z, A-Z, 0-9)
var alphanumMask [4]uint64

func init() {
	// Build bitmask where bit position corresponds to byte value
	for i := 0; i < 256; i++ {
		if megaToLower[i] != 0 {
			alphanumMask[i/64] |= 1 << (i % 64)
		}
	}
}

// FixedTokenizeV3 uses simpler unrolled loop for delimiter scanning
func FixedTokenizeV3(text string, table *FixedHashTable) int {
	if len(text) == 0 {
		return 0
	}

	const fnvOffset = 14695981039346656037
	const fnvPrime = 1099511628211

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	table.Reset()

	for i < n {
		// Skip delimiters - unroll 4x
		for i+4 <= n {
			if megaToLower[data[i]] != 0 {
				break
			}
			if megaToLower[data[i+1]] != 0 {
				i++
				break
			}
			if megaToLower[data[i+2]] != 0 {
				i += 2
				break
			}
			if megaToLower[data[i+3]] != 0 {
				i += 3
				break
			}
			i += 4
		}
		for i < n && megaToLower[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		// Found start of token
		start := i
		hash := uint64(fnvOffset)

		// Hash the token - unroll 4x
		for i+4 <= n {
			c0 := megaToLower[data[i]]
			if c0 == 0 {
				goto done
			}
			hash = (hash ^ uint64(c0)) * fnvPrime

			c1 := megaToLower[data[i+1]]
			if c1 == 0 {
				i++
				goto done
			}
			hash = (hash ^ uint64(c1)) * fnvPrime

			c2 := megaToLower[data[i+2]]
			if c2 == 0 {
				i += 2
				goto done
			}
			hash = (hash ^ uint64(c2)) * fnvPrime

			c3 := megaToLower[data[i+3]]
			if c3 == 0 {
				i += 3
				goto done
			}
			hash = (hash ^ uint64(c3)) * fnvPrime
			i += 4
		}

		for i < n {
			c := megaToLower[data[i]]
			if c == 0 {
				break
			}
			hash = (hash ^ uint64(c)) * fnvPrime
			i++
		}

	done:
		tokenLen := i - start
		if tokenLen >= 2 && tokenLen <= 32 {
			table.Insert(hash)
			tokenCount++
		}
	}

	return tokenCount
}

// FixedTokenizeV4 reduces branches in the hot loop
func FixedTokenizeV4(text string, table *FixedHashTable) int {
	if len(text) == 0 {
		return 0
	}

	const fnvOffset = 14695981039346656037
	const fnvPrime = 1099511628211

	data := unsafe.Slice(unsafe.StringData(text), len(text))
	n := len(data)
	tokenCount := 0
	i := 0

	table.Reset()

	for i < n {
		// Skip delimiters
		for i < n && megaToLower[data[i]] == 0 {
			i++
		}
		if i >= n {
			break
		}

		// Found start of token - compute hash using Duff's device style unrolling
		start := i
		hash := uint64(fnvOffset)

		// Find token end and hash simultaneously
		// Unroll to reduce loop overhead
		tokenEnd := i
		for tokenEnd < n && megaToLower[data[tokenEnd]] != 0 {
			tokenEnd++
		}

		// Now hash the token bytes
		tokenLen := tokenEnd - start
		if tokenLen >= 2 && tokenLen <= 32 {
			// Hash all bytes
			for j := start; j < tokenEnd; j++ {
				hash = (hash ^ uint64(megaToLower[data[j]])) * fnvPrime
			}
			table.Insert(hash)
			tokenCount++
		}
		i = tokenEnd
	}

	return tokenCount
}
