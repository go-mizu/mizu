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
				if _, err := io.WriteString(bw, doc.Text); err != nil {
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
func RunPipelineFromFlatBin(ctx context.Context, engine Engine, packPath string, batchSize int, progress PackProgressFunc) (*PipelineStats, error) {
	f, err := os.Open(packPath)
	if err != nil {
		return nil, fmt.Errorf("open flatbin: %w", err)
	}
	defer f.Close()

	// Validate magic
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
		for {
			if ctx.Err() != nil {
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

			textBuf := make([]byte, textLen)
			if _, err := io.ReadFull(br, textBuf); err != nil {
				return
			}
			select {
			case docCh <- Document{DocID: string(idBuf), Text: string(textBuf)}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return RunPipelineFromChannel(ctx, engine, docCh, 0, batchSize, progress)
}
