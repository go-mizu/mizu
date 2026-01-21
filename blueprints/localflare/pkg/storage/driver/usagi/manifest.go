package usagi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const manifestFileName = "manifest.usagi"

const manifestVersion = 2

type manifest struct {
	Version          int                      `json:"version"`
	Bucket           string                   `json:"bucket"`
	CreatedAt        time.Time                `json:"created_at"`
	SegmentSizeBytes int64                    `json:"segment_size_bytes"`
	LastSegments     []manifestSegment        `json:"last_segments"`
	Index            map[string]manifestEntry `json:"index"`
}

type manifestSegment struct {
	Shard int   `json:"shard"`
	ID    int64 `json:"id"`
	Size  int64 `json:"size"`
}

type manifestEntry struct {
	Shard       int    `json:"shard"`
	SegmentID   int64  `json:"segment_id"`
	Offset      int64  `json:"offset"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	UpdatedUnix int64  `json:"updated_unix_ns"`
	Checksum    uint32 `json:"checksum"`
}

type manifestV1 struct {
	Version          int                        `json:"version"`
	Bucket           string                     `json:"bucket"`
	CreatedAt        time.Time                  `json:"created_at"`
	LastSegmentID    int64                      `json:"last_segment_id"`
	LastSegmentSize  int64                      `json:"last_segment_size"`
	SegmentSizeBytes int64                      `json:"segment_size_bytes"`
	Index            map[string]manifestEntryV1 `json:"index"`
}

type manifestEntryV1 struct {
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
	var probe struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("usagi: decode manifest: %w", err)
	}
	switch probe.Version {
	case 1:
		var m1 manifestV1
		if err := json.Unmarshal(data, &m1); err != nil {
			return nil, fmt.Errorf("usagi: decode manifest: %w", err)
		}
		m := manifest{
			Version:          manifestVersion,
			Bucket:           m1.Bucket,
			CreatedAt:        m1.CreatedAt,
			SegmentSizeBytes: m1.SegmentSizeBytes,
			LastSegments: []manifestSegment{
				{Shard: 0, ID: m1.LastSegmentID, Size: m1.LastSegmentSize},
			},
			Index: make(map[string]manifestEntry, len(m1.Index)),
		}
		for k, v := range m1.Index {
			m.Index[k] = manifestEntry{
				Shard:       0,
				SegmentID:   v.SegmentID,
				Offset:      v.Offset,
				Size:        v.Size,
				ContentType: v.ContentType,
				UpdatedUnix: v.UpdatedUnix,
				Checksum:    v.Checksum,
			}
		}
		return &m, nil
	case manifestVersion:
		var m manifest
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("usagi: decode manifest: %w", err)
		}
		return &m, nil
	default:
		return nil, fmt.Errorf("usagi: manifest version mismatch")
	}
}

func (b *bucket) writeManifest() error {
	entries := make(map[string]manifestEntry)
	for k, v := range b.index.Snapshot() {
		entries[k] = manifestEntry{
			Shard:       v.shard,
			SegmentID:   v.segmentID,
			Offset:      v.offset,
			Size:        v.size,
			ContentType: v.contentType,
			UpdatedUnix: v.updated.UnixNano(),
			Checksum:    v.checksum,
		}
	}
	lastSegments := make([]manifestSegment, 0, len(b.writers))
	for _, w := range b.writers {
		if w == nil {
			continue
		}
		w.mu.Lock()
		if w.id > 0 {
			lastSegments = append(lastSegments, manifestSegment{
				Shard: w.shard,
				ID:    w.id,
				Size:  w.size,
			})
		}
		w.mu.Unlock()
	}
	m := manifest{
		Version:          manifestVersion,
		Bucket:           b.name,
		CreatedAt:        time.Now(),
		SegmentSizeBytes: b.store.segmentSize,
		LastSegments:     lastSegments,
		Index:            entries,
	}
	data, err := json.MarshalIndent(&m, "", "  ")
	if err != nil {
		return fmt.Errorf("usagi: encode manifest: %w", err)
	}
	path := b.manifestPath()
	return os.WriteFile(path, data, 0o644)
}
