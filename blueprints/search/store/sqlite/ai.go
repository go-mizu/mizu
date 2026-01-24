package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/feature/canvas"
	"github.com/go-mizu/mizu/blueprints/search/feature/chunker"
	"github.com/go-mizu/mizu/blueprints/search/feature/session"
)

// createAISchema creates tables for AI features.
func createAISchema(ctx context.Context, db *sql.DB) error {
	schema := `
		-- AI Sessions
		CREATE TABLE IF NOT EXISTS ai_sessions (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_ai_sessions_updated ON ai_sessions(updated_at DESC);

		-- Session Messages
		CREATE TABLE IF NOT EXISTS ai_messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL REFERENCES ai_sessions(id) ON DELETE CASCADE,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			mode TEXT,
			citations TEXT DEFAULT '[]',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_ai_messages_session ON ai_messages(session_id);
		CREATE INDEX IF NOT EXISTS idx_ai_messages_created ON ai_messages(created_at);

		-- Canvas
		CREATE TABLE IF NOT EXISTS ai_canvas (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL UNIQUE REFERENCES ai_sessions(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_ai_canvas_session ON ai_canvas(session_id);

		-- Canvas Blocks
		CREATE TABLE IF NOT EXISTS ai_canvas_blocks (
			id TEXT PRIMARY KEY,
			canvas_id TEXT NOT NULL REFERENCES ai_canvas(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			content TEXT NOT NULL,
			meta TEXT DEFAULT '{}',
			block_order INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_ai_canvas_blocks_canvas ON ai_canvas_blocks(canvas_id);
		CREATE INDEX IF NOT EXISTS idx_ai_canvas_blocks_order ON ai_canvas_blocks(canvas_id, block_order);

		-- Document Chunks (for RAG)
		CREATE TABLE IF NOT EXISTS ai_documents (
			id TEXT PRIMARY KEY,
			url TEXT UNIQUE NOT NULL,
			title TEXT,
			content TEXT,
			fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_ai_documents_url ON ai_documents(url);

		CREATE TABLE IF NOT EXISTS ai_chunks (
			id TEXT PRIMARY KEY,
			document_id TEXT NOT NULL REFERENCES ai_documents(id) ON DELETE CASCADE,
			url TEXT NOT NULL,
			text TEXT NOT NULL,
			embedding BLOB,
			start_pos INTEGER,
			end_pos INTEGER
		);

		CREATE INDEX IF NOT EXISTS idx_ai_chunks_document ON ai_chunks(document_id);
		CREATE INDEX IF NOT EXISTS idx_ai_chunks_url ON ai_chunks(url);
	`

	_, err := db.ExecContext(ctx, schema)
	return err
}

// ========== Session Store ==========

// SessionStore implements session.Store.
type SessionStore struct {
	db *sql.DB
}

// NewSessionStore creates a new session store.
func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) Create(ctx context.Context, sess *session.Session) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO ai_sessions (id, title, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		sess.ID, sess.Title, sess.CreatedAt, sess.UpdatedAt)
	return err
}

func (s *SessionStore) Get(ctx context.Context, id string) (*session.Session, error) {
	var sess session.Session
	err := s.db.QueryRowContext(ctx,
		`SELECT id, title, created_at, updated_at FROM ai_sessions WHERE id = ?`, id).
		Scan(&sess.ID, &sess.Title, &sess.CreatedAt, &sess.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *SessionStore) List(ctx context.Context, limit, offset int) ([]session.Session, int, error) {
	var total int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ai_sessions`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, created_at, updated_at FROM ai_sessions ORDER BY updated_at DESC LIMIT ? OFFSET ?`,
		limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []session.Session
	for rows.Next() {
		var sess session.Session
		if err := rows.Scan(&sess.ID, &sess.Title, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, sess)
	}

	return sessions, total, rows.Err()
}

func (s *SessionStore) Update(ctx context.Context, sess *session.Session) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE ai_sessions SET title = ?, updated_at = ? WHERE id = ?`,
		sess.Title, sess.UpdatedAt, sess.ID)
	return err
}

func (s *SessionStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM ai_sessions WHERE id = ?`, id)
	return err
}

func (s *SessionStore) AddMessage(ctx context.Context, sessionID string, msg *session.Message) error {
	citations, _ := json.Marshal(msg.Citations)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO ai_messages (id, session_id, role, content, mode, citations, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, sessionID, msg.Role, msg.Content, msg.Mode, string(citations), msg.CreatedAt)
	return err
}

func (s *SessionStore) GetMessages(ctx context.Context, sessionID string) ([]session.Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, session_id, role, content, mode, citations, created_at FROM ai_messages WHERE session_id = ? ORDER BY created_at ASC`,
		sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []session.Message
	for rows.Next() {
		var msg session.Message
		var citationsJSON string
		var mode sql.NullString
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &mode, &citationsJSON, &msg.CreatedAt); err != nil {
			return nil, err
		}
		msg.Mode = mode.String
		msg.Citations = session.UnmarshalCitations(citationsJSON)
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// ========== Canvas Store ==========

// CanvasStore implements canvas.Store.
type CanvasStore struct {
	db *sql.DB
}

// NewCanvasStore creates a new canvas store.
func NewCanvasStore(db *sql.DB) *CanvasStore {
	return &CanvasStore{db: db}
}

func (s *CanvasStore) Create(ctx context.Context, c *canvas.Canvas) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO ai_canvas (id, session_id, title, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		c.ID, c.SessionID, c.Title, c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *CanvasStore) Get(ctx context.Context, id string) (*canvas.Canvas, error) {
	var c canvas.Canvas
	err := s.db.QueryRowContext(ctx,
		`SELECT id, session_id, title, created_at, updated_at FROM ai_canvas WHERE id = ?`, id).
		Scan(&c.ID, &c.SessionID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *CanvasStore) GetBySessionID(ctx context.Context, sessionID string) (*canvas.Canvas, error) {
	var c canvas.Canvas
	err := s.db.QueryRowContext(ctx,
		`SELECT id, session_id, title, created_at, updated_at FROM ai_canvas WHERE session_id = ?`, sessionID).
		Scan(&c.ID, &c.SessionID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *CanvasStore) Update(ctx context.Context, c *canvas.Canvas) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE ai_canvas SET title = ?, updated_at = ? WHERE id = ?`,
		c.Title, c.UpdatedAt, c.ID)
	return err
}

func (s *CanvasStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM ai_canvas WHERE id = ?`, id)
	return err
}

func (s *CanvasStore) AddBlock(ctx context.Context, canvasID string, block *canvas.Block) error {
	meta := canvas.MarshalMeta(block.Meta)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO ai_canvas_blocks (id, canvas_id, type, content, meta, block_order, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		block.ID, canvasID, string(block.Type), block.Content, meta, block.Order, block.CreatedAt)
	return err
}

func (s *CanvasStore) UpdateBlock(ctx context.Context, block *canvas.Block) error {
	meta := canvas.MarshalMeta(block.Meta)
	_, err := s.db.ExecContext(ctx,
		`UPDATE ai_canvas_blocks SET type = ?, content = ?, meta = ?, block_order = ? WHERE id = ?`,
		string(block.Type), block.Content, meta, block.Order, block.ID)
	return err
}

func (s *CanvasStore) DeleteBlock(ctx context.Context, blockID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM ai_canvas_blocks WHERE id = ?`, blockID)
	return err
}

func (s *CanvasStore) GetBlocks(ctx context.Context, canvasID string) ([]canvas.Block, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, canvas_id, type, content, meta, block_order, created_at FROM ai_canvas_blocks WHERE canvas_id = ? ORDER BY block_order ASC`,
		canvasID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []canvas.Block
	for rows.Next() {
		var b canvas.Block
		var typeStr, metaJSON string
		if err := rows.Scan(&b.ID, &b.CanvasID, &typeStr, &b.Content, &metaJSON, &b.Order, &b.CreatedAt); err != nil {
			return nil, err
		}
		b.Type = canvas.BlockType(typeStr)
		b.Meta = canvas.UnmarshalMeta(metaJSON)
		blocks = append(blocks, b)
	}

	return blocks, rows.Err()
}

func (s *CanvasStore) ReorderBlocks(ctx context.Context, canvasID string, blockIDs []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, id := range blockIDs {
		_, err := tx.ExecContext(ctx,
			`UPDATE ai_canvas_blocks SET block_order = ? WHERE id = ? AND canvas_id = ?`,
			i, id, canvasID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ========== Chunker Store ==========

// ChunkerStore implements chunker.Store.
type ChunkerStore struct {
	db *sql.DB
}

// NewChunkerStore creates a new chunker store.
func NewChunkerStore(db *sql.DB) *ChunkerStore {
	return &ChunkerStore{db: db}
}

func (s *ChunkerStore) SaveDocument(ctx context.Context, doc *chunker.Document) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Upsert document
	_, err = tx.ExecContext(ctx,
		`INSERT INTO ai_documents (id, url, title, content, fetched_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET title = excluded.title, content = excluded.content, fetched_at = excluded.fetched_at`,
		doc.ID, doc.URL, doc.Title, doc.Content, doc.FetchedAt)
	if err != nil {
		return err
	}

	// Delete old chunks for this document
	_, err = tx.ExecContext(ctx, `DELETE FROM ai_chunks WHERE document_id = ?`, doc.ID)
	if err != nil {
		return err
	}

	// Insert new chunks
	for _, chunk := range doc.Chunks {
		var embedding []byte
		if len(chunk.Embedding) > 0 {
			embedding, _ = json.Marshal(chunk.Embedding)
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO ai_chunks (id, document_id, url, text, embedding, start_pos, end_pos) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			chunk.ID, doc.ID, chunk.URL, chunk.Text, embedding, chunk.StartPos, chunk.EndPos)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *ChunkerStore) GetDocument(ctx context.Context, url string) (*chunker.Document, error) {
	var doc chunker.Document
	err := s.db.QueryRowContext(ctx,
		`SELECT id, url, title, content, fetched_at FROM ai_documents WHERE url = ?`, url).
		Scan(&doc.ID, &doc.URL, &doc.Title, &doc.Content, &doc.FetchedAt)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *ChunkerStore) GetChunks(ctx context.Context, documentID string) ([]chunker.Chunk, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, document_id, url, text, embedding, start_pos, end_pos FROM ai_chunks WHERE document_id = ?`,
		documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []chunker.Chunk
	for rows.Next() {
		var c chunker.Chunk
		var embedding []byte
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.URL, &c.Text, &embedding, &c.StartPos, &c.EndPos); err != nil {
			return nil, err
		}
		if len(embedding) > 0 {
			_ = json.Unmarshal(embedding, &c.Embedding)
		}
		chunks = append(chunks, c)
	}

	return chunks, rows.Err()
}

func (s *ChunkerStore) SearchChunks(ctx context.Context, embedding []float32, limit int) ([]chunker.Chunk, error) {
	// SQLite doesn't support vector search natively
	// For now, return recent chunks - in production, use a vector DB or extension
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.id, c.document_id, c.url, c.text, c.embedding, c.start_pos, c.end_pos
		FROM ai_chunks c
		JOIN ai_documents d ON c.document_id = d.id
		ORDER BY d.fetched_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []chunker.Chunk
	for rows.Next() {
		var c chunker.Chunk
		var embeddingBytes []byte
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.URL, &c.Text, &embeddingBytes, &c.StartPos, &c.EndPos); err != nil {
			return nil, err
		}
		if len(embeddingBytes) > 0 {
			_ = json.Unmarshal(embeddingBytes, &c.Embedding)
		}
		chunks = append(chunks, c)
	}

	return chunks, rows.Err()
}

func (s *ChunkerStore) SaveChunk(ctx context.Context, chunk *chunker.Chunk) error {
	var embedding []byte
	if len(chunk.Embedding) > 0 {
		embedding, _ = json.Marshal(chunk.Embedding)
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO ai_chunks (id, document_id, url, text, embedding, start_pos, end_pos) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		chunk.ID, chunk.DocumentID, chunk.URL, chunk.Text, embedding, chunk.StartPos, chunk.EndPos)
	return err
}

func (s *ChunkerStore) DeleteOldDocuments(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	_, err := s.db.ExecContext(ctx, `DELETE FROM ai_documents WHERE fetched_at < ?`, cutoff)
	return err
}
