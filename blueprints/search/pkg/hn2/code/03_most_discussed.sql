-- Most discussed stories by total comment count
SELECT id, title, by, score, descendants AS comments, url
FROM read_parquet('hf://datasets/open-index/hacker-news/data/2025/*.parquet')
WHERE type = 'story' AND descendants > 0
ORDER BY descendants DESC
LIMIT 20;
