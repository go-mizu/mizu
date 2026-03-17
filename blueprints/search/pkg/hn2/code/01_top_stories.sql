-- Top 20 highest-scored stories of all time
SELECT id, title, by, score, url, time
FROM read_parquet('hf://datasets/open-index/hacker-news/data/*/*.parquet')
WHERE type = 'story' AND title != ''
ORDER BY score DESC
LIMIT 20;
