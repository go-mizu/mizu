package store

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/types"
)

// Re-export types from the types package for backwards compatibility.
// New code should import from types package directly.
type (
	Document             = types.Document
	Sitelink             = types.Sitelink
	SearchResult         = types.SearchResult
	ImageResult          = types.ImageResult
	VideoResult          = types.VideoResult
	NewsResult           = types.NewsResult
	MusicResult          = types.MusicResult
	FileResult           = types.FileResult
	ITResult             = types.ITResult
	ScienceResult        = types.ScienceResult
	SocialResult         = types.SocialResult
	SearchOptions        = types.SearchOptions
	SearchResponse       = types.SearchResponse
	Suggestion           = types.Suggestion
	InstantAnswer        = types.InstantAnswer
	CalculatorResult     = types.CalculatorResult
	UnitConversionResult = types.UnitConversionResult
	CurrencyResult       = types.CurrencyResult
	WeatherResult        = types.WeatherResult
	DefinitionResult     = types.DefinitionResult
	TimeResult           = types.TimeResult
	KnowledgePanel       = types.KnowledgePanel
	Fact                 = types.Fact
	Link                 = types.Link
	Entity               = types.Entity
	UserPreference       = types.UserPreference
	SearchLens           = types.SearchLens
	SearchHistory        = types.SearchHistory
	SearchSettings       = types.SearchSettings
	IndexStats           = types.IndexStats
	DomainStat           = types.DomainStat
)

// Store defines the interface for all storage operations.
type Store interface {
	// Schema management
	Ensure(ctx context.Context) error
	CreateExtensions(ctx context.Context) error
	Close() error

	// Seeding
	SeedDocuments(ctx context.Context) error
	SeedKnowledge(ctx context.Context) error

	// Feature stores
	Search() SearchStore
	Index() IndexStore
	Suggest() SuggestStore
	Knowledge() KnowledgeStore
	History() HistoryStore
	Preference() PreferenceStore
}

// ========== Store Interfaces ==========

// SearchStore handles search operations.
type SearchStore interface {
	// Full-text search
	Search(ctx context.Context, query string, opts SearchOptions) (*SearchResponse, error)
	SearchImages(ctx context.Context, query string, opts SearchOptions) ([]ImageResult, error)
	SearchVideos(ctx context.Context, query string, opts SearchOptions) ([]VideoResult, error)
	SearchNews(ctx context.Context, query string, opts SearchOptions) ([]NewsResult, error)
}

// IndexStore handles document indexing.
type IndexStore interface {
	// Document indexing
	IndexDocument(ctx context.Context, doc *Document) error
	UpdateDocument(ctx context.Context, doc *Document) error
	DeleteDocument(ctx context.Context, id string) error
	GetDocument(ctx context.Context, id string) (*Document, error)
	GetDocumentByURL(ctx context.Context, url string) (*Document, error)
	ListDocuments(ctx context.Context, limit, offset int) ([]*Document, error)

	// Bulk operations
	BulkIndex(ctx context.Context, docs []*Document) error

	// Statistics
	GetIndexStats(ctx context.Context) (*IndexStats, error)

	// Maintenance
	RebuildIndex(ctx context.Context) error
	OptimizeIndex(ctx context.Context) error
}

// SuggestStore handles autocomplete suggestions.
type SuggestStore interface {
	// Autocomplete
	GetSuggestions(ctx context.Context, prefix string, limit int) ([]Suggestion, error)
	RecordQuery(ctx context.Context, query string) error
	GetTrendingQueries(ctx context.Context, limit int) ([]string, error)
}

// KnowledgeStore handles knowledge graph operations.
type KnowledgeStore interface {
	// Knowledge graph
	GetEntity(ctx context.Context, query string) (*KnowledgePanel, error)
	CreateEntity(ctx context.Context, entity *Entity) error
	UpdateEntity(ctx context.Context, entity *Entity) error
	DeleteEntity(ctx context.Context, id string) error
	ListEntities(ctx context.Context, entityType string, limit, offset int) ([]*Entity, error)
}

// HistoryStore handles search history.
type HistoryStore interface {
	// Search history
	RecordSearch(ctx context.Context, history *SearchHistory) error
	GetHistory(ctx context.Context, limit, offset int) ([]*SearchHistory, error)
	ClearHistory(ctx context.Context) error
	DeleteHistoryEntry(ctx context.Context, id string) error
}

// PreferenceStore handles user preferences.
type PreferenceStore interface {
	// Domain preferences
	SetPreference(ctx context.Context, pref *UserPreference) error
	GetPreferences(ctx context.Context) ([]*UserPreference, error)
	DeletePreference(ctx context.Context, domain string) error

	// Lenses
	CreateLens(ctx context.Context, lens *SearchLens) error
	GetLens(ctx context.Context, id string) (*SearchLens, error)
	ListLenses(ctx context.Context) ([]*SearchLens, error)
	UpdateLens(ctx context.Context, lens *SearchLens) error
	DeleteLens(ctx context.Context, id string) error

	// Settings
	GetSettings(ctx context.Context) (*SearchSettings, error)
	UpdateSettings(ctx context.Context, settings *SearchSettings) error
}
