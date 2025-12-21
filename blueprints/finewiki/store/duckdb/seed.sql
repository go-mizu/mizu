-- Seed titles table for fast search
INSERT OR IGNORE INTO titles
SELECT
  id,
  wikiname,
  in_language,
  title,
  lower(title) AS title_lc
FROM read_parquet('__PARQUET_GLOB__')
WHERE NOT EXISTS (SELECT 1 FROM titles);

-- Seed pages table for fast page retrieval
INSERT OR IGNORE INTO pages
SELECT
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
FROM read_parquet('__PARQUET_GLOB__')
WHERE NOT EXISTS (SELECT 1 FROM pages);

INSERT INTO meta (k, v)
SELECT 'seeded_at', cast(now() AS VARCHAR)
WHERE NOT EXISTS (SELECT 1 FROM meta WHERE k = 'seeded_at');
