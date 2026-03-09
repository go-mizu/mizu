package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	mizu "github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline"
	_ "github.com/duckdb/duckdb-go/v2"
)

func duckQuotePath(p string) string {
	return "'" + strings.ReplaceAll(filepath.ToSlash(p), "'", "''") + "'"
}

func isSafeSQL(q string) bool {
	q = strings.TrimSpace(q)
	if q == "" {
		return false
	}
	upper := strings.ToUpper(q)
	return strings.HasPrefix(upper, "SELECT ") || strings.HasPrefix(upper, "SELECT\n") ||
		strings.HasPrefix(upper, "SELECT\t") || strings.HasPrefix(upper, "WITH ") ||
		strings.HasPrefix(upper, "WITH\n") || strings.HasPrefix(upper, "WITH\t")
}

// subsetKPIQueries defines scalar metric queries per subset.
var subsetKPIQueries = map[string][]struct {
	Key   string
	Query string
}{
	"warc": {
		{Key: "unique_domains", Query: "SELECT COUNT(DISTINCT url_host_registered_domain) FROM ccindex WHERE url_host_registered_domain IS NOT NULL"},
		{Key: "unique_tlds", Query: "SELECT COUNT(DISTINCT url_host_tld) FROM ccindex WHERE url_host_tld IS NOT NULL"},
		{Key: "https_pct", Query: "SELECT COALESCE(ROUND(100.0 * SUM(CASE WHEN url_protocol = 'https' THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0), 1), 0) FROM ccindex"},
	},
	"non200responses": {
		{Key: "unique_domains", Query: "SELECT COUNT(DISTINCT url_host_registered_domain) FROM ccindex WHERE url_host_registered_domain IS NOT NULL"},
		{Key: "redirect_pct", Query: "SELECT COALESCE(ROUND(100.0 * SUM(CASE WHEN fetch_redirect IS NOT NULL AND fetch_redirect != '' THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0), 1), 0) FROM ccindex"},
		{Key: "unique_statuses", Query: "SELECT COUNT(DISTINCT fetch_status) FROM ccindex"},
	},
	"robotstxt": {
		{Key: "unique_domains", Query: "SELECT COUNT(DISTINCT url_host_registered_domain) FROM ccindex WHERE url_host_registered_domain IS NOT NULL"},
		{Key: "unique_tlds", Query: "SELECT COUNT(DISTINCT url_host_tld) FROM ccindex WHERE url_host_tld IS NOT NULL"},
		{Key: "https_pct", Query: "SELECT COALESCE(ROUND(100.0 * SUM(CASE WHEN url_protocol = 'https' THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0), 1), 0) FROM ccindex"},
	},
	"crawldiagnostics": {
		{Key: "unique_domains", Query: "SELECT COUNT(DISTINCT url_host_registered_domain) FROM ccindex WHERE url_host_registered_domain IS NOT NULL"},
		{Key: "unique_statuses", Query: "SELECT COUNT(DISTINCT fetch_status) FROM ccindex"},
		{Key: "unique_mimes", Query: "SELECT COUNT(DISTINCT content_mime_detected) FROM ccindex WHERE content_mime_detected IS NOT NULL"},
	},
}

// subsetChartQueries defines distribution queries per subset.
var subsetChartQueries = map[string][]struct {
	Key   string
	Query string
}{
	"warc": {
		{Key: "tld", Query: "SELECT url_host_tld AS label, COUNT(*) AS value FROM ccindex WHERE url_host_tld IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 25"},
		{Key: "domain", Query: "SELECT url_host_registered_domain AS label, COUNT(*) AS value FROM ccindex WHERE url_host_registered_domain IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 25"},
		{Key: "mime", Query: "SELECT content_mime_detected AS label, COUNT(*) AS value FROM ccindex WHERE content_mime_detected IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 20"},
		{Key: "language", Query: "SELECT content_languages AS label, COUNT(*) AS value FROM ccindex WHERE content_languages IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 20"},
		{Key: "status", Query: "SELECT CAST(fetch_status AS VARCHAR) AS label, COUNT(*) AS value FROM ccindex GROUP BY 1 ORDER BY 2 DESC LIMIT 20"},
		{Key: "charset", Query: "SELECT content_charset AS label, COUNT(*) AS value FROM ccindex WHERE content_charset IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 15"},
		{Key: "protocol", Query: "SELECT url_protocol AS label, COUNT(*) AS value FROM ccindex WHERE url_protocol IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 5"},
		{Key: "segment", Query: "SELECT warc_segment AS label, COUNT(*) AS value FROM ccindex WHERE warc_segment IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 10"},
	},
	"non200responses": {
		{Key: "status", Query: "SELECT CAST(fetch_status AS VARCHAR) AS label, COUNT(*) AS value FROM ccindex GROUP BY 1 ORDER BY 2 DESC LIMIT 20"},
		{Key: "domain", Query: "SELECT url_host_registered_domain AS label, COUNT(*) AS value FROM ccindex WHERE url_host_registered_domain IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 25"},
		{Key: "tld", Query: "SELECT url_host_tld AS label, COUNT(*) AS value FROM ccindex WHERE url_host_tld IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 20"},
		{Key: "redirect", Query: "SELECT fetch_redirect AS label, COUNT(*) AS value FROM ccindex WHERE fetch_redirect IS NOT NULL AND fetch_redirect != '' GROUP BY 1 ORDER BY 2 DESC LIMIT 20"},
		{Key: "mime", Query: "SELECT content_mime_detected AS label, COUNT(*) AS value FROM ccindex WHERE content_mime_detected IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 15"},
		{Key: "protocol", Query: "SELECT url_protocol AS label, COUNT(*) AS value FROM ccindex WHERE url_protocol IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 5"},
	},
	"robotstxt": {
		{Key: "domain", Query: "SELECT url_host_registered_domain AS label, COUNT(*) AS value FROM ccindex WHERE url_host_registered_domain IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 25"},
		{Key: "tld", Query: "SELECT url_host_tld AS label, COUNT(*) AS value FROM ccindex WHERE url_host_tld IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 25"},
		{Key: "status", Query: "SELECT CAST(fetch_status AS VARCHAR) AS label, COUNT(*) AS value FROM ccindex GROUP BY 1 ORDER BY 2 DESC LIMIT 15"},
		{Key: "protocol", Query: "SELECT url_protocol AS label, COUNT(*) AS value FROM ccindex WHERE url_protocol IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 5"},
		{Key: "segment", Query: "SELECT warc_segment AS label, COUNT(*) AS value FROM ccindex WHERE warc_segment IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 10"},
	},
	"crawldiagnostics": {
		{Key: "domain", Query: "SELECT url_host_registered_domain AS label, COUNT(*) AS value FROM ccindex WHERE url_host_registered_domain IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 25"},
		{Key: "tld", Query: "SELECT url_host_tld AS label, COUNT(*) AS value FROM ccindex WHERE url_host_tld IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 20"},
		{Key: "status", Query: "SELECT CAST(fetch_status AS VARCHAR) AS label, COUNT(*) AS value FROM ccindex GROUP BY 1 ORDER BY 2 DESC LIMIT 20"},
		{Key: "mime", Query: "SELECT content_mime_detected AS label, COUNT(*) AS value FROM ccindex WHERE content_mime_detected IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 15"},
		{Key: "segment", Query: "SELECT warc_segment AS label, COUNT(*) AS value FROM ccindex WHERE warc_segment IS NOT NULL GROUP BY 1 ORDER BY 2 DESC LIMIT 10"},
	},
}

func getCCConfig(d *Deps) cc.Config {
	if d.CCConfig != nil {
		return d.CCConfig()
	}
	cfg := cc.DefaultConfig()
	cfg.CrawlID = d.CrawlID
	if d.CrawlDir != "" {
		cfg.DataDir = filepath.Dir(d.CrawlDir)
	}
	return cfg
}

func handleParquetManifest(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		cfg := getCCConfig(d)
		subset := c.Query("subset")
		q := c.Query("q")
		offset := queryIntAPI(c, "offset", 0)
		limit := queryIntAPI(c, "limit", 200)

		client := cc.NewClient(cfg.BaseURL, cfg.TransportShards)
		opts := cc.ParquetListOptions{}
		if subset != "" && subset != "all" {
			opts.Subset = subset
		}

		files, err := cc.ListParquetFiles(c.Request().Context(), client, cfg, opts)
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("manifest: %v", err)})
		}

		if q != "" {
			qLower := strings.ToLower(q)
			filtered := files[:0]
			for _, f := range files {
				if strings.Contains(strings.ToLower(f.Filename), qLower) {
					filtered = append(filtered, f)
				}
			}
			files = filtered
		}

		entries := make([]ParquetFileEntry, len(files))
		summary := ParquetManifestSummary{BySubset: make(map[string]SubsetSummary)}
		for i, f := range files {
			localPath := cc.LocalParquetPathForRemote(cfg, f.RemotePath)
			var downloaded, invalid bool
			var localSize int64
			if info, err := os.Stat(localPath); err == nil {
				localSize = info.Size()
				if cc.IsValidParquetFile(localPath) {
					downloaded = true
				} else {
					invalid = true
				}
			}
			entries[i] = ParquetFileEntry{
				ManifestIndex: f.ManifestIndex,
				RemotePath:    f.RemotePath,
				Filename:      f.Filename,
				Subset:        f.Subset,
				Downloaded:    downloaded,
				Invalid:       invalid,
				LocalSize:     localSize,
			}
			summary.Total++
			if downloaded {
				summary.Downloaded++
				summary.DiskBytes += localSize
			} else if invalid {
				summary.Invalid++
				summary.DiskBytes += localSize
			}
			ss := summary.BySubset[f.Subset]
			ss.Total++
			if downloaded {
				ss.Downloaded++
			}
			summary.BySubset[f.Subset] = ss
		}

		total := len(entries)
		if offset > total {
			offset = total
		}
		end := offset + limit
		if end > total {
			end = total
		}

		return c.JSON(200, ParquetManifestResponse{
			Files:   entries[offset:end],
			Summary: summary,
			Total:   total,
			Offset:  offset,
			Limit:   limit,
		})
	}
}

func handleParquetSchema(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		cfg := getCCConfig(d)
		locals, err := cc.LocalParquetFiles(cfg)
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("local files: %v", err)})
		}
		if len(locals) == 0 {
			return c.JSON(400, errResp{"no local parquet files found"})
		}

		source := locals[0]
		db, err := sql.Open("duckdb", "")
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("duckdb open: %v", err)})
		}
		defer db.Close()

		query := fmt.Sprintf("DESCRIBE SELECT * FROM read_parquet(%s)", duckQuotePath(source))
		rows, err := db.QueryContext(c.Request().Context(), query)
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("describe: %v", err)})
		}
		defer rows.Close()

		var cols []ParquetColumnInfo
		order := 0
		for rows.Next() {
			var name, typ string
			var ignore sql.NullString
			if err := rows.Scan(&name, &typ, &ignore, &ignore, &ignore, &ignore); err != nil {
				return c.JSON(500, errResp{fmt.Sprintf("scan: %v", err)})
			}
			cols = append(cols, ParquetColumnInfo{Name: name, Type: typ, Order: order})
			order++
		}

		return c.JSON(200, ParquetSchemaResponse{
			Columns: cols,
			Source:  filepath.Base(source),
		})
	}
}

func handleParquetQuery(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		var req ParquetQueryRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(400, errResp{fmt.Sprintf("invalid json: %v", err)})
		}
		if !isSafeSQL(req.SQL) {
			return c.JSON(400, errResp{"only SELECT or WITH statements are allowed"})
		}
		if req.Limit <= 0 {
			req.Limit = 1000
		}
		if req.Limit > 10000 {
			req.Limit = 10000
		}

		cfg := getCCConfig(d)
		locals, err := cc.LocalParquetFiles(cfg)
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("local files: %v", err)})
		}
		if len(locals) == 0 {
			return c.JSON(400, errResp{"no local parquet files"})
		}

		db, err := sql.Open("duckdb", "")
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("duckdb open: %v", err)})
		}
		defer db.Close()

		quoted := make([]string, len(locals))
		for i, p := range locals {
			quoted[i] = duckQuotePath(p)
		}
		viewSQL := fmt.Sprintf(
			"CREATE VIEW ccindex AS SELECT * FROM read_parquet([%s], union_by_name=true, hive_partitioning=true)",
			strings.Join(quoted, ", "),
		)
		if _, err := db.Exec(viewSQL); err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("create view: %v", err)})
		}

		ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
		defer cancel()

		start := time.Now()
		rows, err := db.QueryContext(ctx, req.SQL)
		if err != nil {
			return c.JSON(400, errResp{fmt.Sprintf("query error: %v", err)})
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("columns: %v", err)})
		}

		var result [][]interface{}
		truncated := false
		for rows.Next() {
			if len(result) >= req.Limit {
				truncated = true
				break
			}
			vals := make([]interface{}, len(columns))
			ptrs := make([]interface{}, len(columns))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return c.JSON(500, errResp{fmt.Sprintf("scan: %v", err)})
			}
			for i, v := range vals {
				if b, ok := v.([]byte); ok {
					vals[i] = string(b)
				}
			}
			result = append(result, vals)
		}

		elapsed := time.Since(start).Milliseconds()
		if result == nil {
			result = [][]interface{}{}
		}

		return c.JSON(200, ParquetQueryResponse{
			Columns:   columns,
			Rows:      result,
			TotalRows: len(result),
			ElapsedMs: elapsed,
			Truncated: truncated,
		})
	}
}

func handleParquetDownload(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		var req ParquetDownloadRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(400, errResp{fmt.Sprintf("invalid json: %v", err)})
		}

		cfg := getCCConfig(d)
		client := cc.NewClient(cfg.BaseURL, cfg.TransportShards)

		opts := cc.ParquetListOptions{}
		if req.Subset != "" && req.Subset != "all" {
			opts.Subset = req.Subset
		}

		files, err := cc.ListParquetFiles(c.Request().Context(), client, cfg, opts)
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("manifest: %v", err)})
		}

		if len(req.Indices) > 0 {
			idxSet := make(map[int]bool, len(req.Indices))
			for _, idx := range req.Indices {
				idxSet[idx] = true
			}
			filtered := make([]cc.ParquetFile, 0, len(req.Indices))
			for _, f := range files {
				if idxSet[f.ManifestIndex] {
					filtered = append(filtered, f)
				}
			}
			files = filtered
		}

		var toDownload []cc.ParquetFile
		for _, f := range files {
			localPath := cc.LocalParquetPathForRemote(cfg, f.RemotePath)
			if !cc.IsValidParquetFile(localPath) {
				toDownload = append(toDownload, f)
			}
		}

		if req.Sample > 0 && req.Sample < len(toDownload) {
			toDownload = toDownload[:req.Sample]
		}

		if len(toDownload) == 0 {
			return c.JSON(200, ParquetDownloadResponse{
				Started:   false,
				Message:   "all matching files already downloaded",
				FileCount: 0,
			})
		}

		idxStrs := make([]string, len(toDownload))
		for i, f := range toDownload {
			idxStrs[i] = strconv.Itoa(f.ManifestIndex)
		}

		if d.Jobs == nil {
			return c.JSON(503, errResp{"job manager not available"})
		}
		job := d.Jobs.Create(pipeline.JobConfig{
			Type:   "parquet_download",
			Source: req.Subset,
			Files:  strings.Join(idxStrs, ","),
		})
		d.Jobs.RunJob(job)

		return c.JSON(200, ParquetDownloadResponse{
			Started:   true,
			Message:   fmt.Sprintf("downloading %d parquet files", len(toDownload)),
			FileCount: len(toDownload),
			JobID:     job.ID,
		})
	}
}

func handleParquetStats(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		cfg := getCCConfig(d)
		locals, err := cc.LocalParquetFiles(cfg)
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("local files: %v", err)})
		}

		var diskBytes int64
		for _, p := range locals {
			if info, err := os.Stat(p); err == nil {
				diskBytes += info.Size()
			}
		}

		resp := ParquetStatsResponse{
			LocalFiles: len(locals),
			DiskBytes:  diskBytes,
			CrawlID:    d.CrawlID,
		}

		if len(locals) > 0 {
			db, err := sql.Open("duckdb", "")
			if err == nil {
				defer db.Close()
				quoted := make([]string, len(locals))
				for i, p := range locals {
					quoted[i] = duckQuotePath(p)
				}
				countSQL := fmt.Sprintf(
					"SELECT COUNT(*) FROM read_parquet([%s], union_by_name=true, hive_partitioning=true)",
					strings.Join(quoted, ", "),
				)
				_ = db.QueryRowContext(c.Request().Context(), countSQL).Scan(&resp.TotalRows)
				descSQL := fmt.Sprintf("DESCRIBE SELECT * FROM read_parquet(%s)", duckQuotePath(locals[0]))
				if rows, err := db.QueryContext(c.Request().Context(), descSQL); err == nil {
					defer rows.Close()
					for rows.Next() {
						resp.SchemaColumns++
					}
				}
			}
		}

		return c.JSON(200, resp)
	}
}

func handleParquetFileDetail(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		idx, err := strconv.Atoi(c.Param("index"))
		if err != nil {
			return c.JSON(400, errResp{"invalid manifest index"})
		}

		cfg := getCCConfig(d)
		client := cc.NewClient(cfg.BaseURL, cfg.TransportShards)
		files, err := cc.ListParquetFiles(c.Request().Context(), client, cfg, cc.ParquetListOptions{})
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("manifest: %v", err)})
		}

		var found *cc.ParquetFile
		for i := range files {
			if files[i].ManifestIndex == idx {
				found = &files[i]
				break
			}
		}
		if found == nil {
			return c.JSON(404, errResp{"parquet file not found in manifest"})
		}

		localPath := cc.LocalParquetPathForRemote(cfg, found.RemotePath)
		resp := ParquetFileDetailResponse{
			ManifestIndex: found.ManifestIndex,
			RemotePath:    found.RemotePath,
			Filename:      found.Filename,
			Subset:        found.Subset,
		}

		if info, statErr := os.Stat(localPath); statErr == nil {
			resp.Downloaded = true
			resp.LocalSize = info.Size()
			resp.LocalPath = localPath

			db, dbErr := sql.Open("duckdb", "")
			if dbErr == nil {
				defer db.Close()
				countSQL := fmt.Sprintf(
					"SELECT SUM(row_group_num_rows) FROM parquet_metadata(%s) WHERE column_id = 0",
					duckQuotePath(localPath),
				)
				_ = db.QueryRowContext(c.Request().Context(), countSQL).Scan(&resp.RowCount)
				descSQL := fmt.Sprintf("DESCRIBE SELECT * FROM read_parquet(%s)", duckQuotePath(localPath))
				if rows, qErr := db.QueryContext(c.Request().Context(), descSQL); qErr == nil {
					defer rows.Close()
					order := 0
					for rows.Next() {
						var name, typ string
						var ignore sql.NullString
						if rows.Scan(&name, &typ, &ignore, &ignore, &ignore, &ignore) == nil {
							resp.Columns = append(resp.Columns, ParquetColumnInfo{Name: name, Type: typ, Order: order})
							order++
						}
					}
				}
			}
		}

		return c.JSON(200, resp)
	}
}

func handleParquetFileData(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		idx, err := strconv.Atoi(c.Param("index"))
		if err != nil {
			return c.JSON(400, errResp{"invalid manifest index"})
		}

		page := queryIntAPI(c, "page", 1)
		pageSize := queryIntAPI(c, "page_size", 100)
		sortBy := c.Query("sort")
		filter := c.Query("filter")

		if page < 1 {
			page = 1
		}
		if pageSize < 1 {
			pageSize = 100
		}
		if pageSize > 500 {
			pageSize = 500
		}

		cfg := getCCConfig(d)
		client := cc.NewClient(cfg.BaseURL, cfg.TransportShards)
		files, err := cc.ListParquetFiles(c.Request().Context(), client, cfg, cc.ParquetListOptions{})
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("manifest: %v", err)})
		}

		var found *cc.ParquetFile
		for i := range files {
			if files[i].ManifestIndex == idx {
				found = &files[i]
				break
			}
		}
		if found == nil {
			return c.JSON(404, errResp{"parquet file not found"})
		}

		localPath := cc.LocalParquetPathForRemote(cfg, found.RemotePath)
		if _, statErr := os.Stat(localPath); statErr != nil {
			return c.JSON(400, errResp{"file not downloaded locally"})
		}

		db, err := sql.Open("duckdb", "")
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("duckdb: %v", err)})
		}
		defer db.Close()

		quotedPath := duckQuotePath(localPath)
		fromClause := fmt.Sprintf("read_parquet(%s, hive_partitioning=true)", quotedPath)

		whereClause := ""
		if filter != "" && !strings.ContainsAny(filter, ";") {
			whereClause = " WHERE " + filter
		}

		ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
		defer cancel()

		start := time.Now()

		var total int64
		countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", fromClause, whereClause)
		if err := db.QueryRowContext(ctx, countSQL).Scan(&total); err != nil {
			return c.JSON(400, errResp{fmt.Sprintf("count: %v", err)})
		}

		orderClause := ""
		if sortBy != "" && !strings.ContainsAny(sortBy, ";'\"") {
			orderClause = " ORDER BY " + sortBy
		}

		offset := (page - 1) * pageSize
		dataSQL := fmt.Sprintf("SELECT * FROM %s%s%s LIMIT %d OFFSET %d",
			fromClause, whereClause, orderClause, pageSize, offset)

		rows, err := db.QueryContext(ctx, dataSQL)
		if err != nil {
			return c.JSON(400, errResp{fmt.Sprintf("query: %v", err)})
		}
		defer rows.Close()

		columns, _ := rows.Columns()
		var result [][]interface{}
		for rows.Next() {
			vals := make([]interface{}, len(columns))
			ptrs := make([]interface{}, len(columns))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return c.JSON(500, errResp{fmt.Sprintf("scan: %v", err)})
			}
			for i, v := range vals {
				if b, ok := v.([]byte); ok {
					vals[i] = string(b)
				}
			}
			result = append(result, vals)
		}
		if result == nil {
			result = [][]interface{}{}
		}

		return c.JSON(200, ParquetFileDataResponse{
			Columns:   columns,
			Rows:      result,
			Total:     total,
			Page:      page,
			PageSize:  pageSize,
			ElapsedMs: time.Since(start).Milliseconds(),
		})
	}
}

func handleParquetFileStats(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		idx, err := strconv.Atoi(c.Param("index"))
		if err != nil {
			return c.JSON(400, errResp{"invalid manifest index"})
		}

		cfg := getCCConfig(d)
		client := cc.NewClient(cfg.BaseURL, cfg.TransportShards)
		files, err := cc.ListParquetFiles(c.Request().Context(), client, cfg, cc.ParquetListOptions{})
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("manifest: %v", err)})
		}

		var found *cc.ParquetFile
		for i := range files {
			if files[i].ManifestIndex == idx {
				found = &files[i]
				break
			}
		}
		if found == nil {
			return c.JSON(404, errResp{"parquet file not found in manifest"})
		}

		localPath := cc.LocalParquetPathForRemote(cfg, found.RemotePath)
		if _, statErr := os.Stat(localPath); statErr != nil {
			return c.JSON(400, errResp{"file not downloaded locally"})
		}

		chartQueries := subsetChartQueries[found.Subset]
		kpiQueries := subsetKPIQueries[found.Subset]

		db, err := sql.Open("duckdb", "")
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("duckdb: %v", err)})
		}
		defer db.Close()

		viewSQL := fmt.Sprintf(
			"CREATE VIEW ccindex AS SELECT * FROM read_parquet(%s, hive_partitioning=true)",
			duckQuotePath(localPath),
		)
		if _, err := db.Exec(viewSQL); err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("view: %v", err)})
		}

		ctx, cancel := context.WithTimeout(c.Request().Context(), 60*time.Second)
		defer cancel()

		start := time.Now()

		var rowCount int64
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ccindex").Scan(&rowCount)

		kpis := make(map[string]float64, len(kpiQueries))
		for _, q := range kpiQueries {
			var val float64
			if db.QueryRowContext(ctx, q.Query).Scan(&val) == nil {
				kpis[q.Key] = val
			}
		}

		charts := make(map[string][]ChartEntry, len(chartQueries))
		for _, q := range chartQueries {
			rows, qErr := db.QueryContext(ctx, q.Query)
			if qErr != nil {
				charts[q.Key] = []ChartEntry{}
				continue
			}
			var entries []ChartEntry
			for rows.Next() {
				var e ChartEntry
				if rows.Scan(&e.Label, &e.Value) == nil {
					entries = append(entries, e)
				}
			}
			rows.Close()
			if entries == nil {
				entries = []ChartEntry{}
			}
			charts[q.Key] = entries
		}

		return c.JSON(200, ParquetFileStatsResponse{
			ManifestIndex: idx,
			Subset:        found.Subset,
			RowCount:      rowCount,
			ElapsedMs:     time.Since(start).Milliseconds(),
			KPIs:          kpis,
			Charts:        charts,
		})
	}
}

func handleParquetSubsetStats(d *Deps) mizu.Handler {
	return func(c *mizu.Ctx) error {
		subset := c.Param("subset")
		queries, ok := subsetChartQueries[subset]
		if !ok {
			return c.JSON(400, errResp{fmt.Sprintf("unknown subset: %s", subset)})
		}

		cfg := getCCConfig(d)
		locals, err := cc.LocalParquetFilesBySubset(cfg, subset)
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("local files: %v", err)})
		}
		if len(locals) == 0 {
			return c.JSON(400, errResp{fmt.Sprintf("no local parquet files for subset %q — download some first", subset)})
		}

		var diskBytes int64
		for _, p := range locals {
			if info, statErr := os.Stat(p); statErr == nil {
				diskBytes += info.Size()
			}
		}

		db, err := sql.Open("duckdb", "")
		if err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("duckdb: %v", err)})
		}
		defer db.Close()

		quoted := make([]string, len(locals))
		for i, p := range locals {
			quoted[i] = duckQuotePath(p)
		}
		viewSQL := fmt.Sprintf(
			"CREATE VIEW ccindex AS SELECT * FROM read_parquet([%s], union_by_name=true, hive_partitioning=true)",
			strings.Join(quoted, ", "),
		)
		if _, err := db.Exec(viewSQL); err != nil {
			return c.JSON(500, errResp{fmt.Sprintf("view: %v", err)})
		}

		ctx, cancel := context.WithTimeout(c.Request().Context(), 60*time.Second)
		defer cancel()

		start := time.Now()

		var totalRows int64
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ccindex").Scan(&totalRows)

		charts := make(map[string][]ChartEntry, len(queries))
		for _, q := range queries {
			rows, qErr := db.QueryContext(ctx, q.Query)
			if qErr != nil {
				charts[q.Key] = []ChartEntry{}
				continue
			}
			var entries []ChartEntry
			for rows.Next() {
				var e ChartEntry
				if rows.Scan(&e.Label, &e.Value) == nil {
					entries = append(entries, e)
				}
			}
			rows.Close()
			if entries == nil {
				entries = []ChartEntry{}
			}
			charts[q.Key] = entries
		}

		return c.JSON(200, ParquetSubsetStatsResponse{
			Subset:    subset,
			TotalRows: totalRows,
			FileCount: len(locals),
			DiskBytes: diskBytes,
			ElapsedMs: time.Since(start).Milliseconds(),
			Charts:    charts,
		})
	}
}
