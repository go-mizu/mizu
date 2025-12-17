package sync

import (
	"testing"
)

func TestStore_GetSet(t *testing.T) {
	s := NewStore()

	// Set
	s.Set("user", "1", []byte(`{"name":"Alice"}`))

	// Get
	data, ok := s.Get("user", "1")
	if !ok {
		t.Fatal("Get returned false")
	}
	if string(data) != `{"name":"Alice"}` {
		t.Errorf("Get() = %s, want %s", data, `{"name":"Alice"}`)
	}

	// Get non-existent
	_, ok = s.Get("user", "999")
	if ok {
		t.Error("Get non-existent should return false")
	}
}

func TestStore_Delete(t *testing.T) {
	s := NewStore()

	s.Set("user", "1", []byte(`{"name":"Alice"}`))
	s.Delete("user", "1")

	_, ok := s.Get("user", "1")
	if ok {
		t.Error("Get after delete should return false")
	}
}

func TestStore_Has(t *testing.T) {
	s := NewStore()

	if s.Has("user", "1") {
		t.Error("Has should return false for non-existent")
	}

	s.Set("user", "1", []byte(`{}`))

	if !s.Has("user", "1") {
		t.Error("Has should return true for existing")
	}
}

func TestStore_List(t *testing.T) {
	s := NewStore()

	s.Set("user", "1", []byte(`{}`))
	s.Set("user", "2", []byte(`{}`))
	s.Set("user", "3", []byte(`{}`))

	ids := s.List("user")
	if len(ids) != 3 {
		t.Errorf("List() len = %d, want 3", len(ids))
	}

	// Empty entity
	ids = s.List("nonexistent")
	if len(ids) != 0 {
		t.Errorf("List() len = %d, want 0", len(ids))
	}
}

func TestStore_All(t *testing.T) {
	s := NewStore()

	s.Set("user", "1", []byte(`{"name":"Alice"}`))
	s.Set("user", "2", []byte(`{"name":"Bob"}`))

	all := s.All("user")
	if len(all) != 2 {
		t.Errorf("All() len = %d, want 2", len(all))
	}
}

func TestStore_Snapshot(t *testing.T) {
	s := NewStore()

	s.Set("user", "1", []byte(`{"name":"Alice"}`))
	s.Set("post", "1", []byte(`{"title":"Hello"}`))

	snap := s.Snapshot()

	if len(snap) != 2 {
		t.Errorf("Snapshot() len = %d, want 2", len(snap))
	}
	if _, ok := snap["user"]["1"]; !ok {
		t.Error("Snapshot missing user/1")
	}
	if _, ok := snap["post"]["1"]; !ok {
		t.Error("Snapshot missing post/1")
	}
}

func TestStore_Load(t *testing.T) {
	s := NewStore()

	data := map[string]map[string][]byte{
		"user": {
			"1": []byte(`{"name":"Alice"}`),
			"2": []byte(`{"name":"Bob"}`),
		},
	}

	s.Load(data)

	if s.Count("user") != 2 {
		t.Errorf("Count() = %d, want 2", s.Count("user"))
	}

	got, ok := s.Get("user", "1")
	if !ok || string(got) != `{"name":"Alice"}` {
		t.Errorf("Get after Load failed")
	}
}

func TestStore_Clear(t *testing.T) {
	s := NewStore()

	s.Set("user", "1", []byte(`{}`))
	s.Set("user", "2", []byte(`{}`))

	s.Clear()

	if s.Count("user") != 0 {
		t.Errorf("Count after Clear = %d, want 0", s.Count("user"))
	}
}

func TestStore_Count(t *testing.T) {
	s := NewStore()

	if s.Count("user") != 0 {
		t.Errorf("Count() = %d, want 0", s.Count("user"))
	}

	s.Set("user", "1", []byte(`{}`))
	s.Set("user", "2", []byte(`{}`))

	if s.Count("user") != 2 {
		t.Errorf("Count() = %d, want 2", s.Count("user"))
	}
}

func TestStore_EntityTypes(t *testing.T) {
	s := NewStore()

	s.Set("user", "1", []byte(`{}`))
	s.Set("post", "1", []byte(`{}`))
	s.Set("comment", "1", []byte(`{}`))

	types := s.EntityTypes()
	if len(types) != 3 {
		t.Errorf("EntityTypes() len = %d, want 3", len(types))
	}
}

func TestStore_Version(t *testing.T) {
	s := NewStore()

	v1 := s.Version().Get()
	s.Set("user", "1", []byte(`{}`))
	v2 := s.Version().Get()

	if v2 <= v1 {
		t.Error("Version should increase after Set")
	}

	s.Delete("user", "1")
	v3 := s.Version().Get()

	if v3 <= v2 {
		t.Error("Version should increase after Delete")
	}
}

func TestStore_OnChange(t *testing.T) {
	s := NewStore()

	var lastEntity, lastID string
	var lastOp Op

	s.SetOnChange(func(entity, id string, op Op) {
		lastEntity = entity
		lastID = id
		lastOp = op
	})

	s.Set("user", "1", []byte(`{}`))
	if lastEntity != "user" || lastID != "1" || lastOp != OpCreate {
		t.Error("OnChange not called correctly for create")
	}

	s.Set("user", "1", []byte(`{"updated":true}`))
	if lastOp != OpUpdate {
		t.Error("OnChange should report update for existing entity")
	}

	s.Delete("user", "1")
	if lastOp != OpDelete {
		t.Error("OnChange should report delete")
	}
}

func TestStore_DataCopy(t *testing.T) {
	s := NewStore()

	original := []byte(`{"name":"Alice"}`)
	s.Set("user", "1", original)

	// Modify original
	original[0] = 'X'

	// Get should return unchanged data
	got, _ := s.Get("user", "1")
	if got[0] == 'X' {
		t.Error("Store should make a copy of data")
	}

	// Modify returned data
	got[0] = 'Y'

	// Get again should return unchanged
	got2, _ := s.Get("user", "1")
	if got2[0] == 'Y' {
		t.Error("Store should return a copy")
	}
}
