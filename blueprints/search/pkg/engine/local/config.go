package local

import (
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/local/engines"
)

// Config holds the configuration for the metasearch engine.
type Config struct {
	// Search settings
	DefaultCategories []engines.Category
	DefaultLanguage   string
	DefaultLocale     string
	SafeSearch        engines.SafeSearchLevel

	// Timeout settings
	RequestTimeout    time.Duration
	MaxRequestTimeout time.Duration

	// Pagination
	DefaultPageSize int
	MaxPage         int

	// Engine settings
	Engines []EngineConfig

	// Plugin settings
	Plugins []PluginConfig

	// Answerer settings
	Answerers []AnswererConfig

	// Cache settings
	CacheEnabled bool
	CacheTTL     time.Duration

	// User agent
	UserAgent string

	// HTTP client settings
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
}

// EngineConfig defines engine configuration.
type EngineConfig struct {
	Name       string
	Engine     string // Engine module name
	Shortcut   string
	Categories []engines.Category
	Timeout    time.Duration
	Weight     float64
	Disabled   bool
	Inactive   bool

	// Optional overrides
	BaseURL  string
	Language string
	Region   string
	Proxies  map[string]string

	// Engine-specific settings
	Extra map[string]any
}

// PluginConfig defines plugin configuration.
type PluginConfig struct {
	ID       string
	Active   bool
	Settings map[string]any
}

// AnswererConfig defines answerer configuration.
type AnswererConfig struct {
	ID     string
	Active bool
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		DefaultCategories: []engines.Category{engines.CategoryGeneral},
		DefaultLanguage:   "en",
		DefaultLocale:     "en-US",
		SafeSearch:        engines.SafeSearchModerate,

		RequestTimeout:    5 * time.Second,
		MaxRequestTimeout: 15 * time.Second,

		DefaultPageSize: 10,
		MaxPage:         50,

		CacheEnabled: true,
		CacheTTL:     time.Hour,

		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",

		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,

		Engines: defaultEngineConfigs(),
		Plugins: defaultPluginConfigs(),
	}
}

func defaultEngineConfigs() []EngineConfig {
	return []EngineConfig{
		// General/Web search
		{Name: "google", Engine: "google", Shortcut: "g", Categories: []engines.Category{engines.CategoryGeneral, engines.CategoryWeb}},
		{Name: "bing", Engine: "bing", Shortcut: "b", Categories: []engines.Category{engines.CategoryGeneral, engines.CategoryWeb}},
		{Name: "duckduckgo", Engine: "duckduckgo", Shortcut: "ddg", Categories: []engines.Category{engines.CategoryGeneral, engines.CategoryWeb}},
		{Name: "brave", Engine: "brave", Shortcut: "br", Categories: []engines.Category{engines.CategoryGeneral, engines.CategoryWeb}},
		{Name: "qwant", Engine: "qwant", Shortcut: "qw", Categories: []engines.Category{engines.CategoryGeneral, engines.CategoryWeb}},
		{Name: "startpage", Engine: "startpage", Shortcut: "sp", Categories: []engines.Category{engines.CategoryGeneral, engines.CategoryWeb}},
		{Name: "mojeek", Engine: "mojeek", Shortcut: "mj", Categories: []engines.Category{engines.CategoryGeneral, engines.CategoryWeb}, Disabled: true},
		{Name: "yahoo", Engine: "yahoo", Shortcut: "yh", Categories: []engines.Category{engines.CategoryGeneral, engines.CategoryWeb}, Disabled: true},

		// Image search
		{Name: "google images", Engine: "google_images", Shortcut: "gi", Categories: []engines.Category{engines.CategoryImages}},
		{Name: "bing images", Engine: "bing_images", Shortcut: "bi", Categories: []engines.Category{engines.CategoryImages}},
		{Name: "duckduckgo images", Engine: "duckduckgo_images", Shortcut: "ddi", Categories: []engines.Category{engines.CategoryImages}},
		{Name: "unsplash", Engine: "unsplash", Shortcut: "us", Categories: []engines.Category{engines.CategoryImages}},
		{Name: "flickr", Engine: "flickr", Shortcut: "fl", Categories: []engines.Category{engines.CategoryImages}, Disabled: true},

		// Video search
		{Name: "google videos", Engine: "google_videos", Shortcut: "gv", Categories: []engines.Category{engines.CategoryVideos}},
		{Name: "bing videos", Engine: "bing_videos", Shortcut: "bv", Categories: []engines.Category{engines.CategoryVideos}},
		{Name: "duckduckgo videos", Engine: "duckduckgo_videos", Shortcut: "ddv", Categories: []engines.Category{engines.CategoryVideos}},
		{Name: "youtube", Engine: "youtube", Shortcut: "yt", Categories: []engines.Category{engines.CategoryVideos}},
		{Name: "vimeo", Engine: "vimeo", Shortcut: "vm", Categories: []engines.Category{engines.CategoryVideos}, Disabled: true},
		{Name: "dailymotion", Engine: "dailymotion", Shortcut: "dm", Categories: []engines.Category{engines.CategoryVideos}, Disabled: true},
		{Name: "peertube", Engine: "peertube", Shortcut: "pt", Categories: []engines.Category{engines.CategoryVideos}, Disabled: true},

		// News search
		{Name: "google news", Engine: "google_news", Shortcut: "gn", Categories: []engines.Category{engines.CategoryNews}},
		{Name: "bing news", Engine: "bing_news", Shortcut: "bn", Categories: []engines.Category{engines.CategoryNews}},
		{Name: "duckduckgo news", Engine: "duckduckgo_news", Shortcut: "ddn", Categories: []engines.Category{engines.CategoryNews}},
		{Name: "wikinews", Engine: "wikinews", Shortcut: "wn", Categories: []engines.Category{engines.CategoryNews}, Disabled: true},

		// Science/Academic
		{Name: "arxiv", Engine: "arxiv", Shortcut: "arx", Categories: []engines.Category{engines.CategoryScience}},
		{Name: "google scholar", Engine: "google_scholar", Shortcut: "gs", Categories: []engines.Category{engines.CategoryScience}},
		{Name: "pubmed", Engine: "pubmed", Shortcut: "pm", Categories: []engines.Category{engines.CategoryScience}, Disabled: true},
		{Name: "semantic scholar", Engine: "semantic_scholar", Shortcut: "ss", Categories: []engines.Category{engines.CategoryScience}, Disabled: true},
		{Name: "crossref", Engine: "crossref", Shortcut: "cr", Categories: []engines.Category{engines.CategoryScience}, Disabled: true},

		// IT/Development
		{Name: "github", Engine: "github", Shortcut: "gh", Categories: []engines.Category{engines.CategoryIT}},
		{Name: "gitlab", Engine: "gitlab", Shortcut: "gl", Categories: []engines.Category{engines.CategoryIT}, Disabled: true},
		{Name: "stackoverflow", Engine: "stackoverflow", Shortcut: "so", Categories: []engines.Category{engines.CategoryIT}},
		{Name: "hacker news", Engine: "hackernews", Shortcut: "hn", Categories: []engines.Category{engines.CategoryIT}},
		{Name: "npm", Engine: "npm", Shortcut: "npm", Categories: []engines.Category{engines.CategoryIT}, Disabled: true},
		{Name: "crates.io", Engine: "crates", Shortcut: "crt", Categories: []engines.Category{engines.CategoryIT}, Disabled: true},
		{Name: "docker hub", Engine: "docker_hub", Shortcut: "dh", Categories: []engines.Category{engines.CategoryIT}, Disabled: true},

		// Files/Torrents
		{Name: "1337x", Engine: "1337x", Shortcut: "1337", Categories: []engines.Category{engines.CategoryFiles}, Disabled: true},
		{Name: "piratebay", Engine: "piratebay", Shortcut: "tpb", Categories: []engines.Category{engines.CategoryFiles}, Disabled: true},
		{Name: "nyaa", Engine: "nyaa", Shortcut: "ny", Categories: []engines.Category{engines.CategoryFiles}, Disabled: true},

		// Maps
		{Name: "openstreetmap", Engine: "openstreetmap", Shortcut: "osm", Categories: []engines.Category{engines.CategoryMaps}},
		{Name: "photon", Engine: "photon", Shortcut: "ph", Categories: []engines.Category{engines.CategoryMaps}, Disabled: true},

		// Music
		{Name: "bandcamp", Engine: "bandcamp", Shortcut: "bc", Categories: []engines.Category{engines.CategoryMusic}, Disabled: true},
		{Name: "soundcloud", Engine: "soundcloud", Shortcut: "sc", Categories: []engines.Category{engines.CategoryMusic}, Disabled: true},
		{Name: "genius", Engine: "genius", Shortcut: "gen", Categories: []engines.Category{engines.CategoryMusic}, Disabled: true},

		// Social Media
		{Name: "reddit", Engine: "reddit", Shortcut: "re", Categories: []engines.Category{engines.CategorySocial}},
		{Name: "mastodon", Engine: "mastodon", Shortcut: "mast", Categories: []engines.Category{engines.CategorySocial}, Disabled: true},
		{Name: "lemmy", Engine: "lemmy", Shortcut: "lem", Categories: []engines.Category{engines.CategorySocial}, Disabled: true},

		// Encyclopedias
		{Name: "wikipedia", Engine: "wikipedia", Shortcut: "w", Categories: []engines.Category{engines.CategoryGeneral}},
		{Name: "wikidata", Engine: "wikidata", Shortcut: "wd", Categories: []engines.Category{engines.CategoryGeneral}, Disabled: true},

		// Translation
		{Name: "deepl", Engine: "deepl", Shortcut: "dl", Categories: []engines.Category{engines.CategoryGeneral}, Disabled: true},
		{Name: "libretranslate", Engine: "libretranslate", Shortcut: "lt", Categories: []engines.Category{engines.CategoryGeneral}, Disabled: true},

		// Currency
		{Name: "currency", Engine: "currency_convert", Shortcut: "cc", Categories: []engines.Category{engines.CategoryGeneral}, Disabled: true},
	}
}

func defaultPluginConfigs() []PluginConfig {
	return []PluginConfig{
		{ID: "tracker_url_remover", Active: true},
		{ID: "hostname_blocker", Active: false, Settings: map[string]any{"hosts": []string{}}},
		{ID: "hostname_replacer", Active: false, Settings: map[string]any{
			"replacements": map[string]string{
				"youtube.com":     "invidious.io",
				"www.youtube.com": "invidious.io",
				"twitter.com":     "nitter.net",
				"www.twitter.com": "nitter.net",
				"reddit.com":      "teddit.net",
				"www.reddit.com":  "teddit.net",
			},
		}},
		{ID: "unit_converter", Active: true},
		{ID: "hash_plugin", Active: true},
		{ID: "self_info", Active: false},
	}
}
