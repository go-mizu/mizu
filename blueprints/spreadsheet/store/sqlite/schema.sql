-- SQLite Schema for Spreadsheet (Tile-based, SwanDB-inspired)
-- Optimized for high performance with tile-based cell storage
--
-- DESIGN:
-- 1) Tile storage is the source of truth for the spreadsheet grid.
--    This makes CSV import and UI viewport reads fast.
-- 2) Overlays remain range-based (formats, validations, merges).
-- 3) API-compatible with duckdb/swandb stores.

-- ============================================================
-- Users, sessions
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    password    TEXT NOT NULL,
    avatar      TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    token       TEXT NOT NULL UNIQUE,
    expires_at  DATETIME NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- ============================================================
-- Workbooks, sheets
-- ============================================================

CREATE TABLE IF NOT EXISTS workbooks (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    owner_id    TEXT NOT NULL,
    settings    TEXT DEFAULT '{}',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS sheets (
    id                 TEXT PRIMARY KEY,
    workbook_id        TEXT NOT NULL,
    name               TEXT NOT NULL,
    index_num          INTEGER NOT NULL,
    hidden             INTEGER DEFAULT 0,
    color              TEXT,
    grid_color         TEXT DEFAULT '#E2E8F0',
    frozen_rows        INTEGER DEFAULT 0,
    frozen_cols        INTEGER DEFAULT 0,
    default_row_height INTEGER DEFAULT 21,
    default_col_width  INTEGER DEFAULT 100,
    row_heights        TEXT DEFAULT '{}',
    col_widths         TEXT DEFAULT '{}',
    hidden_rows        TEXT DEFAULT '[]',
    hidden_cols        TEXT DEFAULT '[]',
    content_version    INTEGER DEFAULT 0,
    created_at         DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workbook_id) REFERENCES workbooks(id)
);

CREATE INDEX IF NOT EXISTS idx_sheets_workbook ON sheets(workbook_id);

-- ============================================================
-- Core: tile storage (source of truth for cell data)
-- ============================================================
-- A tile is a fixed-size block of the grid (256 rows x 64 columns).
-- This table replaces the original per-cell "cells" table for storage,
-- but the cells API is maintained through tile encoding/decoding.

CREATE TABLE IF NOT EXISTS sheet_tiles (
    sheet_id      TEXT NOT NULL,
    tile_row      INTEGER NOT NULL,
    tile_col      INTEGER NOT NULL,
    tile_h        INTEGER NOT NULL DEFAULT 256,
    tile_w        INTEGER NOT NULL DEFAULT 64,
    encoding      TEXT NOT NULL DEFAULT 'json_v1',
    values_blob   BLOB NOT NULL,
    formula_blob  BLOB,
    format_blob   BLOB,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (sheet_id, tile_row, tile_col),
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_sheet_tiles_lookup
ON sheet_tiles(sheet_id, tile_row, tile_col);

-- ============================================================
-- Overlays (range-based)
-- ============================================================

CREATE TABLE IF NOT EXISTS merged_regions (
    id          TEXT PRIMARY KEY,
    sheet_id    TEXT NOT NULL,
    start_row   INTEGER NOT NULL,
    start_col   INTEGER NOT NULL,
    end_row     INTEGER NOT NULL,
    end_col     INTEGER NOT NULL,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_merged_regions_sheet ON merged_regions(sheet_id);

CREATE TABLE IF NOT EXISTS format_ranges (
    id          TEXT PRIMARY KEY,
    sheet_id    TEXT NOT NULL,
    start_row   INTEGER NOT NULL,
    start_col   INTEGER NOT NULL,
    end_row     INTEGER NOT NULL,
    end_col     INTEGER NOT NULL,
    format      TEXT NOT NULL,
    priority    INTEGER DEFAULT 0,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_format_ranges_sheet ON format_ranges(sheet_id);

CREATE TABLE IF NOT EXISTS conditional_formats (
    id           TEXT PRIMARY KEY,
    sheet_id     TEXT NOT NULL,
    ranges       TEXT NOT NULL,
    priority     INTEGER NOT NULL,
    format_type  TEXT NOT NULL,
    rule         TEXT NOT NULL,
    format       TEXT NOT NULL,
    stop_if_true INTEGER DEFAULT 0,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_conditional_formats_sheet ON conditional_formats(sheet_id);

CREATE TABLE IF NOT EXISTS data_validations (
    id              TEXT PRIMARY KEY,
    sheet_id        TEXT NOT NULL,
    ranges          TEXT NOT NULL,
    validation_type TEXT NOT NULL,
    operator        TEXT,
    validation_values TEXT,
    allow_blank     INTEGER DEFAULT 1,
    show_dropdown   INTEGER DEFAULT 1,
    show_error      INTEGER DEFAULT 1,
    error_title     TEXT,
    error_message   TEXT,
    error_style     TEXT DEFAULT 'stop',
    show_input      INTEGER DEFAULT 0,
    input_title     TEXT,
    input_message   TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_data_validations_sheet ON data_validations(sheet_id);

-- ============================================================
-- Named ranges
-- ============================================================

CREATE TABLE IF NOT EXISTS named_ranges (
    id          TEXT PRIMARY KEY,
    workbook_id TEXT NOT NULL,
    sheet_id    TEXT,
    name        TEXT NOT NULL,
    range_ref   TEXT NOT NULL,
    comment     TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workbook_id) REFERENCES workbooks(id),
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_named_ranges_workbook ON named_ranges(workbook_id);

-- ============================================================
-- Comments
-- ============================================================

CREATE TABLE IF NOT EXISTS comments (
    id          TEXT PRIMARY KEY,
    sheet_id    TEXT NOT NULL,
    cell_ref    TEXT NOT NULL,
    author_id   TEXT NOT NULL,
    content     TEXT NOT NULL,
    resolved    INTEGER DEFAULT 0,
    resolved_by TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id),
    FOREIGN KEY (author_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_comments_sheet ON comments(sheet_id);

CREATE TABLE IF NOT EXISTS comment_replies (
    id          TEXT PRIMARY KEY,
    comment_id  TEXT NOT NULL,
    author_id   TEXT NOT NULL,
    content     TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (comment_id) REFERENCES comments(id),
    FOREIGN KEY (author_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_comment_replies_comment ON comment_replies(comment_id);

-- ============================================================
-- Charts, pivots, filters
-- ============================================================

CREATE TABLE IF NOT EXISTS charts (
    id          TEXT PRIMARY KEY,
    sheet_id    TEXT NOT NULL,
    name        TEXT,
    chart_type  TEXT NOT NULL,
    position    TEXT NOT NULL,
    size        TEXT NOT NULL,
    data_ranges TEXT NOT NULL,
    title       TEXT,
    subtitle    TEXT,
    legend      TEXT,
    axes        TEXT,
    series      TEXT,
    options     TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_charts_sheet ON charts(sheet_id);

CREATE TABLE IF NOT EXISTS pivot_tables (
    id              TEXT PRIMARY KEY,
    sheet_id        TEXT NOT NULL,
    name            TEXT,
    source_range    TEXT NOT NULL,
    dest_cell       TEXT NOT NULL,
    pivot_rows      TEXT,
    pivot_columns   TEXT,
    pivot_values    TEXT,
    filters         TEXT,
    options         TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_pivot_tables_sheet ON pivot_tables(sheet_id);

CREATE TABLE IF NOT EXISTS auto_filters (
    id          TEXT PRIMARY KEY,
    sheet_id    TEXT NOT NULL,
    range_ref   TEXT NOT NULL,
    columns     TEXT,
    sort_spec   TEXT,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_auto_filters_sheet ON auto_filters(sheet_id);

-- ============================================================
-- Sharing, versions
-- ============================================================

CREATE TABLE IF NOT EXISTS shares (
    id          TEXT PRIMARY KEY,
    workbook_id TEXT NOT NULL,
    user_id     TEXT,
    email       TEXT,
    permission  TEXT NOT NULL,
    link_token  TEXT,
    expires_at  DATETIME,
    created_by  TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workbook_id) REFERENCES workbooks(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_shares_workbook ON shares(workbook_id);
CREATE INDEX IF NOT EXISTS idx_shares_user ON shares(user_id);

CREATE TABLE IF NOT EXISTS versions (
    id          TEXT PRIMARY KEY,
    workbook_id TEXT NOT NULL,
    name        TEXT,
    snapshot    BLOB,
    created_by  TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workbook_id) REFERENCES workbooks(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_versions_workbook ON versions(workbook_id);
