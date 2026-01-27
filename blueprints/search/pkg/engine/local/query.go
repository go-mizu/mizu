package local

import (
	"regexp"
	"strings"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/local/engines"
)

// ParsedQuery contains the parsed query information.
type ParsedQuery struct {
	Query           string
	OriginalQuery   string
	Bangs           []string
	EngineRefs      []EngineRef
	Categories      []engines.Category
	Language        string
	Region          string
	TimeRange       engines.TimeRange
	SafeSearch      engines.SafeSearchLevel
	ExternalBang    string
	SpecificEngines []string
}

// QueryParser parses search queries.
type QueryParser struct {
	registry *Registry

	bangPattern     *regexp.Regexp
	langPattern     *regexp.Regexp
	categoryPattern *regexp.Regexp
	timePattern     *regexp.Regexp
}

// NewQueryParser creates a new QueryParser.
func NewQueryParser(registry *Registry) *QueryParser {
	return &QueryParser{
		registry:        registry,
		bangPattern:     regexp.MustCompile(`!(\w+)`),
		langPattern:     regexp.MustCompile(`:([a-z]{2}(?:-[A-Z]{2})?)`),
		categoryPattern: regexp.MustCompile(`!(images|videos|news|music|files|it|science|maps)`),
		timePattern:     regexp.MustCompile(`!(day|week|month|year)`),
	}
}

// Parse parses a search query.
func (qp *QueryParser) Parse(query string) *ParsedQuery {
	parsed := &ParsedQuery{
		OriginalQuery: query,
		Query:         query,
		Bangs:         make([]string, 0),
		EngineRefs:    make([]EngineRef, 0),
		Categories:    make([]engines.Category, 0),
	}

	// Extract bangs (e.g., !g, !ddg)
	bangMatches := qp.bangPattern.FindAllStringSubmatch(query, -1)
	for _, match := range bangMatches {
		if len(match) >= 2 {
			bang := strings.ToLower(match[1])
			parsed.Bangs = append(parsed.Bangs, bang)

			// Check if it's an engine shortcut
			if eng, ok := qp.registry.GetByShortcut(bang); ok {
				cats := eng.Categories()
				cat := engines.CategoryGeneral
				if len(cats) > 0 {
					cat = cats[0]
				}
				parsed.SpecificEngines = append(parsed.SpecificEngines, eng.Name())
				parsed.EngineRefs = append(parsed.EngineRefs, EngineRef{
					Name:     eng.Name(),
					Category: cat,
				})
			}

			// Check if it's a category shortcut
			switch bang {
			case "images", "img", "i":
				parsed.Categories = appendIfNotExists(parsed.Categories, engines.CategoryImages)
			case "videos", "video", "v":
				parsed.Categories = appendIfNotExists(parsed.Categories, engines.CategoryVideos)
			case "news", "n":
				parsed.Categories = appendIfNotExists(parsed.Categories, engines.CategoryNews)
			case "music", "m":
				parsed.Categories = appendIfNotExists(parsed.Categories, engines.CategoryMusic)
			case "files", "f":
				parsed.Categories = appendIfNotExists(parsed.Categories, engines.CategoryFiles)
			case "it", "code":
				parsed.Categories = appendIfNotExists(parsed.Categories, engines.CategoryIT)
			case "science", "sci":
				parsed.Categories = appendIfNotExists(parsed.Categories, engines.CategoryScience)
			case "maps", "map":
				parsed.Categories = appendIfNotExists(parsed.Categories, engines.CategoryMaps)
			case "social":
				parsed.Categories = appendIfNotExists(parsed.Categories, engines.CategorySocial)
			}

			// Check if it's a time range
			switch bang {
			case "day", "d":
				parsed.TimeRange = engines.TimeRangeDay
			case "week", "w":
				parsed.TimeRange = engines.TimeRangeWeek
			case "month", "mo":
				parsed.TimeRange = engines.TimeRangeMonth
			case "year", "y":
				parsed.TimeRange = engines.TimeRangeYear
			}
		}
	}

	// Extract language/region (e.g., :en-US, :de)
	langMatches := qp.langPattern.FindAllStringSubmatch(query, -1)
	for _, match := range langMatches {
		if len(match) >= 2 {
			locale := match[1]
			if strings.Contains(locale, "-") {
				parts := strings.Split(locale, "-")
				parsed.Language = strings.ToLower(parts[0])
				parsed.Region = strings.ToUpper(parts[1])
			} else {
				parsed.Language = strings.ToLower(locale)
			}
		}
	}

	// Clean query - remove bangs and language specifiers
	cleanQuery := query
	cleanQuery = qp.bangPattern.ReplaceAllString(cleanQuery, "")
	cleanQuery = qp.langPattern.ReplaceAllString(cleanQuery, "")
	cleanQuery = strings.TrimSpace(cleanQuery)

	// Collapse multiple spaces
	cleanQuery = regexp.MustCompile(`\s+`).ReplaceAllString(cleanQuery, " ")

	parsed.Query = cleanQuery

	return parsed
}

func appendIfNotExists(slice []engines.Category, item engines.Category) []engines.Category {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

// ExternalBangs contains known external bangs (search on other sites).
var ExternalBangs = map[string]string{
	"yt":       "https://www.youtube.com/results?search_query=%s",
	"youtube":  "https://www.youtube.com/results?search_query=%s",
	"tw":       "https://twitter.com/search?q=%s",
	"twitter":  "https://twitter.com/search?q=%s",
	"reddit":   "https://www.reddit.com/search/?q=%s",
	"re":       "https://www.reddit.com/search/?q=%s",
	"amazon":   "https://www.amazon.com/s?k=%s",
	"amz":      "https://www.amazon.com/s?k=%s",
	"ebay":     "https://www.ebay.com/sch/i.html?_nkw=%s",
	"maps":     "https://www.openstreetmap.org/search?query=%s",
	"osm":      "https://www.openstreetmap.org/search?query=%s",
	"gmaps":    "https://www.google.com/maps/search/%s",
	"imdb":     "https://www.imdb.com/find?q=%s",
	"wiki":     "https://en.wikipedia.org/wiki/Special:Search?search=%s",
	"wolfram":  "https://www.wolframalpha.com/input/?i=%s",
	"wa":       "https://www.wolframalpha.com/input/?i=%s",
	"wayback":  "https://web.archive.org/web/*/%s",
	"archive":  "https://web.archive.org/web/*/%s",
}

// GetExternalBangURL returns the URL for an external bang, or empty string if not found.
func GetExternalBangURL(bang, query string) string {
	if template, ok := ExternalBangs[strings.ToLower(bang)]; ok {
		return strings.Replace(template, "%s", query, 1)
	}
	return ""
}
