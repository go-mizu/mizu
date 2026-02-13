-- Book Worker D1 Schema

CREATE TABLE IF NOT EXISTS books (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ol_key TEXT DEFAULT '',
  google_id TEXT DEFAULT '',
  title TEXT NOT NULL,
  original_title TEXT DEFAULT '',
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
  language TEXT DEFAULT '',
  edition_language TEXT DEFAULT '',
  first_published TEXT DEFAULT '',
  page_count INTEGER DEFAULT 0,
  format TEXT DEFAULT '',
  subjects TEXT DEFAULT '[]',
  characters TEXT DEFAULT '[]',
  settings TEXT DEFAULT '[]',
  literary_awards TEXT DEFAULT '[]',
  series TEXT DEFAULT '',
  editions_count INTEGER DEFAULT 0,
  average_rating REAL DEFAULT 0,
  ratings_count INTEGER DEFAULT 0,
  reviews_count INTEGER DEFAULT 0,
  currently_reading INTEGER DEFAULT 0,
  want_to_read INTEGER DEFAULT 0,
  rating_dist TEXT DEFAULT '[]',
  source_id TEXT DEFAULT '',
  source_url TEXT DEFAULT '',
  asin TEXT DEFAULT '',
  enriched INTEGER DEFAULT 0,
  created_at TEXT DEFAULT (datetime('now')),
  updated_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS authors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ol_key TEXT DEFAULT '',
  name TEXT NOT NULL,
  bio TEXT DEFAULT '',
  photo_url TEXT DEFAULT '',
  birth_date TEXT DEFAULT '',
  death_date TEXT DEFAULT '',
  works_count INTEGER DEFAULT 0,
  followers INTEGER DEFAULT 0,
  genres TEXT DEFAULT '',
  influences TEXT DEFAULT '',
  website TEXT DEFAULT '',
  source_id TEXT DEFAULT '',
  enriched INTEGER DEFAULT 0,
  created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS book_authors (
  book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  author_id INTEGER NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
  PRIMARY KEY (book_id, author_id)
);

CREATE TABLE IF NOT EXISTS shelves (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  is_exclusive INTEGER DEFAULT 0,
  is_default INTEGER DEFAULT 0,
  sort_order INTEGER DEFAULT 0,
  created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS shelf_books (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  shelf_id INTEGER NOT NULL REFERENCES shelves(id) ON DELETE CASCADE,
  book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  date_added TEXT DEFAULT (datetime('now')),
  position INTEGER DEFAULT 0,
  date_started TEXT,
  date_read TEXT,
  read_count INTEGER DEFAULT 0,
  UNIQUE(shelf_id, book_id)
);

CREATE TABLE IF NOT EXISTS reviews (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  rating INTEGER DEFAULT 0,
  text TEXT DEFAULT '',
  is_spoiler INTEGER DEFAULT 0,
  likes_count INTEGER DEFAULT 0,
  comments_count INTEGER DEFAULT 0,
  reviewer_name TEXT DEFAULT '',
  source TEXT DEFAULT 'user',
  started_at TEXT,
  finished_at TEXT,
  created_at TEXT DEFAULT (datetime('now')),
  updated_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS review_comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  review_id INTEGER NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
  author_name TEXT DEFAULT '',
  text TEXT NOT NULL,
  created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS reading_progress (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  page INTEGER DEFAULT 0,
  percent REAL DEFAULT 0,
  note TEXT DEFAULT '',
  created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS reading_challenges (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  year INTEGER UNIQUE NOT NULL,
  goal INTEGER DEFAULT 0,
  progress INTEGER DEFAULT 0,
  created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS book_lists (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  description TEXT DEFAULT '',
  item_count INTEGER DEFAULT 0,
  source_url TEXT DEFAULT '',
  voter_count INTEGER DEFAULT 0,
  tag TEXT DEFAULT '',
  enriched INTEGER DEFAULT 0,
  created_at TEXT DEFAULT (datetime('now'))
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
  created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS book_notes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  book_id INTEGER UNIQUE NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  text TEXT NOT NULL,
  created_at TEXT DEFAULT (datetime('now')),
  updated_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS feed_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL,
  book_id INTEGER DEFAULT 0,
  book_title TEXT DEFAULT '',
  data TEXT DEFAULT '',
  created_at TEXT DEFAULT (datetime('now'))
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_books_source_id ON books(source_id);
CREATE INDEX IF NOT EXISTS idx_books_title ON books(title);
CREATE INDEX IF NOT EXISTS idx_books_ol_key ON books(ol_key);
CREATE INDEX IF NOT EXISTS idx_authors_ol_key ON authors(ol_key);
CREATE INDEX IF NOT EXISTS idx_authors_source_id ON authors(source_id);
CREATE INDEX IF NOT EXISTS idx_authors_name ON authors(name);
CREATE INDEX IF NOT EXISTS idx_shelf_books_book ON shelf_books(book_id);
CREATE INDEX IF NOT EXISTS idx_shelf_books_shelf ON shelf_books(shelf_id);
CREATE INDEX IF NOT EXISTS idx_reviews_book ON reviews(book_id);
CREATE INDEX IF NOT EXISTS idx_quotes_book ON quotes(book_id);
CREATE INDEX IF NOT EXISTS idx_feed_created ON feed_items(created_at);

-- Default shelves
INSERT OR IGNORE INTO shelves (name, slug, is_exclusive, is_default, sort_order) VALUES
  ('Want to Read', 'want-to-read', 1, 1, 1),
  ('Currently Reading', 'currently-reading', 1, 1, 2),
  ('Read', 'read', 1, 1, 3);
