package pack

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// flatBinMagic is the 8-byte file header for the flat binary pack format.
const flatBinMagic = "MZFTS1\n\x00"

// flatBinFooterMagic is the 8-byte magic at the end of the footer block.
const flatBinFooterMagic = "MZFTS1F\n"

// flatBinFooterSize is the total byte size of the footer written by PackFlatBin.
//
// Footer layout (30 bytes, appended after the index block and before EOF):
//
//	[0:8]   record_count  int64 LE  — number of records in the file
//	[8:16]  index_offset  int64 LE  — byte offset to index block (0 = no index)
//	[16:20] index_size    int32 LE  — byte size of index block (N×8 bytes)
//	[20]    version       uint8     — format version, currently 1
//	[21]    flags         uint8     — reserved, 0
//	[22:30] footer_magic  [8]byte   — "MZFTS1F\n"
const flatBinFooterSize = 30

// PackFlatBin packs all markdown files from markdownDir into a flat binary file at packPath.
//
// Wire format:
//
//	magic     (8 bytes: "MZFTS1\n\x00")
//	repeated:
//	  id_len  uint16 LE  — doc ID length in bytes (max 65535)
//	  id      [id_len]byte
//	  txt_len uint32 LE  — text length in bytes
//	  text    [txt_len]byte
//	index     (N × uint64 LE) — byte offset from file start to the start of record i
//	footer    (30 bytes)      — see flatBinFooterSize for layout
//
// The index allows readers to divide work across multiple goroutines for parallel
// deserialisation: each worker seeks to offsets[chunkStart] and reads its slice.
func PackFlatBin(ctx context.Context, markdownDir, packPath string, workers, batchSize int, progress ProgressFunc) (*PipelineStats, error) {
	if err := os.MkdirAll(filepath.Dir(packPath), 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(packPath)
	if err != nil {
		return nil, err
	}

	bw := bufio.NewWriterSize(f, 1<<20) // 1 MB write buffer
	if _, err := io.WriteString(bw, flatBinMagic); err != nil {
		f.Close()
		os.Remove(packPath)
		return nil, err
	}

	// offsets records the file-start byte offset of each record.
	// Pre-allocate for a typical corpus size to avoid repeated growth.
	var (
		offsets []uint64
		pos     = uint64(len(flatBinMagic)) // current logical write position
	)

	eng := &funcEngine{
		name: "flatbin-writer",
		indexFn: func(_ context.Context, docs []Document) error {
			var hdr [6]byte
			for _, doc := range docs {
				id := doc.DocID
				if len(id) > 65535 {
					id = id[:65535]
				}
				offsets = append(offsets, pos)
				idLen := uint64(len(id))
				txtLen := uint64(len(doc.Text))
				pos += 2 + idLen + 4 + txtLen

				binary.LittleEndian.PutUint16(hdr[0:2], uint16(idLen))
				binary.LittleEndian.PutUint32(hdr[2:6], uint32(txtLen))
				if _, err := bw.Write(hdr[:2]); err != nil {
					return err
				}
				if _, err := io.WriteString(bw, id); err != nil {
					return err
				}
				if _, err := bw.Write(hdr[2:6]); err != nil {
					return err
				}
				if _, err := bw.Write(doc.Text); err != nil {
					return err
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
		// Append index: N × uint64 LE (byte offsets from file start).
		indexOffset := pos
		var u8 [8]byte
		for _, off := range offsets {
			binary.LittleEndian.PutUint64(u8[:], off)
			bw.Write(u8[:])
		}
		indexSize := uint32(len(offsets) * 8)

		// Append footer.
		var foot [flatBinFooterSize]byte
		binary.LittleEndian.PutUint64(foot[0:8], uint64(len(offsets)))
		binary.LittleEndian.PutUint64(foot[8:16], indexOffset)
		binary.LittleEndian.PutUint32(foot[16:20], indexSize)
		foot[20] = 1 // version
		// foot[21] = flags (0)
		copy(foot[22:30], flatBinFooterMagic)
		bw.Write(foot[:])
	}

	flushErr := bw.Flush()
	closeErr := f.Close()

	if pipeErr != nil {
		os.Remove(packPath)
		return stats, pipeErr
	}
	if flushErr != nil {
		os.Remove(packPath)
		return stats, flushErr
	}
	return stats, closeErr
}

// RunPipelineFromFlatBin reads a flat binary pack file and feeds documents into engine.
//
// When an index block is present (written by the current PackFlatBin), the reader
// loads all record offsets into memory and spawns NumCPU worker goroutines, each
// opening its own file handle and reading a contiguous slice of records in parallel.
// Falls back to a single sequential goroutine for files without an index.
func RunPipelineFromFlatBin(ctx context.Context, engine Engine, packPath string, batchSize int, progress PackProgressFunc) (*PipelineStats, error) {
	f, err := os.Open(packPath)
	if err != nil {
		return nil, fmt.Errorf("open flatbin: %w", err)
	}
	defer f.Close()

	// Read footer (pread — does not affect the sequential read position).
	var total int64
	var offsets []uint64
	if fi, err := f.Stat(); err == nil {
		sz := fi.Size()
		if sz >= int64(len(flatBinMagic)+flatBinFooterSize) {
			var foot [flatBinFooterSize]byte
			if _, err := f.ReadAt(foot[:], sz-flatBinFooterSize); err == nil {
				if string(foot[22:30]) == flatBinFooterMagic {
					total = int64(binary.LittleEndian.Uint64(foot[0:8]))
					indexOffset := int64(binary.LittleEndian.Uint64(foot[8:16]))
					indexSize := int32(binary.LittleEndian.Uint32(foot[16:20]))
					if indexOffset > 0 && indexSize > 0 && total > 0 {
						indexBuf := make([]byte, indexSize)
						if _, err := f.ReadAt(indexBuf, indexOffset); err == nil {
							offsets = make([]uint64, total)
							for i := range offsets {
								offsets[i] = binary.LittleEndian.Uint64(indexBuf[i*8:])
							}
						}
					}
				}
			}
		}
	}

	// Validate magic (advances position to 8; workers seek independently).
	var magic [8]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return nil, fmt.Errorf("read flatbin magic: %w", err)
	}
	if string(magic[:]) != flatBinMagic {
		return nil, fmt.Errorf("invalid flatbin magic: %x", magic)
	}

	docCh := make(chan Document, max(batchSize*4, 4096))

	if len(offsets) > 0 {
		// Parallel path: divide records across NumCPU workers.
		numWorkers := runtime.NumCPU()
		if numWorkers > int(total) {
			numWorkers = int(total)
		}
		var wg sync.WaitGroup
		chunkSize := (total + int64(numWorkers) - 1) / int64(numWorkers)
		for w := 0; w < numWorkers; w++ {
			start := int64(w) * chunkSize
			end := min(start+chunkSize, total)
			if start >= total {
				break
			}
			wg.Add(1)
			go func(start, end int64, seekOff uint64) {
				defer wg.Done()
				wf, err := os.Open(packPath)
				if err != nil {
					return
				}
				defer wf.Close()
				if _, err := wf.Seek(int64(seekOff), io.SeekStart); err != nil {
					return
				}
				br := bufio.NewReaderSize(wf, 512*1024)
				var hdr [6]byte
				for i := start; i < end; i++ {
					if ctx.Err() != nil {
						return
					}
					if _, err := io.ReadFull(br, hdr[:2]); err != nil {
						return
					}
					idLen := int(binary.LittleEndian.Uint16(hdr[:2]))
					idBuf := make([]byte, idLen)
					if _, err := io.ReadFull(br, idBuf); err != nil {
						return
					}
					if _, err := io.ReadFull(br, hdr[2:6]); err != nil {
						return
					}
					textLen := int(binary.LittleEndian.Uint32(hdr[2:6]))
					textBuf := make([]byte, textLen)
					if _, err := io.ReadFull(br, textBuf); err != nil {
						return
					}
					select {
					case docCh <- Document{DocID: string(idBuf), Text: textBuf}:
					case <-ctx.Done():
						return
					}
				}
			}(start, end, offsets[start])
		}
		go func() {
			wg.Wait()
			close(docCh)
		}()
	} else {
		// Sequential fallback: no index available.
		go func() {
			defer close(docCh)
			br := bufio.NewReaderSize(f, 1<<20)
			var hdr [6]byte
			var count int64
			for {
				if ctx.Err() != nil {
					return
				}
				if total > 0 && count >= total {
					return
				}
				if _, err := io.ReadFull(br, hdr[:2]); err != nil {
					return
				}
				idLen := int(binary.LittleEndian.Uint16(hdr[:2]))
				idBuf := make([]byte, idLen)
				if _, err := io.ReadFull(br, idBuf); err != nil {
					return
				}
				if _, err := io.ReadFull(br, hdr[2:6]); err != nil {
					return
				}
				textLen := int(binary.LittleEndian.Uint32(hdr[2:6]))
				textBuf := make([]byte, textLen)
				if _, err := io.ReadFull(br, textBuf); err != nil {
					return
				}
				select {
				case docCh <- Document{DocID: string(idBuf), Text: textBuf}:
					count++
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	return RunPipelineFromChannel(ctx, engine, docCh, total, batchSize, progress)
}
