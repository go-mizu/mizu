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
	Thumbnail            = types.Thumbnail
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
	Widget               = types.Widget
	WidgetType           = types.WidgetType
	WidgetSetting        = types.WidgetSetting
	CheatSheet           = types.CheatSheet
	CheatSection         = types.CheatSection
	CheatItem            = types.CheatItem
	Bang                 = types.Bang
	BangResult           = types.BangResult
	SummaryCache         = types.SummaryCache
	SummaryEngine        = types.SummaryEngine
	SummaryType          = types.SummaryType
	SummarizeRequest     = types.SummarizeRequest
	SummarizeResponse    = types.SummarizeResponse
	EnrichmentResult     = types.EnrichmentResult
	SmallWebEntry        = types.SmallWebEntry
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
	Bang() BangStore
	Summary() SummaryStore
	Widget() WidgetStore
	SmallWeb() SmallWebStore
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
	GetPreference(ctx context.Context, domain string) (*UserPreference, error)
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

// BangStore handles bang shortcuts.
type BangStore interface {
	// Bang CRUD
	CreateBang(ctx context.Context, bang *Bang) error
	GetBang(ctx context.Context, trigger string) (*Bang, error)
	ListBangs(ctx context.Context) ([]*Bang, error)
	ListUserBangs(ctx context.Context, userID string) ([]*Bang, error)
	DeleteBang(ctx context.Context, id int64) error
	SeedBuiltinBangs(ctx context.Context) error
}

// SummaryStore handles URL/text summarization cache.
type SummaryStore interface {
	// Summary cache
	GetSummary(ctx context.Context, urlHash, engine, summaryType, lang string) (*SummaryCache, error)
	SaveSummary(ctx context.Context, summary *SummaryCache) error
	DeleteExpiredSummaries(ctx context.Context) error
}

// WidgetStore handles widget settings.
type WidgetStore interface {
	// Widget settings
	GetWidgetSettings(ctx context.Context, userID string) ([]*WidgetSetting, error)
	SetWidgetSetting(ctx context.Context, setting *WidgetSetting) error
	GetCheatSheet(ctx context.Context, language string) (*CheatSheet, error)
	SaveCheatSheet(ctx context.Context, sheet *CheatSheet) error
	ListCheatSheets(ctx context.Context) ([]*CheatSheet, error)
	SeedCheatSheets(ctx context.Context) error

	// Related searches
	GetRelatedSearches(ctx context.Context, queryHash string) ([]string, error)
	SaveRelatedSearches(ctx context.Context, queryHash, query string, related []string) error
}

// SmallWebStore handles small web index for enrichment.
type SmallWebStore interface {
	// Small web entries
	IndexEntry(ctx context.Context, entry *SmallWebEntry) error
	SearchWeb(ctx context.Context, query string, limit int) ([]*EnrichmentResult, error)
	SearchNews(ctx context.Context, query string, limit int) ([]*EnrichmentResult, error)
	SeedSmallWeb(ctx context.Context) error
}
