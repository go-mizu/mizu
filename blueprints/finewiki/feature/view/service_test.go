package view

import (
	"context"
	"errors"
	"testing"
)

// mockStore implements Store for testing.
type mockStore struct {
	page      *Page
	err       error
	calledID  string
	calledWiki string
	calledTitle string
}

func (m *mockStore) GetByID(ctx context.Context, id string) (*Page, error) {
	m.calledID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.page, nil
}

func (m *mockStore) GetByTitle(ctx context.Context, wikiname, title string) (*Page, error) {
	m.calledWiki = wikiname
	m.calledTitle = title
	if m.err != nil {
		return nil, m.err
	}
	return m.page, nil
}

func TestService_ByID(t *testing.T) {
	tests := []struct {
		name    string
		store   *mockStore
		id      string
		want    *Page
		wantErr bool
	}{
		{
			name:    "empty id returns error",
			store:   &mockStore{},
			id:      "",
			wantErr: true,
		},
		{
			name:    "whitespace only id returns error",
			store:   &mockStore{},
			id:      "   ",
			wantErr: true,
		},
		{
			name: "valid id returns page",
			store: &mockStore{
				page: &Page{ID: "test/1", Title: "Test Page"},
			},
			id:   "test/1",
			want: &Page{ID: "test/1", Title: "Test Page"},
		},
		{
			name:    "store error propagates",
			store:   &mockStore{err: errors.New("not found")},
			id:      "test/1",
			wantErr: true,
		},
		{
			name: "id is trimmed",
			store: &mockStore{
				page: &Page{ID: "test/1", Title: "Test Page"},
			},
			id:   "  test/1  ",
			want: &Page{ID: "test/1", Title: "Test Page"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := New(tt.store)
			got, err := svc.ByID(context.Background(), tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("ByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got.ID != tt.want.ID {
				t.Errorf("ByID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_ByTitle(t *testing.T) {
	tests := []struct {
		name     string
		store    *mockStore
		wikiname string
		title    string
		want     *Page
		wantErr  bool
	}{
		{
			name:     "empty wikiname returns error",
			store:    &mockStore{},
			wikiname: "",
			title:    "Test",
			wantErr:  true,
		},
		{
			name:     "empty title returns error",
			store:    &mockStore{},
			wikiname: "enwiki",
			title:    "",
			wantErr:  true,
		},
		{
			name: "valid params returns page",
			store: &mockStore{
				page: &Page{ID: "enwiki/1", WikiName: "enwiki", Title: "Test Page"},
			},
			wikiname: "enwiki",
			title:    "Test Page",
			want:     &Page{ID: "enwiki/1", WikiName: "enwiki", Title: "Test Page"},
		},
		{
			name:     "store error propagates",
			store:    &mockStore{err: errors.New("not found")},
			wikiname: "enwiki",
			title:    "Test",
			wantErr:  true,
		},
		{
			name: "params are trimmed",
			store: &mockStore{
				page: &Page{ID: "enwiki/1", WikiName: "enwiki", Title: "Test Page"},
			},
			wikiname: "  enwiki  ",
			title:    "  Test Page  ",
			want:     &Page{ID: "enwiki/1", WikiName: "enwiki", Title: "Test Page"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := New(tt.store)
			got, err := svc.ByTitle(context.Background(), tt.wikiname, tt.title)

			if (err != nil) != tt.wantErr {
				t.Errorf("ByTitle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got.ID != tt.want.ID {
				t.Errorf("ByTitle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_NilStore(t *testing.T) {
	svc := New(nil)

	_, err := svc.ByID(context.Background(), "test/1")
	if err == nil {
		t.Error("ByID() expected error for nil store")
	}

	_, err = svc.ByTitle(context.Background(), "enwiki", "Test")
	if err == nil {
		t.Error("ByTitle() expected error for nil store")
	}
}

func TestPage_Fields(t *testing.T) {
	p := Page{
		ID:           "enwiki/12345",
		WikiName:     "enwiki",
		PageID:       12345,
		Title:        "Test Article",
		URL:          "https://en.wikipedia.org/wiki/Test_Article",
		DateModified: "2024-01-15T10:30:00Z",
		InLanguage:   "en",
		Text:         "This is the article content.",
		WikidataID:   "Q123456",
		BytesHTML:    5000,
		HasMath:      false,
	}

	if p.ID != "enwiki/12345" {
		t.Errorf("ID = %q, want %q", p.ID, "enwiki/12345")
	}
	if p.WikiName != "enwiki" {
		t.Errorf("WikiName = %q, want %q", p.WikiName, "enwiki")
	}
	if p.PageID != 12345 {
		t.Errorf("PageID = %d, want %d", p.PageID, 12345)
	}
	if p.Title != "Test Article" {
		t.Errorf("Title = %q, want %q", p.Title, "Test Article")
	}
}
