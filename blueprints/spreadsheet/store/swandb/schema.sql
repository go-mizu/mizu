-- schema.sql
-- SwanDB: Spreadsheet schema (DuckDB oriented, tile-based storage)
--
-- DESIGN:
-- 1) Tile storage is the source of truth for the spreadsheet grid.
--    This makes CSV import and UI viewport reads fast.
-- 2) Overlays remain range-based (formats, validations, merges).
-- 3) API-compatible with duckdb store while using tile-based cells.

-- ============================================================
-- Users, sessions
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
    id            VARCHAR PRIMARY KEY,
    email         VARCHAR NOT NULL UNIQUE,
    name          VARCHAR NOT NULL,
    password      VARCHAR NOT NULL,
    avatar        VARCHAR,
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
    color              VARCHAR,
    grid_color         VARCHAR DEFAULT '#E2E8F0',

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

    FOREIGN KEY (workbook_id) REFERENCES workbooks(id)
);

CREATE INDEX IF NOT EXISTS idx_sheets_workbook ON sheets(workbook_id);

-- ============================================================
-- Core: tile storage (source of truth for cell data)
-- ============================================================
--
-- A tile is a fixed-size block of the grid (256 rows x 64 columns).
-- This table replaces the original per-cell "cells" table for storage,
-- but the cells API is maintained through tile encoding/decoding.

CREATE TABLE IF NOT EXISTS sheet_tiles (
    sheet_id      VARCHAR NOT NULL,
    tile_row      INTEGER NOT NULL,
    tile_col      INTEGER NOT NULL,

    tile_h        INTEGER NOT NULL DEFAULT 256,
    tile_w        INTEGER NOT NULL DEFAULT 64,

    encoding      VARCHAR NOT NULL DEFAULT 'json_v1',
    values_blob   BLOB NOT NULL,
    formula_blob  BLOB,
    format_blob   BLOB,

    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (sheet_id, tile_row, tile_col),
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_sheet_tiles_lookup
ON sheet_tiles(sheet_id, tile_row, tile_col);

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
    range_ref   VARCHAR NOT NULL,
    comment     TEXT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workbook_id) REFERENCES workbooks(id),
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_named_ranges_workbook ON named_ranges(workbook_id);

-- ============================================================
-- Comments
-- ============================================================

CREATE TABLE IF NOT EXISTS comments (
    id          VARCHAR PRIMARY KEY,
    sheet_id    VARCHAR NOT NULL,
    cell_ref    VARCHAR NOT NULL,
    author_id   VARCHAR NOT NULL,
    content     TEXT NOT NULL,
    resolved    BOOLEAN DEFAULT FALSE,
    resolved_by VARCHAR,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id),
    FOREIGN KEY (author_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_comments_sheet ON comments(sheet_id);

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
-- Charts, pivots, filters
-- ============================================================

CREATE TABLE IF NOT EXISTS charts (
    id          VARCHAR PRIMARY KEY,
    sheet_id    VARCHAR NOT NULL,
    name        VARCHAR,
    chart_type  VARCHAR NOT NULL,
    position    JSON NOT NULL,
    size        JSON NOT NULL,
    data_ranges JSON NOT NULL,
    title       JSON,
    subtitle    JSON,
    legend      JSON,
    axes        JSON,
    series      JSON,
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
    rows            JSON,
    columns         JSON,
    values          JSON,
    filters         JSON,
    options         JSON,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

CREATE INDEX IF NOT EXISTS idx_pivot_tables_sheet ON pivot_tables(sheet_id);

CREATE TABLE IF NOT EXISTS auto_filters (
    id          VARCHAR PRIMARY KEY,
    sheet_id    VARCHAR NOT NULL,
    range_ref   VARCHAR NOT NULL,
    columns     JSON,
    sort_spec   JSON,
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
    FOREIGN KEY (user_id) REFERENCES users(id)
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

-- End.
