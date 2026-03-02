package index

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// flatBinMagic is the 8-byte file header for the flat binary pack format.
const flatBinMagic = "MZFTS1\n\x00"

// flatBinFooterMagic is the 8-byte magic at the end of the footer block.
const flatBinFooterMagic = "MZFTS1F\n"

// flatBinFooterSize is the total byte size of the footer written by PackFlatBin.
//
// Footer layout (30 bytes, appended after the last record):
//
//	[0:8]   record_count  int64 LE  — number of records in the file
//	[8:16]  index_offset  int64 LE  — byte offset to index table (0 = no index)
//	[16:20] index_size    int32 LE  — byte size of index table (0 = no index)
//	[20]    version       uint8     — format version, currently 1
//	[21]    flags         uint8     — reserved, 0
//	[22:30] footer_magic  [8]byte   — "MZFTS1F\n"
const flatBinFooterSize = 30

// PackFlatBin packs all markdown files from markdownDir into a flat binary file at packPath.
//
// Wire format:
//
//	magic (8 bytes: "MZFTS1\n\x00")
//	repeated:
//	  id_len  uint16 LE  — doc ID length in bytes (max 65535)
//	  id      [id_len]byte
//	  txt_len uint32 LE  — text length in bytes
//	  text    [txt_len]byte
//	footer (30 bytes)    — see flatBinFooterSize for layout
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

	eng := &funcEngine{
		name: "flatbin-writer",
		indexFn: func(_ context.Context, docs []Document) error {
			var hdr [6]byte
			for _, doc := range docs {
				id := doc.DocID
				if len(id) > 65535 {
					id = id[:65535]
				}
				binary.LittleEndian.PutUint16(hdr[0:2], uint16(len(id)))
				binary.LittleEndian.PutUint32(hdr[2:6], uint32(len(doc.Text)))
				if _, err := bw.Write(hdr[:2]); err != nil {
					return err
				}
				if _, err := io.WriteString(bw, id); err != nil {
					return err
				}
				if _, err := bw.Write(hdr[2:6]); err != nil {
					return err
				}
				if _, err := bw.Write(doc.Text); err != nil { // doc.Text is []byte
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
		// Append footer so readers know total record count upfront.
		var foot [flatBinFooterSize]byte
		binary.LittleEndian.PutUint64(foot[0:8], uint64(stats.DocsIndexed.Load()))
		// foot[8:16]  = index_offset (0 = no index)
		// foot[16:20] = index_size   (0 = no index)
		foot[20] = 1 // version
		// foot[21]    = flags (0)
		copy(foot[22:30], flatBinFooterMagic)
		bw.Write(foot[:]) // flush error caught below
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
// If the file contains a footer (written by PackFlatBin), the total record count is used
// to display percentage progress and the goroutine stops exactly at the footer boundary.
// Falls back gracefully for files written without a footer (total = 0, reads until EOF).
//
// Document.Text slices are allocated fresh per record and passed directly without copying.
func RunPipelineFromFlatBin(ctx context.Context, engine Engine, packPath string, batchSize int, progress PackProgressFunc) (*PipelineStats, error) {
	f, err := os.Open(packPath)
	if err != nil {
		return nil, fmt.Errorf("open flatbin: %w", err)
	}
	defer f.Close()

	// Try to read the footer for total record count.
	// f.ReadAt uses pread(2) and does not affect the file offset.
	var total int64
	if fi, err := f.Stat(); err == nil {
		sz := fi.Size()
		if sz >= int64(len(flatBinMagic)+flatBinFooterSize) {
			var foot [flatBinFooterSize]byte
			if _, err := f.ReadAt(foot[:], sz-flatBinFooterSize); err == nil {
				if string(foot[22:30]) == flatBinFooterMagic {
					total = int64(binary.LittleEndian.Uint64(foot[0:8]))
				}
			}
		}
	}

	// Validate magic (reads from position 0, advances to 8).
	var magic [8]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return nil, fmt.Errorf("read flatbin magic: %w", err)
	}
	if string(magic[:]) != flatBinMagic {
		return nil, fmt.Errorf("invalid flatbin magic: %x", magic)
	}

	docCh := make(chan Document, max(batchSize*2, 1024))
	go func() {
		defer close(docCh)
		br := bufio.NewReaderSize(f, 1<<20)
		var hdr [6]byte

		var count int64
		for {
			if ctx.Err() != nil {
				return
			}
			// When total is known, stop before the footer bytes rather than
			// letting io.ReadFull misinterpret them as a record header.
			if total > 0 && count >= total {
				return
			}
			// Read id_len (2 bytes LE)
			if _, err := io.ReadFull(br, hdr[:2]); err != nil {
				return // clean EOF or read error — stop
			}
			idLen := int(binary.LittleEndian.Uint16(hdr[:2]))

			idBuf := make([]byte, idLen)
			if _, err := io.ReadFull(br, idBuf); err != nil {
				return
			}
			// Read text_len (4 bytes LE)
			if _, err := io.ReadFull(br, hdr[2:6]); err != nil {
				return
			}
			textLen := int(binary.LittleEndian.Uint32(hdr[2:6]))

			// Allocate a fresh []byte for the text and pass it directly as
			// Document.Text — no string conversion, no extra copy.
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

	return RunPipelineFromChannel(ctx, engine, docCh, total, batchSize, progress)
}
