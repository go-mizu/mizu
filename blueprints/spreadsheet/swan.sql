-- schema.sql
-- Spreadsheet schema (DuckDB oriented, simple + performant)
--
-- DESIGN:
-- 1) Tile storage is the source of truth for the spreadsheet grid.
--    This makes CSV import and UI viewport reads fast.
-- 2) Overlays remain range-based (formats, validations, merges).
-- 3) Optional: if the user imports a CSV as a "table", store that region as a real DuckDB table
--    (row-oriented input, columnar storage internally), and render it into the grid via tiles.
--
-- Compared to your original schema:
-- - Original stored one row per cell (cells table). Large imports caused millions of inserts + index updates.
-- - New stores many cells per tile row (sheet_tiles). Imports become hundreds/thousands of rows.
-- - Per-cell JSON disappears from the hot path. Tiles are binary blobs decoded by the app.
-- - Overlays (formats/validations/merges) stay in small range tables.

-- ============================================================
-- Users, sessions
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
    id            VARCHAR PRIMARY KEY,
    email         VARCHAR NOT NULL UNIQUE,
    name          VARCHAR NOT NULL,
    password_hash VARCHAR NOT NULL,
    avatar_url    VARCHAR,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id          VARCHAR PRIMARY KEY,
    user_id     VARCHAR NOT NULL,
    token       VARCHAR NOT NULL UNIQUE,
    expires_at  TIMESTAMP NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- ============================================================
-- Workbooks, sheets
-- ============================================================

CREATE TABLE IF NOT EXISTS workbooks (
    id          VARCHAR PRIMARY KEY,
    name        VARCHAR NOT NULL,
    owner_id    VARCHAR NOT NULL,
    settings    JSON DEFAULT '{}',
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS sheets (
    id                 VARCHAR PRIMARY KEY,
    workbook_id        VARCHAR NOT NULL,
    name               VARCHAR NOT NULL,
    index_num          INTEGER NOT NULL,
    hidden             BOOLEAN DEFAULT FALSE,

    frozen_rows        INTEGER DEFAULT 0,
    frozen_cols        INTEGER DEFAULT 0,
    default_row_height INTEGER DEFAULT 21,
    default_col_width  INTEGER DEFAULT 100,

    -- Sheet-level sparse maps (not hot import path)
    row_heights        JSON DEFAULT '{}',
    col_widths         JSON DEFAULT '{}',
    hidden_rows        JSON DEFAULT '[]',
    hidden_cols        JSON DEFAULT '[]',

    -- Bump when content changes (tile writes)
    content_version    BIGINT DEFAULT 0,

    created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (workbook_id) REFERENCES workbooks(id),
    UNIQUE(workbook_id, index_num),
    UNIQUE(workbook_id, name)
);

CREATE INDEX IF NOT EXISTS idx_sheets_workbook ON sheets(workbook_id);

-- ============================================================
-- Core: tile storage (source of truth)
-- ============================================================
--
-- A tile is a fixed-size block of the grid (example 256x64).
-- This table replaces the original per-cell "cells" table.
--
-- values_blob encoding suggestion (app-level):
-- - per tile, store column vectors:
--   - null bitmap per column
--   - value kind per column or per cell (depending on design)
--   - typed vectors for numbers/ints/bools
--   - string references via tile-local dictionary or sheet dictionary
--
-- formula_blob is optional:
-- - many cells have no formula; keeping formulas separate saves space.

CREATE TABLE IF NOT EXISTS sheet_tiles (
    sheet_id      VARCHAR NOT NULL,
    tile_row      INTEGER NOT NULL,
    tile_col      INTEGER NOT NULL,

    tile_h        INTEGER NOT NULL, -- constant chosen by app, e.g. 256
    tile_w        INTEGER NOT NULL, -- constant chosen by app, e.g. 64

    encoding      VARCHAR NOT NULL, -- e.g. 'tile_v1'
    values_blob   BLOB NOT NULL,
    formula_blob  BLOB,

    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (sheet_id, tile_row, tile_col),
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_sheet_tiles_lookup
ON sheet_tiles(sheet_id, tile_row, tile_col);

-- Optional: dictionaries to compress repeated strings/formulas across the sheet.
-- If you want maximum simplicity, you can omit these and store strings inline per tile.

CREATE TABLE IF NOT EXISTS sheet_string_dict (
    sheet_id   VARCHAR NOT NULL,
    dict_id    BIGINT NOT NULL,
    value      VARCHAR NOT NULL,
    PRIMARY KEY (sheet_id, dict_id),
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE TABLE IF NOT EXISTS sheet_formula_dict (
    sheet_id   VARCHAR NOT NULL,
    dict_id    BIGINT NOT NULL,
    formula    VARCHAR NOT NULL,
    PRIMARY KEY (sheet_id, dict_id),
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

-- ============================================================
-- Overlays (range-based)
-- ============================================================

CREATE TABLE IF NOT EXISTS merged_regions (
    id          VARCHAR PRIMARY KEY,
    sheet_id    VARCHAR NOT NULL,
    start_row   INTEGER NOT NULL,
    start_col   INTEGER NOT NULL,
    end_row     INTEGER NOT NULL,
    end_col     INTEGER NOT NULL,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_merged_regions_sheet ON merged_regions(sheet_id);

CREATE TABLE IF NOT EXISTS format_ranges (
    id          VARCHAR PRIMARY KEY,
    sheet_id    VARCHAR NOT NULL,
    start_row   INTEGER NOT NULL,
    start_col   INTEGER NOT NULL,
    end_row     INTEGER NOT NULL,
    end_col     INTEGER NOT NULL,
    format      JSON NOT NULL,
    priority    INTEGER DEFAULT 0,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_format_ranges_sheet ON format_ranges(sheet_id);

CREATE TABLE IF NOT EXISTS conditional_formats (
    id           VARCHAR PRIMARY KEY,
    sheet_id     VARCHAR NOT NULL,
    ranges       JSON NOT NULL,
    priority     INTEGER NOT NULL,
    format_type  VARCHAR NOT NULL,
    rule         JSON NOT NULL,
    format       JSON NOT NULL,
    stop_if_true BOOLEAN DEFAULT FALSE,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_conditional_formats_sheet ON conditional_formats(sheet_id);

CREATE TABLE IF NOT EXISTS data_validations (
    id              VARCHAR PRIMARY KEY,
    sheet_id        VARCHAR NOT NULL,
    ranges          JSON NOT NULL,
    validation_type VARCHAR NOT NULL,
    operator        VARCHAR,
    values          JSON,
    allow_blank     BOOLEAN DEFAULT TRUE,
    show_dropdown   BOOLEAN DEFAULT TRUE,
    show_error      BOOLEAN DEFAULT TRUE,
    error_title     VARCHAR,
    error_message   TEXT,
    error_style     VARCHAR DEFAULT 'stop',
    show_input      BOOLEAN DEFAULT FALSE,
    input_title     VARCHAR,
    input_message   TEXT,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_data_validations_sheet ON data_validations(sheet_id);

-- ============================================================
-- Named ranges
-- ============================================================

CREATE TABLE IF NOT EXISTS named_ranges (
    id          VARCHAR PRIMARY KEY,
    workbook_id VARCHAR NOT NULL,
    sheet_id    VARCHAR,
    name        VARCHAR NOT NULL,
    start_row   INTEGER,
    start_col   INTEGER,
    end_row     INTEGER,
    end_col     INTEGER,
    comment     TEXT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workbook_id) REFERENCES workbooks(id),
    FOREIGN KEY (sheet_id) REFERENCES sheets(id),
    UNIQUE(workbook_id, name)
);

CREATE INDEX IF NOT EXISTS idx_named_ranges_workbook ON named_ranges(workbook_id);

-- ============================================================
-- Comments
-- ============================================================

CREATE TABLE IF NOT EXISTS comments (
    id          VARCHAR PRIMARY KEY,
    sheet_id    VARCHAR NOT NULL,
    row_num     INTEGER NOT NULL,
    col_num     INTEGER NOT NULL,
    author_id   VARCHAR NOT NULL,
    content     TEXT NOT NULL,
    resolved    BOOLEAN DEFAULT FALSE,
    resolved_by VARCHAR,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id),
    FOREIGN KEY (author_id) REFERENCES users(id),
    FOREIGN KEY (resolved_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_comments_sheet_cell ON comments(sheet_id, row_num, col_num);

CREATE TABLE IF NOT EXISTS comment_replies (
    id          VARCHAR PRIMARY KEY,
    comment_id  VARCHAR NOT NULL,
    author_id   VARCHAR NOT NULL,
    content     TEXT NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (comment_id) REFERENCES comments(id),
    FOREIGN KEY (author_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_comment_replies_comment ON comment_replies(comment_id);

-- ============================================================
-- Charts, pivots, filters (metadata)
-- ============================================================

CREATE TABLE IF NOT EXISTS charts (
    id          VARCHAR PRIMARY KEY,
    sheet_id    VARCHAR NOT NULL,
    name        VARCHAR,
    chart_type  VARCHAR NOT NULL,
    position    JSON NOT NULL,
    size        JSON NOT NULL,
    data_ranges JSON NOT NULL,
    options     JSON,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_charts_sheet ON charts(sheet_id);

CREATE TABLE IF NOT EXISTS pivot_tables (
    id              VARCHAR PRIMARY KEY,
    sheet_id        VARCHAR NOT NULL,
    name            VARCHAR,
    source_range    VARCHAR NOT NULL,
    dest_cell       VARCHAR NOT NULL,
    config          JSON,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_pivot_tables_sheet ON pivot_tables(sheet_id);

CREATE TABLE IF NOT EXISTS auto_filters (
    id          VARCHAR PRIMARY KEY,
    sheet_id    VARCHAR NOT NULL,
    start_row   INTEGER NOT NULL,
    start_col   INTEGER NOT NULL,
    end_row     INTEGER NOT NULL,
    end_col     INTEGER NOT NULL,
    spec        JSON,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_auto_filters_sheet ON auto_filters(sheet_id);

-- ============================================================
-- Sharing, versions
-- ============================================================

CREATE TABLE IF NOT EXISTS shares (
    id          VARCHAR PRIMARY KEY,
    workbook_id VARCHAR NOT NULL,
    user_id     VARCHAR,
    email       VARCHAR,
    permission  VARCHAR NOT NULL,
    link_token  VARCHAR,
    expires_at  TIMESTAMP,
    created_by  VARCHAR NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workbook_id) REFERENCES workbooks(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_shares_workbook ON shares(workbook_id);
CREATE INDEX IF NOT EXISTS idx_shares_user ON shares(user_id);

CREATE TABLE IF NOT EXISTS versions (
    id          VARCHAR PRIMARY KEY,
    workbook_id VARCHAR NOT NULL,
    name        VARCHAR,
    snapshot    BLOB,
    created_by  VARCHAR NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workbook_id) REFERENCES workbooks(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_versions_workbook ON versions(workbook_id);

-- ============================================================
-- Optional: DuckDB-native "table sheets" for analytics (simple alternative to column segments)
-- ============================================================
--
-- This is the key idea: leverage DuckDB for what it is best at.
--
-- Instead of building and maintaining your own column segments, you can:
-- - store a table region as an actual DuckDB table (typed columns)
-- - keep the grid editor in tiles
-- - render the table into tiles for display, and write edits back to the table
--
-- This stays conceptually simple:
-- - Grid storage: tiles
-- - Analytics storage: DuckDB tables (real columns)
--
-- The registry below only tracks the existence and mapping of those tables.

CREATE TABLE IF NOT EXISTS sheet_tables (
    id            VARCHAR PRIMARY KEY,
    sheet_id      VARCHAR NOT NULL,
    name          VARCHAR NOT NULL,

    -- Region where the table is displayed in the grid
    start_row     INTEGER NOT NULL,
    start_col     INTEGER NOT NULL,

    -- The physical DuckDB table name that holds the data.
    -- Example: 'tbl_<sheet_id>_<table_id>'
    duckdb_table  VARCHAR NOT NULL UNIQUE,

    -- Column schema and mapping metadata
    schema_json   JSON NOT NULL,

    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (sheet_id) REFERENCES sheets(id),
    UNIQUE(sheet_id, name)
);

CREATE INDEX IF NOT EXISTS idx_sheet_tables_sheet ON sheet_tables(sheet_id);

-- End.
