package store

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

// Store is the top-level storage interface for the book management system.
type Store interface {
	Ensure(ctx context.Context) error
	Close() error

	Book() BookStore
	Author() AuthorStore
	Shelf() ShelfStore
	Review() ReviewStore
	ReviewComment() ReviewCommentStore
	Progress() ProgressStore
	Challenge() ChallengeStore
	List() ListStore
	Quote() QuoteStore
	Stats() StatsStore
	Feed() FeedStore
}

// BookStore manages books.
type BookStore interface {
	Create(ctx context.Context, book *types.Book) error
	Get(ctx context.Context, id int64) (*types.Book, error)
	GetByISBN(ctx context.Context, isbn string) (*types.Book, error)
	GetByOLKey(ctx context.Context, olKey string) (*types.Book, error)
	GetByGoodreadsID(ctx context.Context, grID string) (*types.Book, error)
	Search(ctx context.Context, query string, page, limit int) (*types.SearchResult, error)
	Update(ctx context.Context, book *types.Book) error
	Delete(ctx context.Context, id int64) error
	GetByGenre(ctx context.Context, genre string, page, limit int) (*types.SearchResult, error)
	GetTrending(ctx context.Context, limit int) ([]types.Book, error)
	GetNewReleases(ctx context.Context, limit int) ([]types.Book, error)
	GetPopular(ctx context.Context, limit int) ([]types.Book, error)
	GetSimilar(ctx context.Context, bookID int64, limit int) ([]types.Book, error)
	ListGenres(ctx context.Context) ([]types.Genre, error)
	Count(ctx context.Context) (int, error)
}

// AuthorStore manages authors.
type AuthorStore interface {
	Create(ctx context.Context, author *types.Author) error
	Get(ctx context.Context, id int64) (*types.Author, error)
	GetByOLKey(ctx context.Context, olKey string) (*types.Author, error)
	GetByGoodreadsID(ctx context.Context, grID string) (*types.Author, error)
	Search(ctx context.Context, query string, limit int) ([]types.Author, error)
	GetBooks(ctx context.Context, authorID int64, page, limit int) (*types.SearchResult, error)
	Update(ctx context.Context, author *types.Author) error
}

// ShelfStore manages bookshelves and their contents.
type ShelfStore interface {
	Create(ctx context.Context, shelf *types.Shelf) error
	Get(ctx context.Context, id int64) (*types.Shelf, error)
	GetBySlug(ctx context.Context, slug string) (*types.Shelf, error)
	List(ctx context.Context) ([]types.Shelf, error)
	Update(ctx context.Context, shelf *types.Shelf) error
	Delete(ctx context.Context, id int64) error
	AddBook(ctx context.Context, shelfID, bookID int64) error
	RemoveBook(ctx context.Context, shelfID, bookID int64) error
	GetBooks(ctx context.Context, shelfID int64, sort string, page, limit int) ([]types.ShelfBook, int, error)
	GetBookShelves(ctx context.Context, bookID int64) ([]types.Shelf, error)
	SeedDefaults(ctx context.Context) error
}

// ReviewStore manages book reviews and ratings.
type ReviewStore interface {
	Create(ctx context.Context, review *types.Review) error
	Get(ctx context.Context, id int64) (*types.Review, error)
	GetByBook(ctx context.Context, bookID int64, page, limit int) ([]types.Review, int, error)
	GetByBookFiltered(ctx context.Context, bookID int64, q types.ReviewQuery) ([]types.Review, int, error)
	Update(ctx context.Context, review *types.Review) error
	Delete(ctx context.Context, id int64) error
	GetUserReview(ctx context.Context, bookID int64) (*types.Review, error)
	AddLike(ctx context.Context, reviewID int64) (int, error)
}

// ReviewCommentStore manages comments on reviews.
type ReviewCommentStore interface {
	Create(ctx context.Context, comment *types.ReviewComment) error
	GetByReview(ctx context.Context, reviewID int64, page, limit int) ([]types.ReviewComment, int, error)
	Delete(ctx context.Context, id int64) error
}

// ProgressStore tracks reading progress.
type ProgressStore interface {
	Create(ctx context.Context, p *types.ReadingProgress) error
	GetByBook(ctx context.Context, bookID int64) ([]types.ReadingProgress, error)
	GetLatest(ctx context.Context, bookID int64) (*types.ReadingProgress, error)
}

// ChallengeStore manages yearly reading challenges.
type ChallengeStore interface {
	Set(ctx context.Context, ch *types.ReadingChallenge) error
	Get(ctx context.Context, year int) (*types.ReadingChallenge, error)
	GetProgress(ctx context.Context, year int) (int, error)
}

// ListStore manages curated book lists.
type ListStore interface {
	Create(ctx context.Context, list *types.BookList) error
	Get(ctx context.Context, id int64) (*types.BookList, error)
	GetAll(ctx context.Context, page, limit int) ([]types.BookList, int, error)
	AddBook(ctx context.Context, listID, bookID int64, position int) error
	SetVotes(ctx context.Context, listID, bookID int64, votes int) error
	RemoveBook(ctx context.Context, listID, bookID int64) error
	Vote(ctx context.Context, listID, bookID int64) error
	Delete(ctx context.Context, id int64) error
}

// QuoteStore manages book quotes.
type QuoteStore interface {
	Create(ctx context.Context, q *types.Quote) error
	GetByBook(ctx context.Context, bookID int64, limit int) ([]types.Quote, error)
	GetAll(ctx context.Context, page, limit int) ([]types.Quote, int, error)
	Delete(ctx context.Context, id int64) error
}

// StatsStore computes reading statistics.
type StatsStore interface {
	GetStats(ctx context.Context, year int) (*types.ReadingStats, error)
	GetOverallStats(ctx context.Context) (*types.ReadingStats, error)
}

// FeedStore manages the activity feed.
type FeedStore interface {
	Add(ctx context.Context, item *types.FeedItem) error
	GetRecent(ctx context.Context, limit int) ([]types.FeedItem, error)
}
