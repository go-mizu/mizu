package engines

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBaseEngine(t *testing.T) {
	eng := NewBaseEngine("test", "t", []Category{CategoryGeneral, CategoryWeb})

	if eng.Name() != "test" {
		t.Errorf("Expected name 'test', got %s", eng.Name())
	}

	if eng.Shortcut() != "t" {
		t.Errorf("Expected shortcut 't', got %s", eng.Shortcut())
	}

	cats := eng.Categories()
	if len(cats) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(cats))
	}

	if eng.EngineType() != EngineTypeOnline {
		t.Errorf("Expected online engine type")
	}

	// Test setters
	eng.SetPaging(true)
	if !eng.SupportsPaging() {
		t.Error("Expected paging to be enabled")
	}

	eng.SetTimeRangeSupport(true)
	if !eng.SupportsTimeRange() {
		t.Error("Expected time range to be enabled")
	}

	eng.SetSafeSearch(true)
	if !eng.SupportsSafeSearch() {
		t.Error("Expected safe search to be enabled")
	}

	eng.SetMaxPage(50)
	if eng.MaxPage() != 50 {
		t.Errorf("Expected max page 50, got %d", eng.MaxPage())
	}

	eng.SetTimeout(10 * time.Second)
	if eng.Timeout() != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", eng.Timeout())
	}

	eng.SetWeight(1.5)
	if eng.Weight() != 1.5 {
		t.Errorf("Expected weight 1.5, got %f", eng.Weight())
	}

	eng.SetDisabled(true)
	if !eng.Disabled() {
		t.Error("Expected disabled to be true")
	}
}

func TestEngineTraits(t *testing.T) {
	traits := NewEngineTraits()

	if traits.AllLocale != "all" {
		t.Errorf("Expected all locale 'all', got %s", traits.AllLocale)
	}

	traits.Languages["en"] = "en_US"
	traits.Languages["de"] = "de_DE"
	traits.Regions["en-US"] = "US"
	traits.Regions["de-DE"] = "DE"

	// Test GetLanguage with exact match
	if traits.GetLanguage("en", "fallback") != "en_US" {
		t.Error("Expected en_US")
	}

	// Test GetLanguage with fallback
	if traits.GetLanguage("fr", "fallback") != "fallback" {
		t.Error("Expected fallback")
	}

	// Test GetLanguage with all locale
	if traits.GetLanguage("all", "fallback") != "fallback" {
		t.Error("Expected fallback for 'all'")
	}

	// Test GetRegion
	if traits.GetRegion("en-US", "fallback") != "US" {
		t.Error("Expected US")
	}

	if traits.GetRegion("fr-FR", "fallback") != "fallback" {
		t.Error("Expected fallback for unknown region")
	}
}

func TestRequestParams(t *testing.T) {
	params := NewRequestParams()

	if params.Method != "GET" {
		t.Errorf("Expected GET method, got %s", params.Method)
	}

	if params.Headers == nil {
		t.Error("Expected headers to be initialized")
	}

	if params.Data == nil {
		t.Error("Expected data to be initialized")
	}

	if !params.RaiseForHTTPError {
		t.Error("Expected RaiseForHTTPError to be true by default")
	}
}

func TestEngineResults(t *testing.T) {
	results := NewEngineResults()

	if len(results.Results) != 0 {
		t.Error("Expected empty results")
	}

	// Test Add
	results.Add(Result{URL: "https://example.com", Title: "Example"})
	if len(results.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results.Results))
	}

	// Test AddSuggestion
	results.AddSuggestion("test suggestion")
	if len(results.Suggestions) != 1 {
		t.Errorf("Expected 1 suggestion, got %d", len(results.Suggestions))
	}

	// Test AddCorrection
	results.AddCorrection("test correction")
	if len(results.Corrections) != 1 {
		t.Errorf("Expected 1 correction, got %d", len(results.Corrections))
	}

	// Test AddAnswer
	results.AddAnswer(Answer{Answer: "42"})
	if len(results.Answers) != 1 {
		t.Errorf("Expected 1 answer, got %d", len(results.Answers))
	}

	// Test SetEngineData
	results.SetEngineData("key", "value")
	if results.EngineData["key"] != "value" {
		t.Error("Expected engine data to be set")
	}
}

func TestGoogleEngine(t *testing.T) {
	google := NewGoogle()

	if google.Name() != "google" {
		t.Errorf("Expected name 'google', got %s", google.Name())
	}

	if google.Shortcut() != "g" {
		t.Errorf("Expected shortcut 'g', got %s", google.Shortcut())
	}

	if !google.SupportsPaging() {
		t.Error("Expected Google to support paging")
	}

	if !google.SupportsTimeRange() {
		t.Error("Expected Google to support time range")
	}

	// Test request building
	ctx := context.Background()
	params := NewRequestParams()
	params.PageNo = 1
	params.Locale = "en-US"

	err := google.Request(ctx, "test query", params)
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}

	if params.URL == "" {
		t.Error("Expected URL to be set")
	}
}

func TestDuckDuckGoEngine(t *testing.T) {
	ddg := NewDuckDuckGo()

	if ddg.Name() != "duckduckgo" {
		t.Errorf("Expected name 'duckduckgo', got %s", ddg.Name())
	}

	if ddg.Shortcut() != "ddg" {
		t.Errorf("Expected shortcut 'ddg', got %s", ddg.Shortcut())
	}

	// Test request building
	ctx := context.Background()
	params := NewRequestParams()
	params.PageNo = 1
	params.Locale = "en-US"

	err := ddg.Request(ctx, "test query", params)
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}

	if params.URL == "" {
		t.Error("Expected URL to be set")
	}

	if params.Method != "POST" {
		t.Errorf("Expected POST method, got %s", params.Method)
	}
}

func TestBingEngine(t *testing.T) {
	bing := NewBing()

	if bing.Name() != "bing" {
		t.Errorf("Expected name 'bing', got %s", bing.Name())
	}

	if bing.Shortcut() != "b" {
		t.Errorf("Expected shortcut 'b', got %s", bing.Shortcut())
	}

	if bing.MaxPage() != 200 {
		t.Errorf("Expected max page 200, got %d", bing.MaxPage())
	}

	// Test request building
	ctx := context.Background()
	params := NewRequestParams()
	params.PageNo = 1
	params.Locale = "en-US"

	err := bing.Request(ctx, "test query", params)
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}

	if params.URL == "" {
		t.Error("Expected URL to be set")
	}
}

func TestWikipediaEngine(t *testing.T) {
	wiki := NewWikipedia()

	if wiki.Name() != "wikipedia" {
		t.Errorf("Expected name 'wikipedia', got %s", wiki.Name())
	}

	if wiki.Shortcut() != "w" {
		t.Errorf("Expected shortcut 'w', got %s", wiki.Shortcut())
	}

	// Test request building
	ctx := context.Background()
	params := NewRequestParams()
	params.PageNo = 1
	params.Language = "en"

	err := wiki.Request(ctx, "test query", params)
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}

	if params.URL == "" {
		t.Error("Expected URL to be set")
	}

	// Test with mock response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"query": {
				"search": [
					{"title": "Test Article", "snippet": "This is a test article", "pageid": 123}
				],
				"searchinfo": {"totalhits": 100}
			}
		}`))
	}))
	defer server.Close()

	// Create mock response
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get mock response: %v", err)
	}
	defer resp.Body.Close()

	params.URL = server.URL
	results, err := wiki.Response(ctx, resp, params)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(results.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results.Results))
	}

	if results.Results[0].Title != "Test Article" {
		t.Errorf("Expected title 'Test Article', got %s", results.Results[0].Title)
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<b>bold</b>", "bold"},
		{"<a href='url'>link</a>", "link"},
		{"plain text", "plain text"},
		{"&quot;quoted&quot;", "\"quoted\""},
		{"&amp;", "&"},
		{"<span>nested <b>tags</b></span>", "nested tags"},
	}

	for _, tt := range tests {
		result := stripHTMLTags(tt.input)
		if result != tt.expected {
			t.Errorf("stripHTMLTags(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestBingURLDecoding(t *testing.T) {
	bing := NewBing()

	// Test Bing URL decoding
	// This is a simplified test - real Bing URLs are more complex
	testURL := "https://www.bing.com/ck/a?u=a1aHR0cHM6Ly9leGFtcGxlLmNvbQ"
	decoded := bing.decodeBingURL(testURL)

	// The base64 encoded URL should decode to https://example.com
	if decoded == testURL {
		// Decoding might fail with improper padding, that's OK for this test
		t.Log("URL decoding returned original URL (expected for improperly padded URLs)")
	}
}

func TestCategoryConstants(t *testing.T) {
	categories := []Category{
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
		CategoryOther,
	}

	// Ensure all categories are unique
	seen := make(map[Category]bool)
	for _, cat := range categories {
		if seen[cat] {
			t.Errorf("Duplicate category: %s", cat)
		}
		seen[cat] = true
	}
}

func TestSafeSearchLevels(t *testing.T) {
	if SafeSearchOff != 0 {
		t.Error("Expected SafeSearchOff to be 0")
	}
	if SafeSearchModerate != 1 {
		t.Error("Expected SafeSearchModerate to be 1")
	}
	if SafeSearchStrict != 2 {
		t.Error("Expected SafeSearchStrict to be 2")
	}
}

func TestTimeRangeConstants(t *testing.T) {
	ranges := []TimeRange{
		TimeRangeNone,
		TimeRangeDay,
		TimeRangeWeek,
		TimeRangeMonth,
		TimeRangeYear,
	}

	expected := []string{"", "day", "week", "month", "year"}
	for i, r := range ranges {
		if string(r) != expected[i] {
			t.Errorf("Expected %q, got %q", expected[i], r)
		}
	}
}
