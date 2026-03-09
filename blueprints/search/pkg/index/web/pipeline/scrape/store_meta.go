package scrape

import (
	"database/sql"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

const metaFile = "scrape_meta.duckdb"

// startBackground launches a goroutine that refreshes domain stats every 60s
// or immediately when s.triggerCh is signaled.
func startBackground(s *store) {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		refreshDomains(s)
		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				refreshDomains(s)
			case <-s.triggerCh:
				refreshDomains(s)
			}
		}
	}()
}

// loadFromMeta reads the meta DuckDB on startup to populate the in-memory
// domain cache. This is fast because it reads a single small file.
func loadFromMeta(s *store) {
	metaPath := filepath.Join(s.dataDir, metaFile)
	if _, err := os.Stat(metaPath); err != nil {
		return
	}

	db, err := sql.Open("duckdb", metaPath+"?access_mode=read_only")
	if err != nil {
		log.Printf("[scrape-meta] open meta DB for read: %v", err)
		return
	}
	defer db.Close()

	rows, err := db.Query(`SELECT domain, pages, success, failed, links,
		html_bytes, md_bytes, index_bytes, has_md, has_index,
		last_crawl::VARCHAR
		FROM domain_stats ORDER BY pages DESC`)
	if err != nil {
		log.Printf("[scrape-meta] query meta DB: %v", err)
		return
	}
	defer rows.Close()

	var domains []Domain
	for rows.Next() {
		var d Domain
		var lastCrawl sql.NullString
		if err := rows.Scan(&d.Domain, &d.Pages, &d.Success, &d.Failed, &d.Links,
			&d.HtmlBytes, &d.MdBytes, &d.IndexBytes, &d.HasMD, &d.HasIndex,
			&lastCrawl); err != nil {
			log.Printf("[scrape-meta] scan row: %v", err)
			continue
		}
		if lastCrawl.Valid && len(lastCrawl.String) >= 19 {
			if t, err := time.Parse("2006-01-02 15:04:05", lastCrawl.String[:19]); err == nil {
				d.LastCrawl = t
			}
		}
		domains = append(domains, d)
	}

	s.mu.Lock()
	s.domains = domains
	s.ready = true
	s.mu.Unlock()

	log.Printf("[scrape-meta] loaded %d domains from meta DB", len(domains))
}

// refreshDomains scans the data directory, computes stats for every domain
// in parallel, persists results to the meta DuckDB, and updates the in-memory cache.
func refreshDomains(s *store) {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("[scrape-meta] read data dir: %v", err)
		return
	}

	type work struct {
		name      string
		domainDir string
	}
	var jobs []work
	for _, e := range entries {
		if !e.IsDir() || e.Name() == metaFile {
			continue
		}
		domainDir := filepath.Join(s.dataDir, e.Name())
		shards, _ := filepath.Glob(filepath.Join(domainDir, "results", "results_*.duckdb"))
		if len(shards) == 0 {
			continue
		}
		jobs = append(jobs, work{name: e.Name(), domainDir: domainDir})
	}

	domains := make([]Domain, len(jobs))
	var wg sync.WaitGroup
	for i, j := range jobs {
		wg.Add(1)
		go func(idx int, j work) {
			defer wg.Done()
			domains[idx] = computeDomainStats(j.domainDir, j.name)
		}(i, j)
	}
	wg.Wait()

	if domains == nil {
		domains = []Domain{}
	}

	saveToMeta(s.dataDir, domains)

	s.mu.Lock()
	s.domains = domains
	s.ready = true
	s.mu.Unlock()
}

// computeDomainStats computes aggregate stats for a single domain directory.
// It queries all result shards for page/success/failed/html_bytes/last_crawl,
// counts links, and measures markdown and FTS directory sizes.
func computeDomainStats(domainDir, name string) Domain {
	d := Domain{Domain: name}

	shards, _ := filepath.Glob(filepath.Join(domainDir, "results", "results_*.duckdb"))
	if len(shards) == 0 {
		return d
	}

	// Aggregate stats across all shards.
	type shardResult struct {
		pages     int64
		success   int64
		failed    int64
		links     int64
		htmlBytes int64
		mdBytes   int64
		lastCrawl time.Time
	}

	results := make([]shardResult, len(shards))
	var wg sync.WaitGroup
	for i, shard := range shards {
		wg.Add(1)
		go func(idx int, path string) {
			defer wg.Done()
			var sr shardResult
			db, err := sql.Open("duckdb", path+"?access_mode=read_only")
			if err != nil {
				// DuckDB cannot open the same file with a different configuration
				// from an existing connection in the same process. During active
				// crawls, writer connections are read-write, so retry without
				// read_only for stats reads.
				db, err = sql.Open("duckdb", path)
				if err != nil {
					log.Printf("[scrape-meta] open shard %s: %v", path, err)
					return
				}
			}
			defer db.Close()

			var lastCrawl sql.NullString
			row := db.QueryRow(`SELECT
				count(*),
				count(*) FILTER (WHERE status_code >= 200 AND status_code < 400),
				count(*) FILTER (WHERE status_code >= 400 OR error != ''),
				COALESCE(SUM(content_length), 0),
				max(crawled_at)::VARCHAR
			FROM pages`)
			if row.Scan(&sr.pages, &sr.success, &sr.failed, &sr.htmlBytes, &lastCrawl) == nil {
				if lastCrawl.Valid && len(lastCrawl.String) >= 19 {
					if t, err := time.Parse("2006-01-02 15:04:05", lastCrawl.String[:19]); err == nil {
						sr.lastCrawl = t
					}
				}
			}

			// Count markdown bytes from in-DB markdown column (worker mode stores markdown directly).
			var mdBytes sql.NullInt64
			if db.QueryRow(`SELECT COALESCE(SUM(length(markdown)), 0) FROM pages WHERE markdown IS NOT NULL AND length(markdown) > 0`).Scan(&mdBytes) == nil && mdBytes.Valid {
				sr.mdBytes = mdBytes.Int64
			}

			// If html column has data (worker mode), use that for htmlBytes instead of content_length.
			var htmlColBytes sql.NullInt64
			if db.QueryRow(`SELECT COALESCE(SUM(octet_length(html)), 0) FROM pages WHERE html IS NOT NULL AND octet_length(html) > 0`).Scan(&htmlColBytes) == nil && htmlColBytes.Valid && htmlColBytes.Int64 > 0 {
				sr.htmlBytes = htmlColBytes.Int64
			}

			var links int64
			if db.QueryRow(`SELECT count(*) FROM links`).Scan(&links) == nil {
				sr.links = links
			}

			results[idx] = sr
		}(i, shard)
	}
	wg.Wait()

	var dbMdBytes int64
	for _, sr := range results {
		d.Pages += sr.pages
		d.Success += sr.success
		d.Failed += sr.failed
		d.Links += sr.links
		d.HtmlBytes += sr.htmlBytes
		dbMdBytes += sr.mdBytes
		if sr.lastCrawl.After(d.LastCrawl) {
			d.LastCrawl = sr.lastCrawl
		}
	}

	// Compute directory sizes for markdown files and FTS.
	d.MdBytes = dirSize(filepath.Join(domainDir, "markdown"))
	// If no markdown directory but DB has inline markdown (worker mode), use DB bytes.
	if d.MdBytes == 0 && dbMdBytes > 0 {
		d.MdBytes = dbMdBytes
	}
	d.IndexBytes = dirSize(filepath.Join(domainDir, "fts"))
	d.HasMD = d.MdBytes > 0
	d.HasIndex = d.IndexBytes > 0

	return d
}

// saveToMeta persists domain stats to the scrape_meta.duckdb file.
// It opens a short-lived write connection, creates the table if needed,
// and upserts all domain rows in a single transaction.
func saveToMeta(dataDir string, domains []Domain) {
	metaPath := filepath.Join(dataDir, metaFile)

	db, err := sql.Open("duckdb", metaPath)
	if err != nil {
		log.Printf("[scrape-meta] open meta DB for write: %v", err)
		return
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS domain_stats (
		domain TEXT PRIMARY KEY,
		pages BIGINT, success BIGINT, failed BIGINT, links BIGINT,
		html_bytes BIGINT, md_bytes BIGINT, index_bytes BIGINT,
		has_md BOOLEAN, has_index BOOLEAN,
		last_crawl TIMESTAMP,
		updated_at TIMESTAMP DEFAULT current_timestamp
	)`)
	if err != nil {
		log.Printf("[scrape-meta] create table: %v", err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("[scrape-meta] begin tx: %v", err)
		return
	}

	// Delete all rows and re-insert. This is simpler than upsert and the
	// dataset is small (typically <100 domains).
	if _, err := tx.Exec(`DELETE FROM domain_stats`); err != nil {
		log.Printf("[scrape-meta] delete: %v", err)
		tx.Rollback()
		return
	}

	stmt, err := tx.Prepare(`INSERT INTO domain_stats
		(domain, pages, success, failed, links, html_bytes, md_bytes, index_bytes, has_md, has_index, last_crawl, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`)
	if err != nil {
		log.Printf("[scrape-meta] prepare insert: %v", err)
		tx.Rollback()
		return
	}
	defer stmt.Close()

	now := time.Now()
	for _, d := range domains {
		var lastCrawl interface{}
		if !d.LastCrawl.IsZero() {
			lastCrawl = d.LastCrawl
		}
		if _, err := stmt.Exec(d.Domain, d.Pages, d.Success, d.Failed, d.Links,
			d.HtmlBytes, d.MdBytes, d.IndexBytes, d.HasMD, d.HasIndex,
			lastCrawl, now); err != nil {
			log.Printf("[scrape-meta] insert %s: %v", d.Domain, err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[scrape-meta] commit: %v", err)
	}
}

// dirSize walks the given directory and returns the sum of all regular file sizes.
// Returns 0 if the directory does not exist or is unreadable.
func dirSize(path string) int64 {
	var total int64
	filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if !d.IsDir() {
			if info, err := d.Info(); err == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}
