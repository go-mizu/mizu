package sync

import (
	"testing"
)

func TestMemoryPersistence_Queue(t *testing.T) {
	p := NewMemoryPersistence()

	mutations := []Mutation{
		{ID: "a", Name: "test-a", Seq: 1},
		{ID: "b", Name: "test-b", Seq: 2},
	}

	if err := p.SaveQueue(mutations, 2); err != nil {
		t.Fatalf("SaveQueue error: %v", err)
	}

	loaded, seq, err := p.LoadQueue()
	if err != nil {
		t.Fatalf("LoadQueue error: %v", err)
	}

	if seq != 2 {
		t.Errorf("seq = %d, want 2", seq)
	}

	if len(loaded) != 2 {
		t.Errorf("len(loaded) = %d, want 2", len(loaded))
	}

	if loaded[0].Name != "test-a" {
		t.Errorf("loaded[0].Name = %s, want test-a", loaded[0].Name)
	}
}

func TestMemoryPersistence_Cursor(t *testing.T) {
	p := NewMemoryPersistence()

	if err := p.SaveCursor(42); err != nil {
		t.Fatalf("SaveCursor error: %v", err)
	}

	cursor, err := p.LoadCursor()
	if err != nil {
		t.Fatalf("LoadCursor error: %v", err)
	}

	if cursor != 42 {
		t.Errorf("cursor = %d, want 42", cursor)
	}
}

func TestMemoryPersistence_Store(t *testing.T) {
	p := NewMemoryPersistence()

	data := map[string]map[string][]byte{
		"user": {
			"1": []byte(`{"name":"Alice"}`),
		},
	}

	if err := p.SaveStore(data); err != nil {
		t.Fatalf("SaveStore error: %v", err)
	}

	loaded, err := p.LoadStore()
	if err != nil {
		t.Fatalf("LoadStore error: %v", err)
	}

	if string(loaded["user"]["1"]) != `{"name":"Alice"}` {
		t.Errorf("loaded data mismatch")
	}
}

func TestMemoryPersistence_ClientID(t *testing.T) {
	p := NewMemoryPersistence()

	if err := p.SaveClientID("my-client"); err != nil {
		t.Fatalf("SaveClientID error: %v", err)
	}

	id, err := p.LoadClientID()
	if err != nil {
		t.Fatalf("LoadClientID error: %v", err)
	}

	if id != "my-client" {
		t.Errorf("id = %s, want my-client", id)
	}
}

func TestMemoryPersistence_DataCopy(t *testing.T) {
	p := NewMemoryPersistence()

	data := map[string]map[string][]byte{
		"user": {
			"1": []byte(`{"name":"Alice"}`),
		},
	}

	_ = p.SaveStore(data)

	// Modify original
	data["user"]["1"][0] = 'X'

	loaded, _ := p.LoadStore()
	if loaded["user"]["1"][0] == 'X' {
		t.Error("Persistence should make copies of data")
	}
}

func TestNopPersistence(t *testing.T) {
	p := NopPersistence{}

	// All operations should succeed silently
	if err := p.SaveQueue(nil, 0); err != nil {
		t.Errorf("SaveQueue error: %v", err)
	}

	mutations, seq, err := p.LoadQueue()
	if err != nil {
		t.Errorf("LoadQueue error: %v", err)
	}
	if mutations != nil || seq != 0 {
		t.Error("LoadQueue should return nil, 0")
	}

	if err := p.SaveCursor(42); err != nil {
		t.Errorf("SaveCursor error: %v", err)
	}

	cursor, err := p.LoadCursor()
	if err != nil || cursor != 0 {
		t.Error("LoadCursor should return 0, nil")
	}

	if err := p.SaveStore(nil); err != nil {
		t.Errorf("SaveStore error: %v", err)
	}

	data, err := p.LoadStore()
	if err != nil || data != nil {
		t.Error("LoadStore should return nil, nil")
	}

	if err := p.SaveClientID("test"); err != nil {
		t.Errorf("SaveClientID error: %v", err)
	}

	id, err := p.LoadClientID()
	if err != nil || id != "" {
		t.Error("LoadClientID should return empty, nil")
	}
}
