// File: lib/storage/transport/sftp/buffer_test.go
package sftp

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestWriteBufferMemory(t *testing.T) {
	buf := newWriteBuffer(1024, os.TempDir())
	defer buf.Close()

	data := []byte("hello world")
	n, err := buf.WriteAt(data, 0)
	if err != nil {
		t.Fatalf("WriteAt: %v", err)
	}
	if n != len(data) {
		t.Errorf("wrote %d bytes, expected %d", n, len(data))
	}

	size, reader, err := buf.Reader()
	if err != nil {
		t.Fatalf("Reader: %v", err)
	}
	if size != int64(len(data)) {
		t.Errorf("size %d, expected %d", size, len(data))
	}

	result, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(result, data) {
		t.Errorf("data mismatch: got %q, want %q", result, data)
	}
}

func TestWriteBufferOffsets(t *testing.T) {
	buf := newWriteBuffer(1024, os.TempDir())
	defer buf.Close()

	// Write at different offsets
	buf.WriteAt([]byte("AAA"), 0)
	buf.WriteAt([]byte("BBB"), 10)
	buf.WriteAt([]byte("CCC"), 20)

	size, reader, err := buf.Reader()
	if err != nil {
		t.Fatalf("Reader: %v", err)
	}
	if size != 23 {
		t.Errorf("size %d, expected 23", size)
	}

	result, _ := io.ReadAll(reader)

	if string(result[0:3]) != "AAA" {
		t.Errorf("offset 0: got %q", result[0:3])
	}
	if string(result[10:13]) != "BBB" {
		t.Errorf("offset 10: got %q", result[10:13])
	}
	if string(result[20:23]) != "CCC" {
		t.Errorf("offset 20: got %q", result[20:23])
	}

	// Check zeros in gaps
	for i := 3; i < 10; i++ {
		if result[i] != 0 {
			t.Errorf("expected zero at %d, got %d", i, result[i])
		}
	}
}

func TestWriteBufferSpillToDisk(t *testing.T) {
	// Small buffer to force disk spill
	buf := newWriteBuffer(100, os.TempDir())
	defer buf.Close()

	// Write more than 100 bytes
	data := bytes.Repeat([]byte("X"), 200)
	n, err := buf.WriteAt(data, 0)
	if err != nil {
		t.Fatalf("WriteAt: %v", err)
	}
	if n != len(data) {
		t.Errorf("wrote %d bytes, expected %d", n, len(data))
	}

	if !buf.useDisk {
		t.Error("expected buffer to spill to disk")
	}

	size, reader, err := buf.Reader()
	if err != nil {
		t.Fatalf("Reader: %v", err)
	}
	if size != int64(len(data)) {
		t.Errorf("size %d, expected %d", size, len(data))
	}

	result, _ := io.ReadAll(reader)
	if !bytes.Equal(result, data) {
		t.Error("data mismatch after disk spill")
	}
}

func TestWriteBufferSpillWithExistingData(t *testing.T) {
	buf := newWriteBuffer(100, os.TempDir())
	defer buf.Close()

	// Write some data first
	buf.WriteAt([]byte("initial"), 0)

	// Then write more to trigger spill
	large := bytes.Repeat([]byte("X"), 150)
	buf.WriteAt(large, 50)

	size, reader, _ := buf.Reader()
	result, _ := io.ReadAll(reader)

	// Verify initial data preserved
	if string(result[0:7]) != "initial" {
		t.Errorf("initial data lost: %q", result[0:7])
	}

	// Verify large data
	for i := 50; i < 200; i++ {
		if result[i] != 'X' {
			t.Errorf("large data corrupted at %d", i)
			break
		}
	}

	if size != 200 {
		t.Errorf("size %d, expected 200", size)
	}
}

func TestWriteBufferOverwrite(t *testing.T) {
	buf := newWriteBuffer(1024, os.TempDir())
	defer buf.Close()

	// Write initial data
	buf.WriteAt([]byte("AAAAAAAAAA"), 0) // 10 A's

	// Overwrite middle
	buf.WriteAt([]byte("BBB"), 3)

	size, reader, _ := buf.Reader()
	result, _ := io.ReadAll(reader)

	expected := "AAABBBAAAA"
	if string(result) != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
	if size != 10 {
		t.Errorf("size %d, expected 10", size)
	}
}

func TestWriteBufferClose(t *testing.T) {
	buf := newWriteBuffer(100, os.TempDir())

	// Force disk spill
	buf.WriteAt(bytes.Repeat([]byte("X"), 200), 0)

	if buf.tempFile == nil {
		t.Fatal("expected temp file to be created")
	}

	tempPath := buf.tempFile.Name()

	// Verify temp file exists
	if _, err := os.Stat(tempPath); os.IsNotExist(err) {
		t.Fatal("temp file should exist before close")
	}

	buf.Close()

	// Verify temp file is deleted
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Error("temp file should be deleted after close")
	}
}

func TestWriteBufferMultipleCloses(t *testing.T) {
	buf := newWriteBuffer(1024, os.TempDir())
	buf.WriteAt([]byte("test"), 0)

	// Multiple closes should be safe
	buf.Close()
	buf.Close()
	buf.Close()
}

func TestWriteBufferEmpty(t *testing.T) {
	buf := newWriteBuffer(1024, os.TempDir())
	defer buf.Close()

	size, reader, err := buf.Reader()
	if err != nil {
		t.Fatalf("Reader: %v", err)
	}
	if size != 0 {
		t.Errorf("size %d, expected 0", size)
	}

	result, _ := io.ReadAll(reader)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d bytes", len(result))
	}
}

func TestWriteBufferLargeOffset(t *testing.T) {
	buf := newWriteBuffer(1024, os.TempDir())
	defer buf.Close()

	// Write at large offset
	buf.WriteAt([]byte("data"), 500)

	size, reader, _ := buf.Reader()
	if size != 504 {
		t.Errorf("size %d, expected 504", size)
	}

	result, _ := io.ReadAll(reader)

	// First 500 bytes should be zero
	for i := 0; i < 500; i++ {
		if result[i] != 0 {
			t.Errorf("expected zero at %d", i)
			break
		}
	}

	if string(result[500:504]) != "data" {
		t.Errorf("data at offset 500: %q", result[500:504])
	}
}

func BenchmarkWriteBufferMemory(b *testing.B) {
	buf := newWriteBuffer(10*1024*1024, os.TempDir()) // 10MB
	defer buf.Close()

	data := bytes.Repeat([]byte("X"), 1024) // 1KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		offset := int64(i * 1024)
		buf.WriteAt(data, offset)
	}
}

func BenchmarkWriteBufferDisk(b *testing.B) {
	buf := newWriteBuffer(100, os.TempDir()) // Small to force disk
	defer buf.Close()

	data := bytes.Repeat([]byte("X"), 1024) // 1KB

	// Force disk mode
	buf.WriteAt(data, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		offset := int64((i + 1) * 1024)
		buf.WriteAt(data, offset)
	}
}
