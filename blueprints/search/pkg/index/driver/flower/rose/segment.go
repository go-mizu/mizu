package rose

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
)

// ---------------------------------------------------------------------------
// Segment format constants
// ---------------------------------------------------------------------------

const (
	segMagic   = uint32(0x524F5345) // "ROSE" in little-endian bytes: 0x52='R', 0x4F='O', 0x53='S', 0x45='E'
	segVersion = uint8(0x01)
)

// segMagicBytes are the four magic bytes as written on disk (little-endian uint32).
// 0x524F5345 LE → bytes [0x45, 0x53, 0x4F, 0x52] → but the spec says [0x52,0x4F,0x53,0x45]
// which matches big-endian layout of the constant. We write it as 4 raw bytes directly.
var segMagicBytes = [4]byte{0x52, 0x4F, 0x53, 0x45}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// memPosting is one posting in the in-memory buffer.
type memPosting struct {
	docID  uint32
	impact uint8 // quantised BM25+ score (set by flushSegment, not during indexing)
}

// termEntry is the in-memory view of one term's dictionary entry, used when reading.
type termEntry struct {
	term          string
	df            uint32
	postingOffset uint32
	numBlocks     uint32
}

// ---------------------------------------------------------------------------
// flushSegment
// ---------------------------------------------------------------------------

// flushSegment writes an immutable segment file to path.
//
// mem maps each term to its sorted (ascending docID) slice of memPosting
// values. The impact field in the incoming postings is ignored; flushSegment
// recomputes quantised BM25+ scores for every posting.
//
// docCount is the number of documents represented in this segment.
// avgDocLen is the average document length (tokens) across those documents.
//
// Binary layout
//
//	[4]  magic bytes 0x52 0x4F 0x53 0x45
//	[1]  version 0x01
//	[4]  docCount (uint32 LE)
//	[4]  avgDocLen (uint32 LE)
//	[4]  dictSize (uint32 LE)
//	[4]  postingBase byte-offset (uint32 LE)
//
//	For each term (lexicographic order):
//	  [4]  termLen (uint32 LE)
//	  [termLen] term UTF-8 bytes
//	  [4]  df (uint32 LE)
//	  [4]  postingOffset (uint32 LE) — relative to postingBase
//	  [4]  numBlocks (uint32 LE)
//
//	Posting region (starts at postingBase):
//	  For each block of up to 128 postings:
//	    [4]  blockBase uint32 LE — first absolute docID in the block
//	    [1]  BlockMaxImpact uint8
//	    [1]  n uint8 — actual count 1..128
//	    [variable] VByte-encoded deltas (docID[i] - blockBase)
//	    [n]  uint8 impact scores
func flushSegment(path string, mem map[string][]memPosting, docCount, avgDocLen uint32) error {
	// -----------------------------------------------------------------------
	// 1. Sort terms lexicographically.
	// -----------------------------------------------------------------------
	terms := make([]string, 0, len(mem))
	for t := range mem {
		terms = append(terms, t)
	}
	sort.Strings(terms)

	// -----------------------------------------------------------------------
	// 2. Encode all posting lists into a single byte buffer, recording the
	//    per-term metadata (df, offset, numBlocks) as we go.
	// -----------------------------------------------------------------------
	type termMeta struct {
		term          string
		df            uint32
		postingOffset uint32
		numBlocks     uint32
	}

	var postingBuf []byte
	metas := make([]termMeta, 0, len(terms))

	for _, term := range terms {
		postings := mem[term]
		df := uint32(len(postings))
		if df == 0 {
			continue
		}

		// Compute raw BM25+ scores for every posting in this term's list.
		// The term-frequency (tf) for each posting is 1 (each posting represents
		// one occurrence per doc; callers that count tf should store multiple
		// postings — but the design uses one posting per (term, doc) pair with
		// the raw tf encoded separately).
		//
		// However, looking at the memPosting struct, there is no tf field —
		// only docID and impact. The caller is expected to pre-compute tf somehow.
		// Since this is a flush function and the spec says "recompute quantised
		// BM25+ scores", we use tf=1 for every posting as the baseline (the
		// caller controls what gets into the map; here we just score each doc
		// as appearing once with this term).
		//
		// A more complete engine would store tf alongside docID, but the spec
		// only gives us impact (uint8) and docID (uint32).  We use tf=1 and the
		// BM25+ scores become IDF-weighted only (with length normalisation from
		// avgDocLen).  This is correct for binary occurrence posting lists.
		scores := make([]float64, df)
		for i := range postings {
			// dl (document length) is unknown per-posting; use avgDocLen as
			// a uniform approximation (produces IDF-only BM25+ weighting).
			scores[i] = bm25Plus(1, df, avgDocLen, avgDocLen, docCount)
		}
		impacts := quantise(scores)

		// Pack into 128-posting blocks.
		offset := uint32(len(postingBuf))
		numBlocks := uint32(0)

		for start := 0; start < int(df); start += blockSize {
			end := start + blockSize
			if end > int(df) {
				end = int(df)
			}
			n := end - start

			// Collect docIDs and impacts for this block.
			blockDocIDs := make([]uint32, n)
			blockImpacts := make([]uint8, n)
			for i := 0; i < n; i++ {
				blockDocIDs[i] = postings[start+i].docID
				blockImpacts[i] = impacts[start+i]
			}

			blockBase := blockDocIDs[0]
			blockData, bmi := packBlock(blockDocIDs, blockImpacts, blockBase)

			// Write: [4] blockBase, [1] BlockMaxImpact, [1] n, [variable] data
			var hdr [6]byte
			binary.LittleEndian.PutUint32(hdr[0:4], blockBase)
			hdr[4] = bmi
			hdr[5] = uint8(n)
			postingBuf = append(postingBuf, hdr[:]...)
			postingBuf = append(postingBuf, blockData...)

			numBlocks++
		}

		metas = append(metas, termMeta{
			term:          term,
			df:            df,
			postingOffset: offset,
			numBlocks:     numBlocks,
		})
	}

	// -----------------------------------------------------------------------
	// 3. Compute postingBase: fixed header (17 bytes) + dict entries.
	// -----------------------------------------------------------------------
	// Header: 4 (magic) + 1 (version) + 4 (docCount) + 4 (avgDocLen) +
	//         4 (dictSize) + 4 (postingBase field) = 21 bytes.
	headerSize := uint32(21)

	dictSize := uint32(len(metas))
	dictBytes := uint32(0)
	for _, m := range metas {
		// 4 (termLen) + len(term) + 4 (df) + 4 (postingOffset) + 4 (numBlocks)
		dictBytes += 4 + uint32(len(m.term)) + 4 + 4 + 4
	}

	postingBase := headerSize + dictBytes

	// -----------------------------------------------------------------------
	// 4. Write the file.
	// -----------------------------------------------------------------------
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("flushSegment create %q: %w", path, err)
	}

	w := bufio.NewWriter(f)

	writeErr := func(e error) error {
		f.Close()
		return fmt.Errorf("flushSegment write %q: %w", path, e)
	}

	// Magic bytes (4 raw bytes, NOT as uint32 LE — spec says [0x52,0x4F,0x53,0x45]).
	if _, err := w.Write(segMagicBytes[:]); err != nil {
		return writeErr(err)
	}

	// Version.
	if err := w.WriteByte(segVersion); err != nil {
		return writeErr(err)
	}

	// docCount, avgDocLen, dictSize, postingBase.
	var u32buf [4]byte
	writeU32 := func(v uint32) error {
		binary.LittleEndian.PutUint32(u32buf[:], v)
		_, e := w.Write(u32buf[:])
		return e
	}

	if err := writeU32(docCount); err != nil {
		return writeErr(err)
	}
	if err := writeU32(avgDocLen); err != nil {
		return writeErr(err)
	}
	if err := writeU32(dictSize); err != nil {
		return writeErr(err)
	}
	if err := writeU32(postingBase); err != nil {
		return writeErr(err)
	}

	// Dictionary entries.
	for _, m := range metas {
		termBytes := []byte(m.term)
		if err := writeU32(uint32(len(termBytes))); err != nil {
			return writeErr(err)
		}
		if _, err := w.Write(termBytes); err != nil {
			return writeErr(err)
		}
		if err := writeU32(m.df); err != nil {
			return writeErr(err)
		}
		if err := writeU32(m.postingOffset); err != nil {
			return writeErr(err)
		}
		if err := writeU32(m.numBlocks); err != nil {
			return writeErr(err)
		}
	}

	// Posting region.
	if _, err := w.Write(postingBuf); err != nil {
		return writeErr(err)
	}

	if err := w.Flush(); err != nil {
		f.Close()
		return fmt.Errorf("flushSegment flush %q: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("flushSegment close %q: %w", path, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// openSegment
// ---------------------------------------------------------------------------

// openSegment reads the segment file at path and returns the term dictionary,
// the raw posting-region bytes, docCount, and avgDocLen.
//
// postingData is loaded entirely into memory; segments are immutable and
// accessed randomly by readPostings.
//
// Returns a descriptive error on magic/version mismatch or I/O failure.
func openSegment(path string) ([]termEntry, []byte, uint32, uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("openSegment open %q: %w", path, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("openSegment read %q: %w", path, err)
	}

	pos := 0
	need := func(n int) error {
		if pos+n > len(data) {
			return fmt.Errorf("openSegment %q: truncated at offset %d (need %d more bytes)", path, pos, n)
		}
		return nil
	}

	// Magic (4 bytes).
	if err := need(4); err != nil {
		return nil, nil, 0, 0, err
	}
	if data[0] != segMagicBytes[0] || data[1] != segMagicBytes[1] ||
		data[2] != segMagicBytes[2] || data[3] != segMagicBytes[3] {
		return nil, nil, 0, 0, fmt.Errorf("openSegment %q: bad magic bytes [%02x %02x %02x %02x], want [52 4f 53 45]",
			path, data[0], data[1], data[2], data[3])
	}
	pos += 4

	// Version (1 byte).
	if err := need(1); err != nil {
		return nil, nil, 0, 0, err
	}
	if data[pos] != segVersion {
		return nil, nil, 0, 0, fmt.Errorf("openSegment %q: unsupported version %d, want %d", path, data[pos], segVersion)
	}
	pos++

	readU32 := func() (uint32, error) {
		if err := need(4); err != nil {
			return 0, err
		}
		v := binary.LittleEndian.Uint32(data[pos : pos+4])
		pos += 4
		return v, nil
	}

	docCount, err := readU32()
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("openSegment %q: reading docCount: %w", path, err)
	}
	avgDocLen, err := readU32()
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("openSegment %q: reading avgDocLen: %w", path, err)
	}
	dictSize, err := readU32()
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("openSegment %q: reading dictSize: %w", path, err)
	}
	postingBase, err := readU32()
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("openSegment %q: reading postingBase: %w", path, err)
	}

	// Dictionary entries.
	dict := make([]termEntry, 0, dictSize)
	for i := uint32(0); i < dictSize; i++ {
		termLen, err := readU32()
		if err != nil {
			return nil, nil, 0, 0, fmt.Errorf("openSegment %q: term %d termLen: %w", path, i, err)
		}
		if err := need(int(termLen)); err != nil {
			return nil, nil, 0, 0, fmt.Errorf("openSegment %q: term %d bytes: %w", path, i, err)
		}
		termStr := string(data[pos : pos+int(termLen)])
		pos += int(termLen)

		df, err := readU32()
		if err != nil {
			return nil, nil, 0, 0, fmt.Errorf("openSegment %q: term %d df: %w", path, i, err)
		}
		postingOffset, err := readU32()
		if err != nil {
			return nil, nil, 0, 0, fmt.Errorf("openSegment %q: term %d postingOffset: %w", path, i, err)
		}
		numBlocks, err := readU32()
		if err != nil {
			return nil, nil, 0, 0, fmt.Errorf("openSegment %q: term %d numBlocks: %w", path, i, err)
		}

		dict = append(dict, termEntry{
			term:          termStr,
			df:            df,
			postingOffset: postingOffset,
			numBlocks:     numBlocks,
		})
	}

	// Posting region: everything from postingBase to end of file.
	if int(postingBase) > len(data) {
		return nil, nil, 0, 0, fmt.Errorf("openSegment %q: postingBase %d exceeds file size %d", path, postingBase, len(data))
	}
	postingData := make([]byte, len(data)-int(postingBase))
	copy(postingData, data[postingBase:])

	return dict, postingData, docCount, avgDocLen, nil
}

// ---------------------------------------------------------------------------
// readPostings
// ---------------------------------------------------------------------------

// readPostings decodes all posting blocks for te from the raw posting region.
//
// Block layout (each block):
//
//	[4]  blockBase uint32 LE
//	[1]  BlockMaxImpact uint8
//	[1]  n uint8 (1..128)
//	[variable] VByte deltas (docID[i] - blockBase)
//	[n]  uint8 impacts
//
// Returns (docIDs, impacts, error).
func readPostings(postingData []byte, te termEntry) ([]uint32, []uint8, error) {
	if te.numBlocks == 0 {
		return nil, nil, nil
	}

	var allDocIDs []uint32
	var allImpacts []uint8

	pos := int(te.postingOffset)

	for b := uint32(0); b < te.numBlocks; b++ {
		// Read blockBase (4 bytes).
		if pos+4 > len(postingData) {
			return nil, nil, fmt.Errorf("readPostings: block %d: truncated reading blockBase (offset=%d, len=%d)", b, pos, len(postingData))
		}
		blockBase := binary.LittleEndian.Uint32(postingData[pos : pos+4])
		pos += 4

		// Read BlockMaxImpact (1 byte).
		if pos >= len(postingData) {
			return nil, nil, fmt.Errorf("readPostings: block %d: truncated reading BlockMaxImpact", b)
		}
		pos++ // skip bmi, not needed for decoding

		// Read n (1 byte).
		if pos >= len(postingData) {
			return nil, nil, fmt.Errorf("readPostings: block %d: truncated reading n", b)
		}
		n := int(postingData[pos])
		pos++

		if n == 0 {
			return nil, nil, fmt.Errorf("readPostings: block %d: n=0 is invalid", b)
		}

		// Decode VByte deltas + impact bytes.
		// We need to know how many bytes the VByte section occupies.
		// unpackBlock reads from data starting at pos=0, so we pass a sub-slice.
		blockDocIDs, blockImpacts, err := unpackBlock(postingData[pos:], blockBase, n)
		if err != nil {
			return nil, nil, fmt.Errorf("readPostings: block %d: %w", b, err)
		}

		// Advance pos past the block data.
		// VByte section length: we need to re-encode to find the byte count,
		// or we can decode manually to track position.
		// Simplest: count VByte bytes by decoding individually.
		deltaEnd := pos
		for i := 0; i < n; i++ {
			_, newDeltaEnd, err := vbyteDecode(postingData, deltaEnd)
			if err != nil {
				return nil, nil, fmt.Errorf("readPostings: block %d delta %d advance: %w", b, i, err)
			}
			deltaEnd = newDeltaEnd
		}
		// After VByte section, n impact bytes follow.
		pos = deltaEnd + n

		allDocIDs = append(allDocIDs, blockDocIDs...)
		allImpacts = append(allImpacts, blockImpacts...)
	}

	return allDocIDs, allImpacts, nil
}
