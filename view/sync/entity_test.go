package sync

import (
	"context"
	"encoding/json"
	"testing"
)

type TestUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func newTestClient() *Client {
	return New(Options{
		BaseURL:     "http://localhost:8080/_sync",
		Scope:       "test",
		Persistence: &NopPersistence{},
	})
}

func TestCollection_Create(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	user := users.Create("1", TestUser{Name: "Alice", Email: "alice@example.com"})

	if user.ID() != "1" {
		t.Errorf("ID() = %s, want 1", user.ID())
	}

	got := user.Get()
	if got.Name != "Alice" {
		t.Errorf("Get().Name = %s, want Alice", got.Name)
	}
}

func TestCollection_Get(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	// Set data directly in store
	data, _ := json.Marshal(TestUser{Name: "Bob"})
	client.store.Set("user", "1", data)

	user := users.Get("1")
	got := user.Get()

	if got.Name != "Bob" {
		t.Errorf("Get().Name = %s, want Bob", got.Name)
	}
}

func TestCollection_All(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	users.Create("1", TestUser{Name: "Alice"})
	users.Create("2", TestUser{Name: "Bob"})
	users.Create("3", TestUser{Name: "Charlie"})

	all := users.All()
	if len(all) != 3 {
		t.Errorf("All() len = %d, want 3", len(all))
	}
}

func TestCollection_Count(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	if users.Count() != 0 {
		t.Errorf("Count() = %d, want 0", users.Count())
	}

	users.Create("1", TestUser{Name: "Alice"})
	users.Create("2", TestUser{Name: "Bob"})

	if users.Count() != 2 {
		t.Errorf("Count() = %d, want 2", users.Count())
	}
}

func TestCollection_Find(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	users.Create("1", TestUser{Name: "Alice", Age: 25})
	users.Create("2", TestUser{Name: "Bob", Age: 30})
	users.Create("3", TestUser{Name: "Charlie", Age: 35})

	adults := users.Find(func(u TestUser) bool {
		return u.Age >= 30
	})

	if len(adults) != 2 {
		t.Errorf("Find() len = %d, want 2", len(adults))
	}
}

func TestCollection_First(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	users.Create("1", TestUser{Name: "Alice", Age: 25})
	users.Create("2", TestUser{Name: "Bob", Age: 30})

	bob := users.First(func(u TestUser) bool {
		return u.Age == 30
	})

	if bob == nil {
		t.Fatal("First() returned nil")
	}
	if bob.Get().Name != "Bob" {
		t.Errorf("First().Get().Name = %s, want Bob", bob.Get().Name)
	}

	// Not found
	none := users.First(func(u TestUser) bool {
		return u.Age == 100
	})
	if none != nil {
		t.Error("First() should return nil for no match")
	}
}

func TestCollection_Has(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	if users.Has("1") {
		t.Error("Has() should return false before create")
	}

	users.Create("1", TestUser{Name: "Alice"})

	if !users.Has("1") {
		t.Error("Has() should return true after create")
	}
}

func TestCollection_IDs(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	users.Create("a", TestUser{Name: "Alice"})
	users.Create("b", TestUser{Name: "Bob"})

	ids := users.IDs()
	if len(ids) != 2 {
		t.Errorf("IDs() len = %d, want 2", len(ids))
	}
}

func TestEntity_Set(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	user := users.Create("1", TestUser{Name: "Alice"})
	user.Set(TestUser{Name: "Alicia", Email: "alicia@example.com"})

	got := user.Get()
	if got.Name != "Alicia" {
		t.Errorf("Get().Name = %s, want Alicia", got.Name)
	}
}

func TestEntity_Delete(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	user := users.Create("1", TestUser{Name: "Alice"})
	user.Delete()

	if user.Exists() {
		t.Error("Exists() should return false after delete")
	}
}

func TestEntity_Exists(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	user := users.Get("nonexistent")
	if user.Exists() {
		t.Error("Exists() should return false for non-existent")
	}

	users.Create("1", TestUser{Name: "Alice"})
	user = users.Get("1")
	if !user.Exists() {
		t.Error("Exists() should return true after create")
	}
}

func TestCollection_MutationQueued(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	users.Create("1", TestUser{Name: "Alice"})

	if client.queue.Len() != 1 {
		t.Errorf("Queue.Len() = %d, want 1", client.queue.Len())
	}

	pending := client.queue.Pending()
	if pending[0].Name != "user.create" {
		t.Errorf("Mutation.Name = %s, want user.create", pending[0].Name)
	}
}

func TestEntity_SetQueuesMutation(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	user := users.Create("1", TestUser{Name: "Alice"})
	user.Set(TestUser{Name: "Alicia"})

	if client.queue.Len() != 2 {
		t.Errorf("Queue.Len() = %d, want 2", client.queue.Len())
	}

	pending := client.queue.Pending()
	if pending[1].Name != "user.update" {
		t.Errorf("Mutation.Name = %s, want user.update", pending[1].Name)
	}
}

func TestEntity_DeleteQueuesMutation(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	user := users.Create("1", TestUser{Name: "Alice"})
	user.Delete()

	if client.queue.Len() != 2 {
		t.Errorf("Queue.Len() = %d, want 2", client.queue.Len())
	}

	pending := client.queue.Pending()
	if pending[1].Name != "user.delete" {
		t.Errorf("Mutation.Name = %s, want user.delete", pending[1].Name)
	}
}

func TestCollection_Reactive(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	callCount := 0
	computed := NewComputed(func() int {
		callCount++
		return users.Count()
	})

	// First access
	if computed.Get() != 0 {
		t.Errorf("Computed.Get() = %d, want 0", computed.Get())
	}

	// Create user - should invalidate computed
	users.Create("1", TestUser{Name: "Alice"})

	// Access again - should recompute
	if computed.Get() != 1 {
		t.Errorf("Computed.Get() = %d, want 1", computed.Get())
	}
}

func TestCollection_Name(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	if users.Name() != "user" {
		t.Errorf("Name() = %s, want user", users.Name())
	}
}

func TestClient_Mutate(t *testing.T) {
	client := newTestClient()

	id := client.Mutate("custom.action", map[string]any{
		"key": "value",
	})

	if id == "" {
		t.Error("Mutate should return an ID")
	}

	if client.queue.Len() != 1 {
		t.Errorf("Queue.Len() = %d, want 1", client.queue.Len())
	}

	pending := client.queue.Pending()
	if pending[0].Name != "custom.action" {
		t.Errorf("Mutation.Name = %s, want custom.action", pending[0].Name)
	}
	if pending[0].Args["key"] != "value" {
		t.Error("Mutation.Args not preserved")
	}
}

func TestClient_Accessors(t *testing.T) {
	client := newTestClient()

	if client.Store() == nil {
		t.Error("Store() should not be nil")
	}

	if client.Queue() == nil {
		t.Error("Queue() should not be nil")
	}

	if client.Cursor() != 0 {
		t.Errorf("Cursor() = %d, want 0", client.Cursor())
	}

	if client.IsOnline() {
		t.Error("IsOnline() should be false before start")
	}
}

func TestClient_NotStarted(t *testing.T) {
	client := newTestClient()

	err := client.Sync()
	if err != ErrNotStarted {
		t.Errorf("Sync() error = %v, want ErrNotStarted", err)
	}
}

func TestClient_Stop(t *testing.T) {
	client := newTestClient()

	// Stop should not panic on non-started client
	client.Stop()
}

func TestClient_StartAlreadyStarted(t *testing.T) {
	client := newTestClient()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// First start will fail (no server) but that's okay
	_ = client.Start(ctx)

	err := client.Start(ctx)
	if err != ErrAlreadyStarted {
		t.Errorf("Start() error = %v, want ErrAlreadyStarted", err)
	}

	client.Stop()
}
