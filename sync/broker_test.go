package sync

import (
	"sync"
	"testing"
)

func TestNopBroker(t *testing.T) {
	b := NopBroker{}
	// Should not panic
	b.Poke("scope", 1)
	b.Poke("scope", 2)
}

func TestNopBroker_Interface(t *testing.T) {
	var _ PokeBroker = NopBroker{}
}

func TestFuncBroker(t *testing.T) {
	var calls []Poke

	b := FuncBroker(func(scope string, cursor uint64) {
		calls = append(calls, Poke{Scope: scope, Cursor: cursor})
	})

	b.Poke("scope1", 1)
	b.Poke("scope2", 2)

	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}

	if calls[0].Scope != "scope1" || calls[0].Cursor != 1 {
		t.Errorf("unexpected first call: %+v", calls[0])
	}
	if calls[1].Scope != "scope2" || calls[1].Cursor != 2 {
		t.Errorf("unexpected second call: %+v", calls[1])
	}
}

func TestFuncBroker_Interface(t *testing.T) {
	var _ PokeBroker = FuncBroker(func(scope string, cursor uint64) {})
}

func TestMultiBroker(t *testing.T) {
	var calls1, calls2 []Poke

	b1 := FuncBroker(func(scope string, cursor uint64) {
		calls1 = append(calls1, Poke{Scope: scope, Cursor: cursor})
	})
	b2 := FuncBroker(func(scope string, cursor uint64) {
		calls2 = append(calls2, Poke{Scope: scope, Cursor: cursor})
	})

	mb := NewMultiBroker(b1, b2)
	mb.Poke("scope", 1)

	if len(calls1) != 1 {
		t.Errorf("expected 1 call to broker1, got %d", len(calls1))
	}
	if len(calls2) != 1 {
		t.Errorf("expected 1 call to broker2, got %d", len(calls2))
	}
}

func TestMultiBroker_Add(t *testing.T) {
	var calls []int

	mb := NewMultiBroker()

	mb.Add(FuncBroker(func(scope string, cursor uint64) {
		calls = append(calls, 1)
	}))
	mb.Add(FuncBroker(func(scope string, cursor uint64) {
		calls = append(calls, 2)
	}))

	mb.Poke("scope", 1)

	if len(calls) != 2 {
		t.Errorf("expected 2 calls, got %d", len(calls))
	}
}

func TestMultiBroker_Empty(t *testing.T) {
	mb := NewMultiBroker()
	// Should not panic
	mb.Poke("scope", 1)
}

func TestPoke_Fields(t *testing.T) {
	p := Poke{
		Scope:  "user:123",
		Cursor: 42,
	}

	if p.Scope != "user:123" {
		t.Errorf("expected scope user:123, got %s", p.Scope)
	}
	if p.Cursor != 42 {
		t.Errorf("expected cursor 42, got %d", p.Cursor)
	}
}

// MockBroker for testing - records all pokes
type MockBroker struct {
	mu    sync.Mutex
	pokes []Poke
}

func (m *MockBroker) Poke(scope string, cursor uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pokes = append(m.pokes, Poke{Scope: scope, Cursor: cursor})
}

func (m *MockBroker) Pokes() []Poke {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Poke{}, m.pokes...)
}

func TestMockBroker(t *testing.T) {
	m := &MockBroker{}

	m.Poke("scope1", 1)
	m.Poke("scope2", 2)

	pokes := m.Pokes()
	if len(pokes) != 2 {
		t.Fatalf("expected 2 pokes, got %d", len(pokes))
	}
}
