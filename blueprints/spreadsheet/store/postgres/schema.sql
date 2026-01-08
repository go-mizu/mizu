-- PostgreSQL Schema for Spreadsheet (Tile-based, SwanDB-inspired)
-- Optimized for high performance with tile-based cell storage

-- DESIGN:
-- 1) Tile storage is the source of truth for the spreadsheet grid.
--    This makes CSV import and UI viewport reads fast.
-- 2) Overlays remain range-based (formats, validations, merges).
-- 3) API-compatible with duckdb/swandb/sqlite stores.

-- Enable btree_gist for exclusion constraints
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- ============================================================
-- Users, sessions
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
    id          VARCHAR(26) PRIMARY KEY,
    email       VARCHAR(255) NOT NULL UNIQUE,
    name        VARCHAR(255) NOT NULL,
    password    VARCHAR(255) NOT NULL,
    avatar      VARCHAR(512),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id          VARCHAR(26) PRIMARY KEY,
    user_id     VARCHAR(26) NOT NULL REFERENCES users(id),
    token       VARCHAR(255) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

-- ============================================================
-- Workbooks, sheets
-- ============================================================

CREATE TABLE IF NOT EXISTS workbooks (
    id          VARCHAR(26) PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    owner_id    VARCHAR(26) NOT NULL REFERENCES users(id),
    settings    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workbooks_owner ON workbooks(owner_id);

CREATE TABLE IF NOT EXISTS sheets (
    id                 VARCHAR(26) PRIMARY KEY,
    workbook_id        VARCHAR(26) NOT NULL REFERENCES workbooks(id),
    name               VARCHAR(255) NOT NULL,
    index_num          INTEGER NOT NULL,
    hidden             BOOLEAN DEFAULT FALSE,
    color              VARCHAR(50),
    grid_color         VARCHAR(50) DEFAULT '#E2E8F0',
    frozen_rows        INTEGER DEFAULT 0,
    frozen_cols        INTEGER DEFAULT 0,
    default_row_height INTEGER DEFAULT 21,
    default_col_width  INTEGER DEFAULT 100,
    row_heights        JSONB DEFAULT '{}',
    col_widths         JSONB DEFAULT '{}',
    hidden_rows        JSONB DEFAULT '[]',
    hidden_cols        JSONB DEFAULT '[]',
    content_version    INTEGER DEFAULT 0,
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    updated_at         TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sheets_workbook ON sheets(workbook_id);

-- ============================================================
-- Core: tile storage (source of truth for cell data)
-- ============================================================
-- A tile is a fixed-size block of the grid (256 rows x 64 columns).
-- This table replaces the original per-cell "cells" table for storage,
-- but the cells API is maintained through tile encoding/decoding.

CREATE TABLE IF NOT EXISTS sheet_tiles (
    sheet_id      VARCHAR(26) NOT NULL REFERENCES sheets(id),
    tile_row      INTEGER NOT NULL,
    tile_col      INTEGER NOT NULL,
    tile_h        INTEGER NOT NULL DEFAULT 256,
    tile_w        INTEGER NOT NULL DEFAULT 64,
    encoding      VARCHAR(20) NOT NULL DEFAULT 'json_v1',
    values_blob   BYTEA NOT NULL,
    formula_blob  BYTEA,
    format_blob   BYTEA,
    updated_at    TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (sheet_id, tile_row, tile_col)
);

CREATE INDEX IF NOT EXISTS idx_sheet_tiles_lookup
ON sheet_tiles(sheet_id, tile_row, tile_col);

-- ============================================================
-- Overlays (range-based)
-- ============================================================

CREATE TABLE IF NOT EXISTS merged_regions (
    id          VARCHAR(26) PRIMARY KEY,
    sheet_id    VARCHAR(26) NOT NULL REFERENCES sheets(id),
    start_row   INTEGER NOT NULL,
    start_col   INTEGER NOT NULL,
    end_row     INTEGER NOT NULL,
    end_col     INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_merged_regions_sheet ON merged_regions(sheet_id);

CREATE TABLE IF NOT EXISTS format_ranges (
    id          VARCHAR(26) PRIMARY KEY,
    sheet_id    VARCHAR(26) NOT NULL REFERENCES sheets(id),
    start_row   INTEGER NOT NULL,
    start_col   INTEGER NOT NULL,
    end_row     INTEGER NOT NULL,
    end_col     INTEGER NOT NULL,
    format      JSONB NOT NULL,
    priority    INTEGER DEFAULT 0,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_format_ranges_sheet ON format_ranges(sheet_id);

CREATE TABLE IF NOT EXISTS conditional_formats (
    id           VARCHAR(26) PRIMARY KEY,
    sheet_id     VARCHAR(26) NOT NULL REFERENCES sheets(id),
    ranges       JSONB NOT NULL,
    priority     INTEGER NOT NULL,
    format_type  VARCHAR(50) NOT NULL,
    rule         JSONB NOT NULL,
    format       JSONB NOT NULL,
    stop_if_true BOOLEAN DEFAULT FALSE,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conditional_formats_sheet ON conditional_formats(sheet_id);

CREATE TABLE IF NOT EXISTS data_validations (
    id                VARCHAR(26) PRIMARY KEY,
    sheet_id          VARCHAR(26) NOT NULL REFERENCES sheets(id),
    ranges            JSONB NOT NULL,
    validation_type   VARCHAR(50) NOT NULL,
    operator          VARCHAR(20),
    validation_values JSONB,
    allow_blank       BOOLEAN DEFAULT TRUE,
    show_dropdown     BOOLEAN DEFAULT TRUE,
    show_error        BOOLEAN DEFAULT TRUE,
    error_title       VARCHAR(255),
    error_message     TEXT,
    error_style       VARCHAR(20) DEFAULT 'stop',
    show_input        BOOLEAN DEFAULT FALSE,
    input_title       VARCHAR(255),
    input_message     TEXT,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_data_validations_sheet ON data_validations(sheet_id);

-- ============================================================
-- Named ranges
-- ============================================================

CREATE TABLE IF NOT EXISTS named_ranges (
    id          VARCHAR(26) PRIMARY KEY,
    workbook_id VARCHAR(26) NOT NULL REFERENCES workbooks(id),
    sheet_id    VARCHAR(26) REFERENCES sheets(id),
    name        VARCHAR(255) NOT NULL,
    range_ref   VARCHAR(255) NOT NULL,
    comment     TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_named_ranges_workbook ON named_ranges(workbook_id);

-- ============================================================
-- Comments
-- ============================================================

CREATE TABLE IF NOT EXISTS comments (
    id          VARCHAR(26) PRIMARY KEY,
    sheet_id    VARCHAR(26) NOT NULL REFERENCES sheets(id),
    cell_ref    VARCHAR(50) NOT NULL,
    author_id   VARCHAR(26) NOT NULL REFERENCES users(id),
    content     TEXT NOT NULL,
    resolved    BOOLEAN DEFAULT FALSE,
    resolved_by VARCHAR(26),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_comments_sheet ON comments(sheet_id);

CREATE TABLE IF NOT EXISTS comment_replies (
    id          VARCHAR(26) PRIMARY KEY,
    comment_id  VARCHAR(26) NOT NULL REFERENCES comments(id),
    author_id   VARCHAR(26) NOT NULL REFERENCES users(id),
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_comment_replies_comment ON comment_replies(comment_id);

-- ============================================================
-- Charts, pivots, filters
-- ============================================================

CREATE TABLE IF NOT EXISTS charts (
    id          VARCHAR(26) PRIMARY KEY,
    sheet_id    VARCHAR(26) NOT NULL REFERENCES sheets(id),
    name        VARCHAR(255),
    chart_type  VARCHAR(50) NOT NULL,
    position    JSONB NOT NULL,
    size        JSONB NOT NULL,
    data_ranges JSONB NOT NULL,
    title       JSONB,
    subtitle    JSONB,
    legend      JSONB,
    axes        JSONB,
    series      JSONB,
    options     JSONB,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_charts_sheet ON charts(sheet_id);

CREATE TABLE IF NOT EXISTS pivot_tables (
    id              VARCHAR(26) PRIMARY KEY,
    sheet_id        VARCHAR(26) NOT NULL REFERENCES sheets(id),
    name            VARCHAR(255),
    source_range    VARCHAR(255) NOT NULL,
    dest_cell       VARCHAR(50) NOT NULL,
    pivot_rows      JSONB,
    pivot_columns   JSONB,
    pivot_values    JSONB,
    filters         JSONB,
    options         JSONB,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pivot_tables_sheet ON pivot_tables(sheet_id);

CREATE TABLE IF NOT EXISTS auto_filters (
    id          VARCHAR(26) PRIMARY KEY,
    sheet_id    VARCHAR(26) NOT NULL REFERENCES sheets(id),
    range_ref   VARCHAR(255) NOT NULL,
    columns     JSONB,
    sort_spec   JSONB
);

CREATE INDEX IF NOT EXISTS idx_auto_filters_sheet ON auto_filters(sheet_id);

-- ============================================================
-- Sharing, versions
-- ============================================================

CREATE TABLE IF NOT EXISTS shares (
    id          VARCHAR(26) PRIMARY KEY,
    workbook_id VARCHAR(26) NOT NULL REFERENCES workbooks(id),
    user_id     VARCHAR(26) REFERENCES users(id),
    email       VARCHAR(255),
    permission  VARCHAR(20) NOT NULL,
    link_token  VARCHAR(255),
    expires_at  TIMESTAMPTZ,
    created_by  VARCHAR(26) NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shares_workbook ON shares(workbook_id);
CREATE INDEX IF NOT EXISTS idx_shares_user ON shares(user_id);

CREATE TABLE IF NOT EXISTS versions (
    id          VARCHAR(26) PRIMARY KEY,
    workbook_id VARCHAR(26) NOT NULL REFERENCES workbooks(id),
    name        VARCHAR(255),
    snapshot    BYTEA,
    created_by  VARCHAR(26) NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_versions_workbook ON versions(workbook_id);

-- BRIN index for time-based queries on versions
CREATE INDEX IF NOT EXISTS idx_versions_created_brin ON versions USING BRIN(created_at);
