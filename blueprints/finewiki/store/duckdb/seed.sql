CREATE OR REPLACE VIEW finewiki AS
SELECT
  id,
  wikiname,
  in_language,
  title
FROM read_parquet('__PARQUET_GLOB__');

INSERT INTO titles
SELECT
  id,
  wikiname,
  in_language,
  title,
  lower(title) AS title_lc
FROM finewiki
WHERE NOT EXISTS (SELECT 1 FROM titles);

INSERT INTO meta (k, v)
SELECT 'seeded_at', cast(now() AS VARCHAR)
WHERE NOT EXISTS (SELECT 1 FROM meta WHERE k = 'seeded_at');
