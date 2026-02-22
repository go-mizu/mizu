package fox

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"testing"

	"github.com/liteio-dev/liteio/pkg/storage"
)

func TestFoxLargeValueRoundTripAndRangeRead(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	st, err := storage.Open(ctx, "fox://"+dir+"?sync=none&page_size=4096&pool_size=1048576")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	b := st.Bucket("b1")
	payload := bytes.Repeat([]byte("abcd1234"), 8192) // 64 KiB

	if _, err := b.Write(ctx, "big.bin", bytes.NewReader(payload), int64(len(payload)), "application/octet-stream", nil); err != nil {
		t.Fatalf("write: %v", err)
	}

	obj, err := b.Stat(ctx, "big.bin", nil)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := obj.Size; got != int64(len(payload)) {
		t.Fatalf("stat size mismatch: got %d want %d", got, len(payload))
	}

	rc, obj, err := b.Open(ctx, "big.bin", 0, 0, nil)
	if err != nil {
		t.Fatalf("open full: %v", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read full: %v", err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("full read mismatch: got %d bytes", len(data))
	}
	if obj.Size != int64(len(payload)) {
		t.Fatalf("open object size mismatch: got %d want %d", obj.Size, len(payload))
	}

	rc2, _, err := b.Open(ctx, "big.bin", 123, 4096, nil)
	if err != nil {
		t.Fatalf("open range: %v", err)
	}
	defer rc2.Close()
	part, err := io.ReadAll(rc2)
	if err != nil {
		t.Fatalf("read range: %v", err)
	}
	if !bytes.Equal(part, payload[123:123+4096]) {
		t.Fatalf("range read mismatch")
	}

	it, err := b.List(ctx, "", 0, 0, nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer it.Close()
	item, err := it.Next()
	if err != nil {
		t.Fatalf("list next: %v", err)
	}
	if item == nil || item.Key != "big.bin" || item.Size != int64(len(payload)) {
		t.Fatalf("list item mismatch: %+v", item)
	}
}

func TestFoxSplitPreservesSmallValues(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	st, err := storage.Open(ctx, "fox://"+dir+"?sync=none&page_size=4096&pool_size=1048576")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	b := st.Bucket("b1")
	val := bytes.Repeat([]byte("x"), 1024)

	for i := 0; i < 500; i++ {
		key := "k/" + strconv.Itoa(i)
		if _, err := b.Write(ctx, key, bytes.NewReader(val), int64(len(val)), "application/octet-stream", nil); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}

	for i := 0; i < 500; i++ {
		key := "k/" + strconv.Itoa(i)
		rc, _, err := b.Open(ctx, key, 0, 0, nil)
		if err != nil {
			t.Fatalf("open %d: %v", i, err)
		}
		got, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if !bytes.Equal(got, val) {
			t.Fatalf("value mismatch for %s", key)
		}
	}
}
