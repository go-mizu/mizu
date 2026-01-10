/**
 * Database schemas for migrations
 *
 * These schemas use tile-based storage for cells (sheet_tiles table)
 */

/**
 * SQLite schema (compatible with D1 and better-sqlite3)
 */
export const sqliteSchema = `
-- SQLite Schema for Spreadsheet (Tile-based storage)

CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    email         TEXT UNIQUE NOT NULL,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TEXT DEFAULT (datetime('now')),
    updated_at    TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    token      TEXT UNIQUE NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS workbooks (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    name       TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_workbooks_user ON workbooks(user_id);

CREATE TABLE IF NOT EXISTS sheets (
    id          TEXT PRIMARY KEY,
    workbook_id TEXT NOT NULL,
    name        TEXT NOT NULL,
    index_num   INTEGER NOT NULL DEFAULT 0,
    row_count   INTEGER DEFAULT 1000,
    col_count   INTEGER DEFAULT 26,
    created_at  TEXT DEFAULT (datetime('now')),
    updated_at  TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (workbook_id) REFERENCES workbooks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sheets_workbook ON sheets(workbook_id);

CREATE TABLE IF NOT EXISTS sheet_tiles (
    sheet_id    TEXT NOT NULL,
    tile_row    INTEGER NOT NULL,
    tile_col    INTEGER NOT NULL,
    tile_h      INTEGER NOT NULL DEFAULT 256,
    tile_w      INTEGER NOT NULL DEFAULT 64,
    encoding    TEXT NOT NULL DEFAULT 'json_v1',
    values_blob TEXT NOT NULL,
    updated_at  TEXT DEFAULT (datetime('now')),
    PRIMARY KEY (sheet_id, tile_row, tile_col),
    FOREIGN KEY (sheet_id) REFERENCES sheets(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sheet_tiles_lookup
ON sheet_tiles(sheet_id, tile_row, tile_col);

CREATE TABLE IF NOT EXISTS merged_regions (
    id        TEXT PRIMARY KEY,
    sheet_id  TEXT NOT NULL,
    start_row INTEGER NOT NULL,
    start_col INTEGER NOT NULL,
    end_row   INTEGER NOT NULL,
    end_col   INTEGER NOT NULL,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_merged_sheet ON merged_regions(sheet_id);

CREATE TABLE IF NOT EXISTS charts (
    id         TEXT PRIMARY KEY,
    sheet_id   TEXT NOT NULL,
    title      TEXT NOT NULL DEFAULT '',
    chart_type TEXT NOT NULL DEFAULT 'bar',
    data_range TEXT NOT NULL,
    config     TEXT,
    position_x INTEGER DEFAULT 0,
    position_y INTEGER DEFAULT 0,
    width      INTEGER DEFAULT 400,
    height     INTEGER DEFAULT 300,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (sheet_id) REFERENCES sheets(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_charts_sheet ON charts(sheet_id);
`;

/**
 * PostgreSQL schema
 */
export const postgresSchema = `
-- PostgreSQL Schema for Spreadsheet (Tile-based storage)

CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    email         TEXT UNIQUE NOT NULL,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMP DEFAULT NOW(),
    updated_at    TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS workbooks (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workbooks_user ON workbooks(user_id);

CREATE TABLE IF NOT EXISTS sheets (
    id          TEXT PRIMARY KEY,
    workbook_id TEXT NOT NULL REFERENCES workbooks(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    index_num   INTEGER NOT NULL DEFAULT 0,
    row_count   INTEGER DEFAULT 1000,
    col_count   INTEGER DEFAULT 26,
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sheets_workbook ON sheets(workbook_id);

CREATE TABLE IF NOT EXISTS sheet_tiles (
    sheet_id    TEXT NOT NULL REFERENCES sheets(id) ON DELETE CASCADE,
    tile_row    INTEGER NOT NULL,
    tile_col    INTEGER NOT NULL,
    tile_h      INTEGER NOT NULL DEFAULT 256,
    tile_w      INTEGER NOT NULL DEFAULT 64,
    encoding    TEXT NOT NULL DEFAULT 'json_v1',
    values_blob TEXT NOT NULL,
    updated_at  TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (sheet_id, tile_row, tile_col)
);

CREATE INDEX IF NOT EXISTS idx_sheet_tiles_lookup
ON sheet_tiles(sheet_id, tile_row, tile_col);

CREATE TABLE IF NOT EXISTS merged_regions (
    id        TEXT PRIMARY KEY,
    sheet_id  TEXT NOT NULL REFERENCES sheets(id) ON DELETE CASCADE,
    start_row INTEGER NOT NULL,
    start_col INTEGER NOT NULL,
    end_row   INTEGER NOT NULL,
    end_col   INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_merged_sheet ON merged_regions(sheet_id);

CREATE TABLE IF NOT EXISTS charts (
    id         TEXT PRIMARY KEY,
    sheet_id   TEXT NOT NULL REFERENCES sheets(id) ON DELETE CASCADE,
    title      TEXT NOT NULL DEFAULT '',
    chart_type TEXT NOT NULL DEFAULT 'bar',
    data_range TEXT NOT NULL,
    config     TEXT,
    position_x INTEGER DEFAULT 0,
    position_y INTEGER DEFAULT 0,
    width      INTEGER DEFAULT 400,
    height     INTEGER DEFAULT 300,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_charts_sheet ON charts(sheet_id);
`;

/**
 * Default schema (SQLite for backward compatibility)
 * @deprecated Use sqliteSchema or postgresSchema directly
 */
export const schema = sqliteSchema;
