package api

import (
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/pkg/goodreads"
)

func TestParseSourceDate(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"May 10, 2021", "2021-05-10"},
		{"May 10 2021", "2021-05-10"},
		{"Jan 2, 2000", "2000-01-02"},
		{"Jan 2 2000", "2000-01-02"},
		{"", ""},
		{"invalid", ""},
	}

	for _, tc := range tests {
		got := parseSourceDate(tc.in)
		if tc.want == "" {
			if got != nil {
				t.Fatalf("expected nil for %q, got %v", tc.in, got)
			}
			continue
		}
		if got == nil {
			t.Fatalf("expected parsed date for %q", tc.in)
		}
		if got.Format("2006-01-02") != tc.want {
			t.Fatalf("expected %s for %q, got %s", tc.want, tc.in, got.Format("2006-01-02"))
		}
	}
}

func TestGoodreadsToBookMapsExpandedFields(t *testing.T) {
	src := &goodreads.GoodreadsBook{
		GoodreadsID:      "123",
		URL:              "https://www.goodreads.com/book/show/123",
		Title:            "Book",
		OriginalTitle:    "Original Book",
		AuthorName:       "Author",
		Description:      "desc",
		EditionLanguage:  "English",
		Characters:       []string{"A", "B"},
		Settings:         []string{"X"},
		LiteraryAwards:   []string{"Award"},
		EditionCount:     50,
		AverageRating:    4.2,
		RatingsCount:     1234,
		ReviewsCount:     222,
		CurrentlyReading: 11,
		WantToRead:       44,
	}

	book := goodreadsToBook(src)
	if book.GoodreadsURL != src.URL {
		t.Fatalf("expected goodreads URL mapped, got %q", book.GoodreadsURL)
	}
	if book.OriginalTitle != src.OriginalTitle {
		t.Fatalf("expected original title mapped, got %q", book.OriginalTitle)
	}
	if book.EditionLanguage != src.EditionLanguage {
		t.Fatalf("expected edition language mapped, got %q", book.EditionLanguage)
	}
	if len(book.Characters) != 2 || len(book.Settings) != 1 || len(book.LiteraryAwards) != 1 {
		t.Fatalf("expected expanded slices mapped, got chars=%#v settings=%#v awards=%#v",
			book.Characters, book.Settings, book.LiteraryAwards)
	}
	if book.EditionsCount != 50 {
		t.Fatalf("expected editions count mapped, got %d", book.EditionsCount)
	}
}

func TestMergeSourceDataPreservesExistingAndAddsExpandedFields(t *testing.T) {
	now := time.Now()
	dst := goodreadsToBook(&goodreads.GoodreadsBook{
		GoodreadsID:   "1",
		URL:           "https://www.goodreads.com/book/show/1",
		Title:         "Book",
		Description:   "existing-desc",
		CoverURL:      "existing-cover",
		AverageRating: 3.0,
		RatingsCount:  10,
	})
	dst.CreatedAt = now

	src := &goodreads.GoodreadsBook{
		GoodreadsID:     "1",
		URL:             "https://www.goodreads.com/book/show/1-new",
		OriginalTitle:   "Original",
		EditionLanguage: "French",
		Characters:      []string{"Hero"},
		Settings:        []string{"Mars"},
		LiteraryAwards:  []string{"Nebula"},
		EditionCount:    99,
		AverageRating:   4.7,
		RatingsCount:    2000,
	}

	mergeSourceData(&dst, src)
	if dst.Description != "existing-desc" {
		t.Fatalf("expected existing description preserved, got %q", dst.Description)
	}
	if dst.CoverURL != "existing-cover" {
		t.Fatalf("expected existing cover preserved, got %q", dst.CoverURL)
	}
	if dst.OriginalTitle != "Original" || dst.EditionLanguage != "French" {
		t.Fatalf("expected expanded fields merged, got original=%q edition=%q", dst.OriginalTitle, dst.EditionLanguage)
	}
	if dst.EditionsCount != 99 {
		t.Fatalf("expected editions count merged, got %d", dst.EditionsCount)
	}
	if dst.AverageRating != 4.7 || dst.RatingsCount != 2000 {
		t.Fatalf("expected rating fields updated, got avg=%f count=%d", dst.AverageRating, dst.RatingsCount)
	}
}
