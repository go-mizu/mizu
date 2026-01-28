// Package local provides a local metasearch engine implementation.
// It is a Go port of SearXNG (https://github.com/searxng/searxng).
package local

import (
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/local/engines"
)

// Re-export types from engines package for convenience.
type (
	Category        = engines.Category
	EngineType      = engines.EngineType
	SafeSearchLevel = engines.SafeSearchLevel
	TimeRange       = engines.TimeRange
	Priority        = engines.Priority
	RequestParams   = engines.RequestParams
	Result          = engines.Result
	Address         = engines.Address
	Answer          = engines.Answer
	Infobox         = engines.Infobox
	InfoboxURL      = engines.InfoboxURL
	InfoboxAttribute = engines.InfoboxAttribute
	EngineAbout     = engines.EngineAbout
	EngineResults   = engines.EngineResults
	EngineTraits    = engines.EngineTraits
	Engine          = engines.Engine
	OnlineEngine    = engines.OnlineEngine
	OfflineEngine   = engines.OfflineEngine
)

// Re-export constants from engines package.
const (
	CategoryGeneral = engines.CategoryGeneral
	CategoryWeb     = engines.CategoryWeb
	CategoryImages  = engines.CategoryImages
	CategoryVideos  = engines.CategoryVideos
	CategoryNews    = engines.CategoryNews
	CategoryMusic   = engines.CategoryMusic
	CategoryFiles   = engines.CategoryFiles
	CategoryIT      = engines.CategoryIT
	CategoryScience = engines.CategoryScience
	CategorySocial  = engines.CategorySocial
	CategoryMaps    = engines.CategoryMaps
	CategoryOther   = engines.CategoryOther

	EngineTypeOnline           = engines.EngineTypeOnline
	EngineTypeOffline          = engines.EngineTypeOffline
	EngineTypeOnlineDictionary = engines.EngineTypeOnlineDictionary
	EngineTypeOnlineCurrency   = engines.EngineTypeOnlineCurrency
	EngineTypeOnlineURLSearch  = engines.EngineTypeOnlineURLSearch

	SafeSearchOff      = engines.SafeSearchOff
	SafeSearchModerate = engines.SafeSearchModerate
	SafeSearchStrict   = engines.SafeSearchStrict

	TimeRangeNone  = engines.TimeRangeNone
	TimeRangeDay   = engines.TimeRangeDay
	TimeRangeWeek  = engines.TimeRangeWeek
	TimeRangeMonth = engines.TimeRangeMonth
	TimeRangeYear  = engines.TimeRangeYear

	PriorityDefault = engines.PriorityDefault
	PriorityLow     = engines.PriorityLow
	PriorityHigh    = engines.PriorityHigh
)

// Re-export functions.
var (
	NewRequestParams  = engines.NewRequestParams
	NewEngineResults  = engines.NewEngineResults
	NewEngineTraits   = engines.NewEngineTraits
	NewBaseEngine     = engines.NewBaseEngine
)

// AllCategories returns all supported categories.
func AllCategories() []Category {
	return []Category{
		CategoryGeneral,
		CategoryWeb,
		CategoryImages,
		CategoryVideos,
		CategoryNews,
		CategoryMusic,
		CategoryFiles,
		CategoryIT,
		CategoryScience,
		CategorySocial,
		CategoryMaps,
	}
}

// EngineTiming contains timing information for an engine.
type EngineTiming struct {
	Engine   string        `json:"engine"`
	Total    time.Duration `json:"total"`
	LoadTime time.Duration `json:"load_time"`
}

// UnresponsiveEngine contains info about an unresponsive engine.
type UnresponsiveEngine struct {
	Engine    string `json:"engine"`
	ErrorType string `json:"error_type"`
	Suspended bool   `json:"suspended"`
}

// SearchResponse is the aggregated response from all engines.
type SearchResponse struct {
	Query          string   `json:"query"`
	CorrectedQuery string   `json:"corrected_query,omitempty"`
	TotalResults   int64    `json:"total_results"`
	Results        []Result `json:"results"`
	Suggestions    []string `json:"suggestions,omitempty"`
	Corrections    []string `json:"corrections,omitempty"`
	Answers        []Answer `json:"answers,omitempty"`
	Infoboxes      []Infobox `json:"infoboxes,omitempty"`
	SearchTimeMs   float64  `json:"search_time_ms"`
	Page           int      `json:"page"`
	PerPage        int      `json:"per_page"`

	// Engine information
	Timings             []EngineTiming       `json:"timings,omitempty"`
	UnresponsiveEngines []UnresponsiveEngine `json:"unresponsive_engines,omitempty"`

	// Pagination
	HasNextPage bool `json:"has_next_page"`

	// Engine data for subsequent requests
	EngineData map[string]map[string]string `json:"engine_data,omitempty"`
}

// SearchOptions contains search configuration options.
type SearchOptions struct {
	Categories []Category
	Engines    []string
	Language   string
	Locale     string
	Page       int
	PerPage    int
	TimeRange  TimeRange
	SafeSearch SafeSearchLevel
	Timeout    time.Duration

	// For plugins
	EnabledPlugins  []string
	DisabledPlugins []string

	// Engine-specific data from previous requests
	EngineData map[string]map[string]string
}

// EngineRef represents a reference to an engine for a search.
type EngineRef struct {
	Name     string
	Category Category
}
