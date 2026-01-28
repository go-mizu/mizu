// Package types contains shared data types for the search blueprint.
package types

import "time"

// Bang represents a search shortcut (e.g., !yt for YouTube).
type Bang struct {
	ID          int64     `json:"id"`
	Trigger     string    `json:"trigger"`      // e.g., "yt" (without !)
	Name        string    `json:"name"`         // e.g., "YouTube"
	URLTemplate string    `json:"url_template"` // e.g., "https://youtube.com/results?search_query={query}"
	Category    string    `json:"category"`     // e.g., "video", "search", "social"
	IsBuiltin   bool      `json:"is_builtin"`
	UserID      string    `json:"user_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// BangResult represents the result of parsing a bang from a query.
type BangResult struct {
	Bang        *Bang  `json:"bang,omitempty"`
	Query       string `json:"query"`           // Query without the bang
	OrigQuery   string `json:"orig_query"`      // Original query with bang
	RedirectURL string `json:"redirect,omitempty"`
	Internal    bool   `json:"internal"`        // true for internal bangs like !i, !n, !v
	Category    string `json:"category,omitempty"` // For internal bangs: images, news, videos, maps
}

// Built-in bang triggers by category
var (
	// External search bangs
	ExternalBangs = map[string]Bang{
		"g":    {Trigger: "g", Name: "Google", URLTemplate: "https://www.google.com/search?q={query}", Category: "search", IsBuiltin: true},
		"ddg":  {Trigger: "ddg", Name: "DuckDuckGo", URLTemplate: "https://duckduckgo.com/?q={query}", Category: "search", IsBuiltin: true},
		"b":    {Trigger: "b", Name: "Bing", URLTemplate: "https://www.bing.com/search?q={query}", Category: "search", IsBuiltin: true},
		"yt":   {Trigger: "yt", Name: "YouTube", URLTemplate: "https://www.youtube.com/results?search_query={query}", Category: "video", IsBuiltin: true},
		"w":    {Trigger: "w", Name: "Wikipedia", URLTemplate: "https://en.wikipedia.org/wiki/Special:Search?search={query}", Category: "reference", IsBuiltin: true},
		"r":    {Trigger: "r", Name: "Reddit", URLTemplate: "https://www.reddit.com/search/?q={query}", Category: "social", IsBuiltin: true},
		"gh":   {Trigger: "gh", Name: "GitHub", URLTemplate: "https://github.com/search?q={query}", Category: "code", IsBuiltin: true},
		"so":   {Trigger: "so", Name: "Stack Overflow", URLTemplate: "https://stackoverflow.com/search?q={query}", Category: "code", IsBuiltin: true},
		"tw":   {Trigger: "tw", Name: "Twitter/X", URLTemplate: "https://twitter.com/search?q={query}", Category: "social", IsBuiltin: true},
		"x":    {Trigger: "x", Name: "Twitter/X", URLTemplate: "https://twitter.com/search?q={query}", Category: "social", IsBuiltin: true},
		"amz":  {Trigger: "amz", Name: "Amazon", URLTemplate: "https://www.amazon.com/s?k={query}", Category: "shopping", IsBuiltin: true},
		"imdb": {Trigger: "imdb", Name: "IMDb", URLTemplate: "https://www.imdb.com/find?q={query}", Category: "media", IsBuiltin: true},
		"npm":  {Trigger: "npm", Name: "npm", URLTemplate: "https://www.npmjs.com/search?q={query}", Category: "code", IsBuiltin: true},
		"go":   {Trigger: "go", Name: "Go Packages", URLTemplate: "https://pkg.go.dev/search?q={query}", Category: "code", IsBuiltin: true},
		"mdn":  {Trigger: "mdn", Name: "MDN Web Docs", URLTemplate: "https://developer.mozilla.org/en-US/search?q={query}", Category: "code", IsBuiltin: true},
		"wa":   {Trigger: "wa", Name: "Wolfram Alpha", URLTemplate: "https://www.wolframalpha.com/input?i={query}", Category: "reference", IsBuiltin: true},
		"ud":   {Trigger: "ud", Name: "Urban Dictionary", URLTemplate: "https://www.urbandictionary.com/define.php?term={query}", Category: "reference", IsBuiltin: true},
		"maps": {Trigger: "maps", Name: "Google Maps", URLTemplate: "https://www.google.com/maps/search/{query}", Category: "maps", IsBuiltin: true},
		"osm":  {Trigger: "osm", Name: "OpenStreetMap", URLTemplate: "https://www.openstreetmap.org/search?query={query}", Category: "maps", IsBuiltin: true},
	}

	// Internal bangs (redirect to internal search categories)
	InternalBangs = map[string]string{
		"i":      "images",
		"images": "images",
		"n":      "news",
		"news":   "news",
		"v":      "videos",
		"videos": "videos",
		"m":      "maps",
		"map":    "maps",
	}

	// AI bangs (redirect to AI assistant)
	AIBangs = []string{"ai", "chat", "assistant", "llm", "asst", "as", "expert", "fast"}

	// Summarizer bangs
	SummarizerBangs = []string{"sum", "summarize", "fgpt", "fastgpt"}

	// Time filter bangs
	TimeFilterBangs = map[string]string{
		"24":    "day",
		"day":   "day",
		"week":  "week",
		"month": "month",
		"year":  "year",
	}
)
