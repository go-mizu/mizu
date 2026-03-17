-- Monthly submission volume for a specific year
-- NOTE: countIf() is ClickHouse syntax; DuckDB uses count(*) FILTER (WHERE ...)
SELECT
    strftime(time, '%Y-%m') AS month,
    count(*) AS items,
    count(*) FILTER (WHERE type = 'story') AS stories,
    count(*) FILTER (WHERE type = 'comment') AS comments
FROM read_parquet('hf://datasets/open-index/hacker-news/data/2024/*.parquet')
GROUP BY month
ORDER BY month;
