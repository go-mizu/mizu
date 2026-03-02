package bodystore

import (
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Store is a content-addressable body store backed by the filesystem.
// Bodies are stored gzip-compressed at {dir}/{sha[0:2]}/{sha[2:4]}/{sha[4:]}.gz.
// The content ID (CID) format is "sha256:{hex64}".
//
// Concurrent Put calls with identical content are safe on POSIX systems:
// each write goes to a unique .tmp file and os.Rename is atomic, so the last
// rename wins and the result is always a valid file.
type Store struct{ dir string }

// Open returns a Store backed by dir, creating it if needed.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("bodystore: mkdir %s: %w", dir, err)
	}
	return &Store{dir: dir}, nil
}

// Put writes body to the store if not already present and returns its CID.
// It is safe to call Put with the same content multiple times (idempotent).
func (s *Store) Put(body []byte) (string, error) {
	sum := sha256.Sum256(body)
	hex := fmt.Sprintf("%x", sum[:])
	cid := "sha256:" + hex

	path := s.cidToPath(hex)
	if _, err := os.Stat(path); err == nil {
		return cid, nil // already exists
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("bodystore: mkdir: %w", err)
	}

	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", fmt.Errorf("bodystore: create: %w", err)
	}
	gz := gzip.NewWriter(f)
	if _, err := gz.Write(body); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", fmt.Errorf("bodystore: gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", fmt.Errorf("bodystore: gzip close: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("bodystore: close: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("bodystore: rename: %w", err)
	}
	return cid, nil
}

// Get decompresses and returns the body for the given CID.
func (s *Store) Get(cid string) ([]byte, error) {
	hex, ok := cidHex(cid)
	if !ok {
		return nil, fmt.Errorf("bodystore: invalid cid %q", cid)
	}
	f, err := os.Open(s.cidToPath(hex))
	if err != nil {
		return nil, fmt.Errorf("bodystore: open: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("bodystore: gzip open: %w", err)
	}
	defer gz.Close()
	b, err := io.ReadAll(gz)
	if err != nil {
		return nil, fmt.Errorf("bodystore: read: %w", err)
	}
	return b, nil
}

// Has reports whether the CID exists in the store.
func (s *Store) Has(cid string) bool {
	hex, ok := cidHex(cid)
	if !ok {
		return false
	}
	_, err := os.Stat(s.cidToPath(hex))
	return err == nil
}

// Path returns the filesystem path for a CID (does not check existence).
func (s *Store) Path(cid string) string {
	hex, ok := cidHex(cid)
	if !ok {
		return ""
	}
	return s.cidToPath(hex)
}

func (s *Store) cidToPath(hex string) string {
	// hex is always 64 chars (SHA-256)
	return filepath.Join(s.dir, hex[0:2], hex[2:4], hex[4:]+".gz")
}

func cidHex(cid string) (string, bool) {
	const prefix = "sha256:"
	if len(cid) != len(prefix)+64 {
		return "", false
	}
	if cid[:len(prefix)] != prefix {
		return "", false
	}
	return cid[len(prefix):], true
}
