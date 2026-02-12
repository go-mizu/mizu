package sqlite

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

func TestBookStorePersistsExpandedGoodreadsFields(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	book := &types.Book{
		Title:           "Expanded Metadata Book",
		AuthorNames:     "Author Name",
		GoodreadsID:     "12345",
		GoodreadsURL:    "https://www.goodreads.com/book/show/12345",
		OriginalTitle:   "Expanded Metadata Book (Original)",
		EditionLanguage: "English",
		Characters:      []string{"Character A", "Character B"},
		Settings:        []string{"City", "Country"},
		LiteraryAwards:  []string{"Award One", "Award Two"},
		EditionsCount:   42,
	}
	if err := st.Book().Create(ctx, book); err != nil {
		t.Fatalf("create book: %v", err)
	}

	got, err := st.Book().Get(ctx, book.ID)
	if err != nil {
		t.Fatalf("get book: %v", err)
	}
	if got == nil {
		t.Fatal("expected book row")
	}
	if got.GoodreadsURL != book.GoodreadsURL {
		t.Fatalf("expected goodreads URL persisted, got %q", got.GoodreadsURL)
	}
	if got.OriginalTitle != book.OriginalTitle {
		t.Fatalf("expected original title persisted, got %q", got.OriginalTitle)
	}
	if got.EditionLanguage != book.EditionLanguage {
		t.Fatalf("expected edition language persisted, got %q", got.EditionLanguage)
	}
	if len(got.Characters) != 2 || got.Characters[0] != "Character A" {
		t.Fatalf("expected characters persisted, got %#v", got.Characters)
	}
	if len(got.Settings) != 2 || got.Settings[0] != "City" {
		t.Fatalf("expected settings persisted, got %#v", got.Settings)
	}
	if len(got.LiteraryAwards) != 2 || got.LiteraryAwards[0] != "Award One" {
		t.Fatalf("expected awards persisted, got %#v", got.LiteraryAwards)
	}
	if got.EditionsCount != 42 {
		t.Fatalf("expected editions_count 42, got %d", got.EditionsCount)
	}
}

func TestListStoreSetVotesAndBookHydration(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	book := &types.Book{
		Title:         "List Book",
		AuthorNames:   "List Author",
		OriginalTitle: "List Book Original",
		Characters:    []string{"Lead"},
	}
	if err := st.Book().Create(ctx, book); err != nil {
		t.Fatalf("create book: %v", err)
	}

	list := &types.BookList{Title: "Imported List"}
	if err := st.List().Create(ctx, list); err != nil {
		t.Fatalf("create list: %v", err)
	}
	if err := st.List().AddBook(ctx, list.ID, book.ID, 1); err != nil {
		t.Fatalf("add book to list: %v", err)
	}
	if err := st.List().SetVotes(ctx, list.ID, book.ID, 999); err != nil {
		t.Fatalf("set votes: %v", err)
	}

	got, err := st.List().Get(ctx, list.ID)
	if err != nil {
		t.Fatalf("get list: %v", err)
	}
	if got == nil || len(got.Items) != 1 {
		t.Fatalf("expected one list item, got %#v", got)
	}
	item := got.Items[0]
	if item.Votes != 999 {
		t.Fatalf("expected seeded votes 999, got %d", item.Votes)
	}
	if item.Book == nil || item.Book.OriginalTitle != "List Book Original" {
		t.Fatalf("expected hydrated book metadata, got %#v", item.Book)
	}
	if len(item.Book.Characters) != 1 || item.Book.Characters[0] != "Lead" {
		t.Fatalf("expected hydrated characters, got %#v", item.Book.Characters)
	}
}

func TestBookStoreGetSimilarFallsBackToAuthorWhenNoSubjects(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	base := &types.Book{
		Title:       "Blood of Elves (The Witcher)",
		AuthorNames: "Andrzej Sapkowski",
		Subjects:    nil, // persisted as [] and should trigger fallback path
	}
	if err := st.Book().Create(ctx, base); err != nil {
		t.Fatalf("create base book: %v", err)
	}

	candidate := &types.Book{
		Title:       "Time of Contempt",
		AuthorNames: "Andrzej Sapkowski",
	}
	if err := st.Book().Create(ctx, candidate); err != nil {
		t.Fatalf("create candidate: %v", err)
	}

	similar, err := st.Book().GetSimilar(ctx, base.ID, 10)
	if err != nil {
		t.Fatalf("get similar: %v", err)
	}
	if len(similar) == 0 {
		t.Fatal("expected author fallback to return at least one similar book")
	}
	if similar[0].ID != candidate.ID {
		t.Fatalf("expected candidate id %d, got %d", candidate.ID, similar[0].ID)
	}
}
