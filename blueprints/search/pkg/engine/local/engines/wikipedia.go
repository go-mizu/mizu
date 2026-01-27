package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Wikipedia implements Wikipedia search.
type Wikipedia struct {
	*BaseEngine
	languageMap map[string]string
}

// NewWikipedia creates a new Wikipedia engine.
func NewWikipedia() *Wikipedia {
	w := &Wikipedia{
		BaseEngine: NewBaseEngine("wikipedia", "w", []Category{CategoryGeneral}),
		languageMap: map[string]string{
			"en":    "en",
			"de":    "de",
			"fr":    "fr",
			"es":    "es",
			"it":    "it",
			"pt":    "pt",
			"ja":    "ja",
			"ko":    "ko",
			"zh":    "zh",
			"ru":    "ru",
			"ar":    "ar",
			"hi":    "hi",
			"nl":    "nl",
			"pl":    "pl",
			"sv":    "sv",
			"vi":    "vi",
			"uk":    "uk",
			"he":    "he",
			"id":    "id",
			"cs":    "cs",
			"fi":    "fi",
			"da":    "da",
			"no":    "no",
			"hu":    "hu",
			"ro":    "ro",
			"tr":    "tr",
			"th":    "th",
			"el":    "el",
			"fa":    "fa",
			"ca":    "ca",
		},
	}

	w.SetPaging(true).
		SetTimeout(5 * time.Second).
		SetAbout(EngineAbout{
			Website:         "https://www.wikipedia.org",
			WikidataID:      "Q52",
			OfficialAPIDocs: "https://www.mediawiki.org/wiki/API:Main_page",
			UseOfficialAPI:  true,
			Results:         "JSON",
		})

	// Set up language mappings
	for lang, wikiLang := range w.languageMap {
		w.traits.Languages[lang] = wikiLang
	}

	return w
}

func (w *Wikipedia) Request(ctx context.Context, query string, params *RequestParams) error {
	// Get language from locale
	lang := "en"
	if params.Language != "" {
		if len(params.Language) >= 2 {
			lang = params.Language[:2]
		}
	}
	if params.Locale != "" {
		if len(params.Locale) >= 2 {
			lang = strings.ToLower(params.Locale[:2])
		}
	}

	wikiLang := w.languageMap[lang]
	if wikiLang == "" {
		wikiLang = "en"
	}

	// Use Wikipedia API
	queryParams := url.Values{}
	queryParams.Set("action", "query")
	queryParams.Set("list", "search")
	queryParams.Set("srsearch", query)
	queryParams.Set("srwhat", "text")
	queryParams.Set("srlimit", "10")
	queryParams.Set("srprop", "snippet|titlesnippet|timestamp")
	queryParams.Set("format", "json")
	queryParams.Set("utf8", "1")

	if params.PageNo > 1 {
		queryParams.Set("sroffset", fmt.Sprintf("%d", (params.PageNo-1)*10))
	}

	params.URL = fmt.Sprintf("https://%s.wikipedia.org/w/api.php?%s", wikiLang, queryParams.Encode())
	params.Headers.Set("Accept", "application/json")

	return nil
}

func (w *Wikipedia) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	var apiResp wikipediaResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	// Get language from URL
	lang := "en"
	if u, err := url.Parse(params.URL); err == nil {
		parts := strings.Split(u.Host, ".")
		if len(parts) > 0 {
			lang = parts[0]
		}
	}

	for _, item := range apiResp.Query.Search {
		result := Result{
			URL:     fmt.Sprintf("https://%s.wikipedia.org/wiki/%s", lang, url.PathEscape(item.Title)),
			Title:   item.Title,
			Content: stripHTMLTags(item.Snippet),
		}
		result.ParsedURL, _ = url.Parse(result.URL)
		results.Add(result)
	}

	return results, nil
}

type wikipediaResponse struct {
	Query struct {
		Search []struct {
			NS        int    `json:"ns"`
			Title     string `json:"title"`
			PageID    int    `json:"pageid"`
			Snippet   string `json:"snippet"`
			Timestamp string `json:"timestamp"`
		} `json:"search"`
		SearchInfo struct {
			TotalHits int `json:"totalhits"`
		} `json:"searchinfo"`
	} `json:"query"`
}

func stripHTMLTags(s string) string {
	// Simple HTML tag stripper
	result := s
	for {
		start := strings.Index(result, "<")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	// Also handle HTML entities
	result = strings.ReplaceAll(result, "&quot;", "\"")
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&#39;", "'")
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	return strings.TrimSpace(result)
}

// Wikidata implements Wikidata search.
type Wikidata struct {
	*BaseEngine
}

// NewWikidata creates a new Wikidata engine.
func NewWikidata() *Wikidata {
	w := &Wikidata{
		BaseEngine: NewBaseEngine("wikidata", "wd", []Category{CategoryGeneral}),
	}

	w.SetPaging(true).
		SetTimeout(5 * time.Second).
		SetDisabled(true).
		SetAbout(EngineAbout{
			Website:         "https://www.wikidata.org",
			WikidataID:      "Q2013",
			OfficialAPIDocs: "https://www.wikidata.org/wiki/Wikidata:Data_access",
			UseOfficialAPI:  true,
			Results:         "JSON",
		})

	return w
}

func (w *Wikidata) Request(ctx context.Context, query string, params *RequestParams) error {
	queryParams := url.Values{}
	queryParams.Set("action", "wbsearchentities")
	queryParams.Set("search", query)
	queryParams.Set("language", "en")
	queryParams.Set("limit", "10")
	queryParams.Set("format", "json")

	if params.PageNo > 1 {
		queryParams.Set("continue", fmt.Sprintf("%d", (params.PageNo-1)*10))
	}

	params.URL = "https://www.wikidata.org/w/api.php?" + queryParams.Encode()
	params.Headers.Set("Accept", "application/json")

	return nil
}

func (w *Wikidata) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	var apiResp wikidataResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	for _, item := range apiResp.Search {
		result := Result{
			URL:     item.ConceptURI,
			Title:   item.Label,
			Content: item.Description,
		}
		if item.ConceptURI == "" {
			result.URL = fmt.Sprintf("https://www.wikidata.org/wiki/%s", item.ID)
		}
		result.ParsedURL, _ = url.Parse(result.URL)
		results.Add(result)
	}

	return results, nil
}

type wikidataResponse struct {
	Search []struct {
		ID          string `json:"id"`
		Label       string `json:"label"`
		Description string `json:"description"`
		ConceptURI  string `json:"concepturi"`
	} `json:"search"`
}
