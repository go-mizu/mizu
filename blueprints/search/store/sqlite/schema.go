package sqlite

import (
	"context"
	"database/sql"
)

// createSchema creates all tables and FTS5 virtual tables.
func createSchema(ctx context.Context, db *sql.DB) error {
	schema := `
		-- Documents table for indexed content
		CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			url TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			content TEXT,
			domain TEXT NOT NULL,
			language TEXT DEFAULT 'en',
			content_type TEXT DEFAULT 'text/html',
			favicon TEXT,
			word_count INTEGER DEFAULT 0,
			crawled_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			metadata TEXT DEFAULT '{}'
		);

		CREATE INDEX IF NOT EXISTS idx_documents_domain ON documents(domain);
		CREATE INDEX IF NOT EXISTS idx_documents_language ON documents(language);
		CREATE INDEX IF NOT EXISTS idx_documents_crawled_at ON documents(crawled_at);

		-- FTS5 virtual table for full-text search
		CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts5(
			title,
			description,
			content,
			content='documents',
			content_rowid='rowid',
			tokenize='porter unicode61'
		);

		-- Triggers to keep FTS in sync
		CREATE TRIGGER IF NOT EXISTS documents_ai AFTER INSERT ON documents BEGIN
			INSERT INTO documents_fts(rowid, title, description, content)
			VALUES (NEW.rowid, NEW.title, NEW.description, NEW.content);
		END;

		CREATE TRIGGER IF NOT EXISTS documents_ad AFTER DELETE ON documents BEGIN
			INSERT INTO documents_fts(documents_fts, rowid, title, description, content)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.description, OLD.content);
		END;

		CREATE TRIGGER IF NOT EXISTS documents_au AFTER UPDATE ON documents BEGIN
			INSERT INTO documents_fts(documents_fts, rowid, title, description, content)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.description, OLD.content);
			INSERT INTO documents_fts(rowid, title, description, content)
			VALUES (NEW.rowid, NEW.title, NEW.description, NEW.content);
		END;

		-- Images table
		CREATE TABLE IF NOT EXISTS images (
			id TEXT PRIMARY KEY,
			url TEXT UNIQUE NOT NULL,
			thumbnail_url TEXT,
			title TEXT,
			source_url TEXT NOT NULL,
			source_domain TEXT NOT NULL,
			width INTEGER,
			height INTEGER,
			file_size INTEGER,
			format TEXT,
			crawled_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_images_source_domain ON images(source_domain);

		-- FTS for image search
		CREATE VIRTUAL TABLE IF NOT EXISTS images_fts USING fts5(
			title,
			content='images',
			content_rowid='rowid',
			tokenize='porter unicode61'
		);

		CREATE TRIGGER IF NOT EXISTS images_ai AFTER INSERT ON images BEGIN
			INSERT INTO images_fts(rowid, title) VALUES (NEW.rowid, NEW.title);
		END;

		CREATE TRIGGER IF NOT EXISTS images_ad AFTER DELETE ON images BEGIN
			INSERT INTO images_fts(images_fts, rowid, title)
			VALUES ('delete', OLD.rowid, OLD.title);
		END;

		-- Videos table
		CREATE TABLE IF NOT EXISTS videos (
			id TEXT PRIMARY KEY,
			url TEXT UNIQUE NOT NULL,
			thumbnail_url TEXT,
			title TEXT NOT NULL,
			description TEXT,
			duration_seconds INTEGER,
			channel TEXT,
			views INTEGER DEFAULT 0,
			published_at DATETIME,
			crawled_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_videos_channel ON videos(channel);
		CREATE INDEX IF NOT EXISTS idx_videos_published_at ON videos(published_at);

		-- FTS for video search
		CREATE VIRTUAL TABLE IF NOT EXISTS videos_fts USING fts5(
			title,
			description,
			channel,
			content='videos',
			content_rowid='rowid',
			tokenize='porter unicode61'
		);

		CREATE TRIGGER IF NOT EXISTS videos_ai AFTER INSERT ON videos BEGIN
			INSERT INTO videos_fts(rowid, title, description, channel)
			VALUES (NEW.rowid, NEW.title, NEW.description, NEW.channel);
		END;

		CREATE TRIGGER IF NOT EXISTS videos_ad AFTER DELETE ON videos BEGIN
			INSERT INTO videos_fts(videos_fts, rowid, title, description, channel)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.description, OLD.channel);
		END;

		-- News table
		CREATE TABLE IF NOT EXISTS news (
			id TEXT PRIMARY KEY,
			url TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			snippet TEXT,
			source TEXT NOT NULL,
			image_url TEXT,
			published_at DATETIME NOT NULL,
			crawled_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_news_source ON news(source);
		CREATE INDEX IF NOT EXISTS idx_news_published_at ON news(published_at);

		-- FTS for news search
		CREATE VIRTUAL TABLE IF NOT EXISTS news_fts USING fts5(
			title,
			snippet,
			source,
			content='news',
			content_rowid='rowid',
			tokenize='porter unicode61'
		);

		CREATE TRIGGER IF NOT EXISTS news_ai AFTER INSERT ON news BEGIN
			INSERT INTO news_fts(rowid, title, snippet, source)
			VALUES (NEW.rowid, NEW.title, NEW.snippet, NEW.source);
		END;

		CREATE TRIGGER IF NOT EXISTS news_ad AFTER DELETE ON news BEGIN
			INSERT INTO news_fts(news_fts, rowid, title, snippet, source)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.snippet, OLD.source);
		END;

		-- Suggestions table for autocomplete
		CREATE TABLE IF NOT EXISTS suggestions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			query TEXT UNIQUE NOT NULL COLLATE NOCASE,
			frequency INTEGER DEFAULT 1,
			last_used DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_suggestions_frequency ON suggestions(frequency DESC);
		CREATE INDEX IF NOT EXISTS idx_suggestions_last_used ON suggestions(last_used DESC);

		-- Knowledge entities table
		CREATE TABLE IF NOT EXISTS entities (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL COLLATE NOCASE,
			type TEXT NOT NULL,
			description TEXT,
			image TEXT,
			facts TEXT DEFAULT '{}',
			links TEXT DEFAULT '[]',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_entities_name ON entities(name);
		CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(type);

		-- FTS for entity search
		CREATE VIRTUAL TABLE IF NOT EXISTS entities_fts USING fts5(
			name,
			description,
			content='entities',
			content_rowid='rowid',
			tokenize='porter unicode61'
		);

		CREATE TRIGGER IF NOT EXISTS entities_ai AFTER INSERT ON entities BEGIN
			INSERT INTO entities_fts(rowid, name, description)
			VALUES (NEW.rowid, NEW.name, NEW.description);
		END;

		CREATE TRIGGER IF NOT EXISTS entities_ad AFTER DELETE ON entities BEGIN
			INSERT INTO entities_fts(entities_fts, rowid, name, description)
			VALUES ('delete', OLD.rowid, OLD.name, OLD.description);
		END;

		CREATE TRIGGER IF NOT EXISTS entities_au AFTER UPDATE ON entities BEGIN
			INSERT INTO entities_fts(entities_fts, rowid, name, description)
			VALUES ('delete', OLD.rowid, OLD.name, OLD.description);
			INSERT INTO entities_fts(rowid, name, description)
			VALUES (NEW.rowid, NEW.name, NEW.description);
		END;

		-- Search history table
		CREATE TABLE IF NOT EXISTS history (
			id TEXT PRIMARY KEY,
			query TEXT NOT NULL,
			results INTEGER DEFAULT 0,
			clicked_url TEXT,
			searched_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_history_searched_at ON history(searched_at DESC);
		CREATE INDEX IF NOT EXISTS idx_history_query ON history(query);

		-- User preferences table
		CREATE TABLE IF NOT EXISTS preferences (
			id TEXT PRIMARY KEY,
			domain TEXT UNIQUE NOT NULL,
			action TEXT NOT NULL CHECK (action IN ('upvote', 'downvote', 'block')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_preferences_domain ON preferences(domain);
		CREATE INDEX IF NOT EXISTS idx_preferences_action ON preferences(action);

		-- Search lenses table
		CREATE TABLE IF NOT EXISTS lenses (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			domains TEXT DEFAULT '[]',
			exclude TEXT DEFAULT '[]',
			keywords TEXT DEFAULT '[]',
			is_public INTEGER DEFAULT 0,
			is_built_in INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_lenses_is_public ON lenses(is_public);

		-- Settings table (singleton)
		CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			safe_search TEXT DEFAULT 'moderate',
			results_per_page INTEGER DEFAULT 10,
			region TEXT DEFAULT 'us',
			language TEXT DEFAULT 'en',
			theme TEXT DEFAULT 'system',
			open_in_new_tab INTEGER DEFAULT 0,
			show_thumbnails INTEGER DEFAULT 1
		);

		-- Insert default settings
		INSERT OR IGNORE INTO settings (id) VALUES (1);

		-- Search cache table with versioning (no TTL)
		CREATE TABLE IF NOT EXISTS search_cache (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hash TEXT NOT NULL,
			query TEXT NOT NULL,
			category TEXT NOT NULL,
			options_json TEXT NOT NULL DEFAULT '{}',
			results_json TEXT NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(hash, version)
		);

		CREATE INDEX IF NOT EXISTS idx_cache_hash ON search_cache(hash);
		CREATE INDEX IF NOT EXISTS idx_cache_query ON search_cache(query);
		CREATE INDEX IF NOT EXISTS idx_cache_created ON search_cache(created_at);
	`

	_, err := db.ExecContext(ctx, schema)
	return err
}
