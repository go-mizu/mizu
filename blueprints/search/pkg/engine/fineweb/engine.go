package fineweb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/local/engines"
)

// Engine is a FineWeb-2 search engine using DuckDB.
type Engine struct {
	*engines.BaseEngine

	config    Config
	stores    map[string]*Store
	mu        sync.RWMutex
	ftsReady  map[string]bool
}

// NewEngine creates a new FineWeb search engine.
func NewEngine(cfg Config) (*Engine, error) {
	// Merge with defaults
	defaults := DefaultConfig()
	if cfg.DataDir == "" {
		cfg.DataDir = defaults.DataDir
	}
	if cfg.SourceDir == "" {
		cfg.SourceDir = defaults.SourceDir
	}
	if cfg.ResultLimit == 0 {
		cfg.ResultLimit = defaults.ResultLimit
	}
	if cfg.ContentSnippetLength == 0 {
		cfg.ContentSnippetLength = defaults.ContentSnippetLength
	}

	base := engines.NewBaseEngine("fineweb", "fw", []engines.Category{engines.CategoryGeneral})
	base.SetEngineType(engines.EngineTypeOffline)
	base.SetTimeout(10 * time.Second)
	base.SetPaging(true)
	base.SetMaxPage(10)
	base.SetLanguageSupport(true)
	base.SetAbout(engines.EngineAbout{
		Website:    "https://huggingface.co/datasets/HuggingFaceFW/fineweb-2",
		WikidataID: "",
		Results:    "Web documents from FineWeb-2 dataset",
	})

	e := &Engine{
		BaseEngine: base,
		config:     cfg,
		stores:     make(map[string]*Store),
		ftsReady:   make(map[string]bool),
	}

	// Initialize stores for downloaded languages
	if err := e.initStores(); err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Engine) initStores() error {
	// List downloaded languages from source directory
	entries, err := os.ReadDir(e.config.SourceDir)
	if os.IsNotExist(err) {
		return nil // No data yet
	}
	if err != nil {
		return fmt.Errorf("reading source directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		lang := entry.Name()

		// Check if this language has parquet files
		trainDir := filepath.Join(e.config.SourceDir, lang, "train")
		if _, err := os.Stat(trainDir); os.IsNotExist(err) {
			continue
		}

		// Skip if language filter is set and doesn't include this language
		if len(e.config.Languages) > 0 {
			found := false
			for _, l := range e.config.Languages {
				if l == lang {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Initialize store
		store, err := NewStore(lang, e.config.DataDir)
		if err != nil {
			return fmt.Errorf("initializing store for %s: %w", lang, err)
		}
		e.stores[lang] = store
	}

	return nil
}

// Search implements OfflineEngine.Search.
func (e *Engine) Search(ctx context.Context, query string, params *engines.RequestParams) (*engines.EngineResults, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.stores) == 0 {
		return engines.NewEngineResults(), nil
	}

	results := engines.NewEngineResults()
	limit := e.config.ResultLimit
	if params.PageNo > 1 {
		limit = e.config.ResultLimit / 2 // Fewer results per subsequent page
	}
	offset := (params.PageNo - 1) * limit

	// Collect results from all stores
	var allDocs []Document
	for lang, store := range e.stores {
		// Use FTS if ready, otherwise fallback to simple search
		var docs []Document
		var err error

		if e.ftsReady[lang] {
			docs, err = store.Search(ctx, query, limit*2, offset) // Get more to allow cross-language ranking
		} else {
			docs, err = store.SearchSimple(ctx, query, limit*2, offset)
		}

		if err != nil {
			continue // Skip errors from individual languages
		}

		for i := range docs {
			docs[i].Language = lang
		}
		allDocs = append(allDocs, docs...)
	}

	// Sort by score
	sortByScore(allDocs)

	// Limit results
	if len(allDocs) > limit {
		allDocs = allDocs[:limit]
	}

	// Convert to engine results
	for _, doc := range allDocs {
		result := e.toResult(doc)
		results.Add(result)
	}

	return results, nil
}

func sortByScore(docs []Document) {
	for i := 0; i < len(docs)-1; i++ {
		for j := i + 1; j < len(docs); j++ {
			if docs[j].Score > docs[i].Score {
				docs[i], docs[j] = docs[j], docs[i]
			}
		}
	}
}

func (e *Engine) toResult(doc Document) engines.Result {
	return engines.Result{
		URL:      doc.URL,
		Title:    extractTitle(doc.Text, doc.URL),
		Content:  truncateText(doc.Text, e.config.ContentSnippetLength),
		Engine:   e.Name(),
		Category: engines.CategoryGeneral,
		Score:    doc.Score,
	}
}

// extractTitle extracts a title from text or URL.
func extractTitle(text, url string) string {
	// Try to get first line as title
	if idx := strings.IndexAny(text, "\n\r"); idx > 0 {
		title := strings.TrimSpace(text[:idx])
		if len(title) > 10 && len(title) < 200 {
			return truncateText(title, 100)
		}
	}

	// Use first N characters
	if len(text) > 50 {
		title := strings.TrimSpace(text[:50])
		// Find a good break point
		if idx := strings.LastIndexAny(title, " .,;:!?"); idx > 20 {
			title = title[:idx]
		}
		return title + "..."
	}

	// Fallback to URL
	if url != "" {
		// Extract domain or path
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimPrefix(url, "http://")
		if idx := strings.Index(url, "/"); idx > 0 {
			return url[:idx]
		}
		return url
	}

	return "Untitled"
}

// truncateText truncates text to maxLen characters at word boundary.
func truncateText(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	// Remove newlines
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	// Collapse multiple spaces
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	if utf8.RuneCountInString(text) <= maxLen {
		return text
	}

	// Truncate at character boundary
	runes := []rune(text)
	if len(runes) > maxLen {
		runes = runes[:maxLen]
	}

	// Find last space
	result := string(runes)
	if idx := strings.LastIndex(result, " "); idx > maxLen/2 {
		result = result[:idx]
	}

	return result + "..."
}

// ImportLanguage imports data for a language.
func (e *Engine) ImportLanguage(ctx context.Context, lang string, progress func(file string, rows int64)) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Get or create store
	store, ok := e.stores[lang]
	if !ok {
		var err error
		store, err = NewStore(lang, e.config.DataDir)
		if err != nil {
			return fmt.Errorf("creating store: %w", err)
		}
		e.stores[lang] = store
	}

	// Import from parquet files
	parquetDir := filepath.Join(e.config.SourceDir, lang, "train")
	if err := store.Import(ctx, parquetDir, progress); err != nil {
		return fmt.Errorf("importing data: %w", err)
	}

	// Create FTS index
	if err := store.CreateFTSIndex(ctx); err != nil {
		// FTS might fail, continue without it
		return nil
	}
	e.ftsReady[lang] = true

	return nil
}

// GetLanguages returns available languages.
func (e *Engine) GetLanguages() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	langs := make([]string, 0, len(e.stores))
	for lang := range e.stores {
		langs = append(langs, lang)
	}
	return langs
}

// GetDocumentCount returns total document count.
func (e *Engine) GetDocumentCount(ctx context.Context) (int64, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var total int64
	for _, store := range e.stores {
		count, err := store.Count(ctx)
		if err != nil {
			continue
		}
		total += count
	}
	return total, nil
}

// Close closes all stores.
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, store := range e.stores {
		store.Close()
	}
	e.stores = nil
	return nil
}
