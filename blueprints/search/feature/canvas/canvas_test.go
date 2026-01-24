package canvas

import (
	"context"
	"database/sql"
	"testing"
)

// mockStore implements Store interface for testing.
type mockStore struct {
	canvases map[string]*Canvas
	blocks   map[string][]Block
}

func newMockStore() *mockStore {
	return &mockStore{
		canvases: make(map[string]*Canvas),
		blocks:   make(map[string][]Block),
	}
}

func (m *mockStore) Create(ctx context.Context, c *Canvas) error {
	m.canvases[c.ID] = c
	m.blocks[c.ID] = []Block{}
	return nil
}

func (m *mockStore) Get(ctx context.Context, id string) (*Canvas, error) {
	if c, ok := m.canvases[id]; ok {
		return c, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockStore) GetBySessionID(ctx context.Context, sessionID string) (*Canvas, error) {
	for _, c := range m.canvases {
		if c.SessionID == sessionID {
			return c, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockStore) Update(ctx context.Context, c *Canvas) error {
	if _, ok := m.canvases[c.ID]; !ok {
		return sql.ErrNoRows
	}
	m.canvases[c.ID] = c
	return nil
}

func (m *mockStore) Delete(ctx context.Context, id string) error {
	delete(m.canvases, id)
	delete(m.blocks, id)
	return nil
}

func (m *mockStore) AddBlock(ctx context.Context, canvasID string, b *Block) error {
	m.blocks[canvasID] = append(m.blocks[canvasID], *b)
	return nil
}

func (m *mockStore) UpdateBlock(ctx context.Context, b *Block) error {
	blocks := m.blocks[b.CanvasID]
	for i := range blocks {
		if blocks[i].ID == b.ID {
			blocks[i] = *b
			m.blocks[b.CanvasID] = blocks
			return nil
		}
	}
	return sql.ErrNoRows
}

func (m *mockStore) DeleteBlock(ctx context.Context, blockID string) error {
	for canvasID, blocks := range m.blocks {
		for i, b := range blocks {
			if b.ID == blockID {
				m.blocks[canvasID] = append(blocks[:i], blocks[i+1:]...)
				return nil
			}
		}
	}
	return nil
}

func (m *mockStore) GetBlocks(ctx context.Context, canvasID string) ([]Block, error) {
	return m.blocks[canvasID], nil
}

func (m *mockStore) ReorderBlocks(ctx context.Context, canvasID string, blockIDs []string) error {
	blocks := m.blocks[canvasID]
	newOrder := make([]Block, 0, len(blocks))
	for order, id := range blockIDs {
		for _, b := range blocks {
			if b.ID == id {
				b.Order = order
				newOrder = append(newOrder, b)
				break
			}
		}
	}
	m.blocks[canvasID] = newOrder
	return nil
}

func TestService_Create(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	canv, err := svc.Create(ctx, "session-1", "My Canvas")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if canv.ID == "" {
		t.Error("Create() ID should not be empty")
	}
	if canv.SessionID != "session-1" {
		t.Errorf("Create() SessionID = %v, want session-1", canv.SessionID)
	}
	if canv.Title != "My Canvas" {
		t.Errorf("Create() Title = %v, want My Canvas", canv.Title)
	}
}

func TestService_GetBySessionID(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	canv, _ := svc.Create(ctx, "session-1", "Test")

	got, err := svc.GetBySessionID(ctx, "session-1")
	if err != nil {
		t.Fatalf("GetBySessionID() error = %v", err)
	}
	if got.ID != canv.ID {
		t.Errorf("GetBySessionID() ID = %v, want %v", got.ID, canv.ID)
	}
}

func TestService_AddBlock(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	canv, _ := svc.Create(ctx, "session-1", "Test")

	block, err := svc.AddBlock(ctx, canv.ID, BlockTypeText, "Hello world", 0)
	if err != nil {
		t.Fatalf("AddBlock() error = %v", err)
	}

	if block.ID == "" {
		t.Error("AddBlock() ID should not be empty")
	}
	if block.Type != BlockTypeText {
		t.Errorf("AddBlock() Type = %v, want text", block.Type)
	}
	if block.Content != "Hello world" {
		t.Errorf("AddBlock() Content = %v, want Hello world", block.Content)
	}
}

func TestService_GetWithBlocks(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	canv, _ := svc.Create(ctx, "session-1", "Test")
	svc.AddBlock(ctx, canv.ID, BlockTypeText, "First", 0)
	svc.AddBlock(ctx, canv.ID, BlockTypeHeading, "Title", 1)

	got, err := svc.Get(ctx, canv.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(got.Blocks) != 2 {
		t.Errorf("Get() blocks len = %v, want 2", len(got.Blocks))
	}
}

func TestService_UpdateBlock(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	canv, _ := svc.Create(ctx, "session-1", "Test")
	block, _ := svc.AddBlock(ctx, canv.ID, BlockTypeText, "Original", 0)
	block.CanvasID = canv.ID // Store mock needs this

	block.Content = "Updated"
	if err := svc.UpdateBlock(ctx, block); err != nil {
		t.Fatalf("UpdateBlock() error = %v", err)
	}

	got, _ := svc.Get(ctx, canv.ID)
	if got.Blocks[0].Content != "Updated" {
		t.Errorf("UpdateBlock() Content = %v, want Updated", got.Blocks[0].Content)
	}
}

func TestService_DeleteBlock(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	canv, _ := svc.Create(ctx, "session-1", "Test")
	block, _ := svc.AddBlock(ctx, canv.ID, BlockTypeText, "To delete", 0)

	if err := svc.DeleteBlock(ctx, block.ID); err != nil {
		t.Fatalf("DeleteBlock() error = %v", err)
	}

	got, _ := svc.Get(ctx, canv.ID)
	if len(got.Blocks) != 0 {
		t.Errorf("DeleteBlock() blocks len = %v, want 0", len(got.Blocks))
	}
}

func TestService_ReorderBlocks(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	canv, _ := svc.Create(ctx, "session-1", "Test")
	block1, _ := svc.AddBlock(ctx, canv.ID, BlockTypeText, "First", 0)
	block2, _ := svc.AddBlock(ctx, canv.ID, BlockTypeText, "Second", 1)

	// Reverse order
	if err := svc.ReorderBlocks(ctx, canv.ID, []string{block2.ID, block1.ID}); err != nil {
		t.Fatalf("ReorderBlocks() error = %v", err)
	}

	got, _ := svc.Get(ctx, canv.ID)
	if got.Blocks[0].ID != block2.ID {
		t.Errorf("ReorderBlocks() first block ID = %v, want %v", got.Blocks[0].ID, block2.ID)
	}
}

func TestService_Export_Markdown(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	canv, _ := svc.Create(ctx, "session-1", "Test Canvas")
	svc.AddBlock(ctx, canv.ID, BlockTypeHeading, "Title", 0)
	svc.AddBlock(ctx, canv.ID, BlockTypeText, "Some text", 1)
	svc.AddBlock(ctx, canv.ID, BlockTypeDivider, "", 2)
	svc.AddBlock(ctx, canv.ID, BlockTypeCode, "const x = 1", 3)

	data, contentType, err := svc.Export(ctx, canv.ID, ExportMarkdown)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if contentType != "text/markdown" {
		t.Errorf("Export() contentType = %v, want text/markdown", contentType)
	}

	md := string(data)
	if len(md) == 0 {
		t.Error("Export() should return non-empty data")
	}
}

func TestService_Export_HTML(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	canv, _ := svc.Create(ctx, "session-1", "Test")
	svc.AddBlock(ctx, canv.ID, BlockTypeText, "Hello", 0)

	data, contentType, err := svc.Export(ctx, canv.ID, ExportHTML)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if contentType != "text/html" {
		t.Errorf("Export() contentType = %v, want text/html", contentType)
	}

	html := string(data)
	if len(html) == 0 {
		t.Error("Export() should return non-empty data")
	}
}

func TestService_Export_JSON(t *testing.T) {
	store := newMockStore()
	svc := New(store)
	ctx := context.Background()

	canv, _ := svc.Create(ctx, "session-1", "Test")
	svc.AddBlock(ctx, canv.ID, BlockTypeText, "Hello", 0)

	data, contentType, err := svc.Export(ctx, canv.ID, ExportJSON)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if contentType != "application/json" {
		t.Errorf("Export() contentType = %v, want application/json", contentType)
	}

	if len(data) == 0 {
		t.Error("Export() should return non-empty data")
	}
}

func TestBlockTypes(t *testing.T) {
	tests := []struct {
		blockType BlockType
		valid     bool
	}{
		{BlockTypeText, true},
		{BlockTypeAIResponse, true},
		{BlockTypeNote, true},
		{BlockTypeCitation, true},
		{BlockTypeHeading, true},
		{BlockTypeDivider, true},
		{BlockTypeCode, true},
		{BlockType("invalid"), true}, // Currently no validation
	}

	for _, tt := range tests {
		t.Run(string(tt.blockType), func(t *testing.T) {
			// Just verify the type can be used
			b := Block{Type: tt.blockType}
			if b.Type != tt.blockType {
				t.Errorf("Block type = %v, want %v", b.Type, tt.blockType)
			}
		})
	}
}

func TestMarshalMeta(t *testing.T) {
	meta := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	json := MarshalMeta(meta)
	got := UnmarshalMeta(json)

	if got["key1"] != "value1" {
		t.Errorf("UnmarshalMeta key1 = %v, want value1", got["key1"])
	}
}

func TestMarshalMeta_Nil(t *testing.T) {
	json := MarshalMeta(nil)
	if json != "{}" {
		t.Errorf("MarshalMeta(nil) = %v, want {}", json)
	}

	got := UnmarshalMeta("")
	if got != nil {
		t.Errorf("UnmarshalMeta(\"\") = %v, want nil", got)
	}
}
