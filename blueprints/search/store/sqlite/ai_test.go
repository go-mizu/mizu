package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/feature/canvas"
	"github.com/go-mizu/mizu/blueprints/search/feature/chunker"
	"github.com/go-mizu/mizu/blueprints/search/feature/session"
)

func TestSessionStore_CRUD(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	store := s.Session()

	// Create session
	sess := &session.Session{
		ID:        "test-session-1",
		Title:     "Test Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.Create(ctx, sess); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get session
	got, err := store.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != sess.ID {
		t.Errorf("Get() ID = %v, want %v", got.ID, sess.ID)
	}
	if got.Title != sess.Title {
		t.Errorf("Get() Title = %v, want %v", got.Title, sess.Title)
	}

	// Update session
	sess.Title = "Updated Title"
	sess.UpdatedAt = time.Now()
	if err := store.Update(ctx, sess); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err = store.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get() after update error = %v", err)
	}
	if got.Title != "Updated Title" {
		t.Errorf("Get() Title = %v, want Updated Title", got.Title)
	}

	// Delete session
	if err := store.Delete(ctx, sess.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.Get(ctx, sess.ID)
	if err == nil {
		t.Error("Get() after delete should return error")
	}
}

func TestSessionStore_List(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	store := s.Session()

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		sess := &session.Session{
			ID:        "session-" + string(rune('a'+i)),
			Title:     "Session " + string(rune('A'+i)),
			CreatedAt: time.Now().Add(time.Duration(-i) * time.Hour),
			UpdatedAt: time.Now().Add(time.Duration(-i) * time.Hour),
		}
		if err := store.Create(ctx, sess); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// List sessions
	sessions, total, err := store.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 5 {
		t.Errorf("List() total = %v, want 5", total)
	}
	if len(sessions) != 5 {
		t.Errorf("List() len = %v, want 5", len(sessions))
	}

	// List with pagination
	sessions, total, err = store.List(ctx, 2, 0)
	if err != nil {
		t.Fatalf("List() with limit error = %v", err)
	}
	if total != 5 {
		t.Errorf("List() total = %v, want 5", total)
	}
	if len(sessions) != 2 {
		t.Errorf("List() len = %v, want 2", len(sessions))
	}

	sessions, _, err = store.List(ctx, 2, 2)
	if err != nil {
		t.Fatalf("List() with offset error = %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("List() with offset len = %v, want 2", len(sessions))
	}
}

func TestSessionStore_Messages(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	store := s.Session()

	// Create session first
	sess := &session.Session{
		ID:        "test-session-msg",
		Title:     "Message Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Create(ctx, sess); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Add messages
	msg1 := &session.Message{
		ID:        "msg-1",
		SessionID: sess.ID,
		Role:      "user",
		Content:   "Hello",
		CreatedAt: time.Now(),
	}
	if err := store.AddMessage(ctx, sess.ID, msg1); err != nil {
		t.Fatalf("AddMessage() error = %v", err)
	}

	msg2 := &session.Message{
		ID:        "msg-2",
		SessionID: sess.ID,
		Role:      "assistant",
		Content:   "Hi there!",
		Mode:      "quick",
		Citations: []session.Citation{
			{Index: 1, URL: "https://example.com", Title: "Example"},
		},
		CreatedAt: time.Now(),
	}
	if err := store.AddMessage(ctx, sess.ID, msg2); err != nil {
		t.Fatalf("AddMessage() error = %v", err)
	}

	// Get messages
	messages, err := store.GetMessages(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}
	if len(messages) != 2 {
		t.Errorf("GetMessages() len = %v, want 2", len(messages))
	}

	// Verify message content
	if messages[0].Role != "user" {
		t.Errorf("Message[0].Role = %v, want user", messages[0].Role)
	}
	if messages[1].Role != "assistant" {
		t.Errorf("Message[1].Role = %v, want assistant", messages[1].Role)
	}
	if len(messages[1].Citations) != 1 {
		t.Errorf("Message[1].Citations len = %v, want 1", len(messages[1].Citations))
	}
}

func TestCanvasStore_CRUD(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	sessStore := s.Session()
	canvStore := s.Canvas()

	// Create session first (canvas references session)
	sess := &session.Session{
		ID:        "canvas-session",
		Title:     "Canvas Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := sessStore.Create(ctx, sess); err != nil {
		t.Fatalf("Session Create() error = %v", err)
	}

	// Create canvas
	canv := &canvas.Canvas{
		ID:        "test-canvas-1",
		SessionID: sess.ID,
		Title:     "My Canvas",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := canvStore.Create(ctx, canv); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get canvas
	got, err := canvStore.Get(ctx, canv.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Title != canv.Title {
		t.Errorf("Get() Title = %v, want %v", got.Title, canv.Title)
	}

	// Get by session ID
	got, err = canvStore.GetBySessionID(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetBySessionID() error = %v", err)
	}
	if got.ID != canv.ID {
		t.Errorf("GetBySessionID() ID = %v, want %v", got.ID, canv.ID)
	}

	// Update canvas
	canv.Title = "Updated Canvas"
	canv.UpdatedAt = time.Now()
	if err := canvStore.Update(ctx, canv); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err = canvStore.Get(ctx, canv.ID)
	if err != nil {
		t.Fatalf("Get() after update error = %v", err)
	}
	if got.Title != "Updated Canvas" {
		t.Errorf("Get() Title = %v, want Updated Canvas", got.Title)
	}

	// Delete canvas
	if err := canvStore.Delete(ctx, canv.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = canvStore.Get(ctx, canv.ID)
	if err == nil {
		t.Error("Get() after delete should return error")
	}
}

func TestCanvasStore_Blocks(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	sessStore := s.Session()
	canvStore := s.Canvas()

	// Create session and canvas
	sess := &session.Session{
		ID:        "blocks-session",
		Title:     "Blocks Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := sessStore.Create(ctx, sess); err != nil {
		t.Fatalf("Session Create() error = %v", err)
	}

	canv := &canvas.Canvas{
		ID:        "blocks-canvas",
		SessionID: sess.ID,
		Title:     "Blocks Canvas",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := canvStore.Create(ctx, canv); err != nil {
		t.Fatalf("Canvas Create() error = %v", err)
	}

	// Add blocks
	block1 := &canvas.Block{
		ID:        "block-1",
		CanvasID:  canv.ID,
		Type:      canvas.BlockTypeText,
		Content:   "Hello world",
		Order:     0,
		CreatedAt: time.Now(),
	}
	if err := canvStore.AddBlock(ctx, canv.ID, block1); err != nil {
		t.Fatalf("AddBlock() error = %v", err)
	}

	block2 := &canvas.Block{
		ID:        "block-2",
		CanvasID:  canv.ID,
		Type:      canvas.BlockTypeHeading,
		Content:   "Title",
		Order:     1,
		CreatedAt: time.Now(),
	}
	if err := canvStore.AddBlock(ctx, canv.ID, block2); err != nil {
		t.Fatalf("AddBlock() error = %v", err)
	}

	// Get blocks
	blocks, err := canvStore.GetBlocks(ctx, canv.ID)
	if err != nil {
		t.Fatalf("GetBlocks() error = %v", err)
	}
	if len(blocks) != 2 {
		t.Errorf("GetBlocks() len = %v, want 2", len(blocks))
	}

	// Verify order
	if blocks[0].Order != 0 {
		t.Errorf("Block[0].Order = %v, want 0", blocks[0].Order)
	}
	if blocks[1].Order != 1 {
		t.Errorf("Block[1].Order = %v, want 1", blocks[1].Order)
	}

	// Update block
	block1.Content = "Updated content"
	if err := canvStore.UpdateBlock(ctx, block1); err != nil {
		t.Fatalf("UpdateBlock() error = %v", err)
	}

	blocks, _ = canvStore.GetBlocks(ctx, canv.ID)
	if blocks[0].Content != "Updated content" {
		t.Errorf("Block content = %v, want Updated content", blocks[0].Content)
	}

	// Reorder blocks
	if err := canvStore.ReorderBlocks(ctx, canv.ID, []string{"block-2", "block-1"}); err != nil {
		t.Fatalf("ReorderBlocks() error = %v", err)
	}

	blocks, _ = canvStore.GetBlocks(ctx, canv.ID)
	if blocks[0].ID != "block-2" {
		t.Errorf("After reorder, Block[0].ID = %v, want block-2", blocks[0].ID)
	}

	// Delete block
	if err := canvStore.DeleteBlock(ctx, block1.ID); err != nil {
		t.Fatalf("DeleteBlock() error = %v", err)
	}

	blocks, _ = canvStore.GetBlocks(ctx, canv.ID)
	if len(blocks) != 1 {
		t.Errorf("GetBlocks() after delete len = %v, want 1", len(blocks))
	}
}

func TestChunkerStore_Documents(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	store := s.Chunker()

	// Create document with chunks
	doc := &chunker.Document{
		ID:        "doc-1",
		URL:       "https://example.com/page",
		Title:     "Example Page",
		Content:   "This is the full content of the page.",
		FetchedAt: time.Now(),
		Chunks: []chunker.Chunk{
			{
				ID:         "chunk-1",
				DocumentID: "doc-1",
				URL:        "https://example.com/page",
				Text:       "This is the full",
				StartPos:   0,
				EndPos:     16,
			},
			{
				ID:         "chunk-2",
				DocumentID: "doc-1",
				URL:        "https://example.com/page",
				Text:       "content of the page.",
				StartPos:   17,
				EndPos:     37,
			},
		},
	}

	if err := store.SaveDocument(ctx, doc); err != nil {
		t.Fatalf("SaveDocument() error = %v", err)
	}

	// Get document
	got, err := store.GetDocument(ctx, doc.URL)
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}
	if got.Title != doc.Title {
		t.Errorf("GetDocument() Title = %v, want %v", got.Title, doc.Title)
	}

	// Get chunks
	chunks, err := store.GetChunks(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetChunks() error = %v", err)
	}
	if len(chunks) != 2 {
		t.Errorf("GetChunks() len = %v, want 2", len(chunks))
	}

	// Update document (upsert)
	doc.Title = "Updated Title"
	doc.Chunks = []chunker.Chunk{
		{
			ID:         "chunk-3",
			DocumentID: "doc-1",
			URL:        "https://example.com/page",
			Text:       "New content",
			StartPos:   0,
			EndPos:     11,
		},
	}
	if err := store.SaveDocument(ctx, doc); err != nil {
		t.Fatalf("SaveDocument() update error = %v", err)
	}

	chunks, _ = store.GetChunks(ctx, doc.ID)
	if len(chunks) != 1 {
		t.Errorf("GetChunks() after update len = %v, want 1", len(chunks))
	}
}

func TestChunkerStore_SearchChunks(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	store := s.Chunker()

	// Create documents with chunks
	for i := 0; i < 3; i++ {
		doc := &chunker.Document{
			ID:        "doc-" + string(rune('a'+i)),
			URL:       "https://example.com/page" + string(rune('a'+i)),
			Title:     "Page " + string(rune('A'+i)),
			Content:   "Content " + string(rune('A'+i)),
			FetchedAt: time.Now().Add(time.Duration(-i) * time.Hour),
			Chunks: []chunker.Chunk{
				{
					ID:         "chunk-" + string(rune('a'+i)),
					DocumentID: "doc-" + string(rune('a'+i)),
					URL:        "https://example.com/page" + string(rune('a'+i)),
					Text:       "Content " + string(rune('A'+i)),
				},
			},
		}
		if err := store.SaveDocument(ctx, doc); err != nil {
			t.Fatalf("SaveDocument() error = %v", err)
		}
	}

	// Search chunks (currently returns recent chunks)
	chunks, err := store.SearchChunks(ctx, nil, 10)
	if err != nil {
		t.Fatalf("SearchChunks() error = %v", err)
	}
	if len(chunks) != 3 {
		t.Errorf("SearchChunks() len = %v, want 3", len(chunks))
	}

	// Search with limit
	chunks, err = store.SearchChunks(ctx, nil, 2)
	if err != nil {
		t.Fatalf("SearchChunks() with limit error = %v", err)
	}
	if len(chunks) != 2 {
		t.Errorf("SearchChunks() with limit len = %v, want 2", len(chunks))
	}
}

func TestChunkerStore_DeleteOldDocuments(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	store := s.Chunker()

	// Create old and new documents
	oldDoc := &chunker.Document{
		ID:        "old-doc",
		URL:       "https://example.com/old",
		Title:     "Old Page",
		Content:   "Old content",
		FetchedAt: time.Now().Add(-48 * time.Hour),
	}
	if err := store.SaveDocument(ctx, oldDoc); err != nil {
		t.Fatalf("SaveDocument() old error = %v", err)
	}

	newDoc := &chunker.Document{
		ID:        "new-doc",
		URL:       "https://example.com/new",
		Title:     "New Page",
		Content:   "New content",
		FetchedAt: time.Now(),
	}
	if err := store.SaveDocument(ctx, newDoc); err != nil {
		t.Fatalf("SaveDocument() new error = %v", err)
	}

	// Delete documents older than 24 hours
	if err := store.DeleteOldDocuments(ctx, 24*time.Hour); err != nil {
		t.Fatalf("DeleteOldDocuments() error = %v", err)
	}

	// Old document should be gone
	_, err := store.GetDocument(ctx, oldDoc.URL)
	if err == nil {
		t.Error("GetDocument() old should return error after cleanup")
	}

	// New document should still exist
	got, err := store.GetDocument(ctx, newDoc.URL)
	if err != nil {
		t.Fatalf("GetDocument() new error = %v", err)
	}
	if got.ID != newDoc.ID {
		t.Errorf("GetDocument() new ID = %v, want %v", got.ID, newDoc.ID)
	}
}

func TestChunkerStore_SaveChunk(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	store := s.Chunker()

	// Create document first
	doc := &chunker.Document{
		ID:        "chunk-test-doc",
		URL:       "https://example.com/chunk-test",
		Title:     "Chunk Test",
		Content:   "Test content",
		FetchedAt: time.Now(),
	}
	if err := store.SaveDocument(ctx, doc); err != nil {
		t.Fatalf("SaveDocument() error = %v", err)
	}

	// Save individual chunk
	chunk := &chunker.Chunk{
		ID:         "individual-chunk",
		DocumentID: doc.ID,
		URL:        doc.URL,
		Text:       "Individual chunk text",
		Embedding:  []float32{0.1, 0.2, 0.3},
		StartPos:   0,
		EndPos:     21,
	}
	if err := store.SaveChunk(ctx, chunk); err != nil {
		t.Fatalf("SaveChunk() error = %v", err)
	}

	// Verify chunk was saved
	chunks, err := store.GetChunks(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetChunks() error = %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("GetChunks() len = %v, want 1", len(chunks))
	}
	if len(chunks[0].Embedding) != 3 {
		t.Errorf("Chunk embedding len = %v, want 3", len(chunks[0].Embedding))
	}
}

func TestAIStores_CascadeDelete(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()
	sessStore := s.Session()
	canvStore := s.Canvas()

	// Create session with messages
	sess := &session.Session{
		ID:        "cascade-session",
		Title:     "Cascade Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := sessStore.Create(ctx, sess); err != nil {
		t.Fatalf("Session Create() error = %v", err)
	}

	msg := &session.Message{
		ID:        "cascade-msg",
		SessionID: sess.ID,
		Role:      "user",
		Content:   "Test message",
		CreatedAt: time.Now(),
	}
	if err := sessStore.AddMessage(ctx, sess.ID, msg); err != nil {
		t.Fatalf("AddMessage() error = %v", err)
	}

	// Create canvas with blocks
	canv := &canvas.Canvas{
		ID:        "cascade-canvas",
		SessionID: sess.ID,
		Title:     "Cascade Canvas",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := canvStore.Create(ctx, canv); err != nil {
		t.Fatalf("Canvas Create() error = %v", err)
	}

	block := &canvas.Block{
		ID:        "cascade-block",
		CanvasID:  canv.ID,
		Type:      canvas.BlockTypeText,
		Content:   "Test block",
		Order:     0,
		CreatedAt: time.Now(),
	}
	if err := canvStore.AddBlock(ctx, canv.ID, block); err != nil {
		t.Fatalf("AddBlock() error = %v", err)
	}

	// Delete session - should cascade to messages and canvas/blocks
	if err := sessStore.Delete(ctx, sess.ID); err != nil {
		t.Fatalf("Session Delete() error = %v", err)
	}

	// Verify session is gone
	_, err := sessStore.Get(ctx, sess.ID)
	if err == nil {
		t.Error("Session should be deleted")
	}

	// Verify messages are gone
	messages, err := sessStore.GetMessages(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}
	if len(messages) != 0 {
		t.Error("Messages should be cascade deleted")
	}

	// Verify canvas is gone
	_, err = canvStore.GetBySessionID(ctx, sess.ID)
	if err == nil {
		t.Error("Canvas should be cascade deleted")
	}
}
