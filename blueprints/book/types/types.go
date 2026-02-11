package types

import "time"

// Book represents a book in the library.
type Book struct {
	ID            int64     `json:"id"`
	OLKey         string    `json:"ol_key"`                // Open Library work key e.g. /works/OL12345W
	GoogleID      string    `json:"google_id"`             // Google Books ID
	Title         string    `json:"title"`
	Subtitle      string    `json:"subtitle,omitempty"`
	Description   string    `json:"description,omitempty"`
	Authors       []Author  `json:"authors"`               // Denormalized for display
	AuthorNames   string    `json:"author_names"`          // Comma-separated, for DB storage
	CoverURL      string    `json:"cover_url,omitempty"`
	CoverID       int       `json:"cover_id,omitempty"`    // Open Library cover ID
	ISBN10        string    `json:"isbn10,omitempty"`
	ISBN13        string    `json:"isbn13,omitempty"`
	Publisher     string    `json:"publisher,omitempty"`
	PublishDate   string    `json:"publish_date,omitempty"`
	PublishYear   int       `json:"publish_year,omitempty"`
	PageCount     int       `json:"page_count,omitempty"`
	Language      string    `json:"language,omitempty"`
	Format        string    `json:"format,omitempty"` // hardcover, paperback, ebook, audiobook
	Subjects      []string  `json:"subjects,omitempty"`
	SubjectsJSON  string    `json:"-"` // JSON stored in DB
	AverageRating float64   `json:"average_rating"`
	RatingsCount  int       `json:"ratings_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	// Computed fields (not stored)
	UserRating int    `json:"user_rating,omitempty"`
	UserShelf  string `json:"user_shelf,omitempty"`
}

// Author represents a book author.
type Author struct {
	ID         int64     `json:"id"`
	OLKey      string    `json:"ol_key"`
	Name       string    `json:"name"`
	Bio        string    `json:"bio,omitempty"`
	PhotoURL   string    `json:"photo_url,omitempty"`
	BirthDate  string    `json:"birth_date,omitempty"`
	DeathDate  string    `json:"death_date,omitempty"`
	WorksCount int       `json:"works_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// Shelf represents a bookshelf (e.g. "Read", "Want to Read").
type Shelf struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	IsExclusive bool      `json:"is_exclusive"`
	IsDefault   bool      `json:"is_default"`
	SortOrder   int       `json:"sort_order"`
	BookCount   int       `json:"book_count"` // Computed
	CreatedAt   time.Time `json:"created_at"`
}

// ShelfBook represents a book placed on a shelf.
type ShelfBook struct {
	ID        int64     `json:"id"`
	ShelfID   int64     `json:"shelf_id"`
	BookID    int64     `json:"book_id"`
	DateAdded time.Time `json:"date_added"`
	Position  int       `json:"position"`
	Book      *Book     `json:"book,omitempty"` // Joined
}

// Review represents a user's review of a book.
type Review struct {
	ID         int64      `json:"id"`
	BookID     int64      `json:"book_id"`
	Rating     int        `json:"rating"` // 1-5, 0 = no rating
	Text       string     `json:"text,omitempty"`
	IsSpoiler  bool       `json:"is_spoiler"`
	LikesCount int        `json:"likes_count"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Book       *Book      `json:"book,omitempty"` // Joined
}

// ReadingProgress tracks page-by-page progress through a book.
type ReadingProgress struct {
	ID        int64     `json:"id"`
	BookID    int64     `json:"book_id"`
	Page      int       `json:"page"`
	Percent   float64   `json:"percent"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ReadingChallenge represents a yearly reading goal.
type ReadingChallenge struct {
	ID        int64     `json:"id"`
	Year      int       `json:"year"`
	Goal      int       `json:"goal"`
	Progress  int       `json:"progress"` // Computed
	CreatedAt time.Time `json:"created_at"`
}

// BookList represents a curated list of books.
type BookList struct {
	ID          int64          `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	ItemCount   int            `json:"item_count"` // Computed
	CreatedAt   time.Time      `json:"created_at"`
	Items       []BookListItem `json:"items,omitempty"`
}

// BookListItem represents a book entry within a list.
type BookListItem struct {
	ID       int64 `json:"id"`
	ListID   int64 `json:"list_id"`
	BookID   int64 `json:"book_id"`
	Position int   `json:"position"`
	Votes    int   `json:"votes"`
	Book     *Book `json:"book,omitempty"` // Joined
}

// Quote represents a notable quote from a book.
type Quote struct {
	ID         int64     `json:"id"`
	BookID     int64     `json:"book_id"`
	AuthorName string    `json:"author_name"`
	Text       string    `json:"text"`
	LikesCount int       `json:"likes_count"`
	CreatedAt  time.Time `json:"created_at"`
	Book       *Book     `json:"book,omitempty"` // Joined
}

// FeedItem represents an activity in the user's reading feed.
type FeedItem struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"` // "rating", "review", "shelve", "progress", "challenge"
	BookID    int64     `json:"book_id,omitempty"`
	BookTitle string    `json:"book_title,omitempty"`
	Data      string    `json:"data"` // JSON payload
	CreatedAt time.Time `json:"created_at"`
}

// ReadingStats holds aggregated reading statistics.
type ReadingStats struct {
	TotalBooks     int            `json:"total_books"`
	TotalPages     int            `json:"total_pages"`
	AverageRating  float64        `json:"average_rating"`
	BooksPerMonth  map[string]int `json:"books_per_month"`
	PagesPerMonth  map[string]int `json:"pages_per_month"`
	GenreBreakdown map[string]int `json:"genre_breakdown"`
	RatingDist     map[int]int    `json:"rating_distribution"` // rating -> count
	ShortestBook   *Book          `json:"shortest_book,omitempty"`
	LongestBook    *Book          `json:"longest_book,omitempty"`
	HighestRated   *Book          `json:"highest_rated,omitempty"`
	MostPopular    *Book          `json:"most_popular,omitempty"`
}

// SearchResult wraps paginated book search results.
type SearchResult struct {
	Books      []Book `json:"books"`
	TotalCount int    `json:"total_count"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
}

// Genre is a browsable genre/subject category.
type Genre struct {
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	BookCount int    `json:"book_count"`
}
