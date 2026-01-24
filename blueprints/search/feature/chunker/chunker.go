// Package chunker provides web page fetching, parsing, and chunking for RAG.
package chunker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/html"

	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
)

// Document represents a fetched and processed web page.
type Document struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Chunks    []Chunk   `json:"chunks"`
	FetchedAt time.Time `json:"fetched_at"`
}

// Chunk represents a text passage from a document.
type Chunk struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	URL        string    `json:"url"`
	Text       string    `json:"text"`
	Embedding  []float32 `json:"embedding,omitempty"`
	StartPos   int       `json:"start_pos"`
	EndPos     int       `json:"end_pos"`
}

// Config holds chunker configuration.
type Config struct {
	ChunkSize    int // Target chunk size in characters (default 1000)
	ChunkOverlap int // Overlap between chunks (default 200)
	MaxChunks    int // Maximum chunks per document (default 100)
}

// Store defines the interface for chunk storage.
type Store interface {
	// SaveDocument saves a document and its chunks.
	SaveDocument(ctx context.Context, doc *Document) error

	// GetDocument retrieves a document by URL.
	GetDocument(ctx context.Context, url string) (*Document, error)

	// GetChunks retrieves chunks for a document.
	GetChunks(ctx context.Context, documentID string) ([]Chunk, error)

	// SearchChunks searches chunks by embedding similarity.
	SearchChunks(ctx context.Context, embedding []float32, limit int) ([]Chunk, error)

	// SaveChunk saves or updates a chunk.
	SaveChunk(ctx context.Context, chunk *Chunk) error

	// DeleteOldDocuments deletes documents older than the given duration.
	DeleteOldDocuments(ctx context.Context, olderThan time.Duration) error
}

// Service manages document fetching and chunking.
type Service struct {
	store      Store
	embedder   llm.Provider
	httpClient *http.Client
	config     Config
}

// New creates a new chunker service.
func New(store Store, embedder llm.Provider, cfg Config) *Service {
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = 1000
	}
	if cfg.ChunkOverlap <= 0 {
		cfg.ChunkOverlap = 200
	}
	if cfg.MaxChunks <= 0 {
		cfg.MaxChunks = 100
	}

	return &Service{
		store:    store,
		embedder: embedder,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: cfg,
	}
}

// Fetch retrieves a page, chunks it, and generates embeddings.
// Returns cached document if available and fresh (< 24 hours).
func (s *Service) Fetch(ctx context.Context, url string) (*Document, error) {
	// Check cache first
	if doc, err := s.store.GetDocument(ctx, url); err == nil {
		// Use cached version if less than 24 hours old
		if time.Since(doc.FetchedAt) < 24*time.Hour {
			chunks, _ := s.store.GetChunks(ctx, doc.ID)
			doc.Chunks = chunks
			return doc, nil
		}
	}

	// Fetch the page
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("chunker: create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MizuSearch/1.0)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chunker: fetch url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chunker: unexpected status %d", resp.StatusCode)
	}

	// Parse HTML
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024)) // 5MB limit
	if err != nil {
		return nil, fmt.Errorf("chunker: read body: %w", err)
	}

	title, content := s.parseHTML(string(body))

	// Create document
	doc := &Document{
		ID:        generateID(),
		URL:       url,
		Title:     title,
		Content:   content,
		FetchedAt: time.Now(),
	}

	// Chunk the content
	chunks := s.chunkText(doc.ID, url, content)
	doc.Chunks = chunks

	// Generate embeddings if embedder is available
	if s.embedder != nil && len(chunks) > 0 {
		texts := make([]string, len(chunks))
		for i, c := range chunks {
			texts[i] = c.Text
		}

		embResp, err := s.embedder.Embedding(ctx, llm.EmbeddingRequest{
			Input: texts,
		})
		if err == nil && len(embResp.Data) == len(chunks) {
			for i := range chunks {
				chunks[i].Embedding = embResp.Data[i].Embedding
			}
		}
	}

	// Save to store
	if err := s.store.SaveDocument(ctx, doc); err != nil {
		return nil, fmt.Errorf("chunker: save document: %w", err)
	}

	return doc, nil
}

// FetchMultiple fetches multiple URLs concurrently.
func (s *Service) FetchMultiple(ctx context.Context, urls []string) ([]*Document, error) {
	docs := make([]*Document, len(urls))
	errs := make([]error, len(urls))

	// Simple sequential fetch (could be parallelized)
	for i, url := range urls {
		doc, err := s.Fetch(ctx, url)
		docs[i] = doc
		errs[i] = err
	}

	// Return docs even if some failed
	return docs, nil
}

// Search finds relevant chunks for a query.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]Chunk, error) {
	if s.embedder == nil {
		return nil, fmt.Errorf("chunker: embedder not configured")
	}

	// Generate query embedding
	embResp, err := s.embedder.Embedding(ctx, llm.EmbeddingRequest{
		Input: []string{query},
	})
	if err != nil {
		return nil, fmt.Errorf("chunker: generate embedding: %w", err)
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("chunker: no embedding returned")
	}

	// Search by similarity
	return s.store.SearchChunks(ctx, embResp.Data[0].Embedding, limit)
}

// GetRelevantChunks finds the most relevant chunks from given documents for a query.
func (s *Service) GetRelevantChunks(ctx context.Context, docs []*Document, query string, limit int) ([]Chunk, error) {
	if s.embedder == nil {
		// Fallback: return first N chunks from each doc
		var chunks []Chunk
		perDoc := limit / len(docs)
		if perDoc < 1 {
			perDoc = 1
		}
		for _, doc := range docs {
			if doc == nil {
				continue
			}
			end := perDoc
			if end > len(doc.Chunks) {
				end = len(doc.Chunks)
			}
			chunks = append(chunks, doc.Chunks[:end]...)
		}
		if len(chunks) > limit {
			chunks = chunks[:limit]
		}
		return chunks, nil
	}

	// Generate query embedding
	embResp, err := s.embedder.Embedding(ctx, llm.EmbeddingRequest{
		Input: []string{query},
	})
	if err != nil {
		return nil, fmt.Errorf("chunker: generate embedding: %w", err)
	}

	queryEmb := embResp.Data[0].Embedding

	// Collect all chunks with scores
	type scoredChunk struct {
		chunk Chunk
		score float32
	}
	var scored []scoredChunk

	for _, doc := range docs {
		if doc == nil {
			continue
		}
		for _, chunk := range doc.Chunks {
			if len(chunk.Embedding) > 0 {
				score := cosineSimilarity(queryEmb, chunk.Embedding)
				scored = append(scored, scoredChunk{chunk, score})
			}
		}
	}

	// Sort by score descending
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Return top N
	result := make([]Chunk, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		result = append(result, scored[i].chunk)
	}

	return result, nil
}

// IndexBackground runs background indexing of URLs.
func (s *Service) IndexBackground(ctx context.Context, urls []string) error {
	for _, url := range urls {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, _ = s.Fetch(ctx, url)
		}
	}
	return nil
}

// Cleanup removes old documents.
func (s *Service) Cleanup(ctx context.Context, olderThan time.Duration) error {
	return s.store.DeleteOldDocuments(ctx, olderThan)
}

// parseHTML extracts title and text content from HTML.
func (s *Service) parseHTML(htmlContent string) (title, content string) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		// Fallback: strip tags manually
		return "", stripTags(htmlContent)
	}

	var titleBuf, contentBuf strings.Builder
	var inTitle, inScript, inStyle bool

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				inTitle = true
			case "script", "style", "noscript", "nav", "footer", "header":
				inScript = true
			}
		}

		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				if inTitle {
					titleBuf.WriteString(text)
				} else if !inScript && !inStyle {
					contentBuf.WriteString(text)
					contentBuf.WriteString(" ")
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}

		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				inTitle = false
			case "script", "style", "noscript", "nav", "footer", "header":
				inScript = false
			case "p", "div", "br", "li", "h1", "h2", "h3", "h4", "h5", "h6":
				contentBuf.WriteString("\n")
			}
		}
	}

	extract(doc)

	title = strings.TrimSpace(titleBuf.String())
	content = normalizeWhitespace(contentBuf.String())

	return title, content
}

// chunkText splits text into overlapping chunks.
func (s *Service) chunkText(docID, url, text string) []Chunk {
	if len(text) == 0 {
		return nil
	}

	var chunks []Chunk
	pos := 0

	for pos < len(text) && len(chunks) < s.config.MaxChunks {
		end := pos + s.config.ChunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to break at sentence boundary
		if end < len(text) {
			for i := end; i > pos+s.config.ChunkSize/2; i-- {
				if text[i] == '.' || text[i] == '!' || text[i] == '?' {
					end = i + 1
					break
				}
			}
		}

		chunkText := strings.TrimSpace(text[pos:end])
		if len(chunkText) > 50 { // Minimum chunk size
			chunks = append(chunks, Chunk{
				ID:         generateID(),
				DocumentID: docID,
				URL:        url,
				Text:       chunkText,
				StartPos:   pos,
				EndPos:     end,
			})
		}

		// Move position with overlap
		pos = end - s.config.ChunkOverlap
		if pos < 0 {
			pos = 0
		}
		if pos <= chunks[len(chunks)-1].StartPos {
			pos = end // Prevent infinite loop
		}
	}

	return chunks
}

func stripTags(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(s, " ")
}

func normalizeWhitespace(s string) string {
	var result strings.Builder
	prevSpace := false

	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				result.WriteRune(' ')
				prevSpace = true
			}
		} else {
			result.WriteRune(r)
			prevSpace = false
		}
	}

	return strings.TrimSpace(result.String())
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (sqrt(normA) * sqrt(normB))
}

func sqrt(x float32) float32 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
