#!/usr/bin/env python3
# /// script
# requires-python = ">=3.11"
# dependencies = ["duckdb", "pandas", "numpy"]
# ///

import duckdb

conn = duckdb.connect()

# Score distribution: what does a "typical" HN story look like?
# type=1 is story (type is stored as TINYINT: 1=story, 2=comment, 3=poll, 4=pollopt, 5=job)
df = conn.sql("""
    SELECT
        percentile_disc(0.50) WITHIN GROUP (ORDER BY score) AS p50,
        percentile_disc(0.90) WITHIN GROUP (ORDER BY score) AS p90,
        percentile_disc(0.99) WITHIN GROUP (ORDER BY score) AS p99,
        percentile_disc(0.999) WITHIN GROUP (ORDER BY score) AS p999
    FROM read_parquet('hf://datasets/open-index/hacker-news/data/2010/*.parquet')
    WHERE type = 1
""").df()
print("Score percentiles (2010 stories):")
print(df)
