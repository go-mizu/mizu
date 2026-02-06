package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// Analyzer runs all analytics queries via DuckDB on parquet files.
type Analyzer struct {
	db          *sql.DB
	parquetGlob string
	startTime   time.Time
}

// NewAnalyzer opens an in-memory DuckDB and creates a view over the parquet files.
func NewAnalyzer(parquetDir string) (*Analyzer, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("opening duckdb: %w", err)
	}

	glob := filepath.Join(parquetDir, "*.parquet")
	a := &Analyzer{db: db, parquetGlob: glob}

	if err := a.createView(); err != nil {
		db.Close()
		return nil, err
	}

	return a, nil
}

func (a *Analyzer) createView() error {
	_, err := a.db.Exec(fmt.Sprintf(`
		CREATE VIEW docs AS
		SELECT *,
			LENGTH(text) AS text_len,
			LENGTH(text) - LENGTH(REPLACE(text, ' ', '')) + 1 AS word_count
		FROM read_parquet('%s')
	`, a.parquetGlob))
	return err
}

// Close releases the DuckDB connection.
func (a *Analyzer) Close() error { return a.db.Close() }

// ProgressFunc reports progress.
type ProgressFunc func(step, total int, label string)

// Run executes all analytics and generates the report sections.
func (a *Analyzer) Run(ctx context.Context, progress ProgressFunc) (*Report, error) {
	a.startTime = time.Now()
	totalSteps := 23 // number of query groups
	step := 0

	report := func(label string) {
		step++
		if progress != nil {
			progress(step, totalSteps, label)
		}
	}

	r := &Report{}

	// Overview
	report("Overview statistics")
	var err error
	r.Summary, err = a.querySummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("summary: %w", err)
	}

	// Text Statistics (Charts 1-12)
	report("Text length distribution")
	r.TextLengthDist, err = a.queryHistogram(ctx, "text_len", []float64{100, 500, 1000, 5000, 10000, 50000, 100000})
	if err != nil {
		return nil, fmt.Errorf("text length: %w", err)
	}

	report("Word count distribution")
	r.WordCountDist, err = a.queryHistogram(ctx, "word_count", []float64{10, 50, 100, 500, 1000, 5000, 10000})
	if err != nil {
		return nil, fmt.Errorf("word count: %w", err)
	}

	report("Sentence & line counts")
	r.SentenceCountDist, err = a.queryExprHistogram(ctx,
		"LENGTH(text) - LENGTH(REPLACE(REPLACE(REPLACE(text, '.', ''), '!', ''), '?', ''))",
		[]float64{1, 5, 10, 25, 50, 100, 500})
	if err != nil {
		return nil, fmt.Errorf("sentence count: %w", err)
	}

	r.LineCountDist, err = a.queryExprHistogram(ctx,
		"LENGTH(text) - LENGTH(REPLACE(text, chr(10), '')) + 1",
		[]float64{1, 5, 10, 25, 50, 100, 500})
	if err != nil {
		return nil, fmt.Errorf("line count: %w", err)
	}

	report("Text length percentiles")
	r.TextPercentiles, err = a.queryPercentiles(ctx, "text_len")
	if err != nil {
		return nil, fmt.Errorf("percentiles: %w", err)
	}

	report("Short document analysis")
	r.ShortDocDist, err = a.querySQL(ctx, `
		SELECT
			CASE
				WHEN text_len < 10 THEN '<10 chars'
				WHEN text_len < 50 THEN '10-49 chars'
				WHEN text_len < 100 THEN '50-99 chars'
				ELSE '>=100 chars'
			END AS label,
			COUNT(*) AS cnt
		FROM docs
		GROUP BY label ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("short docs: %w", err)
	}

	report("Top words and bigrams")
	r.TopWords, err = a.querySQL(ctx, `
		WITH sample AS (SELECT text FROM docs USING SAMPLE 20000),
		     words AS (SELECT UNNEST(STRING_SPLIT(LOWER(LEFT(text, 1000)), ' ')) AS word FROM sample)
		SELECT word AS label, COUNT(*) AS cnt
		FROM words WHERE LENGTH(word) BETWEEN 2 AND 20
		GROUP BY word ORDER BY cnt DESC LIMIT 30
	`)
	if err != nil {
		return nil, fmt.Errorf("top words: %w", err)
	}

	r.TopBigrams, err = a.querySQL(ctx, `
		WITH sample AS (SELECT text FROM docs USING SAMPLE 10000),
		     words AS (SELECT UNNEST(STRING_SPLIT(LOWER(LEFT(text, 500)), ' ')) AS word FROM sample),
		     numbered AS (SELECT word, ROW_NUMBER() OVER () AS rn FROM words),
		     bigrams AS (SELECT a.word || ' ' || b.word AS bigram FROM numbered a JOIN numbered b ON b.rn = a.rn + 1)
		SELECT bigram AS label, COUNT(*) AS cnt
		FROM bigrams WHERE LENGTH(bigram) BETWEEN 4 AND 40
		GROUP BY bigram ORDER BY cnt DESC LIMIT 30
	`)
	if err != nil {
		return nil, fmt.Errorf("bigrams: %w", err)
	}

	// Temporal (Charts 13-23)
	report("Documents per year")
	r.DocsPerYear, err = a.querySQL(ctx, `
		SELECT CAST(EXTRACT(YEAR FROM TRY_CAST(date AS TIMESTAMP)) AS VARCHAR) AS label,
		       COUNT(*) AS cnt
		FROM docs WHERE date IS NOT NULL
		GROUP BY label ORDER BY label
	`)
	if err != nil {
		return nil, fmt.Errorf("docs per year: %w", err)
	}

	report("Monthly trend")
	r.MonthlyTrend, err = a.querySQL(ctx, `
		SELECT STRFTIME(TRY_CAST(date AS TIMESTAMP), '%Y-%m') AS label,
		       COUNT(*) AS cnt
		FROM docs WHERE date IS NOT NULL
		GROUP BY label ORDER BY label
	`)
	if err != nil {
		return nil, fmt.Errorf("monthly: %w", err)
	}

	report("Dumps and temporal details")
	r.TopDumps, err = a.querySQL(ctx, `
		SELECT dump AS label, COUNT(*) AS cnt
		FROM docs GROUP BY dump ORDER BY cnt DESC LIMIT 30
	`)
	if err != nil {
		return nil, fmt.Errorf("dumps: %w", err)
	}

	r.HourDist, err = a.querySQL(ctx, `
		SELECT STRFTIME(TRY_CAST(date AS TIMESTAMP), '%H') AS label,
		       COUNT(*) AS cnt
		FROM docs WHERE date IS NOT NULL
		GROUP BY label ORDER BY label
	`)
	if err != nil {
		return nil, fmt.Errorf("hour: %w", err)
	}

	r.DOWDist, err = a.querySQL(ctx, `
		SELECT DAYNAME(TRY_CAST(date AS TIMESTAMP)) AS label,
		       COUNT(*) AS cnt
		FROM docs WHERE date IS NOT NULL
		GROUP BY label
		ORDER BY CASE label
			WHEN 'Monday' THEN 1 WHEN 'Tuesday' THEN 2 WHEN 'Wednesday' THEN 3
			WHEN 'Thursday' THEN 4 WHEN 'Friday' THEN 5 WHEN 'Saturday' THEN 6
			WHEN 'Sunday' THEN 7 END
	`)
	if err != nil {
		return nil, fmt.Errorf("dow: %w", err)
	}

	r.QuarterlyDist, err = a.querySQL(ctx, `
		SELECT EXTRACT(YEAR FROM TRY_CAST(date AS TIMESTAMP)) || '-Q' ||
		       CAST(CEIL(EXTRACT(MONTH FROM TRY_CAST(date AS TIMESTAMP)) / 3.0) AS INT) AS label,
		       COUNT(*) AS cnt
		FROM docs WHERE date IS NOT NULL
		GROUP BY label ORDER BY label
	`)
	if err != nil {
		return nil, fmt.Errorf("quarterly: %w", err)
	}

	report("Date range & dump timeline")
	r.DateRange, err = a.querySQL(ctx, `
		SELECT
			MIN(CAST(date AS VARCHAR)) AS earliest,
			MAX(CAST(date AS VARCHAR)) AS latest,
			CAST(COUNT(DISTINCT EXTRACT(YEAR FROM TRY_CAST(date AS TIMESTAMP))) AS VARCHAR) AS unique_years,
			CAST(COUNT(DISTINCT STRFTIME(TRY_CAST(date AS TIMESTAMP), '%Y-%m')) AS VARCHAR) AS unique_months,
			CAST(COUNT(DISTINCT dump) AS VARCHAR) AS unique_dumps
		FROM docs
	`)
	if err != nil {
		return nil, fmt.Errorf("date range: %w", err)
	}

	r.DumpTimeline, err = a.querySQL(ctx, `
		SELECT dump AS label, COUNT(*) AS cnt
		FROM docs GROUP BY dump ORDER BY dump
	`)
	if err != nil {
		return nil, fmt.Errorf("dump timeline: %w", err)
	}

	// Domain Analysis (Charts 24-35)
	report("Top domains")
	r.TopDomains, err = a.querySQL(ctx, `
		WITH hosts AS (
			SELECT REGEXP_REPLACE(REGEXP_EXTRACT(url, '://([^/]+)', 1), '^www\.', '') AS domain
			FROM docs WHERE url IS NOT NULL
		)
		SELECT domain AS label, COUNT(*) AS cnt
		FROM hosts WHERE domain IS NOT NULL
		GROUP BY domain ORDER BY cnt DESC LIMIT 30
	`)
	if err != nil {
		return nil, fmt.Errorf("domains: %w", err)
	}

	report("TLD and protocol distribution")
	r.TLDDist, err = a.querySQL(ctx, `
		WITH hosts AS (
			SELECT REGEXP_EXTRACT(url, '://([^/]+)', 1) AS host FROM docs
		)
		SELECT
			CASE
				WHEN host LIKE '%.com.vn' THEN '.com.vn'
				WHEN host LIKE '%.edu.vn' THEN '.edu.vn'
				WHEN host LIKE '%.gov.vn' THEN '.gov.vn'
				WHEN host LIKE '%.org.vn' THEN '.org.vn'
				WHEN host LIKE '%.net.vn' THEN '.net.vn'
				ELSE '.' || REGEXP_EXTRACT(host, '\.([a-z]+)$', 1)
			END AS label,
			COUNT(*) AS cnt
		FROM hosts WHERE host IS NOT NULL
		GROUP BY label ORDER BY cnt DESC LIMIT 10
	`)
	if err != nil {
		return nil, fmt.Errorf("tld: %w", err)
	}

	r.ProtocolDist, err = a.querySQL(ctx, `
		SELECT CASE WHEN url LIKE 'https%' THEN 'HTTPS' ELSE 'HTTP' END AS label,
		       COUNT(*) AS cnt
		FROM docs GROUP BY label ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("protocol: %w", err)
	}

	report("URL structure analysis")
	r.PathDepthDist, err = a.queryExprHistogram(ctx,
		"LENGTH(REGEXP_EXTRACT(url, '://[^/]+(.*)', 1)) - LENGTH(REPLACE(REGEXP_EXTRACT(url, '://[^/]+(.*)', 1), '/', ''))",
		[]float64{1, 2, 3, 4, 5, 6, 8, 10})
	if err != nil {
		return nil, fmt.Errorf("path depth: %w", err)
	}

	r.URLLengthDist, err = a.queryHistogram(ctx, "LENGTH(url)", []float64{20, 40, 60, 80, 100, 150, 200, 300})
	if err != nil {
		return nil, fmt.Errorf("url length: %w", err)
	}

	r.SubdomainDist, err = a.querySQL(ctx, `
		WITH hosts AS (
			SELECT REGEXP_EXTRACT(url, '://([^/]+)', 1) AS host FROM docs
		)
		SELECT
			CASE
				WHEN host LIKE 'www.%' THEN 'www'
				WHEN host NOT LIKE '%.%.%' THEN 'no subdomain'
				ELSE 'other subdomain'
			END AS label,
			COUNT(*) AS cnt
		FROM hosts WHERE host IS NOT NULL
		GROUP BY label ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("subdomain: %w", err)
	}

	r.VNDomains, err = a.querySQL(ctx, `
		WITH hosts AS (
			SELECT REGEXP_REPLACE(REGEXP_EXTRACT(url, '://([^/]+)', 1), '^www\.', '') AS domain
			FROM docs WHERE url LIKE '%.vn%'
		)
		SELECT domain AS label, COUNT(*) AS cnt
		FROM hosts WHERE domain IS NOT NULL
		GROUP BY domain ORDER BY cnt DESC LIMIT 20
	`)
	if err != nil {
		return nil, fmt.Errorf("vn domains: %w", err)
	}

	report("Domain concentration & text length by domain")
	r.DomainConcentration, err = a.querySQL(ctx, `
		WITH hosts AS (
			SELECT REGEXP_REPLACE(REGEXP_EXTRACT(url, '://([^/]+)', 1), '^www\.', '') AS domain
			FROM docs WHERE url IS NOT NULL
		),
		domain_counts AS (
			SELECT domain, COUNT(*) AS cnt FROM hosts GROUP BY domain ORDER BY cnt DESC
		),
		totals AS (SELECT SUM(cnt) AS total FROM domain_counts)
		SELECT
			CAST((SELECT COUNT(*) FROM domain_counts) AS VARCHAR) AS unique_domains,
			CAST(ROUND(SUM(CASE WHEN rn <= 10 THEN cnt ELSE 0 END) * 100.0 / MAX(total), 1) AS VARCHAR) AS top10_pct,
			CAST(ROUND(SUM(CASE WHEN rn <= 100 THEN cnt ELSE 0 END) * 100.0 / MAX(total), 1) AS VARCHAR) AS top100_pct
		FROM (SELECT *, ROW_NUMBER() OVER (ORDER BY cnt DESC) AS rn FROM domain_counts), totals
	`)
	if err != nil {
		return nil, fmt.Errorf("concentration: %w", err)
	}

	r.DomainAvgTextLen, err = a.querySQL(ctx, `
		WITH hosts AS (
			SELECT REGEXP_REPLACE(REGEXP_EXTRACT(url, '://([^/]+)', 1), '^www\.', '') AS domain,
			       text_len
			FROM docs WHERE url IS NOT NULL
		)
		SELECT domain AS label, CAST(ROUND(AVG(text_len)) AS BIGINT) AS cnt
		FROM hosts
		GROUP BY domain HAVING COUNT(*) >= 50
		ORDER BY cnt DESC LIMIT 20
	`)
	if err != nil {
		return nil, fmt.Errorf("domain avg: %w", err)
	}

	r.QueryParamDist, err = a.querySQL(ctx, `
		SELECT CASE WHEN url LIKE '%?%' THEN 'has query' ELSE 'no query' END AS label,
		       COUNT(*) AS cnt
		FROM docs GROUP BY label ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query param: %w", err)
	}

	r.NewDomainsPerYear, err = a.querySQL(ctx, `
		WITH hosts AS (
			SELECT REGEXP_REPLACE(REGEXP_EXTRACT(url, '://([^/]+)', 1), '^www\.', '') AS domain,
			       EXTRACT(YEAR FROM TRY_CAST(date AS TIMESTAMP)) AS yr
			FROM docs WHERE url IS NOT NULL AND date IS NOT NULL
		),
		first_seen AS (
			SELECT domain, MIN(yr) AS first_year FROM hosts GROUP BY domain
		)
		SELECT CAST(first_year AS VARCHAR) AS label, COUNT(*) AS cnt
		FROM first_seen GROUP BY first_year ORDER BY first_year
	`)
	if err != nil {
		return nil, fmt.Errorf("new domains: %w", err)
	}

	r.DomainDiversityByYear, err = a.querySQL(ctx, `
		WITH hosts AS (
			SELECT REGEXP_REPLACE(REGEXP_EXTRACT(url, '://([^/]+)', 1), '^www\.', '') AS domain,
			       CAST(EXTRACT(YEAR FROM TRY_CAST(date AS TIMESTAMP)) AS VARCHAR) AS yr
			FROM docs WHERE url IS NOT NULL AND date IS NOT NULL
		)
		SELECT yr AS label, CAST(COUNT(DISTINCT domain) AS BIGINT) AS cnt
		FROM hosts GROUP BY yr ORDER BY yr
	`)
	if err != nil {
		return nil, fmt.Errorf("domain diversity: %w", err)
	}

	// Quality (Charts 36-45)
	report("Language score distribution")
	r.LangScoreDist, err = a.queryExprHistogram(ctx, "language_score",
		[]float64{0.5, 0.7, 0.8, 0.9, 0.95, 0.99, 0.999, 1.0})
	if err != nil {
		return nil, fmt.Errorf("lang score: %w", err)
	}

	r.LangScorePercentiles, err = a.queryPercentiles(ctx, "language_score")
	if err != nil {
		return nil, fmt.Errorf("lang percentiles: %w", err)
	}

	report("Cluster size distribution")
	r.ClusterSizeDist, err = a.queryExprHistogram(ctx, "minhash_cluster_size",
		[]float64{1, 2, 5, 10, 20, 50, 100, 500})
	if err != nil {
		return nil, fmt.Errorf("cluster: %w", err)
	}

	r.ClusterCategories, err = a.querySQL(ctx, `
		SELECT
			CASE
				WHEN minhash_cluster_size <= 1 THEN 'Unique (1)'
				WHEN minhash_cluster_size <= 5 THEN 'Small (2-5)'
				WHEN minhash_cluster_size <= 20 THEN 'Medium (6-20)'
				WHEN minhash_cluster_size <= 100 THEN 'Large (21-100)'
				ELSE 'Very Large (100+)'
			END AS label,
			COUNT(*) AS cnt
		FROM docs GROUP BY label ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("cluster cats: %w", err)
	}

	report("Quality bands & correlations")
	r.QualityBands, err = a.querySQL(ctx, `
		SELECT
			CASE
				WHEN language_score < 0.8 THEN '<0.80'
				WHEN language_score < 0.9 THEN '0.80-0.90'
				WHEN language_score < 0.95 THEN '0.90-0.95'
				WHEN language_score < 0.99 THEN '0.95-0.99'
				ELSE '0.99-1.00'
			END AS label,
			COUNT(*) AS cnt
		FROM docs GROUP BY label ORDER BY label
	`)
	if err != nil {
		return nil, fmt.Errorf("quality bands: %w", err)
	}

	r.ScoreVsTextLen, err = a.querySQL(ctx, `
		SELECT
			CASE
				WHEN language_score < 0.9 THEN '<0.9'
				WHEN language_score < 0.95 THEN '0.9-0.95'
				WHEN language_score < 0.99 THEN '0.95-0.99'
				ELSE '>=0.99'
			END AS band,
			COUNT(*) AS cnt,
			CAST(ROUND(AVG(text_len)) AS BIGINT) AS avg_len,
			CAST(ROUND(MEDIAN(text_len)) AS BIGINT) AS median_len
		FROM docs GROUP BY band ORDER BY band
	`)
	if err != nil {
		return nil, fmt.Errorf("score vs len: %w", err)
	}

	r.ScoreVsCluster, err = a.querySQL(ctx, `
		SELECT
			ROUND(AVG(language_score), 6) AS avg_score,
			ROUND(STDDEV(language_score), 6) AS std_score,
			ROUND(MEDIAN(language_score), 6) AS med_score,
			ROUND(AVG(minhash_cluster_size), 1) AS avg_cluster,
			ROUND(STDDEV(minhash_cluster_size), 1) AS std_cluster,
			ROUND(MEDIAN(minhash_cluster_size), 1) AS med_cluster
		FROM docs
	`)
	if err != nil {
		return nil, fmt.Errorf("score vs cluster: %w", err)
	}

	report("Top languages in top_langs")
	r.TopLangsField, err = a.querySQL(ctx, `
		SELECT CASE WHEN top_langs = '' OR top_langs = '{}' THEN 'empty' ELSE 'populated' END AS label,
		       COUNT(*) AS cnt
		FROM docs GROUP BY label ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("top langs: %w", err)
	}

	r.AvgClusterByDump, err = a.querySQL(ctx, `
		SELECT dump AS label, CAST(ROUND(AVG(minhash_cluster_size), 1) AS DOUBLE) AS avg_size, COUNT(*) AS cnt
		FROM docs GROUP BY dump HAVING COUNT(*) >= 50
		ORDER BY cnt DESC LIMIT 20
	`)
	if err != nil {
		return nil, fmt.Errorf("cluster by dump: %w", err)
	}

	// Vietnamese Content (Charts 46-55)
	report("Vietnamese tone distribution")
	r.ToneDist, err = a.querySQL(ctx, `
		WITH sample AS (SELECT LEFT(text, 2000) AS t FROM docs USING SAMPLE 20000),
		     chars AS (SELECT UNNEST(STRING_SPLIT(t, '')) AS ch FROM sample),
		     tones AS (
		         SELECT CASE
		             WHEN ch IN ('á','ắ','ấ','é','ế','í','ó','ố','ớ','ú','ứ','ý') THEN 'sắc (rising)'
		             WHEN ch IN ('à','ằ','ầ','è','ề','ì','ò','ồ','ờ','ù','ừ','ỳ') THEN 'huyền (falling)'
		             WHEN ch IN ('ả','ẳ','ẩ','ẻ','ể','ỉ','ỏ','ổ','ở','ủ','ử','ỷ') THEN 'hỏi (questioning)'
		             WHEN ch IN ('ã','ẵ','ẫ','ẽ','ễ','ĩ','õ','ỗ','ỡ','ũ','ữ','ỹ') THEN 'ngã (tumbling)'
		             WHEN ch IN ('ạ','ặ','ậ','ẹ','ệ','ị','ọ','ộ','ợ','ụ','ự','ỵ') THEN 'nặng (heavy)'
		             ELSE NULL
		         END AS tone FROM chars
		     )
		SELECT tone AS label, COUNT(*) AS cnt FROM tones WHERE tone IS NOT NULL
		GROUP BY tone ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("tones: %w", err)
	}

	report("Vietnamese diacritics and vowels")
	r.DiacriticFreq, err = a.querySQL(ctx, `
		WITH sample AS (SELECT LEFT(text, 2000) AS t FROM docs USING SAMPLE 20000),
		     chars AS (SELECT UNNEST(STRING_SPLIT(t, '')) AS ch FROM sample)
		SELECT ch AS label, COUNT(*) AS cnt
		FROM chars
		WHERE ch IN ('á','à','ả','ã','ạ','ắ','ằ','ẳ','ẵ','ặ','ấ','ầ','ẩ','ẫ','ậ',
		             'é','è','ẻ','ẽ','ẹ','ế','ề','ể','ễ','ệ',
		             'í','ì','ỉ','ĩ','ị','ó','ò','ỏ','õ','ọ','ố','ồ','ổ','ỗ','ộ',
		             'ớ','ờ','ở','ỡ','ợ','ú','ù','ủ','ũ','ụ','ứ','ừ','ử','ữ','ự',
		             'ý','ỳ','ỷ','ỹ','ỵ')
		GROUP BY ch ORDER BY cnt DESC LIMIT 30
	`)
	if err != nil {
		return nil, fmt.Errorf("diacritics: %w", err)
	}

	r.VowelFreq, err = a.querySQL(ctx, `
		WITH sample AS (SELECT LEFT(text, 2000) AS t FROM docs USING SAMPLE 20000),
		     chars AS (SELECT LOWER(UNNEST(STRING_SPLIT(t, ''))) AS ch FROM sample)
		SELECT
			CASE
				WHEN ch IN ('a','á','à','ả','ã','ạ') THEN 'a'
				WHEN ch IN ('ă','ắ','ằ','ẳ','ẵ','ặ') THEN 'ă'
				WHEN ch IN ('â','ấ','ầ','ẩ','ẫ','ậ') THEN 'â'
				WHEN ch IN ('e','é','è','ẻ','ẽ','ẹ') THEN 'e'
				WHEN ch IN ('ê','ế','ề','ể','ễ','ệ') THEN 'ê'
				WHEN ch IN ('i','í','ì','ỉ','ĩ','ị') THEN 'i'
				WHEN ch IN ('o','ó','ò','ỏ','õ','ọ') THEN 'o'
				WHEN ch IN ('ô','ố','ồ','ổ','ỗ','ộ') THEN 'ô'
				WHEN ch IN ('ơ','ớ','ờ','ở','ỡ','ợ') THEN 'ơ'
				WHEN ch IN ('u','ú','ù','ủ','ũ','ụ') THEN 'u'
				WHEN ch IN ('ư','ứ','ừ','ử','ữ','ự') THEN 'ư'
				WHEN ch IN ('y','ý','ỳ','ỷ','ỹ','ỵ') THEN 'y'
				ELSE NULL
			END AS label,
			COUNT(*) AS cnt
		FROM chars WHERE ch IS NOT NULL
		GROUP BY label HAVING label IS NOT NULL
		ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("vowels: %w", err)
	}

	report("Stop words and punctuation")
	r.StopWordFreq, err = a.querySQL(ctx, `
		WITH sample AS (SELECT text FROM docs USING SAMPLE 20000),
		     words AS (SELECT UNNEST(STRING_SPLIT(LOWER(LEFT(text, 1000)), ' ')) AS word FROM sample)
		SELECT word AS label, COUNT(*) AS cnt
		FROM words
		WHERE word IN ('của','và','là','các','cho','không','có','được','trong','này',
		               'với','một','đã','những','từ','người','tại','để','theo','về',
		               'khi','đến','cũng','trên','như','năm','còn','sau','vào','nên',
		               'thì','đó','bị','mà','ra','sẽ','rất','hay','nhưng','nhiều')
		GROUP BY word ORDER BY cnt DESC LIMIT 30
	`)
	if err != nil {
		return nil, fmt.Errorf("stop words: %w", err)
	}

	r.PunctuationDist, err = a.querySQL(ctx, `
		SELECT
			CASE ch
				WHEN '.' THEN 'Period (.)'
				WHEN '?' THEN 'Question (?)'
				WHEN '!' THEN 'Exclamation (!)'
			END AS label,
			COUNT(*) AS cnt
		FROM (
			SELECT UNNEST(STRING_SPLIT(LEFT(text, 5000), '')) AS ch
			FROM docs USING SAMPLE 20000
		) sub
		WHERE ch IN ('.', '?', '!')
		GROUP BY label ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("punctuation: %w", err)
	}

	report("Content type and cleanliness")
	r.ContentTypeDist, err = a.querySQL(ctx, `
		SELECT
			CASE
				WHEN url ILIKE '%news%' OR url ILIKE '%bao%' OR url ILIKE '%tin-tuc%' OR url ILIKE '%/tin/%' THEN 'News'
				WHEN url ILIKE '%forum%' OR url ILIKE '%dien-dan%' OR url ILIKE '%thread%' THEN 'Forum'
				WHEN url ILIKE '%blog%' THEN 'Blog'
				WHEN url ILIKE '%shop%' OR url ILIKE '%product%' OR url ILIKE '%san-pham%' THEN 'E-commerce'
				WHEN url ILIKE '%.gov.vn%' THEN 'Government'
				WHEN url ILIKE '%.edu.vn%' THEN 'Education'
				WHEN url ILIKE '%wiki%' THEN 'Wiki/Reference'
				ELSE 'Other'
			END AS label,
			COUNT(*) AS cnt
		FROM docs GROUP BY label ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("content type: %w", err)
	}

	r.BoilerplateDist, err = a.querySQL(ctx, `
		SELECT
			CASE
				WHEN LEFT(LOWER(text), 500) LIKE '%<html%' OR LEFT(LOWER(text), 500) LIKE '%<div%'
				     OR LEFT(LOWER(text), 500) LIKE '%<script%' THEN 'Contains HTML'
				WHEN LEFT(LOWER(text), 500) LIKE '%function(%' OR LEFT(LOWER(text), 500) LIKE '%var %'
				     THEN 'Contains JS/Code'
				ELSE 'Clean text'
			END AS label,
			COUNT(*) AS cnt
		FROM docs GROUP BY label ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("boilerplate: %w", err)
	}

	report("Numeric density & character types")
	r.NumericDensityDist, err = a.querySQL(ctx, `
		WITH sample AS (
			SELECT text_len,
			       LENGTH(text) - LENGTH(REGEXP_REPLACE(text, '[0-9]', '', 'g')) AS digit_count
			FROM docs USING SAMPLE 50000
		)
		SELECT
			CASE
				WHEN text_len = 0 THEN 'N/A'
				WHEN CAST(digit_count AS DOUBLE) / text_len < 0.01 THEN '<1%'
				WHEN CAST(digit_count AS DOUBLE) / text_len < 0.02 THEN '1-2%'
				WHEN CAST(digit_count AS DOUBLE) / text_len < 0.05 THEN '2-5%'
				WHEN CAST(digit_count AS DOUBLE) / text_len < 0.1 THEN '5-10%'
				ELSE '10%+'
			END AS label,
			COUNT(*) AS cnt
		FROM sample GROUP BY label ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("numeric: %w", err)
	}

	r.CharTypeDist, err = a.querySQL(ctx, `
		WITH sample AS (SELECT LEFT(text, 1000) AS t FROM docs USING SAMPLE 10000),
		     chars AS (SELECT UNNEST(STRING_SPLIT(t, '')) AS ch FROM sample)
		SELECT
			CASE
				WHEN ch ~ '[a-zA-Z]' THEN 'ASCII Letter'
				WHEN ch ~ '[0-9]' THEN 'Digit'
				WHEN ch ~ '\s' THEN 'Whitespace'
				WHEN UNICODE(ch) BETWEEN 192 AND 687 OR UNICODE(ch) BETWEEN 7680 AND 7935 THEN 'Vietnamese Diacritic'
				ELSE 'Punctuation/Other'
			END AS label,
			COUNT(*) AS cnt
		FROM chars WHERE ch != ''
		GROUP BY label ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("char types: %w", err)
	}

	r.VNCharDensity, err = a.querySQL(ctx, `
		WITH sample AS (
			SELECT text_len,
			       LENGTH(REGEXP_REPLACE(LEFT(text, 2000), '[^\x{00C0}-\x{024F}\x{1E00}-\x{1EFF}đĐ]', '', 'g')) AS vn_chars
			FROM docs USING SAMPLE 30000
		)
		SELECT
			CASE
				WHEN text_len = 0 THEN 'N/A'
				WHEN CAST(vn_chars AS DOUBLE) / LEAST(text_len, 2000) >= 0.1 THEN 'Vietnamese-heavy (>10%)'
				WHEN CAST(vn_chars AS DOUBLE) / LEAST(text_len, 2000) >= 0.01 THEN 'Some Vietnamese (1-10%)'
				ELSE 'Minimal Vietnamese (<1%)'
			END AS label,
			COUNT(*) AS cnt
		FROM sample GROUP BY label ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("vn char density: %w", err)
	}

	r.AvgComplexityByDump, err = a.querySQL(ctx, `
		WITH sample AS (
			SELECT dump,
			       LENGTH(REGEXP_REPLACE(LEFT(text, 2000), '[^\x{00C0}-\x{024F}\x{1E00}-\x{1EFF}]', '', 'g')) AS diac_count,
			       LEAST(text_len, 2000) AS sample_len
			FROM docs USING SAMPLE 50000
		)
		SELECT dump AS label,
		       ROUND(AVG(CAST(diac_count AS DOUBLE) / NULLIF(sample_len, 0)), 4) AS avg_ratio,
		       COUNT(*) AS cnt
		FROM sample WHERE sample_len > 0
		GROUP BY dump HAVING COUNT(*) >= 50
		ORDER BY cnt DESC LIMIT 20
	`)
	if err != nil {
		return nil, fmt.Errorf("complexity: %w", err)
	}

	r.Duration = time.Since(a.startTime)
	return r, nil
}

// querySQL runs a SQL query returning label+cnt rows.
func (a *Analyzer) querySQL(ctx context.Context, sql string) ([]LabelCount, error) {
	rows, err := a.db.QueryContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var results []LabelCount

	for rows.Next() {
		if len(cols) == 2 {
			var lc LabelCount
			if err := rows.Scan(&lc.Label, &lc.Count); err != nil {
				return nil, err
			}
			results = append(results, lc)
		} else {
			// Multiple columns - scan as strings
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return nil, err
			}
			lc := LabelCount{Extra: make(map[string]any)}
			for i, col := range cols {
				lc.Extra[col] = vals[i]
			}
			if v, ok := lc.Extra[cols[0]]; ok {
				lc.Label = fmt.Sprintf("%v", v)
			}
			results = append(results, lc)
		}
	}
	return results, rows.Err()
}

// querySummary returns overview stats.
func (a *Analyzer) querySummary(ctx context.Context) (SummaryStats, error) {
	var s SummaryStats
	row := a.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(text_len), 0),
			COALESCE(SUM(word_count), 0),
			COUNT(DISTINCT REGEXP_REPLACE(REGEXP_EXTRACT(url, '://([^/]+)', 1), '^www\.', '')),
			COALESCE(MIN(CAST(date AS VARCHAR)), ''),
			COALESCE(MAX(CAST(date AS VARCHAR)), ''),
			ROUND(AVG(language_score), 6),
			ROUND(AVG(text_len), 0),
			ROUND(MEDIAN(text_len), 0),
			ROUND(AVG(minhash_cluster_size), 1)
		FROM docs
	`)
	var earliest, latest string
	if err := row.Scan(&s.TotalDocs, &s.TotalChars, &s.TotalWords, &s.UniqueDomains,
		&earliest, &latest, &s.AvgLangScore, &s.AvgTextLength, &s.MedianTextLen, &s.AvgClusterSize); err != nil {
		return s, err
	}
	// Trim to date only
	if len(earliest) >= 10 {
		earliest = earliest[:10]
	}
	if len(latest) >= 10 {
		latest = latest[:10]
	}
	s.DateRange = earliest + " to " + latest
	return s, nil
}

// queryHistogram builds a histogram by bucketing a column.
func (a *Analyzer) queryHistogram(ctx context.Context, column string, boundaries []float64) ([]LabelCount, error) {
	return a.queryExprHistogram(ctx, column, boundaries)
}

// queryExprHistogram builds a histogram from an arbitrary expression.
func (a *Analyzer) queryExprHistogram(ctx context.Context, expr string, boundaries []float64) ([]LabelCount, error) {
	// Build CASE expression for bucketing
	var cases []string
	for i, b := range boundaries {
		var label string
		if i == 0 {
			label = fmt.Sprintf("0-%s", formatBound(b))
			cases = append(cases, fmt.Sprintf("WHEN val < %v THEN '%s'", b, label))
		} else {
			label = fmt.Sprintf("%s-%s", formatBound(boundaries[i-1]), formatBound(b))
			cases = append(cases, fmt.Sprintf("WHEN val < %v THEN '%s'", b, label))
		}
	}
	overflowLabel := fmt.Sprintf("%s+", formatBound(boundaries[len(boundaries)-1]))
	cases = append(cases, fmt.Sprintf("ELSE '%s'", overflowLabel))

	sql := fmt.Sprintf(`
		WITH vals AS (SELECT (%s) AS val FROM docs)
		SELECT CASE %s END AS label, COUNT(*) AS cnt
		FROM vals WHERE val IS NOT NULL
		GROUP BY label
		ORDER BY MIN(val)
	`, expr, "\n"+joinStrings(cases, "\n"))

	return a.querySQL(ctx, sql)
}

// queryPercentiles returns percentile statistics.
func (a *Analyzer) queryPercentiles(ctx context.Context, column string) ([]LabelCount, error) {
	sql := fmt.Sprintf(`
		SELECT
			ROUND(PERCENTILE_CONT(0.01) WITHIN GROUP (ORDER BY %[1]s), 4) AS p1,
			ROUND(PERCENTILE_CONT(0.05) WITHIN GROUP (ORDER BY %[1]s), 4) AS p5,
			ROUND(PERCENTILE_CONT(0.10) WITHIN GROUP (ORDER BY %[1]s), 4) AS p10,
			ROUND(PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY %[1]s), 4) AS p25,
			ROUND(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY %[1]s), 4) AS p50,
			ROUND(PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY %[1]s), 4) AS p75,
			ROUND(PERCENTILE_CONT(0.90) WITHIN GROUP (ORDER BY %[1]s), 4) AS p90,
			ROUND(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY %[1]s), 4) AS p95,
			ROUND(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY %[1]s), 4) AS p99,
			ROUND(AVG(%[1]s), 4) AS mean,
			ROUND(STDDEV(%[1]s), 4) AS stddev,
			MIN(%[1]s) AS min_val,
			MAX(%[1]s) AS max_val
		FROM docs
	`, column)
	return a.querySQL(ctx, sql)
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
