package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	st, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := st.Ensure(context.Background()); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	return st
}

func TestReviewFiltersAndCounts(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()

	book := types.Book{Title: "Test Book", AuthorNames: "Tester"}
	if err := st.Book().Create(ctx, &book); err != nil {
		t.Fatalf("create book: %v", err)
	}

	reviews := []types.Review{
		{BookID: book.ID, Rating: 5, Text: "Loved it", Source: "goodreads", ReviewerName: "A"},
		{BookID: book.ID, Rating: 4, Text: "", Source: "goodreads", ReviewerName: "B"},
		{BookID: book.ID, Rating: 3, Text: "Spoilers ahead", Source: "goodreads", ReviewerName: "C", IsSpoiler: true},
	}
	for _, r := range reviews {
		r := r
		if err := st.Review().Create(ctx, &r); err != nil {
			t.Fatalf("create review: %v", err)
		}
	}

	list, total, err := st.Review().GetByBookFiltered(ctx, book.ID, types.ReviewQuery{Rating: 5})
	if err != nil {
		t.Fatalf("filter rating: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected 1 rating-5 review, got total=%d len=%d", total, len(list))
	}

	list, total, err = st.Review().GetByBookFiltered(ctx, book.ID, types.ReviewQuery{HasText: ptrBool(true)})
	if err != nil {
		t.Fatalf("filter has_text: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("expected 2 text reviews, got total=%d len=%d", total, len(list))
	}

	list, total, err = st.Review().GetByBookFiltered(ctx, book.ID, types.ReviewQuery{IncludeSpoilers: false})
	if err != nil {
		t.Fatalf("filter spoilers: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("expected 2 non-spoiler reviews, got total=%d len=%d", total, len(list))
	}

	list, total, err = st.Review().GetByBookFiltered(ctx, book.ID, types.ReviewQuery{Query: "Loved"})
	if err != nil {
		t.Fatalf("filter query: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected 1 query match, got total=%d len=%d", total, len(list))
	}

	updated, err := st.Book().Get(ctx, book.ID)
	if err != nil || updated == nil {
		t.Fatalf("get book: %v", err)
	}
	if updated.ReviewsCount != 3 {
		t.Fatalf("expected reviews_count=3, got %d", updated.ReviewsCount)
	}
	if updated.RatingsCount != 3 {
		t.Fatalf("expected ratings_count=3, got %d", updated.RatingsCount)
	}
	if updated.RatingDist[0] != 1 || updated.RatingDist[1] != 1 || updated.RatingDist[2] != 1 {
		t.Fatalf("unexpected rating_dist: %+v", updated.RatingDist)
	}
}

func TestReviewLikesAndComments(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()

	book := types.Book{Title: "Test Book", AuthorNames: "Tester"}
	if err := st.Book().Create(ctx, &book); err != nil {
		t.Fatalf("create book: %v", err)
	}

	review := types.Review{BookID: book.ID, Rating: 5, Text: "Nice", Source: "user"}
	if err := st.Review().Create(ctx, &review); err != nil {
		t.Fatalf("create review: %v", err)
	}

	likes, err := st.Review().AddLike(ctx, review.ID)
	if err != nil {
		t.Fatalf("add like: %v", err)
	}
	if likes != 1 {
		t.Fatalf("expected likes=1, got %d", likes)
	}

	comment := types.ReviewComment{ReviewID: review.ID, AuthorName: "You", Text: "Agree"}
	if err := st.ReviewComment().Create(ctx, &comment); err != nil {
		t.Fatalf("create comment: %v", err)
	}

	updated, err := st.Review().Get(ctx, review.ID)
	if err != nil || updated == nil {
		t.Fatalf("get review: %v", err)
	}
	if updated.CommentsCount != 1 {
		t.Fatalf("expected comments_count=1, got %d", updated.CommentsCount)
	}

	if err := st.ReviewComment().Delete(ctx, comment.ID); err != nil {
		t.Fatalf("delete comment: %v", err)
	}

	updated, err = st.Review().Get(ctx, review.ID)
	if err != nil || updated == nil {
		t.Fatalf("get review: %v", err)
	}
	if updated.CommentsCount != 0 {
		t.Fatalf("expected comments_count=0, got %d", updated.CommentsCount)
	}
}

func ptrBool(v bool) *bool {
	return &v
}
