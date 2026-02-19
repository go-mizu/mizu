package bee

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"testing"
)

func TestWriteReadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	dsn := "bee:///" + filepath.Join(dir, "cluster") + "?nodes=3&replicas=3&w=2&r=1&sync=none"

	st, err := (&driver{}).Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	b := st.Bucket("test")
	data := bytes.Repeat([]byte("x"), 8192)

	obj, err := b.Write(context.Background(), "k1", bytes.NewReader(data), int64(len(data)), "application/octet-stream", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if obj.Size != int64(len(data)) {
		t.Fatalf("size mismatch: got %d want %d", obj.Size, len(data))
	}

	rc, gotObj, err := b.Open(context.Background(), "k1", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("data mismatch")
	}
	if gotObj.Size != int64(len(data)) {
		t.Fatalf("open size mismatch: got %d want %d", gotObj.Size, len(data))
	}
}

func TestWriteWithOneNodeDownStillMeetsQuorum(t *testing.T) {
	dir := t.TempDir()
	dsn := "bee:///" + filepath.Join(dir, "cluster") + "?nodes=3&replicas=3&w=2&r=1&sync=none"

	raw, err := (&driver{}).Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer raw.Close()

	st := raw.(*store)
	if err := st.nodes[0].close(); err != nil {
		t.Fatalf("close node: %v", err)
	}

	b := st.Bucket("test")
	data := []byte("quorum write")
	if _, err := b.Write(context.Background(), "k2", bytes.NewReader(data), int64(len(data)), "text/plain", nil); err != nil {
		t.Fatalf("Write with one node down should succeed: %v", err)
	}

	rc, _, err := b.Open(context.Background(), "k2", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open after quorum write: %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("data mismatch: got %q want %q", got, data)
	}
}

func TestShardingSpreadsKeys(t *testing.T) {
	dir := t.TempDir()
	dsn := "bee:///" + filepath.Join(dir, "cluster") + "?nodes=5&replicas=2&w=1&r=1&sync=none"

	raw, err := (&driver{}).Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer raw.Close()

	st := raw.(*store)
	b := st.Bucket("test")

	for i := 0; i < 200; i++ {
		k := fmt.Sprintf("obj-%03d", i)
		v := []byte(k)
		if _, err := b.Write(context.Background(), k, bytes.NewReader(v), int64(len(v)), "text/plain", nil); err != nil {
			t.Fatalf("Write %s: %v", k, err)
		}
	}

	activeNodes := 0
	for _, n := range st.nodes {
		if len(n.list("test", "", true)) > 0 {
			activeNodes++
		}
	}

	if activeNodes < 2 {
		t.Fatalf("expected keys on >=2 nodes, got %d", activeNodes)
	}
}
