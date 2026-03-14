-- Query the stats.csv directly from Hugging Face
SELECT * FROM read_csv_auto('hf://datasets/open-index/hacker-news/stats.csv')
ORDER BY year, month;
