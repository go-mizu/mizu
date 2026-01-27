// Package engines provides search engine implementations.
package engines

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// Category represents a search category.
type Category string

// Search categories.
const (
	CategoryGeneral Category = "general"
	CategoryWeb     Category = "web"
	CategoryImages  Category = "images"
	CategoryVideos  Category = "videos"
	CategoryNews    Category = "news"
	CategoryMusic   Category = "music"
	CategoryFiles   Category = "files"
	CategoryIT      Category = "it"
	CategoryScience Category = "science"
	CategorySocial  Category = "social media"
	CategoryMaps    Category = "map"
	CategoryOther   Category = "other"
)

// EngineType defines the type of search engine.
type EngineType string

// Engine types.
const (
	EngineTypeOnline           EngineType = "online"
	EngineTypeOffline          EngineType = "offline"
	EngineTypeOnlineDictionary EngineType = "online_dictionary"
	EngineTypeOnlineCurrency   EngineType = "online_currency"
	EngineTypeOnlineURLSearch  EngineType = "online_url_search"
)

// SafeSearchLevel defines safe search filtering level.
type SafeSearchLevel int

// Safe search levels.
const (
	SafeSearchOff      SafeSearchLevel = 0
	SafeSearchModerate SafeSearchLevel = 1
	SafeSearchStrict   SafeSearchLevel = 2
)

// TimeRange defines time range filter.
type TimeRange string

// Time range options.
const (
	TimeRangeNone  TimeRange = ""
	TimeRangeDay   TimeRange = "day"
	TimeRangeWeek  TimeRange = "week"
	TimeRangeMonth TimeRange = "month"
	TimeRangeYear  TimeRange = "year"
)

// Priority defines result priority.
type Priority string

// Priority levels.
const (
	PriorityDefault Priority = ""
	PriorityLow     Priority = "low"
	PriorityHigh    Priority = "high"
)

// RequestParams contains all parameters for a search request.
type RequestParams struct {
	// Basic parameters
	Query      string
	Category   Category
	PageNo     int
	SafeSearch SafeSearchLevel
	TimeRange  TimeRange
	Language   string
	Locale     string

	// HTTP parameters (for online engines)
	Method  string
	URL     string
	Headers http.Header
	Cookies []*http.Cookie
	Data    url.Values
	JSON    map[string]any
	Content []byte

	// Request options
	AllowRedirects    bool
	MaxRedirects      int
	SoftMaxRedirects  int
	Timeout           time.Duration
	RaiseForHTTPError bool
	Verify            *bool
	Auth              string

	// Engine-specific data
	EngineData map[string]string
}

// NewRequestParams creates a new RequestParams with defaults.
func NewRequestParams() *RequestParams {
	return &RequestParams{
		Method:            "GET",
		Headers:           make(http.Header),
		Cookies:           make([]*http.Cookie, 0),
		Data:              make(url.Values),
		JSON:              make(map[string]any),
		AllowRedirects:    false,
		MaxRedirects:      0,
		SoftMaxRedirects:  0,
		RaiseForHTTPError: true,
		EngineData:        make(map[string]string),
	}
}

// Result represents a search result.
type Result struct {
	// Common fields
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Content     string    `json:"content,omitempty"`
	Category    Category  `json:"category,omitempty"`
	Engine      string    `json:"engine"`
	Engines     []string  `json:"engines,omitempty"`
	Score       float64   `json:"score"`
	Priority    Priority  `json:"priority,omitempty"`
	Template    string    `json:"template,omitempty"`
	ParsedURL   *url.URL  `json:"-"`
	PublishedAt time.Time `json:"published_at,omitempty"`

	// Image fields
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	ImageURL     string `json:"img_src,omitempty"`
	ImgFormat    string `json:"img_format,omitempty"`
	Source       string `json:"source,omitempty"`
	Resolution   string `json:"resolution,omitempty"`

	// Video fields
	Duration  string `json:"duration,omitempty"`
	EmbedURL  string `json:"embed_url,omitempty"`
	IFrameSrc string `json:"iframe_src,omitempty"`

	// Music fields
	Artist string `json:"artist,omitempty"`
	Album  string `json:"album,omitempty"`
	Track  string `json:"track,omitempty"`

	// File fields
	FileSize   string `json:"filesize,omitempty"`
	MagnetLink string `json:"magnetlink,omitempty"`
	Seed       int    `json:"seed,omitempty"`
	Leech      int    `json:"leech,omitempty"`

	// Science fields
	DOI         string   `json:"doi,omitempty"`
	ISSN        string   `json:"issn,omitempty"`
	ISBN        string   `json:"isbn,omitempty"`
	Authors     []string `json:"authors,omitempty"`
	Publisher   string   `json:"publisher,omitempty"`
	Journal     string   `json:"journal,omitempty"`
	Type        string   `json:"type,omitempty"`
	AccessRight string   `json:"access_right,omitempty"`

	// Map fields
	Latitude    float64  `json:"latitude,omitempty"`
	Longitude   float64  `json:"longitude,omitempty"`
	BoundingBox []string `json:"boundingbox,omitempty"`
	GeoJSON     any      `json:"geojson,omitempty"`
	Address     *Address `json:"address,omitempty"`
	OSMType     string   `json:"osm_type,omitempty"`
	OSMID       string   `json:"osm_id,omitempty"`

	// Internal fields
	Positions []int  `json:"-"`
	Hash      uint64 `json:"-"`
}

// Address represents a geographic address.
type Address struct {
	Name        string `json:"name,omitempty"`
	Road        string `json:"road,omitempty"`
	Locality    string `json:"locality,omitempty"`
	PostalCode  string `json:"postcode,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	HouseNumber string `json:"house_number,omitempty"`
}

// Answer represents an instant answer.
type Answer struct {
	Answer string `json:"answer"`
	URL    string `json:"url,omitempty"`
}

// Translation represents a translation item.
type Translation struct {
	Text           string   `json:"text"`
	Transliteration string   `json:"transliteration,omitempty"`
	Examples       []string `json:"examples,omitempty"`
	Definitions    []string `json:"definitions,omitempty"`
	Synonyms       []string `json:"synonyms,omitempty"`
}

// TranslationsAnswer represents a translation answer.
type TranslationsAnswer struct {
	SourceLang   string        `json:"source_lang"`
	TargetLang   string        `json:"target_lang"`
	SourceText   string        `json:"source_text"`
	Translations []Translation `json:"translations"`
}

// WeatherCondition represents weather condition type.
type WeatherCondition string

// Weather conditions.
const (
	WeatherClear        WeatherCondition = "clear"
	WeatherCloudy       WeatherCondition = "cloudy"
	WeatherPartlyCloudy WeatherCondition = "partly_cloudy"
	WeatherRain         WeatherCondition = "rain"
	WeatherSnow         WeatherCondition = "snow"
	WeatherThunderstorm WeatherCondition = "thunderstorm"
	WeatherFog          WeatherCondition = "fog"
	WeatherWind         WeatherCondition = "wind"
)

// WeatherItem represents weather data for a specific time.
type WeatherItem struct {
	Location    string           `json:"location"`
	Temperature float64          `json:"temperature"`
	FeelsLike   float64          `json:"feels_like,omitempty"`
	Humidity    int              `json:"humidity,omitempty"`
	Pressure    float64          `json:"pressure,omitempty"`
	WindSpeed   float64          `json:"wind_speed,omitempty"`
	WindDir     string           `json:"wind_direction,omitempty"`
	Condition   WeatherCondition `json:"condition"`
	Summary     string           `json:"summary,omitempty"`
	DateTime    string           `json:"datetime,omitempty"`
	CloudCover  int              `json:"cloud_cover,omitempty"`
	Unit        string           `json:"unit,omitempty"` // "celsius" or "fahrenheit"
}

// WeatherAnswer represents a weather answer.
type WeatherAnswer struct {
	Current   WeatherItem   `json:"current"`
	Forecasts []WeatherItem `json:"forecasts,omitempty"`
	Service   string        `json:"service,omitempty"`
}

// KeyValueAnswer represents a key-value table answer.
type KeyValueAnswer struct {
	Data       map[string]any `json:"data"`
	Caption    string         `json:"caption,omitempty"`
	KeyTitle   string         `json:"key_title,omitempty"`
	ValueTitle string         `json:"value_title,omitempty"`
}

// CodeResult represents a code snippet result.
type CodeResult struct {
	Result
	Repository    string         `json:"repository,omitempty"`
	CodeLines     []CodeLine     `json:"code_lines,omitempty"`
	HighlightLines []int         `json:"highlight_lines,omitempty"`
	Language      string         `json:"code_language,omitempty"`
	Filename      string         `json:"filename,omitempty"`
}

// CodeLine represents a single line of code.
type CodeLine struct {
	LineNumber int    `json:"line_number"`
	Code       string `json:"code"`
}

// PaperResult represents a scientific paper result.
type PaperResult struct {
	Result
	Abstract    string   `json:"abstract,omitempty"`
	Comments    string   `json:"comments,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Editor      string   `json:"editor,omitempty"`
	Volume      string   `json:"volume,omitempty"`
	Pages       string   `json:"pages,omitempty"`
	Number      string   `json:"number,omitempty"`
	PDFUrl      string   `json:"pdf_url,omitempty"`
	HTMLUrl     string   `json:"html_url,omitempty"`
}

// FileResult represents a file/download result.
type FileResult struct {
	Result
	Filename string `json:"filename,omitempty"`
	MimeType string `json:"mimetype,omitempty"`
	Size     string `json:"size,omitempty"`
	ModTime  string `json:"mod_time,omitempty"`
	Embedded string `json:"embedded,omitempty"` // URL for embedded media
	MediaType string `json:"media_type,omitempty"` // "audio" or "video"
}

// Infobox represents a knowledge panel.
type Infobox struct {
	ID         string              `json:"id,omitempty"`
	Title      string              `json:"infobox"`
	Content    string              `json:"content,omitempty"`
	ImageURL   string              `json:"img_src,omitempty"`
	URLs       []InfoboxURL        `json:"urls,omitempty"`
	Attributes []InfoboxAttribute  `json:"attributes,omitempty"`
	Engine     string              `json:"engine"`
	Engines    map[string]struct{} `json:"-"`
}

// InfoboxURL represents a link in an infobox.
type InfoboxURL struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Entity string `json:"entity,omitempty"`
}

// InfoboxAttribute represents a key-value attribute in an infobox.
type InfoboxAttribute struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Entity string `json:"entity,omitempty"`
}

// EngineAbout contains metadata about an engine.
type EngineAbout struct {
	Website         string `json:"website,omitempty"`
	WikidataID      string `json:"wikidata_id,omitempty"`
	OfficialAPIDocs string `json:"official_api_documentation,omitempty"`
	UseOfficialAPI  bool   `json:"use_official_api,omitempty"`
	RequireAPIKey   bool   `json:"require_api_key,omitempty"`
	Results         string `json:"results,omitempty"`
}

// Engine is the base interface all search engines must implement.
type Engine interface {
	// Name returns the engine name.
	Name() string

	// Shortcut returns the bang shortcut (e.g., "g" for !g).
	Shortcut() string

	// Categories returns supported categories.
	Categories() []Category

	// EngineType returns the engine type.
	EngineType() EngineType

	// SupportsPaging returns true if engine supports pagination.
	SupportsPaging() bool

	// SupportsTimeRange returns true if engine supports time filtering.
	SupportsTimeRange() bool

	// SupportsSafeSearch returns true if engine supports safe search.
	SupportsSafeSearch() bool

	// SupportsLanguage returns true if engine supports language selection.
	SupportsLanguage() bool

	// MaxPage returns max supported page (0 = unlimited).
	MaxPage() int

	// Timeout returns the engine timeout.
	Timeout() time.Duration

	// Weight returns the engine weight for scoring.
	Weight() float64

	// Disabled returns true if engine is disabled by default.
	Disabled() bool

	// About returns engine metadata.
	About() EngineAbout

	// Traits returns engine traits (language/region mappings).
	Traits() *EngineTraits
}

// OnlineEngine extends Engine for HTTP-based engines.
type OnlineEngine interface {
	Engine

	// Request builds the HTTP request parameters.
	Request(ctx context.Context, query string, params *RequestParams) error

	// Response parses the HTTP response.
	Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error)
}

// OfflineEngine extends Engine for local data sources.
type OfflineEngine interface {
	Engine

	// Search performs a local search.
	Search(ctx context.Context, query string, params *RequestParams) (*EngineResults, error)
}

// EngineResults contains results from an engine.
type EngineResults struct {
	Results     []Result
	Suggestions []string
	Corrections []string
	Answers     []Answer
	Infoboxes   []Infobox
	EngineData  map[string]string
}

// NewEngineResults creates a new EngineResults.
func NewEngineResults() *EngineResults {
	return &EngineResults{
		Results:     make([]Result, 0),
		Suggestions: make([]string, 0),
		Corrections: make([]string, 0),
		Answers:     make([]Answer, 0),
		Infoboxes:   make([]Infobox, 0),
		EngineData:  make(map[string]string),
	}
}

// Add adds a result to the results list.
func (r *EngineResults) Add(result Result) {
	r.Results = append(r.Results, result)
}

// AddSuggestion adds a suggestion.
func (r *EngineResults) AddSuggestion(suggestion string) {
	r.Suggestions = append(r.Suggestions, suggestion)
}

// AddCorrection adds a correction.
func (r *EngineResults) AddCorrection(correction string) {
	r.Corrections = append(r.Corrections, correction)
}

// AddAnswer adds an answer.
func (r *EngineResults) AddAnswer(answer Answer) {
	r.Answers = append(r.Answers, answer)
}

// AddInfobox adds an infobox.
func (r *EngineResults) AddInfobox(infobox Infobox) {
	r.Infoboxes = append(r.Infoboxes, infobox)
}

// SetEngineData sets engine-specific data for subsequent requests.
func (r *EngineResults) SetEngineData(key, value string) {
	r.EngineData[key] = value
}

// EngineTraits stores language/region mappings for engines.
type EngineTraits struct {
	// AllLocale is the identifier for "all" locales.
	AllLocale string

	// Languages maps SearXNG locale to engine language code.
	Languages map[string]string

	// Regions maps SearXNG locale to engine region code.
	Regions map[string]string

	// Custom holds engine-specific data.
	Custom map[string]any
}

// NewEngineTraits creates a new EngineTraits.
func NewEngineTraits() *EngineTraits {
	return &EngineTraits{
		AllLocale: "all",
		Languages: make(map[string]string),
		Regions:   make(map[string]string),
		Custom:    make(map[string]any),
	}
}

// GetLanguage returns the engine language for a locale.
func (t *EngineTraits) GetLanguage(locale, fallback string) string {
	if locale == "" || locale == "all" {
		return fallback
	}
	if lang, ok := t.Languages[locale]; ok {
		return lang
	}
	// Try just the language part
	if len(locale) >= 2 {
		if lang, ok := t.Languages[locale[:2]]; ok {
			return lang
		}
	}
	return fallback
}

// GetRegion returns the engine region for a locale.
func (t *EngineTraits) GetRegion(locale, fallback string) string {
	if locale == "" || locale == "all" {
		return fallback
	}
	if region, ok := t.Regions[locale]; ok {
		return region
	}
	return fallback
}

// BaseEngine provides common functionality for engines.
type BaseEngine struct {
	name             string
	shortcut         string
	categories       []Category
	engineType       EngineType
	paging           bool
	timeRangeSupport bool
	safeSearch       bool
	languageSupport  bool
	maxPage          int
	timeout          time.Duration
	weight           float64
	disabled         bool
	about            EngineAbout
	traits           *EngineTraits
}

// NewBaseEngine creates a new BaseEngine.
func NewBaseEngine(name, shortcut string, categories []Category) *BaseEngine {
	return &BaseEngine{
		name:            name,
		shortcut:        shortcut,
		categories:      categories,
		engineType:      EngineTypeOnline,
		paging:          false,
		timeRangeSupport: false,
		safeSearch:      false,
		languageSupport: true,
		maxPage:         0,
		timeout:         3 * time.Second,
		weight:          1.0,
		disabled:        false,
		traits:          NewEngineTraits(),
	}
}

func (e *BaseEngine) Name() string             { return e.name }
func (e *BaseEngine) Shortcut() string         { return e.shortcut }
func (e *BaseEngine) Categories() []Category   { return e.categories }
func (e *BaseEngine) EngineType() EngineType   { return e.engineType }
func (e *BaseEngine) SupportsPaging() bool     { return e.paging }
func (e *BaseEngine) SupportsTimeRange() bool  { return e.timeRangeSupport }
func (e *BaseEngine) SupportsSafeSearch() bool { return e.safeSearch }
func (e *BaseEngine) SupportsLanguage() bool   { return e.languageSupport }
func (e *BaseEngine) MaxPage() int             { return e.maxPage }
func (e *BaseEngine) Timeout() time.Duration   { return e.timeout }
func (e *BaseEngine) Weight() float64          { return e.weight }
func (e *BaseEngine) Disabled() bool           { return e.disabled }
func (e *BaseEngine) About() EngineAbout       { return e.about }
func (e *BaseEngine) Traits() *EngineTraits    { return e.traits }

// SetPaging enables/disables paging support.
func (e *BaseEngine) SetPaging(paging bool) *BaseEngine {
	e.paging = paging
	return e
}

// SetTimeRangeSupport enables/disables time range support.
func (e *BaseEngine) SetTimeRangeSupport(support bool) *BaseEngine {
	e.timeRangeSupport = support
	return e
}

// SetSafeSearch enables/disables safe search support.
func (e *BaseEngine) SetSafeSearch(safeSearch bool) *BaseEngine {
	e.safeSearch = safeSearch
	return e
}

// SetLanguageSupport enables/disables language support.
func (e *BaseEngine) SetLanguageSupport(support bool) *BaseEngine {
	e.languageSupport = support
	return e
}

// SetMaxPage sets the maximum page number.
func (e *BaseEngine) SetMaxPage(maxPage int) *BaseEngine {
	e.maxPage = maxPage
	return e
}

// SetTimeout sets the engine timeout.
func (e *BaseEngine) SetTimeout(timeout time.Duration) *BaseEngine {
	e.timeout = timeout
	return e
}

// SetWeight sets the engine weight.
func (e *BaseEngine) SetWeight(weight float64) *BaseEngine {
	e.weight = weight
	return e
}

// SetDisabled sets whether the engine is disabled.
func (e *BaseEngine) SetDisabled(disabled bool) *BaseEngine {
	e.disabled = disabled
	return e
}

// SetAbout sets the engine about info.
func (e *BaseEngine) SetAbout(about EngineAbout) *BaseEngine {
	e.about = about
	return e
}

// SetEngineType sets the engine type.
func (e *BaseEngine) SetEngineType(engineType EngineType) *BaseEngine {
	e.engineType = engineType
	return e
}
