package session

import (
	"context"
	"database/sql"
	"testing"
)

// mockStore implements Store interface for testing.
type mockStore struct {
	sessions map[string]*Session
	messages map[string][]Message
}

func newMockStore() *mockStore {
	return &mockStore{
		sessions: make(map[string]*Session),
		messages: make(map[string][]Message),
	}
}

func (m *mockStore) Create(ctx context.Context, s *Session) error {
	m.sessions[s.ID] = s
	m.messages[s.ID] = []Message{}
	return nil
}

func (m *mockStore) Get(ctx context.Context, id string) (*Session, error) {
	if s, ok := m.sessions[id]; ok {
		return s, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockStore) List(ctx context.Context, limit, offset int) ([]Session, int, error) {
	var result []Session
	for _, s := range m.sessions {
		result = append(result, *s)
	}
	total := len(result)
	if offset >= len(result) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (m *mockStore) Update(ctx context.Context, s *Session) error {
	if _, ok := m.sessions[s.ID]; !ok {
		return sql.ErrNoRows
	}
	m.sessions[s.ID] = s
	return nil
}

func (m *mockStore) Delete(ctx context.Context, id string) error {
	delete(m.sessions, id)
	delete(m.messages, id)
	return nil
}

func (m *mockStore) AddMessage(ctx context.Context, sessionID string, msg *Message) error {
	m.messages[sessionID] = append(m.messages[sessionID], *msg)
	return nil
}

func (m *mockStore) GetMessages(ctx context.Context, sessionID string) ([]Message, error) {
	return m.messages[sessionID], nil
}

func TestService_Create(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	sess, err := svc.Create(ctx, "Test Session")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if sess.ID == "" {
		t.Error("Create() ID should not be empty")
	}
	if sess.Title != "Test Session" {
		t.Errorf("Create() Title = %v, want Test Session", sess.Title)
	}

	// Verify session was stored
	got, err := svc.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != sess.ID {
		t.Errorf("Get() ID = %v, want %v", got.ID, sess.ID)
	}
}

func TestService_Create_DefaultTitle(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	sess, err := svc.Create(ctx, "")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if sess.Title != "New Research Session" {
		t.Errorf("Create() default Title = %v, want New Research Session", sess.Title)
	}
}

func TestService_List(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	// Create sessions
	for i := 0; i < 5; i++ {
		svc.Create(ctx, "Session")
	}

	sessions, total, err := svc.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 5 {
		t.Errorf("List() total = %v, want 5", total)
	}
	if len(sessions) != 5 {
		t.Errorf("List() len = %v, want 5", len(sessions))
	}
}

func TestService_Delete(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	sess, _ := svc.Create(ctx, "Test")
	if err := svc.Delete(ctx, sess.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := svc.Get(ctx, sess.ID)
	if err != sql.ErrNoRows {
		t.Errorf("Get() after delete error = %v, want sql.ErrNoRows", err)
	}
}

func TestService_AddMessage(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	sess, _ := svc.Create(ctx, "Test")

	msg, err := svc.AddMessage(ctx, sess.ID, "user", "Hello", "", nil)
	if err != nil {
		t.Fatalf("AddMessage() error = %v", err)
	}

	if msg.Role != "user" {
		t.Errorf("AddMessage() Role = %v, want user", msg.Role)
	}
	if msg.Content != "Hello" {
		t.Errorf("AddMessage() Content = %v, want Hello", msg.Content)
	}

	// Verify session was updated
	got, _ := svc.Get(ctx, sess.ID)
	if len(got.Messages) != 1 {
		t.Errorf("Session messages len = %v, want 1", len(got.Messages))
	}
}

func TestService_AddMessage_WithCitations(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	sess, _ := svc.Create(ctx, "Test")

	citations := []Citation{
		{Index: 1, URL: "https://example.com", Title: "Example"},
	}
	msg, err := svc.AddMessage(ctx, sess.ID, "assistant", "Response", "quick", citations)
	if err != nil {
		t.Fatalf("AddMessage() error = %v", err)
	}

	if msg.Mode != "quick" {
		t.Errorf("AddMessage() Mode = %v, want quick", msg.Mode)
	}
	if len(msg.Citations) != 1 {
		t.Errorf("AddMessage() Citations len = %v, want 1", len(msg.Citations))
	}
}

func TestService_GetConversationContext(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	sess, _ := svc.Create(ctx, "Test")
	svc.AddMessage(ctx, sess.ID, "user", "Hello", "", nil)
	svc.AddMessage(ctx, sess.ID, "assistant", "Hi there!", "quick", nil)
	svc.AddMessage(ctx, sess.ID, "user", "How are you?", "", nil)

	convCtx, err := svc.GetConversationContext(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetConversationContext() error = %v", err)
	}

	if len(convCtx) != 3 {
		t.Errorf("GetConversationContext() len = %v, want 3", len(convCtx))
	}

	// Verify order
	if convCtx[0]["role"] != "user" {
		t.Errorf("First message role = %v, want user", convCtx[0]["role"])
	}
	if convCtx[1]["role"] != "assistant" {
		t.Errorf("Second message role = %v, want assistant", convCtx[1]["role"])
	}
}

func TestCitation_Marshal(t *testing.T) {
	citations := []Citation{
		{Index: 1, URL: "https://example.com", Title: "Example", Snippet: "Test snippet"},
	}

	json := MarshalCitations(citations)
	got := UnmarshalCitations(json)

	if len(got) != 1 {
		t.Errorf("UnmarshalCitations len = %v, want 1", len(got))
	}
	if got[0].Index != 1 {
		t.Errorf("Citation Index = %v, want 1", got[0].Index)
	}
	if got[0].URL != "https://example.com" {
		t.Errorf("Citation URL = %v, want https://example.com", got[0].URL)
	}
}
