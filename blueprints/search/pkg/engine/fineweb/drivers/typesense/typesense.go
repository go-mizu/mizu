// Package typesense provides a Typesense-based driver for fineweb full-text search.
package typesense

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"strings"
	"time"

	"github.com/typesense/typesense-go/v2/typesense"
	"github.com/typesense/typesense-go/v2/typesense/api"
	"github.com/typesense/typesense-go/v2/typesense/api/pointer"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

func init() {
	fineweb.Register("typesense", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// DefaultHost is the default Typesense address.
const DefaultHost = "http://localhost:8108"

// DefaultAPIKey is the default API key for the benchmark.
const DefaultAPIKey = "fineweb-benchmark-key"

// DefaultCollectionName is the default collection name.
const DefaultCollectionName = "fineweb"

// Driver implements the fineweb.Driver interface using Typesense.
type Driver struct {
	client         *typesense.Client
	collectionName string
	host           string
	apiKey         string
	language       string
}

// TypesenseDocument is the document structure for Typesense.
type TypesenseDocument struct {
	ID            string  `json:"id"`
	URL           string  `json:"url"`
	Text          string  `json:"text"`
	Dump          string  `json:"dump"`
	Date          string  `json:"date"`
	Language      string  `json:"language"`
	LanguageScore float64 `json:"language_score"`
}

// New creates a new Typesense driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	host := cfg.GetString("host", DefaultHost)
	apiKey := cfg.GetString("api_key", DefaultAPIKey)

	collectionName := cfg.GetString("collection", DefaultCollectionName)
	if cfg.Language != "" {
		collectionName = cfg.Language
	}

	client := typesense.NewClient(
		typesense.WithServer(host),
		typesense.WithAPIKey(apiKey),
	)

	d := &Driver{
		client:         client,
		collectionName: collectionName,
		host:           host,
		apiKey:         apiKey,
		language:       cfg.Language,
	}

	// Create or get collection
	if err := d.ensureCollection(context.Background()); err != nil {
		return nil, fmt.Errorf("ensuring collection: %w", err)
	}

	return d, nil
}

func (d *Driver) ensureCollection(ctx context.Context) error {
	// Check if collection exists
	_, err := d.client.Collection(d.collectionName).Retrieve(ctx)
	if err == nil {
		// Collection exists
		return nil
	}

	// Create the collection with schema
	schema := &api.CollectionSchema{
		Name: d.collectionName,
		Fields: []api.Field{
			{Name: "id", Type: "string"},
			{Name: "url", Type: "string"},
			{Name: "text", Type: "string"},
			{Name: "dump", Type: "string"},
			{Name: "date", Type: "string"},
			{Name: "language", Type: "string"},
			{Name: "language_score", Type: "float"},
		},
	}

	_, err = d.client.Collections().Create(ctx, schema)
	if err != nil {
		// Check if it's "already exists" error
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return fmt.Errorf("creating collection: %w", err)
	}

	return nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "typesense"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "typesense",
		Description: "Typesense with typo tolerance, faceting, and instant search",
		Features:    []string{"typo-tolerance", "facets", "filters", "instant-search", "geo-search"},
		External:    true,
	}
}

// Search performs full-text search.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// Build search parameters
	searchParams := &api.SearchCollectionParams{
		Q:       pointer.String(query),
		QueryBy: pointer.String("text"),
		Page:    pointer.Int((offset / limit) + 1),
		PerPage: pointer.Int(limit),
	}

	searchRes, err := d.client.Collection(d.collectionName).Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("executing search: %w", err)
	}

	// Convert results
	docs := make([]fineweb.Document, 0)
	if searchRes.Hits != nil {
		for i, hit := range *searchRes.Hits {
			if hit.Document == nil {
				continue
			}

			doc := fineweb.Document{
				Score: 1.0 / float64(i+1), // Default score based on rank
			}

			// Extract text_match score if available
			if hit.TextMatch != nil {
				doc.Score = float64(*hit.TextMatch)
			}

			// Parse document fields from map
			docMap := *hit.Document

			if id, ok := docMap["id"].(string); ok {
				doc.ID = id
			}
			if url, ok := docMap["url"].(string); ok {
				doc.URL = url
			}
			if text, ok := docMap["text"].(string); ok {
				doc.Text = text
			}
			if dump, ok := docMap["dump"].(string); ok {
				doc.Dump = dump
			}
			if date, ok := docMap["date"].(string); ok {
				doc.Date = date
			}
			if language, ok := docMap["language"].(string); ok {
				doc.Language = language
			}
			if langScore, ok := docMap["language_score"].(float64); ok {
				doc.LanguageScore = langScore
			}

			docs = append(docs, doc)
		}
	}

	var total int64
	if searchRes.Found != nil {
		total = int64(*searchRes.Found)
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "typesense",
		Total:     total,
	}, nil
}

// Import ingests documents from an iterator.
// Uses JSONL batch import for high performance.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	batchSize := 5000 // Increased batch size for better throughput
	batch := make([]any, 0, batchSize)
	var imported int64

	for doc, err := range docs {
		if err != nil {
			return fmt.Errorf("reading document: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		batch = append(batch, TypesenseDocument{
			ID:            doc.ID,
			URL:           doc.URL,
			Text:          doc.Text,
			Dump:          doc.Dump,
			Date:          doc.Date,
			Language:      doc.Language,
			LanguageScore: doc.LanguageScore,
		})

		if len(batch) >= batchSize {
			if err := d.importBatch(ctx, batch); err != nil {
				return fmt.Errorf("importing batch: %w", err)
			}

			imported += int64(len(batch))
			batch = batch[:0]

			if progress != nil {
				progress(imported, 0)
			}
		}
	}

	// Import remaining documents
	if len(batch) > 0 {
		if err := d.importBatch(ctx, batch); err != nil {
			return fmt.Errorf("importing final batch: %w", err)
		}
		imported += int64(len(batch))
	}

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

// importBatch imports a batch of documents using JSONL format.
func (d *Driver) importBatch(ctx context.Context, batch []any) error {
	// Convert batch to JSONL
	var jsonlBuilder strings.Builder
	for _, doc := range batch {
		jsonBytes, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshaling document: %w", err)
		}
		jsonlBuilder.Write(jsonBytes)
		jsonlBuilder.WriteByte('\n')
	}

	// Import using JSONL
	action := "upsert"
	params := &api.ImportDocumentsParams{
		Action: &action,
	}

	_, err := d.client.Collection(d.collectionName).Documents().ImportJsonl(ctx, strings.NewReader(jsonlBuilder.String()), params)
	if err != nil {
		return fmt.Errorf("importing JSONL: %w", err)
	}

	return nil
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	collection, err := d.client.Collection(d.collectionName).Retrieve(ctx)
	if err != nil {
		return 0, fmt.Errorf("getting collection: %w", err)
	}
	if collection.NumDocuments == nil {
		return 0, nil
	}
	return *collection.NumDocuments, nil
}

// Close is a no-op for Typesense (HTTP client).
func (d *Driver) Close() error {
	return nil
}

// WaitForService waits for Typesense to be ready.
func WaitForService(ctx context.Context, host, apiKey string, timeout time.Duration) error {
	client := typesense.NewClient(
		typesense.WithServer(host),
		typesense.WithAPIKey(apiKey),
	)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if _, err := client.Health(ctx, 5*time.Second); err == nil {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("typesense not ready after %v", timeout)
}

// IsServiceAvailable checks if Typesense is reachable.
func IsServiceAvailable(host, apiKey string) bool {
	if host == "" {
		host = DefaultHost
	}
	if apiKey == "" {
		apiKey = DefaultAPIKey
	}
	client := typesense.NewClient(
		typesense.WithServer(host),
		typesense.WithAPIKey(apiKey),
	)
	_, err := client.Health(context.Background(), 5*time.Second)
	return err == nil
}

// NewWithEnv creates a driver using environment variables.
func NewWithEnv(cfg fineweb.DriverConfig) (*Driver, error) {
	if cfg.Options == nil {
		cfg.Options = make(map[string]any)
	}
	if host := os.Getenv("TYPESENSE_URL"); host != "" {
		cfg.Options["host"] = host
	}
	if apiKey := os.Getenv("TYPESENSE_API_KEY"); apiKey != "" {
		cfg.Options["api_key"] = apiKey
	}
	return New(cfg)
}

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
