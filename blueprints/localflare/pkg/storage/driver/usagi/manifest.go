package usagi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const manifestFileName = "manifest.usagi"

const manifestVersion = 1

type manifest struct {
	Version          int                      `json:"version"`
	Bucket           string                   `json:"bucket"`
	CreatedAt        time.Time                `json:"created_at"`
	LastSegmentID    int64                    `json:"last_segment_id"`
	LastSegmentSize  int64                    `json:"last_segment_size"`
	SegmentSizeBytes int64                    `json:"segment_size_bytes"`
	Index            map[string]manifestEntry `json:"index"`
}

type manifestEntry struct {
	SegmentID   int64  `json:"segment_id"`
	Offset      int64  `json:"offset"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	UpdatedUnix int64  `json:"updated_unix_ns"`
	Checksum    uint32 `json:"checksum"`
}

func (b *bucket) manifestPath() string {
	return filepath.Join(b.dir, manifestFileName)
}

func (b *bucket) loadManifest() (*manifest, error) {
	path := b.manifestPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("usagi: decode manifest: %w", err)
	}
	if m.Version != manifestVersion {
		return nil, fmt.Errorf("usagi: manifest version mismatch")
	}
	return &m, nil
}

func (b *bucket) writeManifest() error {
	entries := make(map[string]manifestEntry)
	for k, v := range b.index.Snapshot() {
		entries[k] = manifestEntry{
			SegmentID:   v.segmentID,
			Offset:      v.offset,
			Size:        v.size,
			ContentType: v.contentType,
			UpdatedUnix: v.updated.UnixNano(),
			Checksum:    v.checksum,
		}
	}
	m := manifest{
		Version:          manifestVersion,
		Bucket:           b.name,
		CreatedAt:        time.Now(),
		LastSegmentID:    b.currentSegmentID,
		LastSegmentSize:  b.currentSegmentSize,
		SegmentSizeBytes: b.store.segmentSize,
		Index:            entries,
	}
	data, err := json.MarshalIndent(&m, "", "  ")
	if err != nil {
		return fmt.Errorf("usagi: encode manifest: %w", err)
	}
	path := b.manifestPath()
	return os.WriteFile(path, data, 0o644)
}
