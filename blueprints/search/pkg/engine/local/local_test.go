package local

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/local/engines"
)

func TestNewMetaSearch(t *testing.T) {
	ms := New(nil)
	if ms == nil {
		t.Fatal("Expected non-nil MetaSearch")
	}

	if ms.config == nil {
		t.Error("Expected config to be set")
	}

	if ms.registry == nil {
		t.Error("Expected registry to be set")
	}
}

func TestConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultLanguage != "en" {
		t.Errorf("Expected default language 'en', got %s", cfg.DefaultLanguage)
	}

	if cfg.SafeSearch != engines.SafeSearchModerate {
		t.Errorf("Expected moderate safe search, got %d", cfg.SafeSearch)
	}

	if len(cfg.Engines) == 0 {
		t.Error("Expected default engines to be configured")
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	// Create a test engine
	eng := engines.NewBaseEngine("test", "t", []engines.Category{engines.CategoryGeneral})

	// Test registration
	if err := r.Register(eng); err != nil {
		t.Fatalf("Failed to register engine: %v", err)
	}

	// Test duplicate registration
	if err := r.Register(eng); err == nil {
		t.Error("Expected error for duplicate registration")
	}

	// Test Get
	got, ok := r.Get("test")
	if !ok {
		t.Error("Expected to find registered engine")
	}
	if got.Name() != "test" {
		t.Errorf("Expected engine name 'test', got %s", got.Name())
	}

	// Test GetByShortcut
	got, ok = r.GetByShortcut("t")
	if !ok {
		t.Error("Expected to find engine by shortcut")
	}
	if got.Name() != "test" {
		t.Errorf("Expected engine name 'test', got %s", got.Name())
	}

	// Test GetByCategory
	engs := r.GetByCategory(engines.CategoryGeneral)
	if len(engs) != 1 {
		t.Errorf("Expected 1 engine in category, got %d", len(engs))
	}

	// Test Unregister
	if err := r.Unregister("test"); err != nil {
		t.Fatalf("Failed to unregister engine: %v", err)
	}

	_, ok = r.Get("test")
	if ok {
		t.Error("Expected engine to be unregistered")
	}
}

func TestResultContainer(t *testing.T) {
	rc := NewResultContainer()

	// Add some results
	results := &engines.EngineResults{
		Results: []engines.Result{
			{URL: "https://example.com/1", Title: "Result 1", Content: "Content 1"},
			{URL: "https://example.com/2", Title: "Result 2", Content: "Content 2"},
		},
		Suggestions: []string{"suggestion 1"},
		Answers:     []engines.Answer{{Answer: "42"}},
	}

	rc.Extend("test-engine", results)

	// Check results were added
	ordered := rc.GetOrderedResults()
	if len(ordered) != 2 {
		t.Errorf("Expected 2 results, got %d", len(ordered))
	}

	// Check suggestions
	suggestions := rc.GetSuggestions()
	if len(suggestions) != 1 {
		t.Errorf("Expected 1 suggestion, got %d", len(suggestions))
	}

	// Check answers
	answers := rc.GetAnswers()
	if len(answers) != 1 {
		t.Errorf("Expected 1 answer, got %d", len(answers))
	}
}

func TestResultMerging(t *testing.T) {
	rc := NewResultContainer()

	// Add same URL from two engines
	results1 := &engines.EngineResults{
		Results: []engines.Result{
			{URL: "https://example.com/page", Title: "Short", Content: "Short content"},
		},
	}
	results2 := &engines.EngineResults{
		Results: []engines.Result{
			{URL: "https://example.com/page", Title: "Longer Title Here", Content: "Much longer content that should be preferred"},
		},
	}

	rc.Extend("engine1", results1)
	rc.Extend("engine2", results2)

	ordered := rc.GetOrderedResults()
	if len(ordered) != 1 {
		t.Errorf("Expected 1 merged result, got %d", len(ordered))
	}

	// Check that longer content was used
	if ordered[0].Content != "Much longer content that should be preferred" {
		t.Error("Expected longer content to be used")
	}

	// Check both engines are listed
	if len(ordered[0].Engines) != 2 {
		t.Errorf("Expected 2 engines, got %d", len(ordered[0].Engines))
	}
}

func TestCache(t *testing.T) {
	cache := NewMemoryCache(time.Second)
	defer cache.Close()

	// Test Set and Get
	err := cache.Set("key1", "value1", 0)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	val, ok := cache.Get("key1")
	if !ok {
		t.Error("Expected to find cached value")
	}
	if val != "value1" {
		t.Errorf("Expected 'value1', got %v", val)
	}

	// Test expiration
	err = cache.Set("key2", "value2", time.Millisecond*10)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	time.Sleep(time.Millisecond * 20)
	_, ok = cache.Get("key2")
	if ok {
		t.Error("Expected value to be expired")
	}

	// Test Delete
	cache.Set("key3", "value3", 0)
	cache.Delete("key3")
	_, ok = cache.Get("key3")
	if ok {
		t.Error("Expected value to be deleted")
	}

	// Test SecretHash
	hash1 := cache.SecretHash("test")
	hash2 := cache.SecretHash("test")
	if hash1 != hash2 {
		t.Error("Expected same hash for same input")
	}

	hash3 := cache.SecretHash("different")
	if hash1 == hash3 {
		t.Error("Expected different hash for different input")
	}
}

func TestQueryParser(t *testing.T) {
	r := NewRegistry()
	r.Register(engines.NewBaseEngine("google", "g", []engines.Category{engines.CategoryGeneral}))
	r.Register(engines.NewBaseEngine("duckduckgo", "ddg", []engines.Category{engines.CategoryGeneral}))

	parser := NewQueryParser(r)

	tests := []struct {
		query           string
		expectedQuery   string
		expectedBangs   int
		expectedEngines int
		expectedLang    string
	}{
		{
			query:           "hello world",
			expectedQuery:   "hello world",
			expectedBangs:   0,
			expectedEngines: 0,
		},
		{
			query:           "!g golang tutorial",
			expectedQuery:   "golang tutorial",
			expectedBangs:   1,
			expectedEngines: 1,
		},
		{
			query:           "search query :en-US",
			expectedQuery:   "search query",
			expectedLang:    "en",
		},
		{
			query:           "!images cats",
			expectedQuery:   "cats",
			expectedBangs:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			parsed := parser.Parse(tt.query)

			if parsed.Query != tt.expectedQuery {
				t.Errorf("Expected query %q, got %q", tt.expectedQuery, parsed.Query)
			}

			if len(parsed.Bangs) != tt.expectedBangs {
				t.Errorf("Expected %d bangs, got %d", tt.expectedBangs, len(parsed.Bangs))
			}

			if len(parsed.SpecificEngines) != tt.expectedEngines {
				t.Errorf("Expected %d engines, got %d", tt.expectedEngines, len(parsed.SpecificEngines))
			}

			if tt.expectedLang != "" && parsed.Language != tt.expectedLang {
				t.Errorf("Expected language %q, got %q", tt.expectedLang, parsed.Language)
			}
		})
	}
}

func TestPlugins(t *testing.T) {
	ps := NewPluginStorage()

	// Test tracker URL remover
	tracker := NewTrackerURLRemoverPlugin()
	ps.Register(tracker)

	if !ps.IsEnabled(tracker.ID()) {
		t.Error("Expected tracker remover to be enabled by default")
	}

	// Test hostname blocker
	blocker := NewHostnameBlockerPlugin([]string{"blocked.com"})
	ps.Register(blocker)

	if ps.IsEnabled(blocker.ID()) {
		t.Error("Expected hostname blocker to be disabled by default")
	}

	ps.Enable(blocker.ID())
	if !ps.IsEnabled(blocker.ID()) {
		t.Error("Expected hostname blocker to be enabled after Enable()")
	}

}

func TestAnswerers(t *testing.T) {
	as := NewAnswererStorage()

	// Register answerers
	as.Register(NewRandomAnswerer())
	as.Register(NewHashAnswerer())

	ctx := context.Background()

	// Test random number
	answers := as.Ask(ctx, "random number 1 100")
	if len(answers) == 0 {
		t.Error("Expected answer for random number query")
	}

	// Test UUID
	answers = as.Ask(ctx, "uuid")
	if len(answers) == 0 {
		t.Error("Expected answer for uuid query")
	}

	// Test hash
	answers = as.Ask(ctx, "md5 hello")
	if len(answers) == 0 {
		t.Error("Expected answer for md5 query")
	}
	if len(answers) > 0 && !contains(answers[0].Answer, "5d41402abc4b2a76b9719d911017c592") {
		t.Errorf("Expected MD5 hash of 'hello', got %s", answers[0].Answer)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestMockSearch(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<html>
			<body>
				<div class="g">
					<h2><a href="https://example.com/result1">Result 1</a></h2>
					<div class="VwiC3b">This is the first result content.</div>
				</div>
				<div class="g">
					<h2><a href="https://example.com/result2">Result 2</a></h2>
					<div class="VwiC3b">This is the second result content.</div>
				</div>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	// Test with the mock server
	// In a real test, you would inject the mock server URL into the engine
}

func TestEngineTraits(t *testing.T) {
	traits := engines.NewEngineTraits()

	traits.Languages["en"] = "lang_en"
	traits.Languages["de"] = "lang_de"
	traits.Regions["en-US"] = "US"
	traits.Regions["de-DE"] = "DE"

	// Test GetLanguage
	if traits.GetLanguage("en", "default") != "lang_en" {
		t.Error("Expected lang_en")
	}
	if traits.GetLanguage("unknown", "default") != "default" {
		t.Error("Expected default for unknown language")
	}
	if traits.GetLanguage("all", "default") != "default" {
		t.Error("Expected default for 'all' locale")
	}

	// Test GetRegion
	if traits.GetRegion("en-US", "default") != "US" {
		t.Error("Expected US")
	}
	if traits.GetRegion("unknown", "default") != "default" {
		t.Error("Expected default for unknown region")
	}
}

func TestSuspendedStatus(t *testing.T) {
	ss := NewSuspendedStatus()

	if ss.IsSuspended() {
		t.Error("Expected not suspended initially")
	}

	ss.Suspend(time.Second, "test reason")

	if !ss.IsSuspended() {
		t.Error("Expected suspended after Suspend()")
	}

	if ss.SuspendReason != "test reason" {
		t.Errorf("Expected 'test reason', got %s", ss.SuspendReason)
	}

	ss.Resume()

	if ss.IsSuspended() {
		t.Error("Expected not suspended after Resume()")
	}
}

func TestMetaSearchWithEngines(t *testing.T) {
	// Create MetaSearch with default config
	ms := New(nil)

	// Verify engines are registered
	reg := ms.Registry()

	// Check that Google is registered
	google, ok := reg.Get("google")
	if !ok {
		t.Error("Expected Google engine to be registered")
	}
	if google.Name() != "google" {
		t.Errorf("Expected 'google', got %s", google.Name())
	}

	// Check that Bing is registered
	bing, ok := reg.Get("bing")
	if !ok {
		t.Error("Expected Bing engine to be registered")
	}
	if bing.Name() != "bing" {
		t.Errorf("Expected 'bing', got %s", bing.Name())
	}

	// Check that DuckDuckGo is registered
	ddg, ok := reg.Get("duckduckgo")
	if !ok {
		t.Error("Expected DuckDuckGo engine to be registered")
	}
	if ddg.Name() != "duckduckgo" {
		t.Errorf("Expected 'duckduckgo', got %s", ddg.Name())
	}

	// Check that Wikipedia is registered
	wiki, ok := reg.Get("wikipedia")
	if !ok {
		t.Error("Expected Wikipedia engine to be registered")
	}
	if wiki.Name() != "wikipedia" {
		t.Errorf("Expected 'wikipedia', got %s", wiki.Name())
	}

	// Check that image engines are registered
	googleImages, ok := reg.Get("google images")
	if !ok {
		t.Error("Expected Google Images engine to be registered")
	}
	if googleImages.Name() != "google images" {
		t.Errorf("Expected 'google images', got %s", googleImages.Name())
	}

	// Check engines by category
	generalEngines := reg.GetByCategory(engines.CategoryGeneral)
	if len(generalEngines) == 0 {
		t.Error("Expected at least one general engine")
	}

	imageEngines := reg.GetByCategory(engines.CategoryImages)
	if len(imageEngines) == 0 {
		t.Error("Expected at least one image engine")
	}
}

func TestMetaSearchCategories(t *testing.T) {
	ms := New(nil)
	reg := ms.Registry()

	// Test that each engine type has proper categories
	tests := []struct {
		name       string
		categories []engines.Category
	}{
		{"google", []engines.Category{engines.CategoryGeneral, engines.CategoryWeb}},
		{"bing", []engines.Category{engines.CategoryGeneral, engines.CategoryWeb}},
		{"duckduckgo", []engines.Category{engines.CategoryGeneral, engines.CategoryWeb}},
		{"wikipedia", []engines.Category{engines.CategoryGeneral}},
		{"google images", []engines.Category{engines.CategoryImages}},
		{"bing images", []engines.Category{engines.CategoryImages}},
		{"bing news", []engines.Category{engines.CategoryNews}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng, ok := reg.Get(tt.name)
			if !ok {
				t.Errorf("Engine %s not registered", tt.name)
				return
			}

			cats := eng.Categories()
			if len(cats) != len(tt.categories) {
				t.Errorf("Expected %d categories, got %d", len(tt.categories), len(cats))
				return
			}

			for i, cat := range tt.categories {
				if cats[i] != cat {
					t.Errorf("Expected category %s, got %s", cat, cats[i])
				}
			}
		})
	}
}

func TestEngineShortcuts(t *testing.T) {
	ms := New(nil)
	reg := ms.Registry()

	shortcuts := map[string]string{
		"g":   "google",
		"b":   "bing",
		"ddg": "duckduckgo",
		"w":   "wikipedia",
		"gi":  "google images",
		"bi":  "bing images",
		"bn":  "bing news",
	}

	for shortcut, expectedName := range shortcuts {
		t.Run(shortcut, func(t *testing.T) {
			eng, ok := reg.GetByShortcut(shortcut)
			if !ok {
				t.Errorf("No engine found for shortcut %s", shortcut)
				return
			}
			if eng.Name() != expectedName {
				t.Errorf("Expected %s for shortcut %s, got %s", expectedName, shortcut, eng.Name())
			}
		})
	}
}
