-- Top linked domains, year over year
SELECT
    extract(year FROM time) AS year,
    regexp_extract(url, 'https?://([^/]+)', 1) AS domain,
    count(*) AS stories
FROM read_parquet('hf://datasets/open-index/hacker-news/data/*/*.parquet')
WHERE type = 'story' AND url != ''
GROUP BY year, domain
QUALIFY row_number() OVER (PARTITION BY year ORDER BY stories DESC) <= 5
ORDER BY year, stories DESC;
