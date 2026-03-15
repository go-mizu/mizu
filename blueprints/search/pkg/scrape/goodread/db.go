package goodread

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// DB wraps a DuckDB database for storing Goodreads data.
type DB struct {
	db   *sql.DB
	path string
}

// OpenDB opens or creates a DuckDB database at the given path.
func OpenDB(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb %s: %w", path, err)
	}

	d := &DB{db: db, path: path}
	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return d, nil
}

func (d *DB) initSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS books (
			book_id              VARCHAR PRIMARY KEY,
			title                VARCHAR,
			title_without_series VARCHAR,
			description          VARCHAR,
			author_id            VARCHAR,
			author_name          VARCHAR,
			isbn                 VARCHAR,
			isbn13               VARCHAR,
			asin                 VARCHAR,
			avg_rating           DOUBLE,
			ratings_count        BIGINT,
			reviews_count        BIGINT,
			published_year       INTEGER,
			publisher            VARCHAR,
			language             VARCHAR,
			pages                INTEGER,
			format               VARCHAR,
			series_id            VARCHAR,
			series_name          VARCHAR,
			series_position      VARCHAR,
			genres               VARCHAR,
			cover_url            VARCHAR,
			url                  VARCHAR,
			similar_book_ids     VARCHAR,
			fetched_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS authors (
			author_id       VARCHAR PRIMARY KEY,
			name            VARCHAR,
			bio             VARCHAR,
			photo_url       VARCHAR,
			website         VARCHAR,
			born_date       VARCHAR,
			died_date       VARCHAR,
			hometown        VARCHAR,
			influences      VARCHAR,
			genres          VARCHAR,
			avg_rating      DOUBLE,
			ratings_count   BIGINT,
			books_count     INTEGER,
			followers_count INTEGER,
			url             VARCHAR,
			fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS series (
			series_id          VARCHAR PRIMARY KEY,
			name               VARCHAR,
			description        VARCHAR,
			total_books        INTEGER,
			primary_work_count INTEGER,
			url                VARCHAR,
			fetched_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS series_books (
			series_id VARCHAR NOT NULL,
			book_id   VARCHAR NOT NULL,
			position  INTEGER,
			PRIMARY KEY (series_id, book_id)
		)`,
		`CREATE TABLE IF NOT EXISTS lists (
			list_id         VARCHAR PRIMARY KEY,
			name            VARCHAR,
			description     VARCHAR,
			books_count     INTEGER,
			voters_count    INTEGER,
			tags            VARCHAR,
			created_by_user VARCHAR,
			url             VARCHAR,
			fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS list_books (
			list_id VARCHAR NOT NULL,
			book_id VARCHAR NOT NULL,
			rank    INTEGER,
			votes   INTEGER,
			PRIMARY KEY (list_id, book_id)
		)`,
		`CREATE TABLE IF NOT EXISTS reviews (
			review_id   VARCHAR PRIMARY KEY,
			book_id     VARCHAR,
			user_id     VARCHAR,
			user_name   VARCHAR,
			rating      INTEGER,
			text        VARCHAR,
			date_added  TIMESTAMP,
			likes_count INTEGER,
			is_spoiler  BOOLEAN DEFAULT FALSE,
			url         VARCHAR,
			fetched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS quotes (
			quote_id    VARCHAR PRIMARY KEY,
			text        VARCHAR,
			author_id   VARCHAR,
			author_name VARCHAR,
			book_id     VARCHAR,
			book_title  VARCHAR,
			likes_count INTEGER,
			tags        VARCHAR,
			url         VARCHAR,
			fetched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			user_id           VARCHAR PRIMARY KEY,
			name              VARCHAR,
			username          VARCHAR,
			location          VARCHAR,
			joined_date       TIMESTAMP,
			friends_count     INTEGER,
			books_read_count  INTEGER,
			ratings_count     INTEGER,
			reviews_count     INTEGER,
			avg_rating        DOUBLE,
			bio               VARCHAR,
			website           VARCHAR,
			avatar_url        VARCHAR,
			favorite_book_ids VARCHAR,
			url               VARCHAR,
			fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS genres (
			slug        VARCHAR PRIMARY KEY,
			name        VARCHAR,
			description VARCHAR,
			books_count INTEGER,
			url         VARCHAR,
			fetched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS shelves (
			shelf_id    VARCHAR PRIMARY KEY,
			user_id     VARCHAR,
			name        VARCHAR,
			books_count INTEGER,
			url         VARCHAR,
			fetched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS shelf_books (
			shelf_id   VARCHAR NOT NULL,
			user_id    VARCHAR NOT NULL,
			book_id    VARCHAR NOT NULL,
			date_added TIMESTAMP,
			rating     INTEGER,
			date_read  TIMESTAMP,
			PRIMARY KEY (shelf_id, book_id)
		)`,
	}

	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

// UpsertBook inserts or replaces a book record.
func (d *DB) UpsertBook(b Book) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO books (
		book_id, title, title_without_series, description,
		author_id, author_name, isbn, isbn13, asin,
		avg_rating, ratings_count, reviews_count,
		published_year, publisher, language, pages, format,
		series_id, series_name, series_position,
		genres, cover_url, url, similar_book_ids, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		b.BookID, nullStr(b.Title), nullStr(b.TitleWithoutSeries), nullStr(b.Description),
		nullStr(b.AuthorID), nullStr(b.AuthorName), nullStr(b.ISBN), nullStr(b.ISBN13), nullStr(b.ASIN),
		b.AvgRating, b.RatingsCount, b.ReviewsCount,
		nullInt(b.PublishedYear), nullStr(b.Publisher), nullStr(b.Language), nullInt(b.Pages), nullStr(b.Format),
		nullStr(b.SeriesID), nullStr(b.SeriesName), nullStr(b.SeriesPosition),
		encodeStringSlice(b.Genres), nullStr(b.CoverURL), nullStr(b.URL),
		encodeStringSlice(b.SimilarBookIDs), b.FetchedAt,
	)
	return err
}

// UpsertAuthor inserts or replaces an author record.
func (d *DB) UpsertAuthor(a Author) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO authors (
		author_id, name, bio, photo_url, website,
		born_date, died_date, hometown, influences, genres,
		avg_rating, ratings_count, books_count, followers_count, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.AuthorID, nullStr(a.Name), nullStr(a.Bio), nullStr(a.PhotoURL), nullStr(a.Website),
		nullStr(a.BornDate), nullStr(a.DiedDate), nullStr(a.Hometown),
		encodeStringSlice(a.Influences), encodeStringSlice(a.Genres),
		a.AvgRating, a.RatingsCount, a.BooksCount, a.FollowersCount,
		nullStr(a.URL), a.FetchedAt,
	)
	return err
}

// UpsertSeries inserts or replaces a series record.
func (d *DB) UpsertSeries(s Series) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO series (
		series_id, name, description, total_books, primary_work_count, url, fetched_at
	) VALUES (?,?,?,?,?,?,?)`,
		s.SeriesID, nullStr(s.Name), nullStr(s.Description),
		s.TotalBooks, s.PrimaryWorkCount, nullStr(s.URL), s.FetchedAt,
	)
	return err
}

// InsertSeriesBooks upserts series-book relationships.
func (d *DB) InsertSeriesBooks(books []SeriesBook) error {
	if len(books) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, sb := range books {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO series_books (series_id, book_id, position) VALUES (?,?,?)`,
			sb.SeriesID, sb.BookID, sb.Position,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UpsertList inserts or replaces a list record.
func (d *DB) UpsertList(l List) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO lists (
		list_id, name, description, books_count, voters_count, tags, created_by_user, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?)`,
		l.ListID, nullStr(l.Name), nullStr(l.Description),
		l.BooksCount, l.VotersCount, encodeStringSlice(l.Tags),
		nullStr(l.CreatedByUser), nullStr(l.URL), l.FetchedAt,
	)
	return err
}

// InsertListBooks upserts list-book relationships.
func (d *DB) InsertListBooks(books []ListBook) error {
	if len(books) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, lb := range books {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO list_books (list_id, book_id, rank, votes) VALUES (?,?,?,?)`,
			lb.ListID, lb.BookID, lb.Rank, lb.Votes,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UpsertBookWithReviews inserts a book and all its reviews in a single transaction.
func (d *DB) UpsertBookWithReviews(b Book, reviews []Review) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`INSERT OR REPLACE INTO books (
		book_id, title, title_without_series, description,
		author_id, author_name, isbn, isbn13, asin,
		avg_rating, ratings_count, reviews_count,
		published_year, publisher, language, pages, format,
		series_id, series_name, series_position,
		genres, cover_url, url, similar_book_ids, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		b.BookID, nullStr(b.Title), nullStr(b.TitleWithoutSeries), nullStr(b.Description),
		nullStr(b.AuthorID), nullStr(b.AuthorName), nullStr(b.ISBN), nullStr(b.ISBN13), nullStr(b.ASIN),
		b.AvgRating, b.RatingsCount, b.ReviewsCount,
		nullInt(b.PublishedYear), nullStr(b.Publisher), nullStr(b.Language), nullInt(b.Pages), nullStr(b.Format),
		nullStr(b.SeriesID), nullStr(b.SeriesName), nullStr(b.SeriesPosition),
		encodeStringSlice(b.Genres), nullStr(b.CoverURL), nullStr(b.URL),
		encodeStringSlice(b.SimilarBookIDs), b.FetchedAt,
	); err != nil {
		return err
	}

	for _, r := range reviews {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO reviews (
			review_id, book_id, user_id, user_name, rating, text,
			date_added, likes_count, is_spoiler, url, fetched_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			r.ReviewID, nullStr(r.BookID), nullStr(r.UserID), nullStr(r.UserName),
			r.Rating, nullStr(r.Text), nullTime(r.DateAdded),
			r.LikesCount, r.IsSpoiler, nullStr(r.URL), r.FetchedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpsertReview inserts or replaces a review record.
func (d *DB) UpsertReview(r Review) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO reviews (
		review_id, book_id, user_id, user_name, rating, text,
		date_added, likes_count, is_spoiler, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		r.ReviewID, nullStr(r.BookID), nullStr(r.UserID), nullStr(r.UserName),
		r.Rating, nullStr(r.Text), nullTime(r.DateAdded),
		r.LikesCount, r.IsSpoiler, nullStr(r.URL), r.FetchedAt,
	)
	return err
}

// UpsertQuote inserts or replaces a quote record.
func (d *DB) UpsertQuote(q Quote) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO quotes (
		quote_id, text, author_id, author_name, book_id, book_title,
		likes_count, tags, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		q.QuoteID, nullStr(q.Text), nullStr(q.AuthorID), nullStr(q.AuthorName),
		nullStr(q.BookID), nullStr(q.BookTitle),
		q.LikesCount, encodeStringSlice(q.Tags), nullStr(q.URL), q.FetchedAt,
	)
	return err
}

// UpsertUser inserts or replaces a user record.
func (d *DB) UpsertUser(u User) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO users (
		user_id, name, username, location, joined_date,
		friends_count, books_read_count, ratings_count, reviews_count, avg_rating,
		bio, website, avatar_url, favorite_book_ids, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		u.UserID, nullStr(u.Name), nullStr(u.Username), nullStr(u.Location), nullTime(u.JoinedDate),
		u.FriendsCount, u.BooksReadCount, u.RatingsCount, u.ReviewsCount, u.AvgRating,
		nullStr(u.Bio), nullStr(u.Website), nullStr(u.AvatarURL),
		encodeStringSlice(u.FavoriteBookIDs), nullStr(u.URL), u.FetchedAt,
	)
	return err
}

// UpsertGenre inserts or replaces a genre record.
func (d *DB) UpsertGenre(g Genre) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO genres (
		slug, name, description, books_count, url, fetched_at
	) VALUES (?,?,?,?,?,?)`,
		g.Slug, nullStr(g.Name), nullStr(g.Description),
		g.BooksCount, nullStr(g.URL), g.FetchedAt,
	)
	return err
}

// UpsertShelf inserts or replaces a shelf record.
func (d *DB) UpsertShelf(s Shelf) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO shelves (
		shelf_id, user_id, name, books_count, url, fetched_at
	) VALUES (?,?,?,?,?,?)`,
		s.ShelfID, nullStr(s.UserID), nullStr(s.Name),
		s.BooksCount, nullStr(s.URL), s.FetchedAt,
	)
	return err
}

// InsertShelfBooks upserts shelf-book relationships.
func (d *DB) InsertShelfBooks(books []ShelfBook) error {
	if len(books) == 0 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, sb := range books {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO shelf_books (shelf_id, user_id, book_id, date_added, rating, date_read) VALUES (?,?,?,?,?,?)`,
			sb.ShelfID, sb.UserID, sb.BookID, nullTime(sb.DateAdded), sb.Rating, nullTime(sb.DateRead),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// DBStats holds row counts per table.
type DBStats struct {
	Books    int64
	Authors  int64
	Series   int64
	Lists    int64
	Reviews  int64
	Quotes   int64
	Users    int64
	Genres   int64
	Shelves  int64
	DBSize   int64
}

// GetStats returns row counts for all tables.
func (d *DB) GetStats() (DBStats, error) {
	var s DBStats
	d.db.QueryRow("SELECT COUNT(*) FROM books").Scan(&s.Books)
	d.db.QueryRow("SELECT COUNT(*) FROM authors").Scan(&s.Authors)
	d.db.QueryRow("SELECT COUNT(*) FROM series").Scan(&s.Series)
	d.db.QueryRow("SELECT COUNT(*) FROM lists").Scan(&s.Lists)
	d.db.QueryRow("SELECT COUNT(*) FROM reviews").Scan(&s.Reviews)
	d.db.QueryRow("SELECT COUNT(*) FROM quotes").Scan(&s.Quotes)
	d.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&s.Users)
	d.db.QueryRow("SELECT COUNT(*) FROM genres").Scan(&s.Genres)
	d.db.QueryRow("SELECT COUNT(*) FROM shelves").Scan(&s.Shelves)
	if fi, err := os.Stat(d.path); err == nil {
		s.DBSize = fi.Size()
	}
	return s, nil
}

// RecentBooks returns the most recently fetched books.
func (d *DB) RecentBooks(limit int) ([]Book, error) {
	rows, err := d.db.Query(`
		SELECT book_id, title, author_name, avg_rating, ratings_count, url
		FROM books ORDER BY fetched_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		var title, author, url sql.NullString
		if err := rows.Scan(&b.BookID, &title, &author, &b.AvgRating, &b.RatingsCount, &url); err != nil {
			return books, err
		}
		b.Title = title.String
		b.AuthorName = author.String
		b.URL = url.String
		books = append(books, b)
	}
	return books, rows.Err()
}

// Close closes the database.
func (d *DB) Close() error {
	return d.db.Close()
}

// Path returns the database file path.
func (d *DB) Path() string {
	return d.path
}

// ── tx-based write helpers (used by ImportTask for batch transactions) ────────

func insertBookWithReviewsTx(tx *sql.Tx, b Book, reviews []Review) error {
	if _, err := tx.Exec(`INSERT OR REPLACE INTO books (
		book_id, title, title_without_series, description,
		author_id, author_name, isbn, isbn13, asin,
		avg_rating, ratings_count, reviews_count,
		published_year, publisher, language, pages, format,
		series_id, series_name, series_position,
		genres, cover_url, url, similar_book_ids, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		b.BookID, nullStr(b.Title), nullStr(b.TitleWithoutSeries), nullStr(b.Description),
		nullStr(b.AuthorID), nullStr(b.AuthorName), nullStr(b.ISBN), nullStr(b.ISBN13), nullStr(b.ASIN),
		b.AvgRating, b.RatingsCount, b.ReviewsCount,
		nullInt(b.PublishedYear), nullStr(b.Publisher), nullStr(b.Language), nullInt(b.Pages), nullStr(b.Format),
		nullStr(b.SeriesID), nullStr(b.SeriesName), nullStr(b.SeriesPosition),
		encodeStringSlice(b.Genres), nullStr(b.CoverURL), nullStr(b.URL),
		encodeStringSlice(b.SimilarBookIDs), b.FetchedAt,
	); err != nil {
		return err
	}
	for _, r := range reviews {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO reviews (
			review_id, book_id, user_id, user_name, rating, text,
			date_added, likes_count, is_spoiler, url, fetched_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			r.ReviewID, nullStr(r.BookID), nullStr(r.UserID), nullStr(r.UserName),
			r.Rating, nullStr(r.Text), nullTime(r.DateAdded),
			r.LikesCount, r.IsSpoiler, nullStr(r.URL), r.FetchedAt,
		); err != nil {
			return err
		}
	}
	return nil
}

func insertAuthorTx(tx *sql.Tx, a Author) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO authors (
		author_id, name, bio, photo_url, website,
		born_date, died_date, hometown, influences, genres,
		avg_rating, ratings_count, books_count, followers_count, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.AuthorID, nullStr(a.Name), nullStr(a.Bio), nullStr(a.PhotoURL), nullStr(a.Website),
		nullStr(a.BornDate), nullStr(a.DiedDate), nullStr(a.Hometown),
		encodeStringSlice(a.Influences), encodeStringSlice(a.Genres),
		a.AvgRating, a.RatingsCount, a.BooksCount, a.FollowersCount,
		nullStr(a.URL), a.FetchedAt,
	)
	return err
}

func insertSeriesTx(tx *sql.Tx, s Series, books []SeriesBook) error {
	if _, err := tx.Exec(`INSERT OR REPLACE INTO series (
		series_id, name, description, total_books, primary_work_count, url, fetched_at
	) VALUES (?,?,?,?,?,?,?)`,
		s.SeriesID, nullStr(s.Name), nullStr(s.Description),
		s.TotalBooks, s.PrimaryWorkCount, nullStr(s.URL), s.FetchedAt,
	); err != nil {
		return err
	}
	for _, sb := range books {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO series_books (series_id, book_id, position) VALUES (?,?,?)`,
			sb.SeriesID, sb.BookID, sb.Position,
		); err != nil {
			return err
		}
	}
	return nil
}

func insertListTx(tx *sql.Tx, l List, books []ListBook) error {
	if _, err := tx.Exec(`INSERT OR REPLACE INTO lists (
		list_id, name, description, books_count, voters_count, tags, created_by_user, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?)`,
		l.ListID, nullStr(l.Name), nullStr(l.Description),
		l.BooksCount, l.VotersCount, encodeStringSlice(l.Tags),
		nullStr(l.CreatedByUser), nullStr(l.URL), l.FetchedAt,
	); err != nil {
		return err
	}
	for _, lb := range books {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO list_books (list_id, book_id, rank, votes) VALUES (?,?,?,?)`,
			lb.ListID, lb.BookID, lb.Rank, lb.Votes,
		); err != nil {
			return err
		}
	}
	return nil
}

func insertQuotesTx(tx *sql.Tx, quotes []Quote) error {
	for _, q := range quotes {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO quotes (
			quote_id, text, author_id, author_name, book_id, book_title,
			likes_count, tags, url, fetched_at
		) VALUES (?,?,?,?,?,?,?,?,?,?)`,
			q.QuoteID, nullStr(q.Text), nullStr(q.AuthorID), nullStr(q.AuthorName),
			nullStr(q.BookID), nullStr(q.BookTitle),
			q.LikesCount, encodeStringSlice(q.Tags), nullStr(q.URL), q.FetchedAt,
		); err != nil {
			return err
		}
	}
	return nil
}

func insertUserTx(tx *sql.Tx, u User) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO users (
		user_id, name, username, location, joined_date,
		friends_count, books_read_count, ratings_count, reviews_count, avg_rating,
		bio, website, avatar_url, favorite_book_ids, url, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		u.UserID, nullStr(u.Name), nullStr(u.Username), nullStr(u.Location), nullTime(u.JoinedDate),
		u.FriendsCount, u.BooksReadCount, u.RatingsCount, u.ReviewsCount, u.AvgRating,
		nullStr(u.Bio), nullStr(u.Website), nullStr(u.AvatarURL),
		encodeStringSlice(u.FavoriteBookIDs), nullStr(u.URL), u.FetchedAt,
	)
	return err
}

func insertGenreTx(tx *sql.Tx, g Genre) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO genres (
		slug, name, description, books_count, url, fetched_at
	) VALUES (?,?,?,?,?,?)`,
		g.Slug, nullStr(g.Name), nullStr(g.Description),
		g.BooksCount, nullStr(g.URL), g.FetchedAt,
	)
	return err
}

func insertShelfTx(tx *sql.Tx, s Shelf, books []ShelfBook) error {
	if _, err := tx.Exec(`INSERT OR REPLACE INTO shelves (
		shelf_id, user_id, name, books_count, url, fetched_at
	) VALUES (?,?,?,?,?,?)`,
		s.ShelfID, nullStr(s.UserID), nullStr(s.Name),
		s.BooksCount, nullStr(s.URL), s.FetchedAt,
	); err != nil {
		return err
	}
	for _, sb := range books {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO shelf_books (shelf_id, user_id, book_id, date_added, rating, date_read) VALUES (?,?,?,?,?,?)`,
			sb.ShelfID, sb.UserID, sb.BookID, nullTime(sb.DateAdded), sb.Rating, nullTime(sb.DateRead),
		); err != nil {
			return err
		}
	}
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func encodeStringSlice(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(ss)
	return string(b)
}

func decodeStringSlice(s string) []string {
	if s == "" || s == "[]" {
		return nil
	}
	var out []string
	json.Unmarshal([]byte(s), &out)
	return out
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullInt(i int) any {
	if i == 0 {
		return nil
	}
	return i
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
