package memory

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MemoryManager coordinates file indexing and hybrid search across a
// workspace directory. It combines chunking, FTS5 keyword search, and
// optional vector embeddings for semantic retrieval.
type MemoryManager struct {
	store        *MemoryStore
	embedder     EmbedProvider // nil when no API key is available (FTS5-only mode).
	config       MemoryConfig
	workspaceDir string
}

// MemoryConfig holds all configuration for the memory system.
type MemoryConfig struct {
	Enabled      bool    // Whether memory indexing and search are active.
	WorkspaceDir string  // Root directory to index.
	ChunkTokens  int     // Max tokens per chunk (default 400).
	ChunkOverlap int     // Overlap tokens between chunks (default 80).
	VectorWeight float64 // Weight for vector similarity in hybrid search (default 0.7).
	TextWeight   float64 // Weight for BM25 keyword search (default 0.3).
	MinScore     float64 // Minimum score threshold for results (default 0.35).
	MaxResults   int     // Maximum number of search results (default 6).
}

// DefaultMemoryConfig returns sensible defaults matching OpenClaw.
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		Enabled:      true,
		ChunkTokens:  400,
		ChunkOverlap: 80,
		VectorWeight: 0.7,
		TextWeight:   0.3,
		MinScore:     0.35,
		MaxResults:   6,
	}
}

// NewMemoryManager creates and initialises a MemoryManager. If no OPENAI_API_KEY
// environment variable is set, the manager operates in FTS5-only mode (no
// vector embeddings). The database schema is created automatically.
func NewMemoryManager(dbPath, workspaceDir string, cfg MemoryConfig) (*MemoryManager, error) {
	store, err := NewMemoryStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open memory store: %w", err)
	}

	if err := store.EnsureSchema(); err != nil {
		store.Close()
		return nil, fmt.Errorf("ensure memory schema: %w", err)
	}

	m := &MemoryManager{
		store:        store,
		config:       cfg,
		workspaceDir: workspaceDir,
	}

	// Set up embedder if an OpenAI API key is available.
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		m.embedder = NewOpenAIEmbedder(apiKey)
	}

	return m, nil
}

// IndexFile indexes (or re-indexes) a single file with source="memory".
// It reads the file, computes a content hash, and skips processing if the
// hash has not changed.
func (m *MemoryManager) IndexFile(path string) error {
	return m.IndexFileWithSource(path, "memory")
}

// IndexFileWithSource indexes a single file with the given source tag.
// The source should be "memory" for workspace files or "sessions" for
// session transcripts, matching OpenClaw's convention.
func (m *MemoryManager) IndexFileWithSource(path, source string) error {
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(m.workspaceDir, path)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read file %s: %w", path, err)
	}

	// Compute content hash for change detection.
	hash := fmt.Sprintf("%x", sha256.Sum256(content))

	// Check if file has changed since last index.
	existingHash, err := m.store.GetFileHash(path)
	if err != nil {
		return fmt.Errorf("get file hash %s: %w", path, err)
	}
	if existingHash == hash {
		return nil // unchanged
	}

	// Get file info for mtime/size.
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat file %s: %w", path, err)
	}

	// Delete old chunks for this file.
	if err := m.store.DeleteChunksByPath(path); err != nil {
		return fmt.Errorf("delete old chunks %s: %w", path, err)
	}

	// Chunk the file.
	chunkCfg := ChunkConfig{
		MaxTokens:     m.config.ChunkTokens,
		OverlapTokens: m.config.ChunkOverlap,
	}
	chunks := ChunkMarkdown(string(content), chunkCfg)

	// Generate embeddings if available.
	var embeddings [][]float64
	if m.embedder != nil && len(chunks) > 0 {
		embeddings, err = m.embedChunks(chunks)
		if err != nil {
			// Log but do not fail; fall back to FTS-only for this file.
			embeddings = nil
		}
	}

	// Store each chunk.
	modelName := ""
	if m.embedder != nil {
		modelName = m.embedder.Model()
	}

	for i, c := range chunks {
		chunkID := fmt.Sprintf("%s:%d", path, c.StartLine)
		var emb []float64
		if embeddings != nil && i < len(embeddings) {
			emb = embeddings[i]
		}

		if err := m.store.UpsertChunk(
			chunkID, path, source, c.StartLine, c.EndLine,
			c.Hash, modelName, c.Text, emb,
		); err != nil {
			return fmt.Errorf("upsert chunk %s: %w", chunkID, err)
		}
	}

	// Update file record.
	if err := m.store.UpsertFile(path, source, hash, info.ModTime().Unix(), info.Size()); err != nil {
		return fmt.Errorf("upsert file %s: %w", path, err)
	}

	return nil
}

// IndexAll walks the workspace directory and indexes all supported files.
// It skips hidden directories, common build output directories, and binary files.
func (m *MemoryManager) IndexAll() error {
	if m.workspaceDir == "" {
		return fmt.Errorf("workspace directory not set")
	}

	return filepath.WalkDir(m.workspaceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip files we cannot read
		}

		name := d.Name()

		// Skip hidden directories and common non-content directories.
		if d.IsDir() {
			if shouldSkipDir(name) {
				return filepath.SkipDir
			}
			return nil
		}

		// Only index text-like files.
		if !isIndexableFile(name) {
			return nil
		}

		// Skip very large files (> 1MB).
		info, err := d.Info()
		if err != nil || info.Size() > 1<<20 {
			return nil
		}

		// Use workspace-relative path as the key.
		relPath, err := filepath.Rel(m.workspaceDir, path)
		if err != nil {
			relPath = path
		}

		if err := m.IndexFileWithSource(relPath, "memory"); err != nil {
			// Log but continue indexing other files.
			return nil
		}

		return nil
	})
}

// Search performs hybrid search combining FTS5 keyword matching and (if
// available) vector similarity. Returns up to maxResults results scoring
// above minScore.
func (m *MemoryManager) Search(ctx context.Context, query string, maxResults int, minScore float64) ([]SearchResult, error) {
	if query == "" {
		return nil, nil
	}

	if maxResults <= 0 {
		maxResults = m.config.MaxResults
	}
	if minScore <= 0 {
		minScore = m.config.MinScore
	}

	hybridCfg := HybridConfig{
		VectorWeight:        m.config.VectorWeight,
		TextWeight:          m.config.TextWeight,
		MinScore:            minScore,
		MaxResults:          maxResults,
		CandidateMultiplier: 4,
	}

	candidateLimit := maxResults * hybridCfg.CandidateMultiplier

	// FTS5 keyword search (always available).
	keywordResults, err := m.store.SearchFTS(query, candidateLimit)
	if err != nil {
		return nil, fmt.Errorf("FTS search: %w", err)
	}

	// Vector search (only if embedder is available).
	var vectorResults []VectorResult
	if m.embedder != nil {
		queryEmbs, err := m.embedder.Embed(ctx, []string{query})
		if err == nil && len(queryEmbs) > 0 {
			vectorResults, err = m.store.SearchVector(queryEmbs[0], candidateLimit)
			if err != nil {
				// Degrade gracefully: proceed with FTS-only results.
				vectorResults = nil
			}
		}
	}

	// If no vector results, adjust weights to use FTS only.
	if len(vectorResults) == 0 {
		hybridCfg.VectorWeight = 0
		hybridCfg.TextWeight = 1.0
		hybridCfg.MinScore = minScore * 0.5 // lower threshold for FTS-only
	}

	results := MergeHybridResults(vectorResults, keywordResults, hybridCfg)
	return results, nil
}

// GetLines reads a range of lines from a file. The from parameter is
// 1-based. Returns up to count lines starting from that position.
func (m *MemoryManager) GetLines(path string, from, count int) (string, error) {
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(m.workspaceDir, path)
	}

	f, err := os.Open(absPath)
	if err != nil {
		return "", fmt.Errorf("open file %s: %w", path, err)
	}
	defer f.Close()

	if from < 1 {
		from = 1
	}
	if count <= 0 {
		count = 20
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // up to 1MB lines
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum < from {
			continue
		}
		if lineNum >= from+count {
			break
		}
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan file %s: %w", path, err)
	}

	return strings.Join(lines, "\n"), nil
}

// Stats returns the number of indexed files and chunks.
func (m *MemoryManager) Stats() (fileCount int, chunkCount int, err error) {
	return m.store.Stats()
}

// Close releases all resources held by the MemoryManager.
func (m *MemoryManager) Close() error {
	if m.store != nil {
		return m.store.Close()
	}
	return nil
}

// ReIndex re-indexes all workspace files. This is called after compaction
// or other events that may have updated workspace files (e.g. MEMORY.md).
func (m *MemoryManager) ReIndex() error {
	return m.IndexAll()
}

// EnsureDailyLog creates today's daily memory log file if it doesn't exist.
// The log is created at workspace/memory/YYYY-MM-DD.md matching OpenClaw's
// daily log convention.
func (m *MemoryManager) EnsureDailyLog() error {
	if m.workspaceDir == "" {
		return nil
	}

	memDir := filepath.Join(m.workspaceDir, "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		return fmt.Errorf("create memory dir: %w", err)
	}

	today := time.Now().Format("2006-01-02")
	logPath := filepath.Join(memDir, today+".md")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		header := fmt.Sprintf("# %s\n\n", today)
		return os.WriteFile(logPath, []byte(header), 0o644)
	}
	return nil
}

// IndexSessionTranscript indexes a single JSONL session transcript file.
// It extracts assistant messages and indexes them with source="sessions".
func (m *MemoryManager) IndexSessionTranscript(transcriptPath string) error {
	absPath := transcriptPath
	if !filepath.IsAbs(transcriptPath) {
		absPath = filepath.Join(m.workspaceDir, transcriptPath)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read transcript %s: %w", transcriptPath, err)
	}

	// Compute hash for change detection.
	hash := fmt.Sprintf("%x", sha256.Sum256(content))
	existingHash, err := m.store.GetFileHash(transcriptPath)
	if err != nil {
		return fmt.Errorf("get file hash %s: %w", transcriptPath, err)
	}
	if existingHash == hash {
		return nil // unchanged
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", transcriptPath, err)
	}

	// Delete old chunks.
	if err := m.store.DeleteChunksByPath(transcriptPath); err != nil {
		return fmt.Errorf("delete old chunks %s: %w", transcriptPath, err)
	}

	// Parse JSONL and extract assistant messages.
	var assistantTexts []string
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Role == "assistant" && entry.Content != "" {
			assistantTexts = append(assistantTexts, entry.Content)
		}
	}

	if len(assistantTexts) == 0 {
		// Still record the file so we don't re-process it.
		return m.store.UpsertFile(transcriptPath, "sessions", hash, info.ModTime().Unix(), info.Size())
	}

	// Chunk all assistant text together.
	combined := strings.Join(assistantTexts, "\n\n")
	chunkCfg := ChunkConfig{
		MaxTokens:     m.config.ChunkTokens,
		OverlapTokens: m.config.ChunkOverlap,
	}
	chunks := ChunkMarkdown(combined, chunkCfg)

	// Generate embeddings if available.
	var embeddings [][]float64
	if m.embedder != nil && len(chunks) > 0 {
		embeddings, err = m.embedChunks(chunks)
		if err != nil {
			embeddings = nil
		}
	}

	modelName := ""
	if m.embedder != nil {
		modelName = m.embedder.Model()
	}

	for i, c := range chunks {
		chunkID := fmt.Sprintf("%s:%d", transcriptPath, c.StartLine)
		var emb []float64
		if embeddings != nil && i < len(embeddings) {
			emb = embeddings[i]
		}
		if err := m.store.UpsertChunk(
			chunkID, transcriptPath, "sessions", c.StartLine, c.EndLine,
			c.Hash, modelName, c.Text, emb,
		); err != nil {
			return fmt.Errorf("upsert chunk %s: %w", chunkID, err)
		}
	}

	return m.store.UpsertFile(transcriptPath, "sessions", hash, info.ModTime().Unix(), info.Size())
}

// IndexSessionTranscripts walks a sessions directory and indexes all JSONL
// transcript files with source="sessions".
func (m *MemoryManager) IndexSessionTranscripts(sessionsDir string) error {
	if sessionsDir == "" {
		return nil
	}

	absDir := sessionsDir
	if !filepath.IsAbs(sessionsDir) {
		absDir = filepath.Join(m.workspaceDir, sessionsDir)
	}

	return filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".jsonl" {
			return nil
		}

		relPath, err := filepath.Rel(m.workspaceDir, path)
		if err != nil {
			relPath = path
		}

		if err := m.IndexSessionTranscript(relPath); err != nil {
			// Log but continue indexing other files.
			return nil
		}
		return nil
	})
}

// SearchWithSource performs hybrid search filtered by source.
// If source is empty or "all", all sources are searched.
func (m *MemoryManager) SearchWithSource(ctx context.Context, query, source string, maxResults int, minScore float64) ([]SearchResult, error) {
	if query == "" {
		return nil, nil
	}
	if maxResults <= 0 {
		maxResults = m.config.MaxResults
	}
	if minScore <= 0 {
		minScore = m.config.MinScore
	}

	// If no source filter, delegate to normal search.
	if source == "" || source == "all" {
		return m.Search(ctx, query, maxResults, minScore)
	}

	hybridCfg := HybridConfig{
		VectorWeight:        m.config.VectorWeight,
		TextWeight:          m.config.TextWeight,
		MinScore:            minScore,
		MaxResults:          maxResults,
		CandidateMultiplier: 4,
	}

	candidateLimit := maxResults * hybridCfg.CandidateMultiplier

	// FTS5 keyword search with source filter.
	keywordResults, err := m.store.SearchFTSWithSource(query, source, candidateLimit)
	if err != nil {
		return nil, fmt.Errorf("FTS search: %w", err)
	}

	// Vector search with source filter.
	var vectorResults []VectorResult
	if m.embedder != nil {
		queryEmbs, err := m.embedder.Embed(ctx, []string{query})
		if err == nil && len(queryEmbs) > 0 {
			vectorResults, err = m.store.SearchVectorWithSource(queryEmbs[0], source, candidateLimit)
			if err != nil {
				vectorResults = nil
			}
		}
	}

	if len(vectorResults) == 0 {
		hybridCfg.VectorWeight = 0
		hybridCfg.TextWeight = 1.0
		hybridCfg.MinScore = minScore * 0.5
	}

	return MergeHybridResults(vectorResults, keywordResults, hybridCfg), nil
}

// embedChunks generates embeddings for chunks, using the embedding cache
// to avoid redundant API calls.
func (m *MemoryManager) embedChunks(chunks []Chunk) ([][]float64, error) {
	if m.embedder == nil {
		return nil, nil
	}

	modelName := m.embedder.Model()
	result := make([][]float64, len(chunks))

	// Check cache first and collect uncached indices.
	var uncachedIdxs []int
	var uncachedTexts []string
	for i, c := range chunks {
		cached, err := m.store.GetCachedEmbedding(c.Hash, modelName)
		if err == nil && cached != nil {
			result[i] = cached
			continue
		}
		uncachedIdxs = append(uncachedIdxs, i)
		uncachedTexts = append(uncachedTexts, c.Text)
	}

	if len(uncachedTexts) == 0 {
		return result, nil
	}

	// Batch embed uncached texts. Process in batches of 100 to stay within
	// API limits.
	const batchSize = 100
	for start := 0; start < len(uncachedTexts); start += batchSize {
		end := start + batchSize
		if end > len(uncachedTexts) {
			end = len(uncachedTexts)
		}

		batch := uncachedTexts[start:end]
		embeddings, err := m.embedder.Embed(context.Background(), batch)
		if err != nil {
			return nil, fmt.Errorf("embed batch: %w", err)
		}

		for j, emb := range embeddings {
			globalIdx := uncachedIdxs[start+j]
			result[globalIdx] = emb

			// Cache the embedding.
			_ = m.store.SetCachedEmbedding(chunks[globalIdx].Hash, modelName, emb)
		}
	}

	return result, nil
}

// shouldSkipDir returns true for directories that should not be indexed.
func shouldSkipDir(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	skip := map[string]bool{
		"node_modules": true,
		"vendor":       true,
		"dist":         true,
		"build":        true,
		"__pycache__":  true,
		".git":         true,
		".svn":         true,
		".hg":          true,
	}
	return skip[name]
}

// isIndexableFile returns true if the file should be indexed based on its
// extension. We focus on text/code/documentation files.
func isIndexableFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	indexable := map[string]bool{
		".go":   true,
		".md":   true,
		".txt":  true,
		".py":   true,
		".js":   true,
		".ts":   true,
		".jsx":  true,
		".tsx":  true,
		".rs":   true,
		".java": true,
		".c":    true,
		".cpp":  true,
		".h":    true,
		".hpp":  true,
		".rb":   true,
		".php":  true,
		".sh":   true,
		".bash": true,
		".yaml": true,
		".yml":  true,
		".toml": true,
		".json": true,
		".xml":  true,
		".html": true,
		".css":  true,
		".sql":  true,
		".r":    true,
		".lua":  true,
		".vim":  true,
		".el":   true,
		".cfg":  true,
		".conf": true,
		".ini":  true,
		".env":  true,
		".mod":  true,
		".sum":  true,
	}

	// Also index files with no extension if they are common config files.
	if ext == "" {
		lower := strings.ToLower(name)
		noExtIndexable := map[string]bool{
			"makefile":    true,
			"dockerfile":  true,
			"rakefile":    true,
			"gemfile":     true,
			"procfile":    true,
			"readme":      true,
			"license":     true,
			"changelog":   true,
			"contributing": true,
		}
		return noExtIndexable[lower]
	}

	return indexable[ext]
}
