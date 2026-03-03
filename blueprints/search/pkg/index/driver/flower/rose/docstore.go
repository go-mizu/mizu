package rose

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// docStoreMaxText is the maximum number of bytes of document text stored per entry.
const docStoreMaxText = 512

// docEntry holds an in-memory representation of one stored document.
type docEntry struct {
	externalID string
	text       []byte
}

// docStore is an append-only binary document store backed by a single file.
// Entries are stored in insertion order; the 0-based position is the internal docIdx.
type docStore struct {
	f       *os.File
	entries []docEntry // in-memory index; entry[i] is internal docIdx i
}

// openDocStore opens or creates the file at path and scans all existing entries
// into memory so that get() is O(1).
func openDocStore(path string) (*docStore, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("docstore open %q: %w", path, err)
	}

	ds := &docStore{f: f}
	if err := ds.load(); err != nil {
		f.Close()
		return nil, fmt.Errorf("docstore load %q: %w", path, err)
	}
	return ds, nil
}

// load scans the file from the beginning and populates ds.entries.
// A partial record at the end of the file (e.g. from a crash mid-write) is
// silently discarded; the caller receives no signal about the truncation.
// This is intentional for an append-only store that may be interrupted.
func (ds *docStore) load() error {
	if _, err := ds.f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	var lenBuf [4]byte
	for {
		// Read ExternalIDLen.
		if _, err := io.ReadFull(ds.f, lenBuf[:]); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return fmt.Errorf("reading ExternalIDLen: %w", err)
		}
		extIDLen := binary.LittleEndian.Uint32(lenBuf[:])

		// Read ExternalID bytes.
		extIDBuf := make([]byte, extIDLen)
		if _, err := io.ReadFull(ds.f, extIDBuf); err != nil {
			return fmt.Errorf("reading ExternalID: %w", err)
		}

		// Read TextLen.
		if _, err := io.ReadFull(ds.f, lenBuf[:]); err != nil {
			return fmt.Errorf("reading TextLen: %w", err)
		}
		textLen := binary.LittleEndian.Uint32(lenBuf[:])

		// Read Text bytes.
		textBuf := make([]byte, textLen)
		if textLen > 0 {
			if _, err := io.ReadFull(ds.f, textBuf); err != nil {
				return fmt.Errorf("reading Text: %w", err)
			}
		}

		ds.entries = append(ds.entries, docEntry{
			externalID: string(extIDBuf),
			text:       textBuf,
		})
	}
	return nil
}

// append writes one document to the file and adds it to the in-memory index.
// Returns the 0-based internal docIdx assigned to this document.
func (ds *docStore) append(externalID string, text []byte) (uint32, error) {
	// Truncate text to docStoreMaxText bytes, then walk back to the last
	// rune-start boundary so we never store a partial multi-byte sequence.
	// utf8.RuneStart is O(1) per byte, so the walk-back is O(1) overall
	// (at most 3 continuation bytes before a rune-start) even for binary input.
	if len(text) > docStoreMaxText {
		text = text[:docStoreMaxText]
		for len(text) > 0 && !utf8.RuneStart(text[len(text)-1]) {
			text = text[:len(text)-1]
		}
		// If the final rune-start byte heads a sequence longer than our
		// remaining window (split rune), drop that rune too.
		if len(text) > 0 && !utf8.Valid(text) {
			_, size := utf8.DecodeLastRune(text)
			text = text[:len(text)-size]
		}
	}

	extIDBytes := []byte(externalID)
	extIDLen := uint32(len(extIDBytes))
	textLen := uint32(len(text))

	// Seek to end before writing.
	if _, err := ds.f.Seek(0, io.SeekEnd); err != nil {
		return 0, fmt.Errorf("docstore seek: %w", err)
	}

	var lenBuf [4]byte

	// Write ExternalIDLen.
	binary.LittleEndian.PutUint32(lenBuf[:], extIDLen)
	if _, err := ds.f.Write(lenBuf[:]); err != nil {
		return 0, fmt.Errorf("docstore write ExternalIDLen: %w", err)
	}

	// Write ExternalID.
	if len(extIDBytes) > 0 {
		if _, err := ds.f.Write(extIDBytes); err != nil {
			return 0, fmt.Errorf("docstore write ExternalID: %w", err)
		}
	}

	// Write TextLen.
	binary.LittleEndian.PutUint32(lenBuf[:], textLen)
	if _, err := ds.f.Write(lenBuf[:]); err != nil {
		return 0, fmt.Errorf("docstore write TextLen: %w", err)
	}

	// Write Text.
	if textLen > 0 {
		if _, err := ds.f.Write(text); err != nil {
			return 0, fmt.Errorf("docstore write Text: %w", err)
		}
	}

	idx := uint32(len(ds.entries))
	// Store a copy of text to avoid aliasing if caller mutates the slice.
	stored := make([]byte, len(text))
	copy(stored, text)
	ds.entries = append(ds.entries, docEntry{
		externalID: externalID,
		text:       stored,
	})
	return idx, nil
}

// get returns the docEntry for the given internal docIdx in O(1).
// Returns an error if idx is out of range.
func (ds *docStore) get(idx uint32) (docEntry, error) {
	if int(idx) >= len(ds.entries) {
		return docEntry{}, fmt.Errorf("docstore get: index %d out of range (len=%d)", idx, len(ds.entries))
	}
	return ds.entries[idx], nil
}

// close flushes and closes the underlying file.
// Calling close on a nil docStore or after already closing is safe (no-op / no panic).
func (ds *docStore) close() error {
	if ds == nil || ds.f == nil {
		return nil
	}
	err := ds.f.Close()
	ds.f = nil
	return err
}

// ---------------------------------------------------------------------------
// Snippet generation
// ---------------------------------------------------------------------------

// snippetFor finds the first occurrence of any query term (pre-stemmed) in text
// and returns a window of approximately ±20 words around it.
// If no query term is found, the first 20 words are returned.
// queryTerms must already be in stemmed form (as stored in the inverted index).
func snippetFor(text []byte, queryTerms []string) string {
	if len(text) == 0 {
		return ""
	}

	words := strings.Fields(string(text))
	if len(words) == 0 {
		return ""
	}

	// Build a set of query terms for O(1) lookup.
	termSet := make(map[string]struct{}, len(queryTerms))
	for _, qt := range queryTerms {
		termSet[qt] = struct{}{}
	}

	// Find first word whose stem matches a query term.
	hitIdx := -1
	for i, w := range words {
		stemmed := processTok(w)
		if stemmed == "" {
			continue
		}
		if _, ok := termSet[stemmed]; ok {
			hitIdx = i
			break
		}
	}

	var start, end int
	if hitIdx >= 0 {
		start = hitIdx - 10
		if start < 0 {
			start = 0
		}
		end = hitIdx + 20
		if end > len(words) {
			end = len(words)
		}
	} else {
		// No match: return first 20 words.
		start = 0
		end = 20
		if end > len(words) {
			end = len(words)
		}
	}

	return strings.Join(words[start:end], " ")
}
