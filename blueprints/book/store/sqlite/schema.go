package sqlite

const schema = `
CREATE TABLE IF NOT EXISTS books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ol_key TEXT DEFAULT '',
    google_id TEXT DEFAULT '',
    title TEXT NOT NULL,
    subtitle TEXT DEFAULT '',
    description TEXT DEFAULT '',
    author_names TEXT DEFAULT '',
    cover_url TEXT DEFAULT '',
    cover_id INTEGER DEFAULT 0,
    isbn10 TEXT DEFAULT '',
    isbn13 TEXT DEFAULT '',
    publisher TEXT DEFAULT '',
    publish_date TEXT DEFAULT '',
    publish_year INTEGER DEFAULT 0,
    page_count INTEGER DEFAULT 0,
    language TEXT DEFAULT 'en',
    format TEXT DEFAULT '',
    subjects_json TEXT DEFAULT '[]',
    average_rating REAL DEFAULT 0,
    ratings_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    goodreads_id TEXT DEFAULT '',
    asin TEXT DEFAULT '',
    series TEXT DEFAULT '',
    reviews_count INTEGER DEFAULT 0,
    currently_reading INTEGER DEFAULT 0,
    want_to_read INTEGER DEFAULT 0,
    rating_dist TEXT DEFAULT '[]',
    first_published TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_books_title ON books(title);
CREATE INDEX IF NOT EXISTS idx_books_isbn13 ON books(isbn13);
CREATE INDEX IF NOT EXISTS idx_books_isbn10 ON books(isbn10);
CREATE INDEX IF NOT EXISTS idx_books_ol_key ON books(ol_key);
CREATE INDEX IF NOT EXISTS idx_books_publish_year ON books(publish_year);

CREATE VIRTUAL TABLE IF NOT EXISTS books_fts USING fts5(title, author_names, description, subjects_json, content=books, content_rowid=id);

CREATE TABLE IF NOT EXISTS authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ol_key TEXT UNIQUE DEFAULT '',
    name TEXT NOT NULL,
    bio TEXT DEFAULT '',
    photo_url TEXT DEFAULT '',
    birth_date TEXT DEFAULT '',
    death_date TEXT DEFAULT '',
    works_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    goodreads_id TEXT DEFAULT '',
    followers INTEGER DEFAULT 0,
    genres TEXT DEFAULT '',
    influences TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_authors_name ON authors(name);
CREATE INDEX IF NOT EXISTS idx_authors_goodreads_id ON authors(goodreads_id);

CREATE TABLE IF NOT EXISTS shelves (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    is_exclusive INTEGER DEFAULT 0,
    is_default INTEGER DEFAULT 0,
    sort_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS shelf_books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    shelf_id INTEGER NOT NULL REFERENCES shelves(id) ON DELETE CASCADE,
    book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    date_added DATETIME DEFAULT CURRENT_TIMESTAMP,
    position INTEGER DEFAULT 0,
    UNIQUE(shelf_id, book_id)
);

CREATE INDEX IF NOT EXISTS idx_shelf_books_shelf ON shelf_books(shelf_id);
CREATE INDEX IF NOT EXISTS idx_shelf_books_book ON shelf_books(book_id);

CREATE TABLE IF NOT EXISTS reviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    rating INTEGER DEFAULT 0,
    text TEXT DEFAULT '',
    is_spoiler INTEGER DEFAULT 0,
    likes_count INTEGER DEFAULT 0,
    started_at DATETIME,
    finished_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    reviewer_name TEXT DEFAULT '',
    source TEXT DEFAULT 'user'
);

CREATE INDEX IF NOT EXISTS idx_reviews_book ON reviews(book_id);

CREATE TABLE IF NOT EXISTS reading_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    page INTEGER DEFAULT 0,
    percent REAL DEFAULT 0,
    note TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_progress_book ON reading_progress(book_id);

CREATE TABLE IF NOT EXISTS reading_challenges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL UNIQUE,
    goal INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS book_lists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    description TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    goodreads_url TEXT DEFAULT '',
    voter_count INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS book_list_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    list_id INTEGER NOT NULL REFERENCES book_lists(id) ON DELETE CASCADE,
    book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    position INTEGER DEFAULT 0,
    votes INTEGER DEFAULT 0,
    UNIQUE(list_id, book_id)
);

CREATE TABLE IF NOT EXISTS quotes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    author_name TEXT DEFAULT '',
    text TEXT NOT NULL,
    likes_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_quotes_book ON quotes(book_id);

CREATE TABLE IF NOT EXISTS feed (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,
    book_id INTEGER DEFAULT 0,
    book_title TEXT DEFAULT '',
    data TEXT DEFAULT '{}',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_feed_created ON feed(created_at DESC);
`

// migration adds columns to existing databases that lack them.
const migration = `
ALTER TABLE books ADD COLUMN goodreads_id TEXT DEFAULT '';
ALTER TABLE books ADD COLUMN asin TEXT DEFAULT '';
ALTER TABLE books ADD COLUMN series TEXT DEFAULT '';
ALTER TABLE books ADD COLUMN reviews_count INTEGER DEFAULT 0;
ALTER TABLE books ADD COLUMN currently_reading INTEGER DEFAULT 0;
ALTER TABLE books ADD COLUMN want_to_read INTEGER DEFAULT 0;
ALTER TABLE books ADD COLUMN rating_dist TEXT DEFAULT '[]';
ALTER TABLE books ADD COLUMN first_published TEXT DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_books_goodreads_id ON books(goodreads_id);
ALTER TABLE reviews ADD COLUMN reviewer_name TEXT DEFAULT '';
ALTER TABLE reviews ADD COLUMN source TEXT DEFAULT 'user';
ALTER TABLE authors ADD COLUMN goodreads_id TEXT DEFAULT '';
ALTER TABLE authors ADD COLUMN followers INTEGER DEFAULT 0;
ALTER TABLE authors ADD COLUMN genres TEXT DEFAULT '';
ALTER TABLE authors ADD COLUMN influences TEXT DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_authors_goodreads_id ON authors(goodreads_id);
ALTER TABLE book_lists ADD COLUMN goodreads_url TEXT DEFAULT '';
ALTER TABLE book_lists ADD COLUMN voter_count INTEGER DEFAULT 0;
`
