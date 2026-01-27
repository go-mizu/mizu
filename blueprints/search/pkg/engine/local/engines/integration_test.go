//go:build integration

package engines

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// These tests run against REAL search engines.
// Run with: go test -tags=integration -v ./...
// Run a specific test: go test -tags=integration -v -run TestIntegration_Google ./...

const testQuery = "frieren"

func makeRealRequest(t *testing.T, eng OnlineEngine, query string) *EngineResults {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := NewRequestParams()
	params.PageNo = 1
	params.Locale = "en-US"
	params.Language = "en"
	params.SafeSearch = SafeSearchOff

	err := eng.Request(ctx, query, params)
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}

	t.Logf("Request URL: %s", params.URL)
	t.Logf("Request Method: %s", params.Method)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var req *http.Request
	if params.Method == "POST" {
		if params.Data != nil && len(params.Data) > 0 {
			req, err = http.NewRequestWithContext(ctx, params.Method, params.URL,
				io.NopCloser(strings.NewReader(params.Data.Encode())))
			if err == nil {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
		} else {
			req, err = http.NewRequestWithContext(ctx, params.Method, params.URL, nil)
		}
	} else {
		req, err = http.NewRequestWithContext(ctx, params.Method, params.URL, nil)
	}
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Set headers
	for key, values := range params.Headers {
		for _, v := range values {
			req.Header.Set(key, v)
		}
	}

	// Add cookies
	for _, cookie := range params.Cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Response Status: %d", resp.StatusCode)

	// Accept 200 OK and 202 Accepted (some engines like DuckDuckGo may return 202)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Non-OK status: %d, body: %s", resp.StatusCode, string(body[:minInt(500, len(body))]))
	}

	results, err := eng.Response(ctx, resp, params)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	return results
}

func logResults(t *testing.T, engineName string, results *EngineResults, maxShow int) {
	t.Logf("%s returned %d results", engineName, len(results.Results))
	for i, r := range results.Results {
		if i >= maxShow {
			t.Logf("  ... and %d more results", len(results.Results)-maxShow)
			break
		}
		t.Logf("  [%d] %s - %s", i+1, r.Title, r.URL)
	}
}

// ========== Wikipedia/Wikidata Tests ==========

func TestIntegration_Wikipedia_Frieren(t *testing.T) {
	eng := NewWikipedia()
	results := makeRealRequest(t, eng, testQuery)
	logResults(t, "Wikipedia", results, 5)

	if len(results.Results) == 0 {
		t.Error("Wikipedia returned no results for 'frieren'")
	}
}

func TestIntegration_Wikidata_Frieren(t *testing.T) {
	eng := NewWikidata()
	results := makeRealRequest(t, eng, testQuery)
	logResults(t, "Wikidata", results, 5)

	if len(results.Results) == 0 {
		t.Error("Wikidata returned no results for 'frieren'")
	}
}

// ========== GitHub Tests ==========

func TestIntegration_GitHub_Frieren(t *testing.T) {
	eng := NewGitHub()
	results := makeRealRequest(t, eng, testQuery)
	logResults(t, "GitHub", results, 5)

	if len(results.Results) == 0 {
		t.Error("GitHub returned no results for 'frieren'")
	}
}

// ========== ArXiv Tests ==========

func TestIntegration_ArXiv_MachineLearning(t *testing.T) {
	eng := NewArXiv()
	results := makeRealRequest(t, eng, "machine learning")

	t.Logf("ArXiv returned %d results", len(results.Results))
	for i, r := range results.Results {
		if i >= 5 {
			break
		}
		t.Logf("  [%d] %s", i+1, r.Title)
		if len(r.Authors) > 0 {
			t.Logf("      Authors: %v", r.Authors)
		}
	}

	if len(results.Results) == 0 {
		t.Error("ArXiv returned no results for 'machine learning'")
	}
}

// ========== Reddit Tests ==========

func TestIntegration_Reddit_Frieren(t *testing.T) {
	eng := NewReddit()
	results := makeRealRequest(t, eng, testQuery)
	logResults(t, "Reddit", results, 5)

	if len(results.Results) == 0 {
		t.Error("Reddit returned no results for 'frieren'")
	}
}

// ========== YouTube Tests ==========

func TestIntegration_YouTube_Frieren(t *testing.T) {
	eng := NewYouTube()
	results := makeRealRequest(t, eng, testQuery)

	t.Logf("YouTube returned %d results", len(results.Results))
	for i, r := range results.Results {
		if i >= 5 {
			break
		}
		t.Logf("  [%d] %s", i+1, r.Title)
		if r.Duration != "" {
			t.Logf("      Duration: %s", r.Duration)
		}
		t.Logf("      URL: %s", r.URL)
	}

	if len(results.Results) == 0 {
		t.Error("YouTube returned no results for 'frieren'")
	}
}

// ========== Brave Tests ==========

func TestIntegration_Brave_Frieren(t *testing.T) {
	eng := NewBrave()
	results := makeRealRequest(t, eng, testQuery)
	logResults(t, "Brave", results, 5)

	if len(results.Results) == 0 {
		t.Error("Brave returned no results for 'frieren'")
	}
}

// ========== Google Tests ==========
// Uses SearXNG's bypass techniques: GSA user agent, arc_id async request

func TestIntegration_Google_Frieren(t *testing.T) {
	eng := NewGoogle()
	results := makeRealRequest(t, eng, testQuery)
	logResults(t, "Google", results, 5)

	if len(results.Results) == 0 {
		t.Error("Google returned no results for 'frieren'")
	}
}

func TestIntegration_GoogleImages_Frieren(t *testing.T) {
	eng := NewGoogleImages()
	results := makeRealRequest(t, eng, testQuery)

	t.Logf("Google Images returned %d results", len(results.Results))
	for i, r := range results.Results {
		if i >= 5 {
			break
		}
		t.Logf("  [%d] %s - %s", i+1, r.Title, r.ImageURL)
	}

	if len(results.Results) == 0 {
		t.Error("Google Images returned no results for 'frieren'")
	}
}

// ========== Bing Tests ==========

func TestIntegration_Bing_Frieren(t *testing.T) {
	eng := NewBing()
	results := makeRealRequest(t, eng, testQuery)
	logResults(t, "Bing", results, 5)

	if len(results.Results) == 0 {
		t.Error("Bing returned no results for 'frieren'")
	}
}

func TestIntegration_BingImages_Frieren(t *testing.T) {
	eng := NewBingImages()
	results := makeRealRequest(t, eng, testQuery)

	t.Logf("Bing Images returned %d results", len(results.Results))
	for i, r := range results.Results {
		if i >= 5 {
			break
		}
		t.Logf("  [%d] %s - %s", i+1, r.Title, r.ImageURL)
	}

	if len(results.Results) == 0 {
		t.Error("Bing Images returned no results for 'frieren'")
	}
}

func TestIntegration_BingNews_Frieren(t *testing.T) {
	eng := NewBingNews()
	results := makeRealRequest(t, eng, testQuery)

	t.Logf("Bing News returned %d results", len(results.Results))
	for i, r := range results.Results {
		if i >= 5 {
			break
		}
		t.Logf("  [%d] %s", i+1, r.Title)
		if !r.PublishedAt.IsZero() {
			t.Logf("      Published: %s", r.PublishedAt.Format(time.RFC3339))
		}
	}

	if len(results.Results) == 0 {
		t.Error("Bing News returned no results for 'frieren'")
	}
}

// ========== DuckDuckGo Tests ==========
// Note: DuckDuckGo has very aggressive bot protection that blocks direct requests.
// SearXNG uses session management and proxy rotation that we don't have.

func TestIntegration_DuckDuckGo_Frieren(t *testing.T) {
	t.Skip("DuckDuckGo HTML web search has CAPTCHA - use Images/News/Videos APIs instead")
	eng := NewDuckDuckGo()
	results := makeRealRequest(t, eng, testQuery)
	logResults(t, "DuckDuckGo", results, 5)

	if len(results.Results) == 0 {
		t.Error("DuckDuckGo returned no results for 'frieren'")
	}
}

func TestIntegration_DuckDuckGoImages_Frieren(t *testing.T) {
	// JSON API works! No CAPTCHA
	eng := NewDuckDuckGoImages()
	results := makeRealRequest(t, eng, testQuery)

	t.Logf("DuckDuckGo Images returned %d results", len(results.Results))
	for i, r := range results.Results {
		if i >= 5 {
			break
		}
		t.Logf("  [%d] %s - %s", i+1, r.Title, r.ImageURL)
	}

	if len(results.Results) == 0 {
		t.Error("DuckDuckGo Images returned no results for 'frieren'")
	}
}

func TestIntegration_DuckDuckGoNews_Frieren(t *testing.T) {
	// JSON API works! No CAPTCHA
	eng := NewDuckDuckGoNews()
	results := makeRealRequest(t, eng, testQuery)

	t.Logf("DuckDuckGo News returned %d results", len(results.Results))
	for i, r := range results.Results {
		if i >= 5 {
			break
		}
		t.Logf("  [%d] %s", i+1, r.Title)
		if !r.PublishedAt.IsZero() {
			t.Logf("      Published: %s", r.PublishedAt.Format(time.RFC3339))
		}
	}

	if len(results.Results) == 0 {
		t.Error("DuckDuckGo News returned no results for 'frieren'")
	}
}

func TestIntegration_DuckDuckGoVideos_Frieren(t *testing.T) {
	// JSON API works! No CAPTCHA
	eng := NewDuckDuckGoVideos()
	results := makeRealRequest(t, eng, testQuery)

	t.Logf("DuckDuckGo Videos returned %d results", len(results.Results))
	for i, r := range results.Results {
		if i >= 5 {
			break
		}
		t.Logf("  [%d] %s", i+1, r.Title)
		if r.Duration != "" {
			t.Logf("      Duration: %s", r.Duration)
		}
	}

	if len(results.Results) == 0 {
		t.Error("DuckDuckGo Videos returned no results for 'frieren'")
	}
}

// ========== Summary Test for All Working Engines ==========

func TestIntegration_AllWorkingEngines_Frieren(t *testing.T) {
	type engineTest struct {
		name   string
		engine OnlineEngine
		query  string
	}

	// These engines are expected to return results reliably
	// Using SearXNG bypass techniques for Google
	// DuckDuckGo HTML web search has CAPTCHA, but JSON APIs (images/news/videos) work!
	tests := []engineTest{
		{"Google", NewGoogle(), testQuery},
		{"GoogleImages", NewGoogleImages(), testQuery},
		{"Bing", NewBing(), testQuery},
		{"BingImages", NewBingImages(), testQuery},
		{"BingNews", NewBingNews(), testQuery},
		// DuckDuckGo JSON APIs work (web search excluded - CAPTCHA)
		{"DuckDuckGoImages", NewDuckDuckGoImages(), testQuery},
		{"DuckDuckGoNews", NewDuckDuckGoNews(), testQuery},
		{"DuckDuckGoVideos", NewDuckDuckGoVideos(), testQuery},
		{"Wikipedia", NewWikipedia(), testQuery},
		{"Wikidata", NewWikidata(), testQuery},
		{"GitHub", NewGitHub(), testQuery},
		{"ArXiv", NewArXiv(), "machine learning"},
		{"Reddit", NewReddit(), testQuery},
		{"YouTube", NewYouTube(), testQuery},
		{"Brave", NewBrave(), testQuery},
	}

	var passed, failed int

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := makeRealRequest(t, tt.engine, tt.query)
			count := len(results.Results)

			if count > 0 {
				t.Logf("%s: PASS - %d results", tt.name, count)
				passed++
			} else {
				t.Errorf("%s: FAIL - no results", tt.name)
				failed++
			}
		})
	}

	t.Logf("\n=== SUMMARY ===")
	t.Logf("Passed: %d / %d", passed, len(tests))
	t.Logf("Failed: %d", failed)

	if failed > 0 {
		t.Errorf("%d engines failed to return results", failed)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
