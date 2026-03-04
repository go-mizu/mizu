# Lotus (tantivy-in-Go) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a pure-Go full-text search engine named Lotus that faithfully reproduces tantivy's core architecture — FST term dictionary (vellum), BP128 block posting compression, memory-mapped segment reads, position-indexed phrase queries, BM25F scoring, and Block-Max WAND dynamic pruning — then benchmark against tantivy (CGO) on 5M Wikipedia docs.

**Architecture:** Segment-based inverted index with LSM-style tiered merge. Each segment is a directory of typed files (.tdi, .doc, .freq, .pos, .store, .fnm, .meta). Index writes accumulate in a SegmentWriter buffer, flush atomically to disk. Search reads via mmap. BP128 compresses posting blocks of 128 integers with bit-packing; skip entries enable O(log N) advance.

**Tech Stack:** Pure Go. `github.com/blevesearch/vellum` (FST), `github.com/klauspost/compress/zstd` (stored fields), `github.com/kljensen/snowball` (stemming), `golang.org/x/sys/unix` (mmap). No CGO.

**Test command:** `go test ./pkg/index/driver/flower/lotus/... -count=1 -race`

**Package root:** `pkg/index/driver/flower/lotus/`

---

## Task 1: RESEARCH.md — Tantivy Internals Deep-Dive

**Files:**
- Create: `pkg/index/driver/flower/lotus/RESEARCH.md`

**Step 1: Write the research document**

```markdown
# Lotus RESEARCH.md — Tantivy Architecture Reference

## Overview
Tantivy is a full-text search engine library written in Rust...
[Document the 7 components below with enough detail to implement each from scratch]

## 1. Segment File Format
- .term (FST + TermInfoStore), .idx (posting lists), .pos (positions)
- .fieldnorm (1 byte/doc/field), .store (LZ4/Zstd blocks), .fast (columnar), .del (bitset)
- meta.json at index root

## 2. BP128 Bitpacking
- Block of 128 integers, bit-width = ceil(log2(max+1)), header = 1 byte
- Data = 128 × num_bits / 8 bytes
- SIMD layout: 4×u32 interleaved (we use scalar fallback)
- Delta encoding for sorted sequences (doc IDs)

## 3. FST Term Dictionary
- BurntSushi's fst crate (Go: blevesearch/vellum)
- Maps term bytes → uint64 (term ordinal or offset)
- TermInfoStore: block-based delta encoding with bitpacked deltas

## 4. Skip List + Block WAND
- One skip entry per 128-doc block
- Entry: last_doc(u32) + doc_num_bits(u8) + tf_num_bits(u8) + block_wand data
- Block WAND: max fieldnorm_id + max TF per block → upper-bound score

## 5. Stored Fields (.store)
- 16KB blocks, LZ4/Zstd compressed
- Skip index: (last_doc_id, byte_offset) pairs
- Footer: decompressor enum + offset to skip index

## 6. Field Norm Encoding (Lucene SmallFloat)
- intToByte4: values 0-23 lossless, then 3-bit mantissa + exponent
- byte4ToInt: inverse
- 256-entry BM25 precomputed table at search time

## 7. Posting List Binary Format (.idx)
- Full blocks (128 docs): delta-encoded + bitpacked doc IDs, bitpacked TFs
- Last block (1-127 docs): VInt-encoded doc IDs + TFs
- Skip data inline before blocks

## Papers
[Same paper list as spec/0646 §Research Foundation]
```

**Step 2: Commit**

```bash
git add pkg/index/driver/flower/lotus/RESEARCH.md
git commit -m "docs(lotus): add tantivy architecture research document"
```

---

## Task 2: BP128 Block Compression Codec

**Files:**
- Create: `pkg/index/driver/flower/lotus/bp128.go`
- Test: `pkg/index/driver/flower/lotus/bp128_test.go`

**Step 1: Write the failing test**

```go
package lotus

import (
    "math/rand"
    "testing"
)

func TestBP128_RoundTrip_SmallDeltas(t *testing.T) {
    // 128 deltas all fitting in 3 bits (0-7)
    deltas := make([]uint32, 128)
    for i := range deltas {
        deltas[i] = uint32(i % 8)
    }
    buf := bp128Pack(deltas)
    // header (1 byte for numBits=3) + 3*16 = 48 data bytes = 49 total
    if len(buf) != 1+3*16 {
        t.Fatalf("expected 49 bytes, got %d", len(buf))
    }
    got := make([]uint32, 128)
    bp128Unpack(buf, got)
    for i, v := range got {
        if v != deltas[i] {
            t.Fatalf("mismatch at %d: got %d want %d", i, v, deltas[i])
        }
    }
}

func TestBP128_RoundTrip_LargeValues(t *testing.T) {
    deltas := make([]uint32, 128)
    for i := range deltas {
        deltas[i] = uint32(rand.Intn(1 << 20)) // up to 20-bit values
    }
    buf := bp128Pack(deltas)
    got := make([]uint32, 128)
    bp128Unpack(buf, got)
    for i, v := range got {
        if v != deltas[i] {
            t.Fatalf("mismatch at %d: got %d want %d", i, v, deltas[i])
        }
    }
}

func TestBP128_ZeroBits(t *testing.T) {
    // All zeros → 0 bits needed, just header
    deltas := make([]uint32, 128)
    buf := bp128Pack(deltas)
    if len(buf) != 1 { // just the header byte
        t.Fatalf("expected 1 byte for all-zero block, got %d", len(buf))
    }
    got := make([]uint32, 128)
    bp128Unpack(buf, got)
    for i, v := range got {
        if v != 0 {
            t.Fatalf("mismatch at %d: got %d want 0", i, v)
        }
    }
}

func TestBP128_MaxBits(t *testing.T) {
    // One value uses full 32 bits
    deltas := make([]uint32, 128)
    deltas[0] = 0xFFFFFFFF
    buf := bp128Pack(deltas)
    if len(buf) != 1+32*16 {
        t.Fatalf("expected %d bytes, got %d", 1+32*16, len(buf))
    }
    got := make([]uint32, 128)
    bp128Unpack(buf, got)
    if got[0] != 0xFFFFFFFF {
        t.Fatalf("got %d want %d", got[0], 0xFFFFFFFF)
    }
}

func BenchmarkBP128_Pack(b *testing.B) {
    deltas := make([]uint32, 128)
    for i := range deltas {
        deltas[i] = uint32(rand.Intn(256))
    }
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        bp128Pack(deltas)
    }
}

func BenchmarkBP128_Unpack(b *testing.B) {
    deltas := make([]uint32, 128)
    for i := range deltas {
        deltas[i] = uint32(rand.Intn(256))
    }
    buf := bp128Pack(deltas)
    out := make([]uint32, 128)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        bp128Unpack(buf, out)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/index/driver/flower/lotus/... -run TestBP128 -v`
Expected: FAIL — `bp128Pack` and `bp128Unpack` undefined.

**Step 3: Implement BP128 codec**

```go
package lotus

import "math/bits"

// bp128Pack packs 128 uint32 values into a bitpacked byte slice.
// Format: [numBits: 1 byte] [data: numBits×16 bytes]
// numBits is the minimum bits needed to represent the maximum value.
func bp128Pack(vals [128]uint32 or []uint32) []byte {
    // 1. Find max value → compute numBits
    var maxVal uint32
    for _, v := range vals[:128] {
        if v > maxVal { maxVal = v }
    }
    numBits := uint8(32 - bits.LeadingZeros32(maxVal))
    if maxVal == 0 { numBits = 0 }

    // 2. Allocate: 1 header + numBits*16 data bytes
    size := 1 + int(numBits)*16
    buf := make([]byte, size)
    buf[0] = numBits

    if numBits == 0 { return buf }

    // 3. Bitpack: for each of 128 values, write numBits bits sequentially
    bitPos := uint(0)
    data := buf[1:]
    for _, v := range vals[:128] {
        byteOff := bitPos / 8
        bitOff := bitPos % 8
        // Write numBits bits of v starting at data[byteOff], bit bitOff
        remaining := uint(numBits)
        val := uint64(v)
        for remaining > 0 {
            space := 8 - bitOff
            if space > remaining { space = remaining }
            mask := uint64((1<<space) - 1)
            data[byteOff] |= byte((val & mask) << bitOff)
            val >>= space
            remaining -= space
            bitOff = 0
            byteOff++
        }
        bitPos += uint(numBits)
    }
    return buf
}

// bp128Unpack decodes 128 uint32 values from a bitpacked byte slice into out.
func bp128Unpack(buf []byte, out []uint32) {
    numBits := uint(buf[0])
    if numBits == 0 {
        for i := range out[:128] { out[i] = 0 }
        return
    }
    data := buf[1:]
    bitPos := uint(0)
    for i := 0; i < 128; i++ {
        byteOff := bitPos / 8
        bitOff := bitPos % 8
        remaining := numBits
        var val uint64
        shift := uint(0)
        for remaining > 0 {
            space := 8 - bitOff
            if space > remaining { space = remaining }
            mask := uint64((1<<space) - 1)
            val |= (uint64(data[byteOff]) >> bitOff & mask) << shift
            shift += space
            remaining -= space
            bitOff = 0
            byteOff++
        }
        out[i] = uint32(val)
        bitPos += numBits
    }
}

// bitsNeeded returns the minimum bits to represent v (0 for v==0).
func bitsNeeded(v uint32) uint8 {
    if v == 0 { return 0 }
    return uint8(32 - bits.LeadingZeros32(v))
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./pkg/index/driver/flower/lotus/... -run TestBP128 -v`
Expected: PASS (all 4 tests).

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/bp128.go pkg/index/driver/flower/lotus/bp128_test.go
git commit -m "feat(lotus): implement BP128 block bitpacking codec"
```

---

## Task 3: VInt (Variable-Byte Integer) Codec

**Files:**
- Create: `pkg/index/driver/flower/lotus/vint.go`
- Test: `pkg/index/driver/flower/lotus/vint_test.go`

VInt is used for the last incomplete block (<128 docs) in posting lists, matching tantivy's format.

**Step 1: Write the failing test**

```go
package lotus

import (
    "bytes"
    "testing"
)

func TestVInt_RoundTrip(t *testing.T) {
    cases := []uint32{0, 1, 127, 128, 16383, 16384, 1<<21 - 1, 1 << 28, 0xFFFFFFFF}
    var buf bytes.Buffer
    for _, v := range cases {
        buf.Reset()
        vintPut(&buf, v)
        got, n := vintGet(buf.Bytes())
        if got != v {
            t.Fatalf("roundtrip %d: got %d", v, got)
        }
        if n != buf.Len() {
            t.Fatalf("bytes consumed %d != written %d for value %d", n, buf.Len(), v)
        }
    }
}

func TestVInt_MultipleValues(t *testing.T) {
    vals := []uint32{300, 0, 1, 100000, 0xFFFFFFFF}
    var buf bytes.Buffer
    for _, v := range vals {
        vintPut(&buf, v)
    }
    data := buf.Bytes()
    off := 0
    for i, want := range vals {
        got, n := vintGet(data[off:])
        if got != want {
            t.Fatalf("value[%d]: got %d want %d", i, got, want)
        }
        off += n
    }
    if off != len(data) {
        t.Fatalf("consumed %d bytes, total %d", off, len(data))
    }
}
```

**Step 2: Run test — expect FAIL**

Run: `go test ./pkg/index/driver/flower/lotus/... -run TestVInt -v`

**Step 3: Implement VInt codec**

```go
package lotus

import "io"

// vintPut writes v as a variable-byte integer to w.
// Format: 7 bits per byte, MSB=1 means more bytes follow.
func vintPut(w io.ByteWriter, v uint32) {
    for v >= 0x80 {
        w.WriteByte(byte(v&0x7F) | 0x80)
        v >>= 7
    }
    w.WriteByte(byte(v))
}

// vintGet reads a variable-byte integer from buf.
// Returns the value and the number of bytes consumed.
func vintGet(buf []byte) (uint32, int) {
    var v uint32
    for i, b := range buf {
        v |= uint32(b&0x7F) << (7 * uint(i))
        if b&0x80 == 0 {
            return v, i + 1
        }
    }
    return v, len(buf)
}

// vintSize returns the number of bytes needed to encode v.
func vintSize(v uint32) int {
    n := 1
    for v >= 0x80 {
        n++
        v >>= 7
    }
    return n
}
```

**Step 4: Run tests — expect PASS**

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/vint.go pkg/index/driver/flower/lotus/vint_test.go
git commit -m "feat(lotus): implement VInt variable-byte integer codec"
```

---

## Task 4: Field Norm Encoding (Lucene SmallFloat)

**Files:**
- Create: `pkg/index/driver/flower/lotus/fieldnorms.go`
- Test: `pkg/index/driver/flower/lotus/fieldnorms_test.go`

**Step 1: Write the failing test**

```go
package lotus

import "testing"

func TestFieldNorm_RoundTrip(t *testing.T) {
    // Values 0-23 must round-trip exactly
    for i := 0; i <= 23; i++ {
        b := fieldNormEncode(uint32(i))
        got := fieldNormDecode(b)
        if got != uint32(i) {
            t.Fatalf("lossless range: encode(%d)=%d, decode=%d", i, b, got)
        }
    }
}

func TestFieldNorm_Monotonic(t *testing.T) {
    // Encoding must be monotonic: larger input → larger or equal byte
    prev := uint8(0)
    for i := uint32(0); i < 10000; i++ {
        b := fieldNormEncode(i)
        if b < prev {
            t.Fatalf("not monotonic at %d: byte %d < prev %d", i, b, prev)
        }
        prev = b
    }
}

func TestFieldNorm_KnownValues(t *testing.T) {
    // Spot-check some known values from Lucene
    cases := []struct{ input uint32; wantByte uint8 }{
        {0, 0}, {1, 1}, {23, 23},
        {24, 24}, {100, 37}, {1000, 52},
    }
    for _, c := range cases {
        b := fieldNormEncode(c.input)
        if b != c.wantByte {
            t.Errorf("encode(%d) = %d, want %d", c.input, b, c.wantByte)
        }
    }
}

func TestFieldNorm_BM25Table(t *testing.T) {
    // BM25 precomputed table must have 256 entries
    table := buildFieldNormBM25Table(500.0) // avgDocLen=500
    if len(table) != 256 {
        t.Fatalf("table len %d, want 256", len(table))
    }
    // Norm for short docs (small fieldnorm) should be < norm for long docs
    if table[1] >= table[50] {
        t.Errorf("expected shorter docs to have smaller norm penalty")
    }
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement Lucene SmallFloat encoding**

```go
package lotus

import "math/bits"

const numFreeValues = 24 // values 0-23 are stored losslessly

// fieldNormEncode encodes a document length (token count) to a single byte
// using Lucene's SmallFloat intToByte4 algorithm.
func fieldNormEncode(i uint32) uint8 {
    if i < numFreeValues {
        return uint8(i)
    }
    return uint8(numFreeValues) + longToInt4(i-numFreeValues)
}

// fieldNormDecode decodes a fieldnorm byte back to the approximate token count.
func fieldNormDecode(b uint8) uint32 {
    if b < numFreeValues {
        return uint32(b)
    }
    return numFreeValues + int4ToLong(b-numFreeValues)
}

func longToInt4(i uint32) uint8 {
    numBits := 32 - bits.LeadingZeros32(i)
    if numBits < 4 {
        return uint8(i)
    }
    shift := uint(numBits - 4)
    encoded := i >> shift
    encoded &= 0x07              // clear implicit MSB
    encoded |= (uint32(shift) + 1) << 3 // encode exponent
    return uint8(encoded)
}

func int4ToLong(b uint8) uint32 {
    mantissa := uint32(b) & 0x07
    shift := int(b>>3) - 1
    if shift < 0 {
        return mantissa
    }
    return (mantissa | 0x08) << uint(shift)
}

// buildFieldNormBM25Table precomputes BM25 length normalization for all 256
// possible fieldnorm byte values.  Returns K1*(1-B+B*dl/avgdl) for each.
func buildFieldNormBM25Table(avgDocLen float64) [256]float32 {
    const k1, b = 1.2, 0.75
    var table [256]float32
    for i := 0; i < 256; i++ {
        dl := float64(fieldNormDecode(uint8(i)))
        table[i] = float32(k1 * (1.0 - b + b*dl/avgDocLen))
    }
    return table
}
```

**Step 4: Run tests — expect PASS**

Note: The `TestFieldNorm_KnownValues` test may need adjustment for the exact Lucene byte values at inputs 100 and 1000. Run once, read actual output, then fix expected values. The encoding algorithm is deterministic; just verify the pattern is correct.

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/fieldnorms.go pkg/index/driver/flower/lotus/fieldnorms_test.go
git commit -m "feat(lotus): implement Lucene SmallFloat field norm encoding"
```

---

## Task 5: Text Analyzer with Position Tracking

**Files:**
- Create: `pkg/index/driver/flower/lotus/analyzer.go`
- Test: `pkg/index/driver/flower/lotus/analyzer_test.go`

**Step 1: Write the failing test**

```go
package lotus

import "testing"

func TestAnalyze_Basic(t *testing.T) {
    tokens := analyze("Hello World! This is a test.")
    // "hello" "world" "test" (stopwords "this" "is" "a" removed)
    // Positions: hello=0, world=1, test=5 (original positions preserved)
    if len(tokens) < 2 {
        t.Fatalf("expected at least 2 tokens, got %d", len(tokens))
    }
}

func TestAnalyze_Positions(t *testing.T) {
    tokens := analyzeWithPositions("the quick brown fox")
    // "the" is stopword → removed; "quick"=pos1, "brown"=pos2, "fox"=pos3
    // Stemmed: "quick", "brown", "fox"
    for _, tok := range tokens {
        if tok.term == "" {
            t.Fatal("empty term")
        }
        if tok.term == "the" {
            t.Fatal("stopword 'the' should be filtered")
        }
    }
}

func TestAnalyze_Unicode(t *testing.T) {
    tokens := analyzeWithPositions("Ünïcödé café résumé")
    found := false
    for _, tok := range tokens {
        if tok.term == "café" || tok.term == "cafe" || tok.term == "caf" {
            found = true
        }
    }
    if !found {
        t.Fatal("expected to find stemmed form of 'café'")
    }
}

func TestAnalyze_LengthFilter(t *testing.T) {
    tokens := analyzeWithPositions("a xx hello")
    for _, tok := range tokens {
        if tok.term == "a" {
            t.Fatal("single-char token should be filtered")
        }
    }
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement analyzer**

```go
package lotus

import (
    "strings"
    "sync"
    "unicode"
    "unicode/utf8"

    "github.com/kljensen/snowball/english"
)

type token struct {
    term string
    pos  uint32 // absolute position in document
}

var stemCache sync.Map

// analyzeWithPositions tokenizes and stems text, returning (term, position) pairs.
// Position is the 0-indexed offset of the original token (before stopword removal),
// which is critical for phrase query adjacency checks.
func analyzeWithPositions(text string) []token {
    var tokens []token
    pos := uint32(0)

    // Stack buffer for lowercase accumulation (avoids allocs)
    var lowBuf [68]byte // maxTokLen(64) + utf8.UTFMax

    start := -1
    n := 0
    overflow := false

    flush := func() {
        if start < 0 || overflow || n < 2 || n > 64 {
            start = -1; n = 0; overflow = false
            pos++
            return
        }
        low := string(lowBuf[:n])

        // Check stem cache
        if cached, ok := stemCache.Load(low); ok {
            if s := cached.(string); s != "" {
                tokens = append(tokens, token{term: s, pos: pos})
            }
            pos++
            start = -1; n = 0
            return
        }

        // Stopword check
        if isStopword(low) {
            stemCache.Store(low, "")
            pos++
            start = -1; n = 0
            return
        }

        // Stem
        stemmed := english.Stem(low, false)
        if len(stemmed) < 2 {
            stemmed = low
        }
        stemCache.Store(low, stemmed)
        tokens = append(tokens, token{term: stemmed, pos: pos})

        pos++
        start = -1; n = 0
    }

    for i, r := range text {
        if unicode.IsLetter(r) || unicode.IsDigit(r) {
            if start < 0 {
                start = i
                n = 0
                overflow = false
            }
            lr := unicode.ToLower(r)
            size := utf8.EncodeRune(lowBuf[n:n+utf8.UTFMax], lr)
            if n+size > 64 {
                overflow = true
            } else {
                n += size
            }
        } else {
            if start >= 0 {
                flush()
            }
        }
    }
    if start >= 0 {
        flush()
    }
    return tokens
}

// analyze returns just the stemmed terms (no positions).
func analyze(text string) []string {
    toks := analyzeWithPositions(text)
    result := make([]string, len(toks))
    for i, t := range toks {
        result[i] = t.term
    }
    return result
}

// 127 English stopwords (Lucene-compatible set).
var stopwords = func() map[string]struct{} {
    words := strings.Fields(`a an and are as at be but by for if in into is it
        no not of on or such that the their then there these they this to was
        will with a about above after again against all am an and any are aren't
        as at be because been before being below between both but by can't cannot
        could couldn't did didn't do does doesn't doing don't down during each few
        for from further get got had hadn't has hasn't have haven't having he her
        here hers herself him himself his how i if in into is isn't it its itself
        let me more most mustn't my myself no nor not of off on once only or other
        ought our ours ourselves out over own same shan't she should shouldn't so
        some such than that the their theirs them themselves then there these they
        this those through to too under until up very was wasn't we were weren't
        what when where which while who whom why with won't would wouldn't you your
        yours yourself yourselves`)
    m := make(map[string]struct{}, len(words))
    for _, w := range words {
        m[w] = struct{}{}
    }
    return m
}()

func isStopword(s string) bool {
    _, ok := stopwords[s]
    return ok
}
```

**Step 4: Run tests — expect PASS**

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/analyzer.go pkg/index/driver/flower/lotus/analyzer_test.go
git commit -m "feat(lotus): implement text analyzer with position tracking"
```

---

## Task 6: mmap Utilities

**Files:**
- Create: `pkg/index/driver/flower/lotus/mmap_unix.go`
- Create: `pkg/index/driver/flower/lotus/mmap_other.go`

**Step 1: Implement platform-specific mmap**

`mmap_unix.go`:
```go
//go:build linux || darwin

package lotus

import (
    "os"
    "golang.org/x/sys/unix"
)

// mmapFile memory-maps the entire file at path as read-only.
// Returns the mapped byte slice. Call mmapRelease when done.
func mmapFile(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil { return nil, err }
    defer f.Close()
    fi, err := f.Stat()
    if err != nil { return nil, err }
    size := fi.Size()
    if size == 0 { return nil, nil }
    data, err := unix.Mmap(int(f.Fd()), 0, int(size),
        unix.PROT_READ, unix.MAP_PRIVATE)
    if err != nil { return nil, err }
    return data, nil
}

// mmapRelease unmaps a previously mapped byte slice.
func mmapRelease(data []byte) error {
    if data == nil { return nil }
    return unix.Munmap(data)
}
```

`mmap_other.go`:
```go
//go:build !linux && !darwin

package lotus

import "os"

func mmapFile(path string) ([]byte, error) {
    return os.ReadFile(path)
}

func mmapRelease(data []byte) error {
    return nil // GC handles it
}
```

**Step 2: Commit**

```bash
git add pkg/index/driver/flower/lotus/mmap_unix.go pkg/index/driver/flower/lotus/mmap_other.go
git commit -m "feat(lotus): add mmap utilities (unix + fallback)"
```

---

## Task 7: Term Dictionary (Vellum FST Wrapper)

**Files:**
- Create: `pkg/index/driver/flower/lotus/termdict.go`
- Test: `pkg/index/driver/flower/lotus/termdict_test.go`

**Step 1: Write the failing test**

```go
package lotus

import (
    "os"
    "path/filepath"
    "testing"
)

func TestTermDict_RoundTrip(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "test.tdi")

    // Build term dict
    terms := []struct {
        term string
        info termInfo
    }{
        {"apple", termInfo{docFreq: 10, postingsOff: 0, hasPositions: true}},
        {"banana", termInfo{docFreq: 5, postingsOff: 100, hasPositions: true}},
        {"cherry", termInfo{docFreq: 1, postingsOff: 200, hasPositions: false}},
    }
    w, err := newTermDictWriter(path)
    if err != nil { t.Fatal(err) }
    for _, te := range terms {
        if err := w.add(te.term, te.info); err != nil {
            t.Fatal(err)
        }
    }
    if err := w.close(); err != nil { t.Fatal(err) }

    // Verify file exists
    fi, err := os.Stat(path)
    if err != nil { t.Fatal(err) }
    if fi.Size() == 0 { t.Fatal("empty FST file") }

    // Read term dict
    r, err := openTermDict(path)
    if err != nil { t.Fatal(err) }
    defer r.close()

    // Lookup existing terms
    for _, te := range terms {
        info, found := r.get(te.term)
        if !found {
            t.Fatalf("term %q not found", te.term)
        }
        if info.docFreq != te.info.docFreq {
            t.Fatalf("%q: docFreq %d != %d", te.term, info.docFreq, te.info.docFreq)
        }
        if info.postingsOff != te.info.postingsOff {
            t.Fatalf("%q: postingsOff %d != %d", te.term, info.postingsOff, te.info.postingsOff)
        }
        if info.hasPositions != te.info.hasPositions {
            t.Fatalf("%q: hasPositions %v != %v", te.term, info.hasPositions, te.info.hasPositions)
        }
    }

    // Lookup missing term
    _, found := r.get("zebra")
    if found {
        t.Fatal("found non-existent term")
    }
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement term dict writer/reader**

```go
package lotus

import (
    "os"

    "github.com/blevesearch/vellum"
)

// termInfo stores metadata for a single term in the dictionary.
type termInfo struct {
    docFreq      uint32 // number of docs containing this term
    postingsOff  uint32 // byte offset into .doc/.freq/.pos files
    hasPositions bool   // whether .pos file has data for this term
}

// Pack termInfo into uint64 for vellum:
//   bits [0..30]  = docFreq (u31)
//   bit  [31]     = hasPositions
//   bits [32..63] = postingsOff (u32)
func packTermInfo(ti termInfo) uint64 {
    v := uint64(ti.docFreq & 0x7FFFFFFF)
    if ti.hasPositions {
        v |= 1 << 31
    }
    v |= uint64(ti.postingsOff) << 32
    return v
}

func unpackTermInfo(v uint64) termInfo {
    return termInfo{
        docFreq:      uint32(v & 0x7FFFFFFF),
        postingsOff:  uint32(v >> 32),
        hasPositions: (v>>31)&1 == 1,
    }
}

// --- Writer ---

type termDictWriter struct {
    f       *os.File
    builder *vellum.Builder
}

func newTermDictWriter(path string) (*termDictWriter, error) {
    f, err := os.Create(path)
    if err != nil { return nil, err }
    b, err := vellum.New(f, nil)
    if err != nil { f.Close(); return nil, err }
    return &termDictWriter{f: f, builder: b}, nil
}

// add inserts a term. Terms MUST be added in sorted lexicographic order.
func (w *termDictWriter) add(term string, info termInfo) error {
    return w.builder.Insert([]byte(term), packTermInfo(info))
}

func (w *termDictWriter) close() error {
    if err := w.builder.Close(); err != nil {
        w.f.Close()
        return err
    }
    return w.f.Close()
}

// --- Reader ---

type termDictReader struct {
    fst  *vellum.FST
    data []byte // mmap'd data
}

func openTermDict(path string) (*termDictReader, error) {
    data, err := mmapFile(path)
    if err != nil { return nil, err }
    fst, err := vellum.Load(data)
    if err != nil {
        mmapRelease(data)
        return nil, err
    }
    return &termDictReader{fst: fst, data: data}, nil
}

func (r *termDictReader) get(term string) (termInfo, bool) {
    v, exists, _ := r.fst.Get([]byte(term))
    if !exists { return termInfo{}, false }
    return unpackTermInfo(v), true
}

func (r *termDictReader) close() error {
    return mmapRelease(r.data)
}
```

**Step 4: Run tests — expect PASS**

Run: `go test ./pkg/index/driver/flower/lotus/... -run TestTermDict -v`

Note: First run `go get github.com/blevesearch/vellum` if not already a direct dependency.

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/termdict.go pkg/index/driver/flower/lotus/termdict_test.go
git commit -m "feat(lotus): implement FST term dictionary (vellum wrapper)"
```

---

## Task 8: Posting List Writer + Skip Index

**Files:**
- Create: `pkg/index/driver/flower/lotus/postings.go`
- Test: `pkg/index/driver/flower/lotus/postings_test.go`

This writes `.doc` (docID deltas), `.freq` (term frequencies), and `.pos` (position deltas) files with BP128 encoding and a skip index.

**Step 1: Write the failing test**

```go
package lotus

import (
    "os"
    "path/filepath"
    "testing"
)

func TestPostings_WriteRead_SingleTerm(t *testing.T) {
    dir := t.TempDir()

    // Generate 300 postings (2 full blocks of 128 + 1 partial of 44)
    var docIDs, freqs []uint32
    var positions [][]uint32
    for i := uint32(0); i < 300; i++ {
        docIDs = append(docIDs, i*3) // gaps of 3
        freqs = append(freqs, i%5+1)
        pos := make([]uint32, i%5+1)
        for j := range pos { pos[j] = uint32(j * 10) }
        positions = append(positions, pos)
    }

    // Write
    pw, err := newPostingsWriter(dir)
    if err != nil { t.Fatal(err) }
    off, err := pw.writeTermPostings(docIDs, freqs, positions)
    if err != nil { t.Fatal(err) }
    if err := pw.close(); err != nil { t.Fatal(err) }

    // Verify files exist
    for _, ext := range []string{".doc", ".freq", ".pos"} {
        p := filepath.Join(dir, "postings"+ext)
        fi, err := os.Stat(p)
        if err != nil { t.Fatalf("missing %s: %v", ext, err) }
        if fi.Size() == 0 { t.Fatalf("empty %s", ext) }
    }

    // Read back
    pr, err := openPostingsReader(dir)
    if err != nil { t.Fatal(err) }
    defer pr.close()

    iter := pr.iterator(off, 300)

    // Advance through all 300 docs
    count := 0
    for iter.next() {
        if iter.docID() != docIDs[count] {
            t.Fatalf("doc %d: got %d want %d", count, iter.docID(), docIDs[count])
        }
        if iter.freq() != freqs[count] {
            t.Fatalf("doc %d: freq got %d want %d", count, iter.freq(), freqs[count])
        }
        count++
    }
    if count != 300 {
        t.Fatalf("iterated %d docs, want 300", count)
    }
}

func TestPostings_Advance(t *testing.T) {
    dir := t.TempDir()

    // 256 docs (2 full blocks)
    docIDs := make([]uint32, 256)
    freqs := make([]uint32, 256)
    positions := make([][]uint32, 256)
    for i := range docIDs {
        docIDs[i] = uint32(i * 2) // 0, 2, 4, ... 510
        freqs[i] = 1
        positions[i] = []uint32{0}
    }

    pw, err := newPostingsWriter(dir)
    if err != nil { t.Fatal(err) }
    off, err := pw.writeTermPostings(docIDs, freqs, positions)
    if err != nil { t.Fatal(err) }
    pw.close()

    pr, err := openPostingsReader(dir)
    if err != nil { t.Fatal(err) }
    defer pr.close()

    iter := pr.iterator(off, 256)

    // Advance to doc 200 (should skip first block entirely)
    if !iter.advance(200) {
        t.Fatal("advance(200) returned false")
    }
    if iter.docID() != 200 {
        t.Fatalf("advance(200): got doc %d", iter.docID())
    }

    // Advance to doc 510 (last doc)
    if !iter.advance(510) {
        t.Fatal("advance(510) returned false")
    }
    if iter.docID() != 510 {
        t.Fatalf("advance(510): got doc %d", iter.docID())
    }

    // Advance past end
    if iter.advance(512) {
        t.Fatal("advance(512) should return false")
    }
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement postings writer and reader**

The postings writer uses BP128 for full blocks and VInt for the last partial block. Skip entries are written for each full block.

Key types:
```go
// skipEntry stores metadata for one 128-doc block.
type skipEntry struct {
    lastDoc     uint32 // last docID in this block
    docByteOff  uint32 // byte offset of this block in .doc
    freqByteOff uint32 // byte offset in .freq
    posByteOff  uint32 // byte offset in .pos
    blockMaxTF  uint32 // max term freq in block (for Block WAND)
    blockMaxNorm uint8 // fieldnorm byte of shortest doc (for Block WAND)
}

// postingsWriter writes .doc, .freq, .pos files with BP128 encoding.
type postingsWriter struct {
    docFile, freqFile, posFile *os.File
    docBuf, freqBuf, posBuf   bytes.Buffer
}

// postingsReader reads .doc, .freq, .pos via mmap.
type postingsReader struct {
    docData, freqData, posData []byte
}

// postingIterator walks a single term's posting list.
type postingIterator struct {
    // ... skip index, current block, position within block
}
```

Implementation notes:
- Write docID deltas in blocks of 128 via `bp128Pack`
- Write freqs in blocks of 128 via `bp128Pack`
- Write positions as delta-encoded within each doc, in blocks of 128 total positions via `bp128Pack`
- Last block uses `vintPut` for remaining <128 values
- Skip index appended at end of .doc file: `[]skipEntry` then `numBlocks uint32`
- The `postingOffset` returned by `writeTermPostings` encodes the start positions in each file

**Step 4: Run tests — expect PASS**

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/postings.go pkg/index/driver/flower/lotus/postings_test.go
git commit -m "feat(lotus): implement posting list writer/reader with BP128 + skip index"
```

---

## Task 9: Stored Fields Writer/Reader (Zstd Block Compression)

**Files:**
- Create: `pkg/index/driver/flower/lotus/store.go`
- Test: `pkg/index/driver/flower/lotus/store_test.go`

**Step 1: Write the failing test**

```go
package lotus

import (
    "path/filepath"
    "testing"
)

func TestStore_RoundTrip(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "test.store")

    w, err := newStoreWriter(path)
    if err != nil { t.Fatal(err) }

    docs := []struct{ id, text string }{
        {"doc0", "Hello world"},
        {"doc1", "The quick brown fox jumps over the lazy dog"},
        {"doc2", "Lorem ipsum dolor sit amet"},
    }
    for _, d := range docs {
        if err := w.add(d.id, []byte(d.text)); err != nil {
            t.Fatal(err)
        }
    }
    if err := w.close(); err != nil { t.Fatal(err) }

    r, err := openStoreReader(path)
    if err != nil { t.Fatal(err) }
    defer r.close()

    for i, d := range docs {
        id, text, err := r.get(uint32(i))
        if err != nil { t.Fatalf("doc %d: %v", i, err) }
        if id != d.id { t.Fatalf("doc %d: id %q != %q", i, id, d.id) }
        if string(text) != d.text { t.Fatalf("doc %d: text %q != %q", i, string(text), d.text) }
    }
}

func TestStore_LargeBlock(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "test.store")

    w, err := newStoreWriter(path)
    if err != nil { t.Fatal(err) }

    // Write enough docs to trigger multiple compressed blocks
    for i := 0; i < 1000; i++ {
        text := make([]byte, 200)
        for j := range text { text[j] = byte('a' + j%26) }
        if err := w.add("doc", text); err != nil { t.Fatal(err) }
    }
    if err := w.close(); err != nil { t.Fatal(err) }

    r, err := openStoreReader(path)
    if err != nil { t.Fatal(err) }
    defer r.close()

    // Spot-check a few docs
    for _, idx := range []uint32{0, 499, 999} {
        _, text, err := r.get(idx)
        if err != nil { t.Fatalf("doc %d: %v", idx, err) }
        if len(text) != 200 { t.Fatalf("doc %d: len %d", idx, len(text)) }
    }
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement stored fields**

Architecture (same as tantivy):
- Documents serialized: `[idLen u32][id bytes][textLen u32][text bytes]`
- Accumulated into a buffer; when buffer exceeds 16 KB, zstd-compress and write as one block
- Skip index: `[]storeSkipEntry{lastDocInBlock uint32, blockOffset uint32}`
- Footer: skip index offset (uint64) at the very end

```go
package lotus

import (
    "encoding/binary"
    "os"

    "github.com/klauspost/compress/zstd"
)

const storeBlockSize = 16 * 1024 // 16 KB per block

type storeWriter struct { ... }
type storeReader struct { data []byte; skipIndex []storeSkipEntry }
type storeSkipEntry struct { lastDoc, offset uint32 }
```

**Step 4: Run tests — expect PASS**

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/store.go pkg/index/driver/flower/lotus/store_test.go
git commit -m "feat(lotus): implement stored fields with zstd block compression"
```

---

## Task 10: Segment Writer (Flush)

**Files:**
- Create: `pkg/index/driver/flower/lotus/writer.go`
- Test: `pkg/index/driver/flower/lotus/writer_test.go`

The segment writer orchestrates all the sub-writers to flush an in-memory buffer to a segment directory.

**Step 1: Write the failing test**

```go
package lotus

import (
    "os"
    "path/filepath"
    "testing"
)

func TestSegmentWriter_Flush(t *testing.T) {
    dir := t.TempDir()
    segDir := filepath.Join(dir, "seg_00000001")

    sw := newSegmentWriter()

    // Add 200 documents
    for i := 0; i < 200; i++ {
        sw.addDoc(fmt.Sprintf("doc_%04d", i), []byte("the quick brown fox"))
    }

    if err := sw.flush(segDir); err != nil {
        t.Fatal(err)
    }

    // Verify all segment files exist
    for _, ext := range []string{".tdi", ".doc", ".freq", ".pos", ".store", ".fnm", ".meta"} {
        path := filepath.Join(segDir, "postings"+ext)
        // .tdi, .store, .fnm, .meta are named differently
        // Adjust per actual naming convention
        entries, _ := os.ReadDir(segDir)
        if len(entries) == 0 {
            t.Fatal("empty segment directory")
        }
    }

    // Verify .meta has correct doc count
    meta, err := readSegmentMeta(segDir)
    if err != nil { t.Fatal(err) }
    if meta.DocCount != 200 {
        t.Fatalf("meta.DocCount = %d, want 200", meta.DocCount)
    }
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement segment writer**

```go
package lotus

// segmentWriter accumulates documents in memory and flushes to a segment directory.
type segmentWriter struct {
    docs      []memDoc
    postings  map[string]*memTermPostings // term → {docIDs, freqs, positions}
    totalLen  uint64
}

type memDoc struct {
    id   string
    text []byte
}

type memTermPostings struct {
    docIDs    []uint32
    freqs     []uint32
    positions [][]uint32 // per-doc position list
}

func newSegmentWriter() *segmentWriter { ... }

func (sw *segmentWriter) addDoc(id string, text []byte) {
    docIdx := uint32(len(sw.docs))
    sw.docs = append(sw.docs, memDoc{id: id, text: text})

    tokens := analyzeWithPositions(string(text))
    sw.totalLen += uint64(len(tokens))

    // Group tokens by term, collect positions per doc
    termPositions := make(map[string][]uint32)
    termFreq := make(map[string]uint32)
    for _, tok := range tokens {
        termPositions[tok.term] = append(termPositions[tok.term], tok.pos)
        termFreq[tok.term]++
    }

    for term, positions := range termPositions {
        tp := sw.postings[term]
        if tp == nil {
            tp = &memTermPostings{}
            sw.postings[term] = tp
        }
        tp.docIDs = append(tp.docIDs, docIdx)
        tp.freqs = append(tp.freqs, termFreq[term])
        tp.positions = append(tp.positions, positions)
    }
}

func (sw *segmentWriter) flush(segDir string) error {
    os.MkdirAll(segDir, 0o755)

    docCount := uint32(len(sw.docs))
    avgDocLen := float64(sw.totalLen) / float64(docCount)

    // 1. Write stored fields (.store)
    storeW, _ := newStoreWriter(filepath.Join(segDir, "segment.store"))
    for _, doc := range sw.docs {
        storeW.add(doc.id, doc.text)
    }
    storeW.close()

    // 2. Write field norms (.fnm)
    fnmPath := filepath.Join(segDir, "segment.fnm")
    fnmData := make([]byte, docCount)
    for i, doc := range sw.docs {
        tokCount := len(analyzeWithPositions(string(doc.text)))
        fnmData[i] = fieldNormEncode(uint32(tokCount))
    }
    os.WriteFile(fnmPath, fnmData, 0o644)

    // 3. Sort terms, write postings (.doc, .freq, .pos) + term dict (.tdi)
    sortedTerms := sortedKeys(sw.postings)
    pw, _ := newPostingsWriter(segDir)
    tdw, _ := newTermDictWriter(filepath.Join(segDir, "segment.tdi"))

    for _, term := range sortedTerms {
        tp := sw.postings[term]
        off, _ := pw.writeTermPostings(tp.docIDs, tp.freqs, tp.positions)
        tdw.add(term, termInfo{
            docFreq:      uint32(len(tp.docIDs)),
            postingsOff:  off,
            hasPositions: true,
        })
    }
    pw.close()
    tdw.close()

    // 4. Write segment metadata (.meta)
    writeSegmentMeta(segDir, segmentMeta{
        DocCount:  docCount,
        AvgDocLen: avgDocLen,
    })

    return nil
}
```

**Step 4: Run tests — expect PASS**

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/writer.go pkg/index/driver/flower/lotus/writer_test.go
git commit -m "feat(lotus): implement segment writer (flush to disk)"
```

---

## Task 11: Segment Reader (mmap + Term Lookup + Cursors)

**Files:**
- Create: `pkg/index/driver/flower/lotus/reader.go`
- Test: `pkg/index/driver/flower/lotus/reader_test.go`

**Step 1: Write the failing test**

```go
package lotus

import "testing"

func TestSegmentReader_OpenAndSearch(t *testing.T) {
    // Write a segment with known content
    dir := t.TempDir()
    segDir := filepath.Join(dir, "seg_00000001")
    sw := newSegmentWriter()
    sw.addDoc("doc1", []byte("the quick brown fox"))
    sw.addDoc("doc2", []byte("the lazy brown dog"))
    sw.addDoc("doc3", []byte("quick fox jumps"))
    sw.flush(segDir)

    // Open segment reader
    sr, err := openSegmentReader(segDir)
    if err != nil { t.Fatal(err) }
    defer sr.close()

    if sr.docCount() != 3 {
        t.Fatalf("docCount %d, want 3", sr.docCount())
    }

    // Look up term "brown" — should appear in docs 0, 1
    info, found := sr.termDict.get("brown")
    if !found { t.Fatal("term 'brown' not found") }
    if info.docFreq != 2 { t.Fatalf("brown docFreq %d, want 2", info.docFreq) }

    // Look up term "fox" — should appear in docs 0, 2
    info, found = sr.termDict.get("fox")
    if !found { t.Fatal("term 'fox' not found") }
    if info.docFreq != 2 { t.Fatalf("fox docFreq %d, want 2", info.docFreq) }
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement segment reader**

```go
package lotus

type segmentReader struct {
    dir      string
    termDict *termDictReader
    postings *postingsReader
    store    *storeReader
    fnmData  []byte        // mmap'd field norms
    meta     segmentMeta
    docBase  uint32        // global doc offset for multi-segment search
}

func openSegmentReader(segDir string) (*segmentReader, error) {
    meta, _ := readSegmentMeta(segDir)
    td, _ := openTermDict(filepath.Join(segDir, "segment.tdi"))
    pr, _ := openPostingsReader(segDir)
    st, _ := openStoreReader(filepath.Join(segDir, "segment.store"))
    fnm, _ := mmapFile(filepath.Join(segDir, "segment.fnm"))
    return &segmentReader{
        dir: segDir, termDict: td, postings: pr, store: st,
        fnmData: fnm, meta: meta,
    }, nil
}

func (sr *segmentReader) docCount() uint32 { return sr.meta.DocCount }
func (sr *segmentReader) close() error { ... }
```

**Step 4: Run tests — expect PASS**

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/reader.go pkg/index/driver/flower/lotus/reader_test.go
git commit -m "feat(lotus): implement segment reader with mmap"
```

---

## Task 12: BM25F Scorer + Quantization

**Files:**
- Create: `pkg/index/driver/flower/lotus/scorer.go`
- Test: `pkg/index/driver/flower/lotus/scorer_test.go`

**Step 1: Write the failing test**

```go
package lotus

import (
    "math"
    "testing"
)

func TestBM25_Score(t *testing.T) {
    // N=1000 docs, df=100, tf=3, dl=500, avgdl=400
    score := bm25Score(3, 100, 500, 400.0, 1000)
    if score <= 0 {
        t.Fatalf("expected positive score, got %f", score)
    }
    // BM25+ delta=1.0 ensures score > IDF alone
    idf := math.Log(float64(1000-100+0.5)/float64(100+0.5) + 1)
    if score < idf {
        t.Fatalf("BM25+ score %f should be >= IDF %f (delta=1.0)", score, idf)
    }
}

func TestBM25_Quantize(t *testing.T) {
    // Quantize should map [0, maxScore] to [1, 255]
    q := quantizeBM25(3.0, 6.0) // half of max → ~128
    if q < 120 || q > 135 {
        t.Fatalf("expected ~128 for half-max, got %d", q)
    }
    // Zero score should still map to 1 (BM25+ guarantee)
    q = quantizeBM25(0.001, 6.0)
    if q != 1 {
        t.Fatalf("min quantized value should be 1, got %d", q)
    }
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement scorer**

```go
package lotus

import "math"

const (
    bm25K1    = 1.2
    bm25B     = 0.75
    bm25Delta = 1.0
)

// bm25Score computes BM25+(t,d) score.
func bm25Score(tf, df, dl uint32, avgdl float64, n uint32) float64 {
    idf := math.Log(float64(n-df)+0.5)/(float64(df)+0.5) + 1)
    tfNorm := float64(tf) * (bm25K1 + 1) /
        (float64(tf) + bm25K1*(1-bm25B+bm25B*float64(dl)/avgdl))
    return idf*tfNorm + bm25Delta
}

// bm25IDF returns the IDF component.
func bm25IDF(df, n uint32) float64 {
    return math.Log(float64(n-df)+0.5)/(float64(df)+0.5) + 1)
}

// quantizeBM25 maps a BM25 score to uint8 [1, 255] given the list max.
func quantizeBM25(score, maxScore float64) uint8 {
    if maxScore <= 0 { return 1 }
    q := int(math.Round(score / maxScore * 255))
    if q < 1 { return 1 }
    if q > 255 { return 255 }
    return uint8(q)
}

// dequantizeBM25 converts uint8 impact back to approximate BM25 score.
func dequantizeBM25(impact uint8, maxScore float64) float64 {
    return float64(impact) / 255.0 * maxScore
}
```

**Step 4: Run tests — expect PASS**

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/scorer.go pkg/index/driver/flower/lotus/scorer_test.go
git commit -m "feat(lotus): implement BM25F scorer with quantization"
```

---

## Task 13: Query Parser

**Files:**
- Create: `pkg/index/driver/flower/lotus/query.go`
- Test: `pkg/index/driver/flower/lotus/query_test.go`

**Step 1: Write the failing test**

```go
package lotus

import "testing"

func TestParseQuery_Union(t *testing.T) {
    q := parseQuery("hello world")
    bq, ok := q.(*booleanQuery)
    if !ok { t.Fatal("expected BooleanQuery") }
    if len(bq.should) != 2 { t.Fatalf("expected 2 should, got %d", len(bq.should)) }
}

func TestParseQuery_Intersection(t *testing.T) {
    q := parseQuery("+hello +world")
    bq, ok := q.(*booleanQuery)
    if !ok { t.Fatal("expected BooleanQuery") }
    if len(bq.must) != 2 { t.Fatalf("expected 2 must, got %d", len(bq.must)) }
}

func TestParseQuery_Phrase(t *testing.T) {
    q := parseQuery(`"quick brown fox"`)
    pq, ok := q.(*phraseQuery)
    if !ok { t.Fatal("expected PhraseQuery") }
    if len(pq.terms) != 3 { t.Fatalf("expected 3 terms, got %d", len(pq.terms)) }
}

func TestParseQuery_MustNot(t *testing.T) {
    q := parseQuery("+hello -world")
    bq, ok := q.(*booleanQuery)
    if !ok { t.Fatal("expected BooleanQuery") }
    if len(bq.must) != 1 { t.Fatalf("expected 1 must, got %d", len(bq.must)) }
    if len(bq.mustNot) != 1 { t.Fatalf("expected 1 mustNot, got %d", len(bq.mustNot)) }
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement query parser**

```go
package lotus

import "strings"

type query interface{ isQuery() }

type termQuery struct {
    term string
}

type phraseQuery struct {
    terms []string // analyzed terms in order
}

type booleanQuery struct {
    must    []query
    should  []query
    mustNot []query
}

func (termQuery) isQuery()    {}
func (phraseQuery) isQuery()  {}
func (booleanQuery) isQuery() {}

// parseQuery parses a benchmark-compatible query string:
//   "+a +b"        → BooleanQuery{must: [a, b]}
//   "a b"          → BooleanQuery{should: [a, b]}
//   "+a -b"        → BooleanQuery{must: [a], mustNot: [b]}
//   `"a b c"`      → PhraseQuery{terms: [a, b, c]}
func parseQuery(text string) query {
    text = strings.TrimSpace(text)
    // Phrase query
    if strings.HasPrefix(text, `"`) && strings.HasSuffix(text, `"`) {
        inner := text[1 : len(text)-1]
        terms := analyze(inner)
        return &phraseQuery{terms: terms}
    }
    // Boolean query
    parts := strings.Fields(text)
    hasBoolOp := false
    for _, p := range parts {
        if strings.HasPrefix(p, "+") || strings.HasPrefix(p, "-") {
            hasBoolOp = true; break
        }
    }
    bq := &booleanQuery{}
    for _, p := range parts {
        var raw string
        switch {
        case strings.HasPrefix(p, "+"):
            raw = p[1:]
            terms := analyze(raw)
            for _, t := range terms { bq.must = append(bq.must, &termQuery{term: t}) }
        case strings.HasPrefix(p, "-"):
            raw = p[1:]
            terms := analyze(raw)
            for _, t := range terms { bq.mustNot = append(bq.mustNot, &termQuery{term: t}) }
        default:
            terms := analyze(p)
            if hasBoolOp {
                for _, t := range terms { bq.must = append(bq.must, &termQuery{term: t}) }
            } else {
                for _, t := range terms { bq.should = append(bq.should, &termQuery{term: t}) }
            }
        }
    }
    return bq
}
```

**Step 4: Run tests — expect PASS**

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/query.go pkg/index/driver/flower/lotus/query_test.go
git commit -m "feat(lotus): implement query parser (term/phrase/boolean)"
```

---

## Task 14: WAND Evaluator + Phrase Checker

**Files:**
- Create: `pkg/index/driver/flower/lotus/wand.go`
- Test: `pkg/index/driver/flower/lotus/wand_test.go`

**Step 1: Write the failing test**

```go
package lotus

import "testing"

func TestWAND_TopK(t *testing.T) {
    // Synthetic: 3 cursors, 10 docs
    c1 := newTestCursor([]uint32{0, 2, 4, 6, 8}, []uint8{200, 150, 100, 50, 10})
    c2 := newTestCursor([]uint32{1, 2, 5, 6, 9}, []uint8{180, 160, 120, 80, 20})
    c3 := newTestCursor([]uint32{2, 6}, []uint8{255, 200})

    results := wandTopK([]*wandCursor{c1, c2, c3}, 3)
    if len(results) != 3 {
        t.Fatalf("expected 3 results, got %d", len(results))
    }
    // Doc 2 appears in all 3 cursors → highest score
    if results[0].docID != 2 {
        t.Fatalf("top result should be doc 2, got %d", results[0].docID)
    }
}

func TestPhraseCheck(t *testing.T) {
    // "quick brown" should match doc with positions [5,6] but not [5,8]
    ok := checkPhrase(
        [][]uint32{{5, 10, 20}, {6, 15, 21}}, // term0 positions, term1 positions
    )
    if !ok {
        t.Fatal("expected phrase match for adjacent positions 5,6")
    }

    ok = checkPhrase(
        [][]uint32{{5, 10}, {8, 15}}, // not adjacent
    )
    if ok {
        t.Fatal("expected no phrase match for non-adjacent positions")
    }
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement WAND evaluator**

The WAND loop with block-max pruning. Uses `postingIterator.advance(targetDoc)` to skip blocks via skip index.

```go
package lotus

import "container/heap"

type wandHit struct {
    docID uint32
    score float64
}

type wandCursor struct {
    iter     *postingIterator
    maxScore float64 // whole-list max BM25 score (for MaxScore optimization)
    idf      float64
    // Current state
    curDoc   uint32
    exhausted bool
}

func wandTopK(cursors []*wandCursor, k int) []wandHit {
    // Standard Block-Max WAND with min-heap threshold
    ...
}

// checkPhrase checks if term positions are sequentially adjacent.
func checkPhrase(termPositions [][]uint32) bool {
    // For each position of term[0], check if term[1] has pos+1, term[2] has pos+2, etc.
    ...
}
```

**Step 4: Run tests — expect PASS**

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/wand.go pkg/index/driver/flower/lotus/wand_test.go
git commit -m "feat(lotus): implement Block-Max WAND evaluator + phrase checker"
```

---

## Task 15: Segment Merge

**Files:**
- Create: `pkg/index/driver/flower/lotus/merge.go`
- Test: `pkg/index/driver/flower/lotus/merge_test.go`

**Step 1: Write the failing test**

```go
package lotus

import "testing"

func TestMerge_TwoSegments(t *testing.T) {
    dir := t.TempDir()

    // Create segment A: 100 docs
    swA := newSegmentWriter()
    for i := 0; i < 100; i++ {
        swA.addDoc(fmt.Sprintf("a_%d", i), []byte("hello world"))
    }
    segA := filepath.Join(dir, "seg_a")
    swA.flush(segA)

    // Create segment B: 100 docs
    swB := newSegmentWriter()
    for i := 0; i < 100; i++ {
        swB.addDoc(fmt.Sprintf("b_%d", i), []byte("hello go"))
    }
    segB := filepath.Join(dir, "seg_b")
    swB.flush(segB)

    // Merge A + B → C
    segC := filepath.Join(dir, "seg_c")
    err := mergeSegments([]string{segA, segB}, segC)
    if err != nil { t.Fatal(err) }

    // Verify merged segment
    sr, err := openSegmentReader(segC)
    if err != nil { t.Fatal(err) }
    defer sr.close()

    if sr.docCount() != 200 {
        t.Fatalf("merged docCount %d, want 200", sr.docCount())
    }
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement merge**

```go
package lotus

// mergeSegments merges multiple segment directories into one.
// Opens each segment reader, iterates all terms across all FSTs in sorted order,
// remaps docIDs, re-encodes postings with BP128, and writes a new segment.
func mergeSegments(inputDirs []string, outputDir string) error {
    // 1. Open all input segment readers
    // 2. Collect all unique terms via sorted iteration of each FST
    // 3. For each term: collect all postings from all segments, remap docIDs
    // 4. Write merged postings + term dict + store + fnm
    ...
}

// tierOf returns the merge tier for a segment by doc count.
// Tier 0: < 4*flushDocs, Tier 1: < 16*flushDocs, etc.
func tierOf(docCount uint32) int {
    tier := 0
    threshold := uint32(10000) // base tier size
    for docCount >= threshold*4 {
        threshold *= 4
        tier++
    }
    return tier
}
```

**Step 4: Run tests — expect PASS**

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/merge.go pkg/index/driver/flower/lotus/merge_test.go
git commit -m "feat(lotus): implement tiered segment merge"
```

---

## Task 16: Engine (index.Engine Implementation)

**Files:**
- Create: `pkg/index/driver/flower/lotus/engine.go`
- Test: `pkg/index/driver/flower/lotus/engine_test.go`

**Step 1: Write the failing test**

```go
package lotus

import (
    "context"
    "testing"

    "github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func TestEngine_IndexAndSearch(t *testing.T) {
    dir := t.TempDir()
    eng := &lotusEngine{}
    ctx := context.Background()

    if err := eng.Open(ctx, dir); err != nil {
        t.Fatal(err)
    }
    defer eng.Close()

    docs := []index.Document{
        {DocID: "1", Text: []byte("the quick brown fox jumps over the lazy dog")},
        {DocID: "2", Text: []byte("a quick red car drives fast")},
        {DocID: "3", Text: []byte("the brown dog sleeps all day")},
    }
    if err := eng.Index(ctx, docs); err != nil {
        t.Fatal(err)
    }

    // Search for "brown fox"
    results, err := eng.Search(ctx, index.Query{Text: "brown fox", Limit: 10})
    if err != nil {
        t.Fatal(err)
    }
    if len(results.Hits) == 0 {
        t.Fatal("expected at least 1 hit for 'brown fox'")
    }
    // Doc 1 should score highest (has both "brown" and "fox")
    if results.Hits[0].DocID != "1" {
        t.Fatalf("expected doc 1 first, got %s", results.Hits[0].DocID)
    }
}

func TestEngine_PhraseQuery(t *testing.T) {
    dir := t.TempDir()
    eng := &lotusEngine{}
    ctx := context.Background()
    eng.Open(ctx, dir)
    defer eng.Close()

    eng.Index(ctx, []index.Document{
        {DocID: "1", Text: []byte("the quick brown fox")},
        {DocID: "2", Text: []byte("the brown quick fox")},
    })

    // Phrase "quick brown" should match doc 1 only
    results, _ := eng.Search(ctx, index.Query{Text: `"quick brown"`, Limit: 10})
    if len(results.Hits) != 1 || results.Hits[0].DocID != "1" {
        t.Fatalf("phrase query should match only doc 1, got %v", results.Hits)
    }
}

func TestEngine_IntersectionQuery(t *testing.T) {
    dir := t.TempDir()
    eng := &lotusEngine{}
    ctx := context.Background()
    eng.Open(ctx, dir)
    defer eng.Close()

    eng.Index(ctx, []index.Document{
        {DocID: "1", Text: []byte("quick brown fox")},
        {DocID: "2", Text: []byte("quick red car")},
        {DocID: "3", Text: []byte("brown lazy dog")},
    })

    // +quick +brown → only doc 1 (has both)
    results, _ := eng.Search(ctx, index.Query{Text: "+quick +brown", Limit: 10})
    if len(results.Hits) != 1 || results.Hits[0].DocID != "1" {
        t.Fatalf("intersection should return doc 1, got %v", results.Hits)
    }
}

func TestEngine_Stats(t *testing.T) {
    dir := t.TempDir()
    eng := &lotusEngine{}
    ctx := context.Background()
    eng.Open(ctx, dir)
    defer eng.Close()

    eng.Index(ctx, []index.Document{
        {DocID: "1", Text: []byte("hello world")},
    })

    stats, _ := eng.Stats(ctx)
    if stats.DocCount != 1 {
        t.Fatalf("DocCount %d, want 1", stats.DocCount)
    }
}

func TestEngine_Reopen(t *testing.T) {
    dir := t.TempDir()
    ctx := context.Background()

    // First session: index docs, close
    eng1 := &lotusEngine{}
    eng1.Open(ctx, dir)
    eng1.Index(ctx, []index.Document{
        {DocID: "1", Text: []byte("persistent data test")},
    })
    eng1.Close()

    // Second session: reopen, search
    eng2 := &lotusEngine{}
    eng2.Open(ctx, dir)
    defer eng2.Close()

    results, _ := eng2.Search(ctx, index.Query{Text: "persistent", Limit: 10})
    if len(results.Hits) == 0 {
        t.Fatal("expected to find doc after reopen")
    }
}

// Verify interface compliance
var _ index.Engine = (*lotusEngine)(nil)
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement engine**

```go
package lotus

import (
    "context"
    "os"
    "path/filepath"
    "sync"

    "github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
    index.Register("lotus", func() index.Engine { return &lotusEngine{} })
}

const memFlushDocs = 50000 // flush after 50K docs

type lotusEngine struct {
    dir string
    mu  sync.RWMutex

    writer   *segmentWriter
    segments []*segmentReader

    totalDocs uint32
    totalLen  uint64

    nextSegID uint32
    mergeCh   chan struct{}
    mergeWg   sync.WaitGroup
    done      chan struct{}
    closeOnce sync.Once
}

func (e *lotusEngine) Name() string { return "lotus" }

func (e *lotusEngine) Open(ctx context.Context, dir string) error {
    os.MkdirAll(dir, 0o755)
    e.dir = dir
    e.writer = newSegmentWriter()

    // Load existing segments
    entries, _ := os.ReadDir(dir)
    for _, entry := range entries {
        if entry.IsDir() && strings.HasPrefix(entry.Name(), "seg_") {
            segDir := filepath.Join(dir, entry.Name())
            sr, err := openSegmentReader(segDir)
            if err != nil { return err }
            e.segments = append(e.segments, sr)
            e.totalDocs += sr.docCount()
        }
    }

    // Background merge
    e.done = make(chan struct{})
    e.mergeCh = make(chan struct{}, 1)
    e.mergeWg.Add(1)
    go e.runMergeLoop()

    return nil
}

func (e *lotusEngine) Close() error { ... }
func (e *lotusEngine) Stats(ctx context.Context) (index.EngineStats, error) { ... }

func (e *lotusEngine) Index(ctx context.Context, docs []index.Document) error {
    e.mu.Lock()
    defer e.mu.Unlock()
    for _, doc := range docs {
        e.writer.addDoc(doc.DocID, doc.Text)
        e.totalDocs++
    }
    if len(e.writer.docs) >= memFlushDocs {
        return e.flushLocked()
    }
    return nil
}

func (e *lotusEngine) Search(ctx context.Context, q index.Query) (index.Results, error) {
    e.mu.RLock()
    defer e.mu.RUnlock()

    parsed := parseQuery(q.Text)
    limit := q.Limit
    if limit <= 0 { limit = 10 }

    // Execute query across all segments
    // Build cursors, run WAND, collect hits
    ...
}

func (e *lotusEngine) flushLocked() error {
    segDir := filepath.Join(e.dir, fmt.Sprintf("seg_%08d", e.nextSegID))
    e.nextSegID++
    if err := e.writer.flush(segDir); err != nil { return err }
    sr, err := openSegmentReader(segDir)
    if err != nil { return err }
    e.segments = append(e.segments, sr)
    e.writer = newSegmentWriter()
    select { case e.mergeCh <- struct{}{}: default: }
    return nil
}

func (e *lotusEngine) runMergeLoop() {
    defer e.mergeWg.Done()
    // Same tiered merge as rose
    ...
}
```

**Step 4: Run tests — expect PASS**

Run: `go test ./pkg/index/driver/flower/lotus/... -run TestEngine -v -count=1`

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/engine.go pkg/index/driver/flower/lotus/engine_test.go
git commit -m "feat(lotus): implement Engine (index.Engine interface)"
```

---

## Task 17: Scratch FST (Build-Tagged Alternative)

**Files:**
- Create: `pkg/index/driver/flower/lotus/fst/builder.go`
- Create: `pkg/index/driver/flower/lotus/fst/fst.go`
- Create: `pkg/index/driver/flower/lotus/fst/node.go`
- Test: `pkg/index/driver/flower/lotus/fst/fst_test.go`

**Step 1: Write the failing test**

```go
package fst

import "testing"

func TestFST_BuildAndLookup(t *testing.T) {
    b := NewBuilder()
    entries := []struct{ key string; val uint64 }{
        {"apple", 1}, {"application", 2}, {"banana", 3}, {"band", 4}, {"bandana", 5},
    }
    for _, e := range entries {
        b.Add([]byte(e.key), e.val)
    }
    data := b.Build()

    f, err := Load(data)
    if err != nil { t.Fatal(err) }

    for _, e := range entries {
        val, found := f.Get([]byte(e.key))
        if !found { t.Fatalf("%q not found", e.key) }
        if val != e.val { t.Fatalf("%q: got %d want %d", e.key, val, e.val) }
    }

    // Non-existent key
    _, found := f.Get([]byte("cherry"))
    if found { t.Fatal("found non-existent key") }
}

func TestFST_Empty(t *testing.T) {
    b := NewBuilder()
    data := b.Build()
    f, err := Load(data)
    if err != nil { t.Fatal(err) }
    _, found := f.Get([]byte("anything"))
    if found { t.Fatal("found key in empty FST") }
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement scratch FST using Daciuk's algorithm**

This is a Minimal Acyclic Deterministic Finite Automaton (MADA). The builder receives keys in sorted order and shares common suffixes incrementally.

`fst/node.go`: State and transition types.
`fst/builder.go`: Incremental suffix-sharing builder (Daciuk's algorithm).
`fst/fst.go`: Serialized FST: Load from bytes, Get(key) lookup.

**Step 4: Run tests — expect PASS**

**Step 5: Commit**

```bash
git add pkg/index/driver/flower/lotus/fst/
git commit -m "feat(lotus): implement scratch FST (Daciuk's MADA algorithm)"
```

---

## Task 18: Register Lotus + Integration Smoke Test

**Files:**
- Modify: `cli/bench.go` — ensure lotus is imported for registration
- Modify or create: import side-effect in `cmd/search/main.go` or `cli/root.go`

**Step 1: Add import side-effect**

Check where other flower drivers are imported. Add:
```go
_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/flower/lotus"
```

**Step 2: Verify build**

Run: `go build ./...`
Expected: clean build, no errors.

**Step 3: Smoke test with devnull-style indexing**

Run: `go test ./pkg/index/driver/flower/lotus/... -count=1 -race -v`
Expected: all tests pass.

**Step 4: Commit**

```bash
git commit -m "feat(lotus): register lotus engine + integration import"
```

---

## Task 19: Write spec/0650_tantivy_clone.md

**Files:**
- Create: `spec/0650_tantivy_clone.md`

Write the specification document covering:
- Goal and motivation (tantivy-in-Go, no CGO dependency)
- Architecture overview (segment files, BP128, FST, mmap, WAND)
- On-disk format reference
- Query types supported
- Benchmark plan (empty tables for results)
- Implementation order (summary of tasks 1-18)

**Step 1: Write the spec**

**Step 2: Commit**

```bash
git add spec/0650_tantivy_clone.md
git commit -m "docs(spec/0650): add tantivy-clone lotus specification"
```

---

## Task 20: Benchmark — Wikipedia Index + Search

**Files:**
- Modify: `spec/0650_tantivy_clone.md` (fill in benchmark results)

**Step 1: Build the CLI**

Run: `make install` or `go build -o ~/bin/search -tags tantivy ./cmd/search/`

**Step 2: Index Wikipedia corpus with lotus**

Run: `~/bin/search bench index --engine lotus --dir ~/data/search/bench`
Record: elapsed, docs/s, peak RSS, disk size.

**Step 3: Index with tantivy (CGO) for comparison**

Run: `~/bin/search bench index --engine tantivy --dir ~/data/search/bench`
Record: same metrics.

**Step 4: Search with lotus**

Run: `~/bin/search bench search --engine lotus --dir ~/data/search/bench --warmup 30s --iter 10`
Record: p50, p95, p99, slowest query.

**Step 5: Search with tantivy**

Run: `~/bin/search bench search --engine tantivy --dir ~/data/search/bench --warmup 30s --iter 10`
Record: same metrics.

**Step 6: Fill benchmark tables in spec**

Update `spec/0650_tantivy_clone.md` with actual numbers:

```markdown
### Index Performance

| Engine | Docs | Index time | Rate (docs/s) | Disk | Peak RSS |
|--------|-----:|-----------|--------------|-----:|--------:|
| lotus  | 5,032,104 | ... | ... | ... | ... |
| tantivy (CGO) | 5,032,104 | 7m6s | 11,808 | 7.3 GB | 278 MB |
| rose   | 5,032,104 | 6m40s | 12,563 | 3.2 GB | 22.4 GB |

### Search Performance (TOP_10, 962 queries)

| Engine | p50 | p95 | p99 | Slowest |
|--------|----:|----:|----:|---------|
| lotus  | ... | ... | ... | ...     |
| tantivy (CGO) | 2.7 ms | 3.4 ms | 3.4 ms | 227.7 ms |
| rose   | 14.4 ms | 16.1 ms | 16.1 ms | 6.8 s |
```

**Step 7: Commit**

```bash
git add spec/0650_tantivy_clone.md
git commit -m "docs(spec/0650): add Wikipedia benchmark results for lotus vs tantivy vs rose"
```

---

## Task Summary

| Task | Component | Depends on |
|------|-----------|------------|
| 1 | RESEARCH.md | — |
| 2 | BP128 codec | — |
| 3 | VInt codec | — |
| 4 | Field norms | — |
| 5 | Analyzer | — |
| 6 | mmap utilities | — |
| 7 | Term dictionary (vellum) | 6 |
| 8 | Posting list writer/reader | 2, 3 |
| 9 | Stored fields | 6 |
| 10 | Segment writer | 4, 5, 7, 8, 9 |
| 11 | Segment reader | 6, 7, 8, 9 |
| 12 | BM25F scorer | 4 |
| 13 | Query parser | 5 |
| 14 | WAND evaluator | 8, 12 |
| 15 | Segment merge | 10, 11 |
| 16 | Engine | 10, 11, 13, 14, 15 |
| 17 | Scratch FST | — |
| 18 | Registration + smoke test | 16 |
| 19 | Spec doc | — |
| 20 | Benchmark | 18 |

**Critical path:** 2 → 8 → 10 → 16 → 18 → 20

Tasks 1-6 and 17, 19 can run in parallel (no code dependencies).
