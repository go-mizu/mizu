// File: lib/storage/transport/sftp/buffer.go
package sftp

import (
	"bytes"
	"io"
	"os"
	"sync"
)

// writeBuffer buffers data for writing, spilling to disk for large files.
type writeBuffer struct {
	mu           sync.Mutex
	data         *bytes.Buffer
	tempFile     *os.File
	maxMemory    int64
	tempDir      string
	totalWritten int64
	useDisk      bool
}

func newWriteBuffer(maxMemory int64, tempDir string) *writeBuffer {
	return &writeBuffer{
		data:      bytes.NewBuffer(nil),
		maxMemory: maxMemory,
		tempDir:   tempDir,
	}
}

func (w *writeBuffer) WriteAt(p []byte, off int64) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Calculate required size
	required := off + int64(len(p))

	// Check if we need to switch to disk
	if !w.useDisk && required > w.maxMemory {
		if err := w.spillToDisk(); err != nil {
			return 0, err
		}
	}

	if w.useDisk {
		// Write to temp file
		n, err := w.tempFile.WriteAt(p, off)
		if off+int64(n) > w.totalWritten {
			w.totalWritten = off + int64(n)
		}
		return n, err
	}

	// Write to memory buffer
	// Extend buffer if needed
	if off > int64(w.data.Len()) {
		// Fill gap with zeros
		gap := make([]byte, off-int64(w.data.Len()))
		w.data.Write(gap)
	}

	// If writing beyond current length
	if off == int64(w.data.Len()) {
		w.data.Write(p)
	} else {
		// Writing in the middle - need to handle carefully
		buf := w.data.Bytes()
		if off+int64(len(p)) > int64(len(buf)) {
			// Extend
			newBuf := make([]byte, off+int64(len(p)))
			copy(newBuf, buf)
			copy(newBuf[off:], p)
			w.data.Reset()
			w.data.Write(newBuf)
		} else {
			copy(buf[off:], p)
		}
	}

	if int64(w.data.Len()) > w.totalWritten {
		w.totalWritten = int64(w.data.Len())
	}

	return len(p), nil
}

func (w *writeBuffer) spillToDisk() error {
	f, err := os.CreateTemp(w.tempDir, "sftp-upload-*")
	if err != nil {
		return err
	}

	// Copy existing data
	if w.data.Len() > 0 {
		if _, err := f.Write(w.data.Bytes()); err != nil {
			f.Close()
			os.Remove(f.Name())
			return err
		}
	}

	w.tempFile = f
	w.useDisk = true
	w.data.Reset() // Free memory

	return nil
}

func (w *writeBuffer) Reader() (int64, io.Reader, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.useDisk {
		// Seek to beginning of temp file
		if _, err := w.tempFile.Seek(0, 0); err != nil {
			return 0, nil, err
		}
		return w.totalWritten, w.tempFile, nil
	}

	return int64(w.data.Len()), bytes.NewReader(w.data.Bytes()), nil
}

func (w *writeBuffer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.tempFile != nil {
		name := w.tempFile.Name()
		w.tempFile.Close()
		os.Remove(name)
		w.tempFile = nil
	}

	w.data.Reset()
	return nil
}
