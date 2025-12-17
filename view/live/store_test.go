package live

import (
	"testing"
	"time"
)

func TestMemoryStore(t *testing.T) {
	t.Run("set and get", func(t *testing.T) {
		store := NewMemoryStore()
		session := &sessionBase{session: &Session[int]{ID: "test"}}

		store.Set("test", session)

		got, ok := store.Get("test")
		if !ok {
			t.Fatal("expected to find session")
		}
		if got != session {
			t.Error("session mismatch")
		}
	})

	t.Run("get missing", func(t *testing.T) {
		store := NewMemoryStore()

		_, ok := store.Get("missing")
		if ok {
			t.Error("expected not found")
		}
	})

	t.Run("delete", func(t *testing.T) {
		store := NewMemoryStore()
		session := &sessionBase{session: &Session[int]{ID: "test"}}

		store.Set("test", session)
		store.Delete("test")

		_, ok := store.Get("test")
		if ok {
			t.Error("expected session to be deleted")
		}
	})

	t.Run("touch updates last seen", func(t *testing.T) {
		store := NewMemoryStore()
		session := &sessionBase{session: &Session[int]{ID: "test"}}

		store.Set("test", session)
		time.Sleep(10 * time.Millisecond)
		store.Touch("test")

		// Touch should update the internal lastSeen time
		// We can verify this indirectly via cleanup
	})

	t.Run("count", func(t *testing.T) {
		store := NewMemoryStore()

		store.Set("s1", &sessionBase{session: &Session[int]{ID: "s1"}})
		store.Set("s2", &sessionBase{session: &Session[int]{ID: "s2"}})
		store.Set("s3", &sessionBase{session: &Session[int]{ID: "s3"}})

		if store.Count() != 3 {
			t.Errorf("expected count 3, got %d", store.Count())
		}

		store.Delete("s2")

		if store.Count() != 2 {
			t.Errorf("expected count 2, got %d", store.Count())
		}
	})

	t.Run("cleanup removes expired", func(t *testing.T) {
		store := NewMemoryStore()

		store.Set("old", &sessionBase{session: &Session[int]{ID: "old"}})
		time.Sleep(50 * time.Millisecond)
		store.Set("new", &sessionBase{session: &Session[int]{ID: "new"}})

		// Cleanup sessions older than 40ms
		cleaned := store.Cleanup(40 * time.Millisecond)

		if cleaned != 1 {
			t.Errorf("expected 1 cleaned, got %d", cleaned)
		}

		_, oldExists := store.Get("old")
		_, newExists := store.Get("new")

		if oldExists {
			t.Error("expected old session to be cleaned")
		}
		if !newExists {
			t.Error("expected new session to exist")
		}
	})

	t.Run("all returns all sessions", func(t *testing.T) {
		store := NewMemoryStore()

		store.Set("s1", &sessionBase{session: &Session[int]{ID: "s1"}})
		store.Set("s2", &sessionBase{session: &Session[int]{ID: "s2"}})

		all := store.All()
		if len(all) != 2 {
			t.Errorf("expected 2 sessions, got %d", len(all))
		}
	})
}
