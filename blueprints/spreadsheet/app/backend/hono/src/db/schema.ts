/**
 * Database schema as a string for programmatic use
 */
export const schema = `
-- Spreadsheet Application Schema
-- Compatible with PostgreSQL and SQLite/D1

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Workbooks table
CREATE TABLE IF NOT EXISTS workbooks (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Sheets table
CREATE TABLE IF NOT EXISTS sheets (
    id TEXT PRIMARY KEY,
    workbook_id TEXT NOT NULL,
    name TEXT NOT NULL,
    index_num INTEGER NOT NULL DEFAULT 0,
    row_count INTEGER DEFAULT 1000,
    col_count INTEGER DEFAULT 26,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (workbook_id) REFERENCES workbooks(id) ON DELETE CASCADE
);

-- Cells table (sparse storage - only non-empty cells stored)
CREATE TABLE IF NOT EXISTS cells (
    id TEXT PRIMARY KEY,
    sheet_id TEXT NOT NULL,
    row_num INTEGER NOT NULL,
    col_num INTEGER NOT NULL,
    value TEXT,
    formula TEXT,
    display TEXT,
    format TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (sheet_id) REFERENCES sheets(id) ON DELETE CASCADE,
    UNIQUE(sheet_id, row_num, col_num)
);

-- Merged regions table
CREATE TABLE IF NOT EXISTS merged_regions (
    id TEXT PRIMARY KEY,
    sheet_id TEXT NOT NULL,
    start_row INTEGER NOT NULL,
    start_col INTEGER NOT NULL,
    end_row INTEGER NOT NULL,
    end_col INTEGER NOT NULL,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id) ON DELETE CASCADE
);

-- Charts table
CREATE TABLE IF NOT EXISTS charts (
    id TEXT PRIMARY KEY,
    sheet_id TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    chart_type TEXT NOT NULL DEFAULT 'bar',
    data_range TEXT NOT NULL,
    config TEXT,
    position_x INTEGER DEFAULT 0,
    position_y INTEGER DEFAULT 0,
    width INTEGER DEFAULT 400,
    height INTEGER DEFAULT 300,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (sheet_id) REFERENCES sheets(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_workbooks_user ON workbooks(user_id);
CREATE INDEX IF NOT EXISTS idx_sheets_workbook ON sheets(workbook_id);
CREATE INDEX IF NOT EXISTS idx_cells_sheet ON cells(sheet_id);
CREATE INDEX IF NOT EXISTS idx_cells_position ON cells(sheet_id, row_num, col_num);
CREATE INDEX IF NOT EXISTS idx_merged_sheet ON merged_regions(sheet_id);
CREATE INDEX IF NOT EXISTS idx_charts_sheet ON charts(sheet_id);
`;

/**
 * PostgreSQL-specific schema (uses TIMESTAMP instead of TEXT for dates)
 */
export const postgresSchema = `
-- Spreadsheet Application Schema (PostgreSQL)

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Workbooks table
CREATE TABLE IF NOT EXISTS workbooks (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Sheets table
CREATE TABLE IF NOT EXISTS sheets (
    id TEXT PRIMARY KEY,
    workbook_id TEXT NOT NULL REFERENCES workbooks(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    index_num INTEGER NOT NULL DEFAULT 0,
    row_count INTEGER DEFAULT 1000,
    col_count INTEGER DEFAULT 26,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Cells table (sparse storage)
CREATE TABLE IF NOT EXISTS cells (
    id TEXT PRIMARY KEY,
    sheet_id TEXT NOT NULL REFERENCES sheets(id) ON DELETE CASCADE,
    row_num INTEGER NOT NULL,
    col_num INTEGER NOT NULL,
    value TEXT,
    formula TEXT,
    display TEXT,
    format TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(sheet_id, row_num, col_num)
);

-- Merged regions table
CREATE TABLE IF NOT EXISTS merged_regions (
    id TEXT PRIMARY KEY,
    sheet_id TEXT NOT NULL REFERENCES sheets(id) ON DELETE CASCADE,
    start_row INTEGER NOT NULL,
    start_col INTEGER NOT NULL,
    end_row INTEGER NOT NULL,
    end_col INTEGER NOT NULL
);

-- Charts table
CREATE TABLE IF NOT EXISTS charts (
    id TEXT PRIMARY KEY,
    sheet_id TEXT NOT NULL REFERENCES sheets(id) ON DELETE CASCADE,
    title TEXT NOT NULL DEFAULT '',
    chart_type TEXT NOT NULL DEFAULT 'bar',
    data_range TEXT NOT NULL,
    config TEXT,
    position_x INTEGER DEFAULT 0,
    position_y INTEGER DEFAULT 0,
    width INTEGER DEFAULT 400,
    height INTEGER DEFAULT 300,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_workbooks_user ON workbooks(user_id);
CREATE INDEX IF NOT EXISTS idx_sheets_workbook ON sheets(workbook_id);
CREATE INDEX IF NOT EXISTS idx_cells_sheet ON cells(sheet_id);
CREATE INDEX IF NOT EXISTS idx_cells_position ON cells(sheet_id, row_num, col_num);
CREATE INDEX IF NOT EXISTS idx_merged_sheet ON merged_regions(sheet_id);
CREATE INDEX IF NOT EXISTS idx_charts_sheet ON charts(sheet_id);
`;
