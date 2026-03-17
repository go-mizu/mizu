// Package goodread scrapes public Goodreads data into a local DuckDB database.
package goodread

import "time"

// Book represents a Goodreads book page.
type Book struct {
	BookID             string    `json:"book_id"`
	Title              string    `json:"title"`
	TitleWithoutSeries string    `json:"title_without_series"`
	Description        string    `json:"description"`
	AuthorID           string    `json:"author_id"`
	AuthorName         string    `json:"author_name"`
	ISBN               string    `json:"isbn"`
	ISBN13             string    `json:"isbn13"`
	ASIN               string    `json:"asin"`
	AvgRating          float64   `json:"avg_rating"`
	RatingsCount       int64     `json:"ratings_count"`
	ReviewsCount       int64     `json:"reviews_count"`
	PublishedYear      int       `json:"published_year"`
	Publisher          string    `json:"publisher"`
	Language           string    `json:"language"`
	Pages              int       `json:"pages"`
	Format             string    `json:"format"`
	SeriesID           string    `json:"series_id"`
	SeriesName         string    `json:"series_name"`
	SeriesPosition     string    `json:"series_position"`
	Genres             []string  `json:"genres"`
	CoverURL           string    `json:"cover_url"`
	URL                string    `json:"url"`
	SimilarBookIDs     []string  `json:"similar_book_ids"`
	FetchedAt          time.Time `json:"fetched_at"`
}

// Author represents a Goodreads author page.
type Author struct {
	AuthorID       string    `json:"author_id"`
	Name           string    `json:"name"`
	Bio            string    `json:"bio"`
	PhotoURL       string    `json:"photo_url"`
	Website        string    `json:"website"`
	BornDate       string    `json:"born_date"`
	DiedDate       string    `json:"died_date"`
	Hometown       string    `json:"hometown"`
	Influences     []string  `json:"influences"`
	Genres         []string  `json:"genres"`
	AvgRating      float64   `json:"avg_rating"`
	RatingsCount   int64     `json:"ratings_count"`
	BooksCount     int       `json:"books_count"`
	FollowersCount int       `json:"followers_count"`
	URL            string    `json:"url"`
	FetchedAt      time.Time `json:"fetched_at"`
}

// Series represents a Goodreads book series.
type Series struct {
	SeriesID         string    `json:"series_id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	TotalBooks       int       `json:"total_books"`
	PrimaryWorkCount int       `json:"primary_work_count"`
	URL              string    `json:"url"`
	FetchedAt        time.Time `json:"fetched_at"`
}

// SeriesBook links a series to a book with position.
type SeriesBook struct {
	SeriesID string `json:"series_id"`
	BookID   string `json:"book_id"`
	Position int    `json:"position"`
}

// List represents a Goodreads listopia list.
type List struct {
	ListID        string    `json:"list_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	BooksCount    int       `json:"books_count"`
	VotersCount   int       `json:"voters_count"`
	Tags          []string  `json:"tags"`
	CreatedByUser string    `json:"created_by_user"`
	URL           string    `json:"url"`
	FetchedAt     time.Time `json:"fetched_at"`
}

// ListBook links a list to a book with rank and votes.
type ListBook struct {
	ListID string `json:"list_id"`
	BookID string `json:"book_id"`
	Rank   int    `json:"rank"`
	Votes  int    `json:"votes"`
}

// Review represents a Goodreads book review.
type Review struct {
	ReviewID   string    `json:"review_id"`
	BookID     string    `json:"book_id"`
	UserID     string    `json:"user_id"`
	UserName   string    `json:"user_name"`
	Rating     int       `json:"rating"`
	Text       string    `json:"text"`
	DateAdded  time.Time `json:"date_added"`
	LikesCount int       `json:"likes_count"`
	IsSpoiler  bool      `json:"is_spoiler"`
	URL        string    `json:"url"`
	FetchedAt  time.Time `json:"fetched_at"`
}

// Quote represents a Goodreads quote.
type Quote struct {
	QuoteID    string    `json:"quote_id"`
	Text       string    `json:"text"`
	AuthorID   string    `json:"author_id"`
	AuthorName string    `json:"author_name"`
	BookID     string    `json:"book_id"`
	BookTitle  string    `json:"book_title"`
	LikesCount int       `json:"likes_count"`
	Tags       []string  `json:"tags"`
	URL        string    `json:"url"`
	FetchedAt  time.Time `json:"fetched_at"`
}

// User represents a Goodreads user profile.
type User struct {
	UserID          string    `json:"user_id"`
	Name            string    `json:"name"`
	Username        string    `json:"username"`
	Location        string    `json:"location"`
	JoinedDate      time.Time `json:"joined_date"`
	FriendsCount    int       `json:"friends_count"`
	BooksReadCount  int       `json:"books_read_count"`
	RatingsCount    int       `json:"ratings_count"`
	ReviewsCount    int       `json:"reviews_count"`
	AvgRating       float64   `json:"avg_rating"`
	Bio             string    `json:"bio"`
	Website         string    `json:"website"`
	AvatarURL       string    `json:"avatar_url"`
	FavoriteBookIDs []string  `json:"favorite_book_ids"`
	URL             string    `json:"url"`
	FetchedAt       time.Time `json:"fetched_at"`
}

// Genre represents a Goodreads genre/shelf category.
type Genre struct {
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	BooksCount  int       `json:"books_count"`
	URL         string    `json:"url"`
	FetchedAt   time.Time `json:"fetched_at"`
}

// Shelf represents a user's reading shelf.
type Shelf struct {
	ShelfID    string    `json:"shelf_id"` // "{user_id}/{shelf_name}"
	UserID     string    `json:"user_id"`
	Name       string    `json:"name"`
	BooksCount int       `json:"books_count"`
	URL        string    `json:"url"`
	FetchedAt  time.Time `json:"fetched_at"`
}

// ShelfBook links a shelf to a book with reading metadata.
type ShelfBook struct {
	ShelfID   string    `json:"shelf_id"`
	UserID    string    `json:"user_id"`
	BookID    string    `json:"book_id"`
	DateAdded time.Time `json:"date_added"`
	Rating    int       `json:"rating"`
	DateRead  time.Time `json:"date_read"`
}

// QueueItem is a row from the state.duckdb queue table.
type QueueItem struct {
	ID         int64
	URL        string
	EntityType string
	Priority   int
	HtmlPath   string // path to cached .html.gz; set when status='fetched'
}

// SearchResult is a single result from a search page.
type SearchResult struct {
	URL        string
	EntityType string // book or author
	Title      string
}
