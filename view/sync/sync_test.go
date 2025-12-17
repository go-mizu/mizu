package sync

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"
)

// -----------------------------------------------------------------------------
// Signal Tests
// -----------------------------------------------------------------------------

func TestSignal_GetSet(t *testing.T) {
	s := NewSignal(42)

	if got := s.Get(); got != 42 {
		t.Errorf("Get() = %d, want 42", got)
	}

	s.Set(100)

	if got := s.Get(); got != 100 {
		t.Errorf("Get() = %d, want 100", got)
	}
}

func TestSignal_Update(t *testing.T) {
	s := NewSignal(10)

	s.Update(func(v int) int { return v * 2 })

	if got := s.Get(); got != 20 {
		t.Errorf("Get() = %d, want 20", got)
	}
}

func TestComputed_Basic(t *testing.T) {
	count := NewSignal(5)
	doubled := NewComputed(func() int {
		return count.Get() * 2
	})

	if got := doubled.Get(); got != 10 {
		t.Errorf("Computed.Get() = %d, want 10", got)
	}

	count.Set(7)

	if got := doubled.Get(); got != 14 {
		t.Errorf("Computed.Get() = %d, want 14", got)
	}
}

func TestComputed_Caching(t *testing.T) {
	callCount := 0
	count := NewSignal(5)

	doubled := NewComputed(func() int {
		callCount++
		return count.Get() * 2
	})

	_ = doubled.Get()
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}

	_ = doubled.Get()
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (cached)", callCount)
	}

	count.Set(10)
	_ = doubled.Get()
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (recomputed)", callCount)
	}
}

func TestComputed_Chained(t *testing.T) {
	a := NewSignal(2)
	b := NewComputed(func() int { return a.Get() * 2 })
	c := NewComputed(func() int { return b.Get() + 1 })

	if got := c.Get(); got != 5 {
		t.Errorf("c.Get() = %d, want 5", got)
	}

	a.Set(3)

	if got := c.Get(); got != 7 {
		t.Errorf("c.Get() = %d, want 7", got)
	}
}

func TestEffect_Basic(t *testing.T) {
	count := NewSignal(0)
	var effectCount atomic.Int32

	effect := NewEffect(func() {
		_ = count.Get()
		effectCount.Add(1)
	})
	defer effect.Stop()

	if got := effectCount.Load(); got != 1 {
		t.Errorf("effectCount = %d, want 1", got)
	}

	count.Set(1)

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if effectCount.Load() >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := effectCount.Load(); got < 2 {
		t.Errorf("effectCount = %d, want >= 2", got)
	}
}

func TestEffect_Stop(t *testing.T) {
	count := NewSignal(0)
	var effectCount atomic.Int32

	effect := NewEffect(func() {
		_ = count.Get()
		effectCount.Add(1)
	})

	time.Sleep(10 * time.Millisecond)
	before := effectCount.Load()

	effect.Stop()
	count.Set(1)
	time.Sleep(50 * time.Millisecond)

	after := effectCount.Load()
	if after > before {
		t.Errorf("Effect should not run after Stop")
	}
}

func TestSignal_ConcurrentAccess(t *testing.T) {
	s := NewSignal(0)
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			s.Set(i)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = s.Get()
		}
		done <- true
	}()

	<-done
	<-done
}

// -----------------------------------------------------------------------------
// Store Tests
// -----------------------------------------------------------------------------

func TestStore_GetSet(t *testing.T) {
	s := newStore()

	s.set("user", "1", []byte(`{"name":"Alice"}`))

	data, ok := s.get("user", "1")
	if !ok {
		t.Fatal("Get returned false")
	}
	if string(data) != `{"name":"Alice"}` {
		t.Errorf("Get() = %s, want %s", data, `{"name":"Alice"}`)
	}

	_, ok = s.get("user", "999")
	if ok {
		t.Error("Get non-existent should return false")
	}
}

func TestStore_Delete(t *testing.T) {
	s := newStore()

	s.set("user", "1", []byte(`{"name":"Alice"}`))
	s.delete("user", "1")

	_, ok := s.get("user", "1")
	if ok {
		t.Error("Get after delete should return false")
	}
}

func TestStore_Has(t *testing.T) {
	s := newStore()

	if s.has("user", "1") {
		t.Error("Has should return false for non-existent")
	}

	s.set("user", "1", []byte(`{}`))

	if !s.has("user", "1") {
		t.Error("Has should return true for existing")
	}
}

func TestStore_List(t *testing.T) {
	s := newStore()

	s.set("user", "1", []byte(`{}`))
	s.set("user", "2", []byte(`{}`))
	s.set("user", "3", []byte(`{}`))

	ids := s.list("user")
	if len(ids) != 3 {
		t.Errorf("List() len = %d, want 3", len(ids))
	}

	ids = s.list("nonexistent")
	if len(ids) != 0 {
		t.Errorf("List() len = %d, want 0", len(ids))
	}
}

func TestStore_Snapshot(t *testing.T) {
	s := newStore()

	s.set("user", "1", []byte(`{"name":"Alice"}`))
	s.set("post", "1", []byte(`{"title":"Hello"}`))

	snap := s.snapshot()

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
	s := newStore()

	data := map[string]map[string][]byte{
		"user": {
			"1": []byte(`{"name":"Alice"}`),
			"2": []byte(`{"name":"Bob"}`),
		},
	}

	s.load(data)

	if s.count("user") != 2 {
		t.Errorf("Count() = %d, want 2", s.count("user"))
	}

	got, ok := s.get("user", "1")
	if !ok || string(got) != `{"name":"Alice"}` {
		t.Errorf("Get after Load failed")
	}
}

func TestStore_Version(t *testing.T) {
	s := newStore()

	v1 := s.version.Get()
	s.set("user", "1", []byte(`{}`))
	v2 := s.version.Get()

	if v2 <= v1 {
		t.Error("Version should increase after Set")
	}

	s.delete("user", "1")
	v3 := s.version.Get()

	if v3 <= v2 {
		t.Error("Version should increase after Delete")
	}
}

func TestStore_DataCopy(t *testing.T) {
	s := newStore()

	original := []byte(`{"name":"Alice"}`)
	s.set("user", "1", original)

	original[0] = 'X'

	got, _ := s.get("user", "1")
	if got[0] == 'X' {
		t.Error("Store should make a copy of data")
	}

	got[0] = 'Y'

	got2, _ := s.get("user", "1")
	if got2[0] == 'Y' {
		t.Error("Store should return a copy")
	}
}

// -----------------------------------------------------------------------------
// Queue Tests
// -----------------------------------------------------------------------------

func TestQueue_Push(t *testing.T) {
	q := newQueue()

	id := q.push(mutation{Name: "test"})

	if id == "" {
		t.Error("Push should return an ID")
	}

	if q.len() != 1 {
		t.Errorf("Len() = %d, want 1", q.len())
	}
}

func TestQueue_PushWithID(t *testing.T) {
	q := newQueue()

	id := q.push(mutation{ID: "custom-id", Name: "test"})

	if id != "custom-id" {
		t.Errorf("Push() = %s, want custom-id", id)
	}
}

func TestQueue_PushDuplicate(t *testing.T) {
	q := newQueue()

	q.push(mutation{ID: "dup", Name: "test1"})
	q.push(mutation{ID: "dup", Name: "test2"})

	if q.len() != 1 {
		t.Errorf("Len() = %d, want 1 (no duplicate)", q.len())
	}
}

func TestQueue_Pending(t *testing.T) {
	q := newQueue()

	q.push(mutation{Name: "a"})
	q.push(mutation{Name: "b"})
	q.push(mutation{Name: "c"})

	pending := q.pending()
	if len(pending) != 3 {
		t.Errorf("Pending() len = %d, want 3", len(pending))
	}

	if pending[0].Name != "a" || pending[1].Name != "b" || pending[2].Name != "c" {
		t.Error("Pending should preserve order")
	}

	if pending[0].Seq != 1 || pending[1].Seq != 2 || pending[2].Seq != 3 {
		t.Error("Pending should have correct sequence numbers")
	}
}

func TestQueue_Clear(t *testing.T) {
	q := newQueue()

	q.push(mutation{Name: "a"})
	q.push(mutation{Name: "b"})

	q.clear()

	if q.len() != 0 {
		t.Errorf("Len() = %d, want 0", q.len())
	}
}

func TestQueue_Remove(t *testing.T) {
	q := newQueue()

	q.push(mutation{ID: "a", Name: "a"})
	q.push(mutation{ID: "b", Name: "b"})
	q.push(mutation{ID: "c", Name: "c"})

	q.remove("b")

	if q.len() != 2 {
		t.Errorf("Len() = %d, want 2", q.len())
	}

	pending := q.pending()
	if pending[0].ID != "a" || pending[1].ID != "c" {
		t.Error("Remove should preserve remaining order")
	}
}

func TestQueue_ClientID(t *testing.T) {
	q := newQueue()

	id1 := q.getClientID()
	if id1 == "" {
		t.Error("ClientID should not be empty")
	}

	q.setClientID("custom-client")

	if q.getClientID() != "custom-client" {
		t.Errorf("ClientID() = %s, want custom-client", q.getClientID())
	}
}

func TestQueue_MutationHasClientID(t *testing.T) {
	q := newQueue()
	q.setClientID("my-client")

	q.push(mutation{Name: "test"})

	pending := q.pending()
	if pending[0].Client != "my-client" {
		t.Errorf("Mutation.Client = %s, want my-client", pending[0].Client)
	}
}

func TestQueue_CurrentSeq(t *testing.T) {
	q := newQueue()

	if q.currentSeq() != 0 {
		t.Errorf("CurrentSeq() = %d, want 0", q.currentSeq())
	}

	q.push(mutation{Name: "a"})
	q.push(mutation{Name: "b"})

	if q.currentSeq() != 2 {
		t.Errorf("CurrentSeq() = %d, want 2", q.currentSeq())
	}
}

func TestQueue_Load(t *testing.T) {
	q := newQueue()

	mutations := []mutation{
		{ID: "a", Name: "a", Seq: 10},
		{ID: "b", Name: "b", Seq: 11},
	}

	q.loadMutations(mutations)

	if q.len() != 2 {
		t.Errorf("Len() = %d, want 2", q.len())
	}

	if q.currentSeq() != 11 {
		t.Errorf("CurrentSeq() = %d, want 11", q.currentSeq())
	}

	q.push(mutation{Name: "c"})
	pending := q.pending()
	if pending[2].Seq != 12 {
		t.Errorf("New mutation seq = %d, want 12", pending[2].Seq)
	}
}

func TestQueue_CreatedAt(t *testing.T) {
	q := newQueue()

	q.push(mutation{Name: "test"})

	pending := q.pending()
	if pending[0].CreatedAt.IsZero() {
		t.Error("CreatedAt should be set automatically")
	}
}

// -----------------------------------------------------------------------------
// Entity/Collection Tests
// -----------------------------------------------------------------------------

type TestUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func newTestClient() *Client {
	return New(Options{
		BaseURL: "http://localhost:8080/_sync",
		Scope:   "test",
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

	data, _ := json.Marshal(TestUser{Name: "Bob"})
	client.store.set("user", "1", data)

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

	if client.queue.len() != 1 {
		t.Errorf("Queue.Len() = %d, want 1", client.queue.len())
	}

	pending := client.queue.pending()
	if pending[0].Name != "user.create" {
		t.Errorf("Mutation.Name = %s, want user.create", pending[0].Name)
	}
}

func TestEntity_SetQueuesMutation(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	user := users.Create("1", TestUser{Name: "Alice"})
	user.Set(TestUser{Name: "Alicia"})

	if client.queue.len() != 2 {
		t.Errorf("Queue.Len() = %d, want 2", client.queue.len())
	}

	pending := client.queue.pending()
	if pending[1].Name != "user.update" {
		t.Errorf("Mutation.Name = %s, want user.update", pending[1].Name)
	}
}

func TestEntity_DeleteQueuesMutation(t *testing.T) {
	client := newTestClient()
	users := NewCollection[TestUser](client, "user")

	user := users.Create("1", TestUser{Name: "Alice"})
	user.Delete()

	if client.queue.len() != 2 {
		t.Errorf("Queue.Len() = %d, want 2", client.queue.len())
	}

	pending := client.queue.pending()
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

	if computed.Get() != 0 {
		t.Errorf("Computed.Get() = %d, want 0", computed.Get())
	}

	users.Create("1", TestUser{Name: "Alice"})

	if computed.Get() != 1 {
		t.Errorf("Computed.Get() = %d, want 1", computed.Get())
	}
}

// -----------------------------------------------------------------------------
// Client Tests
// -----------------------------------------------------------------------------

func TestClient_Mutate(t *testing.T) {
	client := newTestClient()

	id := client.Mutate("custom.action", map[string]any{
		"key": "value",
	})

	if id == "" {
		t.Error("Mutate should return an ID")
	}

	if client.queue.len() != 1 {
		t.Errorf("Queue.Len() = %d, want 1", client.queue.len())
	}

	pending := client.queue.pending()
	if pending[0].Name != "custom.action" {
		t.Errorf("Mutation.Name = %s, want custom.action", pending[0].Name)
	}
	if pending[0].Args["key"] != "value" {
		t.Error("Mutation.Args not preserved")
	}
}

func TestClient_Accessors(t *testing.T) {
	client := newTestClient()

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
	client.Stop()
}

func TestClient_StartAlreadyStarted(t *testing.T) {
	client := newTestClient()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = client.Start(ctx)

	err := client.Start(ctx)
	if err != ErrAlreadyStarted {
		t.Errorf("Start() error = %v, want ErrAlreadyStarted", err)
	}

	client.Stop()
}
