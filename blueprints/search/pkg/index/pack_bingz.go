package index

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"sync"

	kgzip "github.com/klauspost/compress/gzip"
)

// binGzDocsPerMember is the maximum number of documents per gzip member.
const binGzDocsPerMember = 1000

// memberEntry records the byte offset and document count for a single gzip member.
type memberEntry struct {
	offset   uint64
	docCount uint32
}

// countingWriter wraps an io.Writer and tracks the total number of bytes written.
type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

// PackFlatBinGz packs all markdown files from markdownDir into packPath as
// concatenated gzip members. Each member holds up to binGzDocsPerMember (1000)
// documents compressed with BestCompression.
//
// Wire format per member (raw flatbin stream inside each gzip member):
//
//	repeated per doc:
//	  uint16 LE  id_len
//	  []byte     id  (id_len bytes)
//	  uint32 LE  txt_len
//	  []byte     text (txt_len bytes)
//
// After writing packPath, a companion index file (packPath+".idx") is written
// by writeBinGzIdx to enable parallel random access by member.
func PackFlatBinGz(ctx context.Context, markdownDir, packPath string, workers, batchSize int, progress ProgressFunc) (*PipelineStats, error) {
	if err := os.MkdirAll(filepath.Dir(packPath), 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(packPath)
	if err != nil {
		return nil, err
	}

	cw := &countingWriter{w: bufio.NewWriterSize(f, 1<<20)} // 1 MB write buffer

	var (
		members []memberEntry
		mu      sync.Mutex // protects members + pending between batch calls

		// pending accumulates docs for the current in-progress member.
		pending []Document
	)

	// flushMember writes all docs in pending as one gzip member to cw.
	// Caller must hold mu (or call from the single-threaded indexFn path).
	flushMember := func() error {
		if len(pending) == 0 {
			return nil
		}
		offset := uint64(cw.n)

		gz, err := kgzip.NewWriterLevel(cw, kgzip.BestCompression)
		if err != nil {
			return err
		}

		var hdr [6]byte
		for _, doc := range pending {
			id := doc.DocID
			if len(id) > 65535 {
				id = id[:65535]
			}
			binary.LittleEndian.PutUint16(hdr[0:2], uint16(len(id)))
			binary.LittleEndian.PutUint32(hdr[2:6], uint32(len(doc.Text)))
			if _, err := gz.Write(hdr[:2]); err != nil {
				return err
			}
			if _, err := io.WriteString(gz, id); err != nil {
				return err
			}
			if _, err := gz.Write(hdr[2:6]); err != nil {
				return err
			}
			if _, err := gz.Write(doc.Text); err != nil {
				return err
			}
		}

		if err := gz.Close(); err != nil {
			return err
		}

		members = append(members, memberEntry{
			offset:   offset,
			docCount: uint32(len(pending)),
		})
		pending = pending[:0]
		return nil
	}

	eng := &funcEngine{
		name: "bingz-writer",
		indexFn: func(_ context.Context, docs []Document) error {
			mu.Lock()
			defer mu.Unlock()
			for _, doc := range docs {
				pending = append(pending, doc)
				if len(pending) >= binGzDocsPerMember {
					if err := flushMember(); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}

	stats, pipeErr := RunPipeline(ctx, eng, PipelineConfig{
		SourceDir: markdownDir,
		BatchSize: batchSize,
		Workers:   workers,
	}, progress)

	if pipeErr == nil {
		// Flush any remaining docs as the final (partial) member.
		mu.Lock()
		pipeErr = flushMember()
		mu.Unlock()
	}

	// Flush the bufio writer wrapping cw.
	var flushErr error
	if bw, ok := cw.w.(*bufio.Writer); ok {
		flushErr = bw.Flush()
	}
	closeErr := f.Close()

	if pipeErr != nil {
		os.Remove(packPath)
		return stats, pipeErr
	}
	if flushErr != nil {
		os.Remove(packPath)
		return stats, flushErr
	}
	if closeErr != nil {
		return stats, closeErr
	}

	// Write the companion index file.
	if err := writeBinGzIdx(packPath+".idx", members); err != nil {
		return stats, err
	}

	return stats, nil
}

// writeBinGzIdx writes a member index to path.
//
// Format:
//
//	per member (12 bytes each):
//	  uint64 LE  byte_offset_in_bin_gz
//	  uint32 LE  doc_count
//	footer (4 bytes):
//	  uint32 LE  member_count
func writeBinGzIdx(path string, members []memberEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	bw := bufio.NewWriterSize(f, 64*1024)
	var buf [12]byte
	for _, m := range members {
		binary.LittleEndian.PutUint64(buf[0:8], m.offset)
		binary.LittleEndian.PutUint32(buf[8:12], m.docCount)
		if _, err := bw.Write(buf[:]); err != nil {
			f.Close()
			os.Remove(path)
			return err
		}
	}

	var foot [4]byte
	binary.LittleEndian.PutUint32(foot[:], uint32(len(members)))
	if _, err := bw.Write(foot[:]); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}

	if err := bw.Flush(); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	return f.Close()
}
