package dahlia

import (
	"fmt"
	"testing"
)

func TestStoreWriteRead(t *testing.T) {
	sw := &storeWriter{}

	docs := make([]struct {
		id   string
		text string
	}, 100)
	for i := range docs {
		docs[i].id = fmt.Sprintf("doc_%04d", i)
		docs[i].text = fmt.Sprintf("This is the text content of document number %d with some extra words to make it longer", i)
	}

	for i, d := range docs {
		sw.addDoc(uint32(i), d.id, []byte(d.text))
	}

	data := sw.finish()
	sr, err := openStoreReader(data)
	if err != nil {
		t.Fatal(err)
	}

	// Verify all docs
	for i, d := range docs {
		id, text, err := sr.getDoc(uint32(i))
		if err != nil {
			t.Fatalf("getDoc(%d): %v", i, err)
		}
		if id != d.id {
			t.Fatalf("doc %d: id=%q, want %q", i, id, d.id)
		}
		if string(text) != d.text {
			t.Fatalf("doc %d: text mismatch", i)
		}
	}
}

func TestStoreWriteRead1000(t *testing.T) {
	sw := &storeWriter{}

	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("doc_%06d", i)
		text := fmt.Sprintf("Document %d with enough text to test block boundaries and compression efficiency in the store component", i)
		sw.addDoc(uint32(i), id, []byte(text))
	}

	data := sw.finish()
	sr, err := openStoreReader(data)
	if err != nil {
		t.Fatal(err)
	}

	// Spot check a few docs
	for _, idx := range []int{0, 1, 50, 100, 500, 999} {
		id, _, err := sr.getDoc(uint32(idx))
		if err != nil {
			t.Fatalf("getDoc(%d): %v", idx, err)
		}
		want := fmt.Sprintf("doc_%06d", idx)
		if id != want {
			t.Fatalf("doc %d: id=%q, want %q", idx, id, want)
		}
	}
}

func TestStoreEmpty(t *testing.T) {
	sw := &storeWriter{}
	data := sw.finish()
	if len(data) != 0 {
		// Empty store should just be the footer
		sr, err := openStoreReader(data)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = sr.getDoc(0)
		if err == nil {
			t.Fatal("should error on empty store")
		}
	}
}

func TestStoreSingleDoc(t *testing.T) {
	sw := &storeWriter{}
	sw.addDoc(0, "only-doc", []byte("only document in store"))
	data := sw.finish()

	sr, err := openStoreReader(data)
	if err != nil {
		t.Fatal(err)
	}

	id, text, err := sr.getDoc(0)
	if err != nil {
		t.Fatal(err)
	}
	if id != "only-doc" {
		t.Fatalf("id=%q, want %q", id, "only-doc")
	}
	if string(text) != "only document in store" {
		t.Fatalf("text=%q", text)
	}
}
