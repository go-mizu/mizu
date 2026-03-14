-- Track how often a topic appears on HN over time
SELECT
    extract(year FROM time) AS year,
    count(*) AS mentions
FROM read_parquet('hf://datasets/open-index/hacker-news/data/*/*.parquet')
WHERE type = 'story' AND lower(title) LIKE '%rust%'
GROUP BY year
ORDER BY year;
