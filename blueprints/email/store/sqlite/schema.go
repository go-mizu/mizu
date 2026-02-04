package sqlite

import (
	"context"
	"database/sql"
)

// createSchema creates all tables and FTS5 virtual tables.
func createSchema(ctx context.Context, db *sql.DB) error {
	schema := `
		-- Emails table
		CREATE TABLE IF NOT EXISTS emails (
			id TEXT PRIMARY KEY,
			thread_id TEXT NOT NULL,
			message_id TEXT UNIQUE NOT NULL,
			in_reply_to TEXT,
			reference_ids TEXT DEFAULT '[]',
			from_address TEXT NOT NULL,
			from_name TEXT NOT NULL DEFAULT '',
			to_addresses TEXT NOT NULL DEFAULT '[]',
			cc_addresses TEXT DEFAULT '[]',
			bcc_addresses TEXT DEFAULT '[]',
			subject TEXT NOT NULL DEFAULT '',
			body_text TEXT DEFAULT '',
			body_html TEXT DEFAULT '',
			snippet TEXT DEFAULT '',
			is_read INTEGER DEFAULT 0,
			is_starred INTEGER DEFAULT 0,
			is_important INTEGER DEFAULT 0,
			is_draft INTEGER DEFAULT 0,
			is_sent INTEGER DEFAULT 0,
			has_attachments INTEGER DEFAULT 0,
			size_bytes INTEGER DEFAULT 0,
			sent_at DATETIME,
			received_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			snoozed_until DATETIME,
			scheduled_at DATETIME,
			is_muted INTEGER DEFAULT 0
		);

		CREATE INDEX IF NOT EXISTS idx_emails_thread_id ON emails(thread_id);
		CREATE INDEX IF NOT EXISTS idx_emails_from_address ON emails(from_address);
		CREATE INDEX IF NOT EXISTS idx_emails_is_read ON emails(is_read);
		CREATE INDEX IF NOT EXISTS idx_emails_is_starred ON emails(is_starred);
		CREATE INDEX IF NOT EXISTS idx_emails_is_important ON emails(is_important);
		CREATE INDEX IF NOT EXISTS idx_emails_is_draft ON emails(is_draft);
		CREATE INDEX IF NOT EXISTS idx_emails_is_sent ON emails(is_sent);
		CREATE INDEX IF NOT EXISTS idx_emails_received_at ON emails(received_at);
		CREATE INDEX IF NOT EXISTS idx_emails_sent_at ON emails(sent_at);
		CREATE INDEX IF NOT EXISTS idx_emails_snoozed_until ON emails(snoozed_until);
		CREATE INDEX IF NOT EXISTS idx_emails_scheduled_at ON emails(scheduled_at);
		CREATE INDEX IF NOT EXISTS idx_emails_is_muted ON emails(is_muted);

		-- FTS5 virtual table for full-text email search
		CREATE VIRTUAL TABLE IF NOT EXISTS emails_fts USING fts5(
			subject,
			body_text,
			from_name,
			from_address,
			content='emails',
			content_rowid='rowid',
			tokenize='porter unicode61'
		);

		-- Triggers to keep FTS in sync
		CREATE TRIGGER IF NOT EXISTS emails_fts_ai AFTER INSERT ON emails BEGIN
			INSERT INTO emails_fts(rowid, subject, body_text, from_name, from_address)
			VALUES (NEW.rowid, NEW.subject, NEW.body_text, NEW.from_name, NEW.from_address);
		END;

		CREATE TRIGGER IF NOT EXISTS emails_fts_ad AFTER DELETE ON emails BEGIN
			INSERT INTO emails_fts(emails_fts, rowid, subject, body_text, from_name, from_address)
			VALUES ('delete', OLD.rowid, OLD.subject, OLD.body_text, OLD.from_name, OLD.from_address);
		END;

		CREATE TRIGGER IF NOT EXISTS emails_fts_au AFTER UPDATE ON emails BEGIN
			INSERT INTO emails_fts(emails_fts, rowid, subject, body_text, from_name, from_address)
			VALUES ('delete', OLD.rowid, OLD.subject, OLD.body_text, OLD.from_name, OLD.from_address);
			INSERT INTO emails_fts(rowid, subject, body_text, from_name, from_address)
			VALUES (NEW.rowid, NEW.subject, NEW.body_text, NEW.from_name, NEW.from_address);
		END;

		-- Email labels junction table
		CREATE TABLE IF NOT EXISTS email_labels (
			email_id TEXT NOT NULL,
			label_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (email_id, label_id),
			FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE,
			FOREIGN KEY (label_id) REFERENCES labels(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_email_labels_email_id ON email_labels(email_id);
		CREATE INDEX IF NOT EXISTS idx_email_labels_label_id ON email_labels(label_id);

		-- Labels table
		CREATE TABLE IF NOT EXISTS labels (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			color TEXT DEFAULT '',
			type TEXT NOT NULL DEFAULT 'user' CHECK (type IN ('system', 'user')),
			visible INTEGER DEFAULT 1,
			position INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_labels_type ON labels(type);
		CREATE INDEX IF NOT EXISTS idx_labels_position ON labels(position);

		-- Contacts table
		CREATE TABLE IF NOT EXISTS contacts (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			avatar_url TEXT DEFAULT '',
			is_frequent INTEGER DEFAULT 0,
			last_contacted DATETIME,
			contact_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_contacts_email ON contacts(email);
		CREATE INDEX IF NOT EXISTS idx_contacts_is_frequent ON contacts(is_frequent);
		CREATE INDEX IF NOT EXISTS idx_contacts_contact_count ON contacts(contact_count DESC);

		-- FTS for contact search
		CREATE VIRTUAL TABLE IF NOT EXISTS contacts_fts USING fts5(
			name,
			email,
			content='contacts',
			content_rowid='rowid',
			tokenize='porter unicode61'
		);

		CREATE TRIGGER IF NOT EXISTS contacts_fts_ai AFTER INSERT ON contacts BEGIN
			INSERT INTO contacts_fts(rowid, name, email)
			VALUES (NEW.rowid, NEW.name, NEW.email);
		END;

		CREATE TRIGGER IF NOT EXISTS contacts_fts_ad AFTER DELETE ON contacts BEGIN
			INSERT INTO contacts_fts(contacts_fts, rowid, name, email)
			VALUES ('delete', OLD.rowid, OLD.name, OLD.email);
		END;

		CREATE TRIGGER IF NOT EXISTS contacts_fts_au AFTER UPDATE ON contacts BEGIN
			INSERT INTO contacts_fts(contacts_fts, rowid, name, email)
			VALUES ('delete', OLD.rowid, OLD.name, OLD.email);
			INSERT INTO contacts_fts(rowid, name, email)
			VALUES (NEW.rowid, NEW.name, NEW.email);
		END;

		-- Attachments table
		CREATE TABLE IF NOT EXISTS attachments (
			id TEXT PRIMARY KEY,
			email_id TEXT NOT NULL,
			filename TEXT NOT NULL,
			content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
			size_bytes INTEGER DEFAULT 0,
			data BLOB,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_attachments_email_id ON attachments(email_id);

		-- Settings table (singleton)
		CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			display_name TEXT DEFAULT 'Me',
			email_address TEXT DEFAULT 'me@example.com',
			signature TEXT DEFAULT '',
			theme TEXT DEFAULT 'light',
			density TEXT DEFAULT 'default',
			conversation_view INTEGER DEFAULT 1,
			auto_advance TEXT DEFAULT 'newer',
			undo_send_seconds INTEGER DEFAULT 5
		);

		-- Insert default settings
		INSERT OR IGNORE INTO settings (id) VALUES (1);
	`

	_, err := db.ExecContext(ctx, schema)
	return err
}
