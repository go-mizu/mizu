-- Idempotent seed: DELETE + INSERT pattern
-- This file is executed when DB count differs from parquet count

-- Clear existing data for fresh seed
DELETE FROM titles;
DELETE FROM pages;

-- Seed titles table from parquet (dedupe by id, keep first occurrence)
INSERT INTO titles
SELECT DISTINCT ON (id)
  id,
  wikiname,
  in_language,
  title,
  lower(title) AS title_lc
FROM read_parquet('__PARQUET_GLOB__');

-- Seed pages table from parquet (dedupe by id, keep first occurrence)
INSERT INTO pages
SELECT DISTINCT ON (id)
  id,
  wikiname,
  page_id,
  title,
  lower(title) AS title_lc,
  url,
  COALESCE(date_modified, ''),
  in_language,
  COALESCE(text, ''),
  COALESCE(wikidata_id, ''),
  COALESCE(bytes_html, 0),
  COALESCE(has_math, false),
  COALESCE(wikitext, ''),
  COALESCE(version, ''),
  COALESCE(infoboxes::VARCHAR, '[]')
FROM read_parquet('__PARQUET_GLOB__');

-- Update metadata
DELETE FROM meta WHERE k IN ('seeded_at', 'parquet_count', 'parquet_glob');

INSERT INTO meta (k, v) VALUES
  ('seeded_at', cast(now() AS VARCHAR)),
  ('parquet_count', (SELECT cast(count(*) AS VARCHAR) FROM pages)),
  ('parquet_glob', '__PARQUET_GLOB__');
