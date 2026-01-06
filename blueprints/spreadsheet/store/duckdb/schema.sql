-- Spreadsheet Database Schema

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id          VARCHAR PRIMARY KEY,
    email       VARCHAR NOT NULL UNIQUE,
    name        VARCHAR NOT NULL,
    password    VARCHAR NOT NULL,
    avatar      VARCHAR,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id          VARCHAR PRIMARY KEY,
    user_id     VARCHAR NOT NULL,
    token       VARCHAR NOT NULL UNIQUE,
    expires_at  TIMESTAMP NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Workbooks table
CREATE TABLE IF NOT EXISTS workbooks (
    id          VARCHAR PRIMARY KEY,
    name        VARCHAR NOT NULL,
    owner_id    VARCHAR NOT NULL,
    settings    JSON,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(id)
);

-- Sheets table
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
    row_heights        JSON,
    col_widths         JSON,
    hidden_rows        JSON,
    hidden_cols        JSON,
    created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workbook_id) REFERENCES workbooks(id)
);

-- Cells table (sparse storage - only non-empty cells)
CREATE TABLE IF NOT EXISTS cells (
    id          VARCHAR PRIMARY KEY,
    sheet_id    VARCHAR NOT NULL,
    row_num     INTEGER NOT NULL,
    col_num     INTEGER NOT NULL,
    value       JSON,
    formula     VARCHAR,
    display     VARCHAR,
    cell_type   VARCHAR,
    format      JSON,
    hyperlink   JSON,
    note        TEXT,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id),
    UNIQUE(sheet_id, row_num, col_num)
);

-- Index for efficient cell lookups
CREATE INDEX IF NOT EXISTS idx_cells_sheet_position ON cells(sheet_id, row_num, col_num);

-- Merged regions table
CREATE TABLE IF NOT EXISTS merged_regions (
    id          VARCHAR PRIMARY KEY,
    sheet_id    VARCHAR NOT NULL,
    start_row   INTEGER NOT NULL,
    start_col   INTEGER NOT NULL,
    end_row     INTEGER NOT NULL,
    end_col     INTEGER NOT NULL,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

-- Named ranges table
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

-- Conditional formats table
CREATE TABLE IF NOT EXISTS conditional_formats (
    id           VARCHAR PRIMARY KEY,
    sheet_id     VARCHAR NOT NULL,
    ranges       JSON NOT NULL,
    priority     INTEGER NOT NULL,
    format_type  VARCHAR NOT NULL,
    rule         JSON NOT NULL,
    format       JSON NOT NULL,
    stop_if_true BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

-- Data validations table
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
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

-- Charts table
CREATE TABLE IF NOT EXISTS charts (
    id          VARCHAR PRIMARY KEY,
    sheet_id    VARCHAR NOT NULL,
    name        VARCHAR,
    chart_type  VARCHAR NOT NULL,
    position    JSON NOT NULL,
    size        JSON NOT NULL,
    data_ranges JSON NOT NULL,
    title       JSON,
    legend      JSON,
    axes        JSON,
    series      JSON,
    options     JSON,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

-- Pivot tables table
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

-- Comments table
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

-- Comment replies table
CREATE TABLE IF NOT EXISTS comment_replies (
    id          VARCHAR PRIMARY KEY,
    comment_id  VARCHAR NOT NULL,
    author_id   VARCHAR NOT NULL,
    content     TEXT NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (comment_id) REFERENCES comments(id),
    FOREIGN KEY (author_id) REFERENCES users(id)
);

-- Auto filters table
CREATE TABLE IF NOT EXISTS auto_filters (
    id          VARCHAR PRIMARY KEY,
    sheet_id    VARCHAR NOT NULL,
    range_ref   VARCHAR NOT NULL,
    columns     JSON,
    sort_spec   JSON,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
);

-- Sharing table
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

-- Version history table
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
