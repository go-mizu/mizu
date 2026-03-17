-- Who posts the most Ask HN questions?
SELECT by, count(*) AS posts
FROM read_parquet('hf://datasets/open-index/hacker-news/data/*/*.parquet')
WHERE type = 'story' AND title LIKE 'Ask HN:%'
GROUP BY by
ORDER BY posts DESC
LIMIT 20;
