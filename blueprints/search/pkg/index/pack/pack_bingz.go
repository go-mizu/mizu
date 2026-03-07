package pack

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"runtime"
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

// RunPipelineFromFlatBinGz reads a concatenated gzip pack file (written by
// PackFlatBinGz) and feeds documents into engine using parallel goroutines.
//
// It loads (or rebuilds) the companion .idx file to discover member offsets,
// then spawns min(NumCPU, len(members)) reader goroutines each with their own
// file handle. Each goroutine decompresses its assigned gzip members and sends
// the decoded documents to a shared channel that is drained by
// RunPipelineFromChannel.
func RunPipelineFromFlatBinGz(ctx context.Context, engine Engine, packPath string, batchSize int, progress PackProgressFunc) (*PipelineStats, error) {
	members, err := loadOrBuildBinGzIdx(packPath)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return &PipelineStats{}, nil
	}

	// Compute total docs for progress reporting.
	var total int64
	for _, m := range members {
		total += int64(m.docCount)
	}

	docCh := make(chan Document, max(batchSize*4, 4096))

	// memberCh distributes work across reader goroutines.
	memberCh := make(chan memberEntry, len(members))
	for _, m := range members {
		memberCh <- m
	}
	close(memberCh)

	numWorkers := runtime.NumCPU()
	if numWorkers > len(members) {
		numWorkers = len(members)
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wf, err := os.Open(packPath)
			if err != nil {
				return
			}
			defer wf.Close()

			var hdr [6]byte
			for m := range memberCh {
				if ctx.Err() != nil {
					return
				}
				if _, err := wf.Seek(int64(m.offset), io.SeekStart); err != nil {
					return
				}
				gr, err := kgzip.NewReader(wf)
				if err != nil {
					return
				}
				br := bufio.NewReaderSize(gr, 512*1024)

				// Read all flatbin records from this member.
				// If docCount == 0 (rebuilt index), read until EOF.
				var count uint32
				for {
					if ctx.Err() != nil {
						gr.Close()
						return
					}
					if m.docCount > 0 && count >= m.docCount {
						break
					}
					if _, err := io.ReadFull(br, hdr[:2]); err != nil {
						break // EOF or error — done with this member
					}
					idLen := int(binary.LittleEndian.Uint16(hdr[:2]))
					idBuf := make([]byte, idLen)
					if _, err := io.ReadFull(br, idBuf); err != nil {
						break
					}
					if _, err := io.ReadFull(br, hdr[2:6]); err != nil {
						break
					}
					textLen := int(binary.LittleEndian.Uint32(hdr[2:6]))
					textBuf := make([]byte, textLen)
					if _, err := io.ReadFull(br, textBuf); err != nil {
						break
					}
					count++
					select {
					case docCh <- Document{DocID: string(idBuf), Text: textBuf}:
					case <-ctx.Done():
						gr.Close()
						return
					}
				}
				gr.Close()
			}
		}()
	}

	go func() {
		wg.Wait()
		close(docCh)
	}()

	return RunPipelineFromChannel(ctx, engine, docCh, total, batchSize, progress)
}

// loadOrBuildBinGzIdx loads the member index from packPath+".idx". If the idx
// file is missing or unreadable, it falls back to scanBinGzMembers and
// best-effort writes a new idx file for future calls.
func loadOrBuildBinGzIdx(packPath string) ([]memberEntry, error) {
	idxPath := packPath + ".idx"
	if members, err := readBinGzIdx(idxPath); err == nil && len(members) > 0 {
		return members, nil
	}
	members, err := scanBinGzMembers(packPath)
	if err != nil {
		return nil, err
	}
	// Best-effort cache: ignore write errors.
	_ = writeBinGzIdx(idxPath, members)
	return members, nil
}

// readBinGzIdx parses a member index written by writeBinGzIdx.
//
// Format: N×12-byte entries (uint64 LE offset + uint32 LE docCount) followed
// by a 4-byte uint32 LE member count footer.
func readBinGzIdx(path string) ([]memberEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 4 {
		return nil, io.ErrUnexpectedEOF
	}
	memberCount := int(binary.LittleEndian.Uint32(data[len(data)-4:]))
	expected := memberCount*12 + 4
	if len(data) != expected {
		return nil, io.ErrUnexpectedEOF
	}
	members := make([]memberEntry, memberCount)
	for i := range members {
		base := i * 12
		members[i] = memberEntry{
			offset:   binary.LittleEndian.Uint64(data[base : base+8]),
			docCount: binary.LittleEndian.Uint32(data[base+8 : base+12]),
		}
	}
	return members, nil
}

// scanBinGzMembers scans packPath for gzip magic bytes (0x1f 0x8b) and
// returns a memberEntry for each found member. docCount is 0 for all entries
// since it is not known without decompression. This is the fallback path used
// when the companion .idx file is missing.
func scanBinGzMembers(packPath string) ([]memberEntry, error) {
	data, err := os.ReadFile(packPath)
	if err != nil {
		return nil, err
	}
	var members []memberEntry
	for i := 0; i+1 < len(data); i++ {
		if data[i] == 0x1f && data[i+1] == 0x8b {
			members = append(members, memberEntry{offset: uint64(i)})
			i++ // skip past the 0x8b byte to avoid re-matching
		}
	}
	return members, nil
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
