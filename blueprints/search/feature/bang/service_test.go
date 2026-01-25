package bang

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/types"
)

// mockBangStore implements store.BangStore for testing.
type mockBangStore struct {
	bangs map[string]*types.Bang
}

func newMockBangStore() *mockBangStore {
	return &mockBangStore{
		bangs: make(map[string]*types.Bang),
	}
}

func (m *mockBangStore) CreateBang(ctx context.Context, bang *types.Bang) error {
	m.bangs[bang.Trigger] = bang
	return nil
}

func (m *mockBangStore) GetBang(ctx context.Context, trigger string) (*types.Bang, error) {
	return m.bangs[trigger], nil
}

func (m *mockBangStore) ListBangs(ctx context.Context) ([]*types.Bang, error) {
	var bangs []*types.Bang
	for _, b := range m.bangs {
		bangs = append(bangs, b)
	}
	return bangs, nil
}

func (m *mockBangStore) ListUserBangs(ctx context.Context, userID string) ([]*types.Bang, error) {
	var bangs []*types.Bang
	for _, b := range m.bangs {
		if b.UserID == userID {
			bangs = append(bangs, b)
		}
	}
	return bangs, nil
}

func (m *mockBangStore) DeleteBang(ctx context.Context, id int64) error {
	for trigger, b := range m.bangs {
		if b.ID == id {
			delete(m.bangs, trigger)
			break
		}
	}
	return nil
}

func (m *mockBangStore) SeedBuiltinBangs(ctx context.Context) error {
	return nil
}

func TestService_Parse_ExternalBangs(t *testing.T) {
	store := newMockBangStore()
	svc := NewService(store)
	ctx := context.Background()

	tests := []struct {
		name        string
		query       string
		wantQuery   string
		wantBang    string
		wantRedir   bool
	}{
		{
			name:      "google bang prefix",
			query:     "!g test query",
			wantQuery: "test query",
			wantBang:  "g",
			wantRedir: true,
		},
		{
			name:      "google bang suffix",
			query:     "test query !g",
			wantQuery: "test query",
			wantBang:  "g",
			wantRedir: true,
		},
		{
			name:      "youtube bang",
			query:     "!yt funny cats",
			wantQuery: "funny cats",
			wantBang:  "yt",
			wantRedir: true,
		},
		{
			name:      "wikipedia bang",
			query:     "!w programming",
			wantQuery: "programming",
			wantBang:  "w",
			wantRedir: true,
		},
		{
			name:      "github bang",
			query:     "!gh mizu framework",
			wantQuery: "mizu framework",
			wantBang:  "gh",
			wantRedir: true,
		},
		{
			name:      "no bang",
			query:     "regular search query",
			wantQuery: "regular search query",
			wantBang:  "",
			wantRedir: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Parse(ctx, tt.query)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if result.Query != tt.wantQuery {
				t.Errorf("Parse() query = %q, want %q", result.Query, tt.wantQuery)
			}

			if tt.wantBang != "" {
				if result.Bang == nil {
					t.Errorf("Parse() bang = nil, want %q", tt.wantBang)
				} else if result.Bang.Trigger != tt.wantBang {
					t.Errorf("Parse() bang.Trigger = %q, want %q", result.Bang.Trigger, tt.wantBang)
				}
			}

			if tt.wantRedir && result.RedirectURL == "" {
				t.Error("Parse() expected redirect URL, got empty")
			}
			if !tt.wantRedir && result.RedirectURL != "" {
				t.Errorf("Parse() expected no redirect, got %q", result.RedirectURL)
			}
		})
	}
}

func TestService_Parse_InternalBangs(t *testing.T) {
	store := newMockBangStore()
	svc := NewService(store)
	ctx := context.Background()

	tests := []struct {
		name         string
		query        string
		wantQuery    string
		wantInternal bool
		wantCategory string
	}{
		{
			name:         "images bang short",
			query:        "!i cats",
			wantQuery:    "cats",
			wantInternal: true,
			wantCategory: "images",
		},
		{
			name:         "images bang long",
			query:        "!images dogs",
			wantQuery:    "dogs",
			wantInternal: true,
			wantCategory: "images",
		},
		{
			name:         "news bang",
			query:        "!n breaking news",
			wantQuery:    "breaking news",
			wantInternal: true,
			wantCategory: "news",
		},
		{
			name:         "videos bang",
			query:        "!v tutorial",
			wantQuery:    "tutorial",
			wantInternal: true,
			wantCategory: "videos",
		},
		{
			name:         "maps bang",
			query:        "!m tokyo",
			wantQuery:    "tokyo",
			wantInternal: true,
			wantCategory: "maps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Parse(ctx, tt.query)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if result.Query != tt.wantQuery {
				t.Errorf("Parse() query = %q, want %q", result.Query, tt.wantQuery)
			}

			if result.Internal != tt.wantInternal {
				t.Errorf("Parse() internal = %v, want %v", result.Internal, tt.wantInternal)
			}

			if result.Category != tt.wantCategory {
				t.Errorf("Parse() category = %q, want %q", result.Category, tt.wantCategory)
			}
		})
	}
}

func TestService_Parse_AIBangs(t *testing.T) {
	store := newMockBangStore()
	svc := NewService(store)
	ctx := context.Background()

	aiBangs := []string{"!ai", "!chat", "!assistant", "!llm", "!fast"}

	for _, bang := range aiBangs {
		t.Run(bang, func(t *testing.T) {
			result, err := svc.Parse(ctx, bang+" explain quantum computing")
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if !result.Internal {
				t.Error("Parse() expected internal = true")
			}

			if result.Category != "ai" {
				t.Errorf("Parse() category = %q, want %q", result.Category, "ai")
			}

			if result.Query != "explain quantum computing" {
				t.Errorf("Parse() query = %q, want %q", result.Query, "explain quantum computing")
			}
		})
	}
}

func TestService_Parse_TimeFilterBangs(t *testing.T) {
	store := newMockBangStore()
	svc := NewService(store)
	ctx := context.Background()

	tests := []struct {
		bang     string
		wantTime string
	}{
		{"!24", "day"},
		{"!day", "day"},
		{"!week", "week"},
		{"!month", "month"},
		{"!year", "year"},
	}

	for _, tt := range tests {
		t.Run(tt.bang, func(t *testing.T) {
			result, err := svc.Parse(ctx, tt.bang+" news today")
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if !result.Internal {
				t.Error("Parse() expected internal = true")
			}

			expectedCategory := "time:" + tt.wantTime
			if result.Category != expectedCategory {
				t.Errorf("Parse() category = %q, want %q", result.Category, expectedCategory)
			}

			if result.Query != "news today" {
				t.Errorf("Parse() query = %q, want %q", result.Query, "news today")
			}
		})
	}
}

func TestService_Parse_SummarizerBangs(t *testing.T) {
	store := newMockBangStore()
	svc := NewService(store)
	ctx := context.Background()

	sumBangs := []string{"!sum", "!summarize", "!fgpt", "!fastgpt"}

	for _, bang := range sumBangs {
		t.Run(bang, func(t *testing.T) {
			result, err := svc.Parse(ctx, bang+" https://example.com/article")
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if !result.Internal {
				t.Error("Parse() expected internal = true")
			}

			if result.Category != "summarize" {
				t.Errorf("Parse() category = %q, want %q", result.Category, "summarize")
			}
		})
	}
}

func TestService_Parse_FeelingLucky(t *testing.T) {
	store := newMockBangStore()
	svc := NewService(store)
	ctx := context.Background()

	tests := []struct {
		name  string
		query string
	}{
		{"prefix lucky", "! golang tutorial"},
		{"suffix lucky", "golang tutorial !"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Parse(ctx, tt.query)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if !result.Internal {
				t.Error("Parse() expected internal = true")
			}

			if result.Category != "lucky" {
				t.Errorf("Parse() category = %q, want %q", result.Category, "lucky")
			}
		})
	}
}

func TestService_Parse_CustomBang(t *testing.T) {
	store := newMockBangStore()
	store.bangs["custom"] = &types.Bang{
		ID:          1,
		Trigger:     "custom",
		Name:        "Custom Site",
		URLTemplate: "https://custom.com/search?q={query}",
		Category:    "custom",
	}

	svc := NewService(store)
	ctx := context.Background()

	result, err := svc.Parse(ctx, "!custom my search")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.Query != "my search" {
		t.Errorf("Parse() query = %q, want %q", result.Query, "my search")
	}

	if result.Bang == nil {
		t.Fatal("Parse() expected bang, got nil")
	}

	if result.Bang.Trigger != "custom" {
		t.Errorf("Parse() bang.Trigger = %q, want %q", result.Bang.Trigger, "custom")
	}

	if result.RedirectURL != "https://custom.com/search?q=my+search" {
		t.Errorf("Parse() redirect = %q, want %q", result.RedirectURL, "https://custom.com/search?q=my+search")
	}
}

func TestExtractBang(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{"prefix with exclaim", "!g search", "g"},
		{"prefix with trailing exclaim", "g! search", "g"},
		{"suffix with exclaim", "search !g", "g"},
		{"suffix with trailing exclaim", "search g!", "g"},
		{"no bang", "regular search", ""},
		{"exclaim in middle", "search ! more", ""},
		{"long trigger prefix", "!stackoverflow query", "stackoverflow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBang(tt.query)
			if got != tt.want {
				t.Errorf("extractBang(%q) = %q, want %q", tt.query, got, tt.want)
			}
		})
	}
}

func TestRemoveBang(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		trigger string
		want    string
	}{
		{"prefix", "!g search query", "g", "search query"},
		{"suffix", "search query !g", "g", "search query"},
		{"prefix trailing", "g! search query", "g", "search query"},
		{"suffix trailing", "search query g!", "g", "search query"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeBang(tt.query, tt.trigger)
			if got != tt.want {
				t.Errorf("removeBang(%q, %q) = %q, want %q", tt.query, tt.trigger, got, tt.want)
			}
		})
	}
}

func TestBuildRedirectURL(t *testing.T) {
	tests := []struct {
		name     string
		template string
		query    string
		want     string
	}{
		{
			name:     "simple query",
			template: "https://google.com/search?q={query}",
			query:    "test",
			want:     "https://google.com/search?q=test",
		},
		{
			name:     "query with spaces",
			template: "https://google.com/search?q={query}",
			query:    "hello world",
			want:     "https://google.com/search?q=hello+world",
		},
		{
			name:     "query with special chars",
			template: "https://google.com/search?q={query}",
			query:    "test&foo=bar",
			want:     "https://google.com/search?q=test%26foo%3Dbar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRedirectURL(tt.template, tt.query)
			if got != tt.want {
				t.Errorf("buildRedirectURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
