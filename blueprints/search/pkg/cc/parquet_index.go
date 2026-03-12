package cc

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	ccIndexTableManifestKind = "cc-index-table.paths.gz"
	ccIndexTableManifestAll  = ccIndexTableManifestKind + ":all"
	ccIndexWARCPathPrefix    = "cc-index/table/cc-main/warc/"
)

type parquetColumnMeta struct {
	Name  string
	Type  string
	Order int
}

type parquetImportMeta struct {
	RemotePath   string
	LocalPath    string
	ShardDBPath  string
	Subset       string
	ManifestIdx  int
	RowCount     int64
	ColumnCount  int
	ImportedAt   time.Time
	SchemaJSON   string
	SchemaFields []parquetColumnMeta
}

// ParquetManifest returns all parquet paths in the crawl's columnar-index manifest (all subsets).
// The manifest is cached locally to avoid repeated downloads.
func ParquetManifest(ctx context.Context, client *Client, cfg Config) ([]string, error) {
	cache := NewCache(cfg.DataDir)
	cd := cache.Load()
	if cd != nil {
		if paths := cache.GetManifest(cd, cfg.CrawlID, ccIndexTableManifestAll); len(paths) > 0 {
			return append([]string(nil), paths...), nil
		}
	}

	paths, err := client.DownloadManifest(ctx, cfg.CrawlID, ccIndexTableManifestKind)
	if err != nil {
		return nil, fmt.Errorf("downloading index manifest: %w", err)
	}

	if cd == nil {
		cd = &CacheData{}
	}
	cache.SetManifest(cd, cfg.CrawlID, ccIndexTableManifestAll, paths)
	_ = cache.Save(cd)

	return append([]string(nil), paths...), nil
}

// ListParquetFiles lists parquet manifest entries, optionally filtered by subset.
func ListParquetFiles(ctx context.Context, client *Client, cfg Config, opts ParquetListOptions) ([]ParquetFile, error) {
	paths, err := ParquetManifest(ctx, client, cfg)
	if err != nil {
		return nil, err
	}

	wantSubset := strings.TrimSpace(opts.Subset)
	out := make([]ParquetFile, 0, len(paths))
	for i, p := range paths {
		subset := ParquetSubsetFromPath(p)
		if wantSubset != "" && subset != wantSubset {
			continue
		}
		out = append(out, ParquetFile{
			ManifestIndex: i,
			RemotePath:    p,
			Filename:      filepath.Base(p),
			Subset:        subset,
		})
	}
	return out, nil
}

// ParquetSubsetFromPath extracts the "subset=..." partition value from a manifest or local path.
func ParquetSubsetFromPath(path string) string {
	path = filepath.ToSlash(path)
	for _, part := range strings.Split(path, "/") {
		if strings.HasPrefix(part, "subset=") {
			return strings.TrimPrefix(part, "subset=")
		}
	}
	return ""
}

// LocalParquetPathForRemote maps a manifest remote parquet path to the local crawl index directory.
// Paths are preserved (minus the static prefix) to retain hive partitions like crawl=.../subset=....
func LocalParquetPathForRemote(cfg Config, remotePath string) string {
	rel := filepath.ToSlash(remotePath)
	if strings.HasPrefix(rel, ccIndexWARCPathPrefix) {
		rel = strings.TrimPrefix(rel, ccIndexWARCPathPrefix)
	}
	return filepath.Join(cfg.IndexDir(), filepath.FromSlash(rel))
}

// IsValidParquetFile checks whether path has the PAR1 magic bytes at the end
// of the file, indicating a complete (non-truncated) parquet file.
// Returns false if the file does not exist, is too small, or is truncated.
func IsValidParquetFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var buf [4]byte
	if _, err := f.Seek(-4, io.SeekEnd); err != nil {
		return false
	}
	if _, err := f.Read(buf[:]); err != nil {
		return false
	}
	return buf == [4]byte{'P', 'A', 'R', '1'}
}

// LocalParquetFiles recursively finds parquet files under the crawl's local index directory.
func LocalParquetFiles(cfg Config) ([]string, error) {
	var files []string
	err := filepath.WalkDir(cfg.IndexDir(), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".parquet") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// LocalParquetFilesBySubset recursively finds parquet files under the crawl's local index directory
// and filters them by subset partition if provided.
func LocalParquetFilesBySubset(cfg Config, subset string) ([]string, error) {
	files, err := LocalParquetFiles(cfg)
	if err != nil || subset == "" {
		return files, err
	}
	var filtered []string
	for _, f := range files {
		if ParquetSubsetFromPath(f) == subset {
			filtered = append(filtered, f)
		}
	}
	return filtered, nil
}

// IndexShardDBPathForParquet returns the per-parquet DuckDB path for a local parquet file.
func IndexShardDBPathForParquet(cfg Config, parquetPath string) (string, error) {
	rel, err := filepath.Rel(cfg.IndexDir(), parquetPath)
	if err != nil {
		return "", err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", fmt.Errorf("parquet path %s is outside index dir %s", parquetPath, cfg.IndexDir())
	}
	rel = strings.TrimSuffix(rel, filepath.Ext(rel))
	return filepath.Join(cfg.IndexShardDir(), rel, "index.duckdb"), nil
}

func sampleParquetFiles(files []ParquetFile, sampleSize int) []ParquetFile {
	if sampleSize <= 0 || sampleSize >= len(files) {
		return files
	}
	sampled := make([]ParquetFile, 0, sampleSize)
	step := float64(len(files)) / float64(sampleSize)
	for i := range sampleSize {
		idx := int(float64(i) * step)
		if idx >= len(files) {
			idx = len(files) - 1
		}
		sampled = append(sampled, files[idx])
	}
	return sampled
}

// DownloadParquetFiles downloads the provided manifest entries to local storage, preserving
// the hive-style partition layout under cfg.IndexDir().
func DownloadParquetFiles(ctx context.Context, client *Client, cfg Config, files []ParquetFile, workers int, progress ProgressFn) error {
	if len(files) == 0 {
		return fmt.Errorf("no parquet files to download")
	}
	if workers <= 0 {
		workers = cfg.IndexWorkers
	}
	if workers <= 0 {
		workers = 10
	}

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var completed atomic.Int64

	var mu sync.Mutex
	var firstErr error

	for i, file := range files {
		if ctx.Err() != nil {
			break
		}

		localPath := LocalParquetPathForRemote(cfg, file.RemotePath)
		fileIndex := i + 1

		if fi, err := os.Stat(localPath); err == nil && fi.Size() > 0 && IsValidParquetFile(localPath) {
			_ = completed.Add(1)
			if progress != nil {
				progress(DownloadProgress{
					File:          file.Filename,
					RemotePath:    file.RemotePath,
					FileIndex:     fileIndex,
					TotalFiles:    len(files),
					BytesReceived: fi.Size(),
					TotalBytes:    fi.Size(),
					Skipped:       true,
					Done:          true,
				})
			}
			continue
		}

		if progress != nil {
			progress(DownloadProgress{
				File:       file.Filename,
				RemotePath: file.RemotePath,
				FileIndex:  fileIndex,
				TotalFiles: len(files),
				Started:    true,
			})
		}

		sem <- struct{}{}
		wg.Add(1)
		go func(pf ParquetFile, localPath string, fileIndex int) {
			defer func() {
				<-sem
				wg.Done()
			}()

			err := client.DownloadFile(ctx, pf.RemotePath, localPath, func(received, total int64) {
				if progress != nil {
					progress(DownloadProgress{
						File:          pf.Filename,
						RemotePath:    pf.RemotePath,
						FileIndex:     fileIndex,
						TotalFiles:    len(files),
						BytesReceived: received,
						TotalBytes:    total,
					})
				}
			})
			_ = completed.Add(1)

			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("downloading %s: %w", pf.RemotePath, err)
				}
				mu.Unlock()
			}

			if progress != nil {
				progress(DownloadProgress{
					File:       pf.Filename,
					RemotePath: pf.RemotePath,
					FileIndex:  fileIndex,
					TotalFiles: len(files),
					Done:       err == nil,
					Error:      err,
				})
			}
		}(file, localPath, fileIndex)
	}

	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}

// DownloadManifestParquetFile downloads a single parquet file selected by its manifest index.
// The index is based on the full cc-index-table.paths.gz manifest (all subsets).
func DownloadManifestParquetFile(ctx context.Context, client *Client, cfg Config, manifestIndex int, progress ProgressFn) (string, error) {
	paths, err := ParquetManifest(ctx, client, cfg)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("no parquet files found in manifest")
	}
	if manifestIndex < 0 {
		manifestIndex = 0
	}
	if manifestIndex >= len(paths) {
		return "", fmt.Errorf("manifest index %d out of range (manifest has %d files)", manifestIndex, len(paths))
	}

	file := ParquetFile{
		ManifestIndex: manifestIndex,
		RemotePath:    paths[manifestIndex],
		Filename:      filepath.Base(paths[manifestIndex]),
		Subset:        ParquetSubsetFromPath(paths[manifestIndex]),
	}
	localPath := LocalParquetPathForRemote(cfg, file.RemotePath)

	if fi, err := os.Stat(localPath); err == nil && fi.Size() > 0 {
		if progress != nil {
			progress(DownloadProgress{
				File:          file.Filename,
				RemotePath:    file.RemotePath,
				FileIndex:     1,
				TotalFiles:    1,
				BytesReceived: fi.Size(),
				TotalBytes:    fi.Size(),
				Skipped:       true,
				Done:          true,
			})
		}
		return localPath, nil
	}

	if progress != nil {
		progress(DownloadProgress{
			File:       file.Filename,
			RemotePath: file.RemotePath,
			FileIndex:  1,
			TotalFiles: 1,
			Started:    true,
		})
	}

	err = client.DownloadFile(ctx, file.RemotePath, localPath, func(received, total int64) {
		if progress != nil {
			progress(DownloadProgress{
				File:          file.Filename,
				RemotePath:    file.RemotePath,
				FileIndex:     1,
				TotalFiles:    1,
				BytesReceived: received,
				TotalBytes:    total,
			})
		}
	})
	if err != nil {
		if progress != nil {
			progress(DownloadProgress{
				File:       file.Filename,
				RemotePath: file.RemotePath,
				FileIndex:  1,
				TotalFiles: 1,
				Error:      err,
			})
		}
		return "", fmt.Errorf("downloading %s: %w", file.RemotePath, err)
	}

	if progress != nil {
		progress(DownloadProgress{
			File:       file.Filename,
			RemotePath: file.RemotePath,
			FileIndex:  1,
			TotalFiles: 1,
			Done:       true,
		})
	}
	return localPath, nil
}

// ImportIndexWithProgress imports local parquet files into per-parquet DuckDB databases and
// builds a catalog DuckDB file at cfg.IndexDBPath() with metadata and a `ccindex` view.
func ImportIndexWithProgress(ctx context.Context, cfg Config, progress ImportProgressFn) (int64, error) {
	parquetPaths, err := LocalParquetFiles(cfg)
	if err != nil {
		return 0, fmt.Errorf("reading index dir: %w", err)
	}
	return ImportParquetPathsWithProgress(ctx, cfg, parquetPaths, progress)
}

// ImportParquetPathsWithProgress imports the provided local parquet paths into per-parquet DuckDB databases.
func ImportParquetPathsWithProgress(ctx context.Context, cfg Config, parquetPaths []string, progress ImportProgressFn) (int64, error) {
	if progress != nil {
		progress(ImportProgress{Stage: "discover", Message: "Discovering local parquet files..."})
	}

	var files []string
	for _, p := range parquetPaths {
		if strings.HasSuffix(strings.ToLower(p), ".parquet") {
			files = append(files, p)
		}
	}
	sort.Strings(files)

	if len(files) == 0 {
		return 0, fmt.Errorf("no parquet files found in %s", cfg.IndexDir())
	}

	if err := os.MkdirAll(cfg.IndexShardDir(), 0755); err != nil {
		return 0, fmt.Errorf("creating index shard dir: %w", err)
	}

	startAll := time.Now()
	metas := make([]parquetImportMeta, 0, len(files))
	totalRows := int64(0)

	for i, parquetPath := range files {
		if ctx.Err() != nil {
			return totalRows, ctx.Err()
		}
		fileStart := time.Now()

		shardDBPath, err := IndexShardDBPathForParquet(cfg, parquetPath)
		if err != nil {
			return totalRows, err
		}
		if err := os.MkdirAll(filepath.Dir(shardDBPath), 0755); err != nil {
			return totalRows, fmt.Errorf("creating shard dir: %w", err)
		}
		_ = os.Remove(shardDBPath)
		_ = os.Remove(shardDBPath + ".wal")

		if progress != nil {
			progress(ImportProgress{
				Stage:      "start",
				File:       parquetPath,
				FileIndex:  i + 1,
				TotalFiles: len(files),
				Message:    "Importing parquet into per-file DuckDB",
			})
		}

		db, err := sql.Open("duckdb", shardDBPath)
		if err != nil {
			return totalRows, fmt.Errorf("opening duckdb %s: %w", shardDBPath, err)
		}

		importSQL := fmt.Sprintf(
			"CREATE TABLE ccindex AS SELECT * FROM read_parquet(%s, union_by_name=true, hive_partitioning=true, filename=true)",
			duckQuoteString(filepath.ToSlash(parquetPath)),
		)
		if err := execWithHeartbeat(ctx, db, importSQL, 5*time.Second, func(elapsed time.Duration) {
			if progress != nil {
				progress(ImportProgress{
					Stage:      "heartbeat",
					File:       parquetPath,
					FileIndex:  i + 1,
					TotalFiles: len(files),
					Elapsed:    elapsed,
					Message:    "Still importing parquet into DuckDB...",
				})
			}
		}); err != nil {
			db.Close()
			return totalRows, fmt.Errorf("importing parquet %s: %w", parquetPath, err)
		}

		var rowCount int64
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ccindex").Scan(&rowCount); err != nil {
			db.Close()
			return totalRows, fmt.Errorf("counting rows for %s: %w", parquetPath, err)
		}

		cols, err := describeDuckDBTable(ctx, db, "ccindex")
		if err != nil {
			db.Close()
			return totalRows, fmt.Errorf("describing ccindex in %s: %w", shardDBPath, err)
		}
		colSet := make(map[string]bool, len(cols))
		for _, col := range cols {
			colSet[col.Name] = true
		}

		if progress != nil {
			progress(ImportProgress{
				Stage:      "indexes",
				File:       parquetPath,
				FileIndex:  i + 1,
				TotalFiles: len(files),
				Rows:       rowCount,
				Columns:    len(cols),
				Message:    "Creating per-file indexes",
			})
		}
		createDuckDBIndexes(ctx, db, colSet)

		if err := db.Close(); err != nil {
			return totalRows, fmt.Errorf("closing duckdb %s: %w", shardDBPath, err)
		}

		schemaJSONBytes, _ := json.Marshal(cols)
		meta := parquetImportMeta{
			RemotePath:   localParquetPathToRemote(parquetPath, cfg),
			LocalPath:    parquetPath,
			ShardDBPath:  shardDBPath,
			Subset:       ParquetSubsetFromPath(parquetPath),
			ManifestIdx:  -1, // local import may not have a manifest lookup context
			RowCount:     rowCount,
			ColumnCount:  len(cols),
			ImportedAt:   time.Now().UTC(),
			SchemaJSON:   string(schemaJSONBytes),
			SchemaFields: cols,
		}
		metas = append(metas, meta)
		totalRows += rowCount

		if progress != nil {
			progress(ImportProgress{
				Stage:      "file_done",
				File:       parquetPath,
				FileIndex:  i + 1,
				TotalFiles: len(files),
				Rows:       rowCount,
				Columns:    len(cols),
				Elapsed:    time.Since(fileStart),
				Done:       true,
				Message:    "Parquet imported",
			})
		}
	}

	if progress != nil {
		progress(ImportProgress{
			Stage:      "catalog",
			TotalFiles: len(files),
			Rows:       totalRows,
			Message:    "Building catalog DuckDB (metadata + ccindex view)...",
		})
	}
	if err := rebuildIndexCatalog(ctx, cfg, files, metas); err != nil {
		return totalRows, err
	}

	if progress != nil {
		progress(ImportProgress{
			Stage:      "done",
			TotalFiles: len(files),
			Rows:       totalRows,
			Elapsed:    time.Since(startAll),
			Done:       true,
			Message:    "Import complete",
		})
	}
	return totalRows, nil
}

func execWithHeartbeat(ctx context.Context, db *sql.DB, stmt string, interval time.Duration, beat func(time.Duration)) error {
	errCh := make(chan error, 1)
	start := time.Now()
	go func() {
		_, err := db.ExecContext(ctx, stmt)
		errCh <- err
	}()

	if interval <= 0 || beat == nil {
		return <-errCh
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case err := <-errCh:
			return err
		case <-ticker.C:
			beat(time.Since(start))
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func describeDuckDBTable(ctx context.Context, db *sql.DB, table string) ([]parquetColumnMeta, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("DESCRIBE SELECT * FROM %s", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []parquetColumnMeta
	order := 0
	for rows.Next() {
		var name, typ string
		var nullable, key, def, extra sql.NullString
		if err := rows.Scan(&name, &typ, &nullable, &key, &def, &extra); err != nil {
			return nil, err
		}
		cols = append(cols, parquetColumnMeta{Name: name, Type: typ, Order: order})
		order++
	}
	return cols, rows.Err()
}

func createDuckDBIndexes(ctx context.Context, db *sql.DB, colSet map[string]bool) {
	indexes := []struct {
		Name   string
		Column string
	}{
		{Name: "idx_domain", Column: "url_host_registered_domain"},
		{Name: "idx_tld", Column: "url_host_tld"},
		{Name: "idx_status", Column: "fetch_status"},
		{Name: "idx_mime", Column: "content_mime_detected"},
		{Name: "idx_url", Column: "url"},
	}

	for _, idx := range indexes {
		if !colSet[idx.Column] {
			continue
		}
		_, _ = db.ExecContext(ctx, fmt.Sprintf("CREATE INDEX %s ON ccindex(%s)", idx.Name, idx.Column))
	}
}

func rebuildIndexCatalog(ctx context.Context, cfg Config, parquetPaths []string, metas []parquetImportMeta) error {
	dbPath := cfg.IndexDBPath()
	_ = os.Remove(dbPath)
	_ = os.Remove(dbPath + ".wal")

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("opening catalog duckdb: %w", err)
	}
	defer db.Close()

	ddl := []string{
		`CREATE TABLE parquet_imports (
			manifest_index BIGINT,
			subset VARCHAR,
			remote_path VARCHAR,
			local_parquet_path VARCHAR,
			shard_db_path VARCHAR,
			row_count BIGINT,
			column_count BIGINT,
			schema_json JSON,
			imported_at TIMESTAMP
		)`,
		`CREATE TABLE parquet_columns (
			shard_db_path VARCHAR,
			local_parquet_path VARCHAR,
			column_order BIGINT,
			column_name VARCHAR,
			column_type VARCHAR
		)`,
	}
	for _, stmt := range ddl {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("creating catalog metadata tables: %w", err)
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting catalog transaction: %w", err)
	}

	stmtImports, err := tx.PrepareContext(ctx, `INSERT INTO parquet_imports
		(manifest_index, subset, remote_path, local_parquet_path, shard_db_path, row_count, column_count, schema_json, imported_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare parquet_imports insert: %w", err)
	}
	defer stmtImports.Close()

	stmtCols, err := tx.PrepareContext(ctx, `INSERT INTO parquet_columns
		(shard_db_path, local_parquet_path, column_order, column_name, column_type)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare parquet_columns insert: %w", err)
	}
	defer stmtCols.Close()

	for _, m := range metas {
		if _, err := stmtImports.ExecContext(ctx,
			m.ManifestIdx,
			m.Subset,
			m.RemotePath,
			filepath.ToSlash(m.LocalPath),
			filepath.ToSlash(m.ShardDBPath),
			m.RowCount,
			m.ColumnCount,
			m.SchemaJSON,
			m.ImportedAt,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert parquet_imports: %w", err)
		}
		for _, col := range m.SchemaFields {
			if _, err := stmtCols.ExecContext(ctx,
				filepath.ToSlash(m.ShardDBPath),
				filepath.ToSlash(m.LocalPath),
				col.Order,
				col.Name,
				col.Type,
			); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("insert parquet_columns: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit catalog transaction: %w", err)
	}

	// Build a catalog view over all local parquet files to preserve existing query/stats commands.
	pathsExpr := duckQuotedList(parquetPaths)
	viewSQL := fmt.Sprintf(
		`CREATE VIEW ccindex AS
		 SELECT * FROM read_parquet(%s, union_by_name=true, hive_partitioning=true, filename=true)`,
		pathsExpr,
	)
	if _, err := db.ExecContext(ctx, viewSQL); err != nil {
		return fmt.Errorf("creating ccindex view: %w", err)
	}

	_, _ = db.ExecContext(ctx, "CREATE VIEW ccindex_warc AS SELECT * FROM ccindex WHERE warc_filename IS NOT NULL")
	_, _ = db.ExecContext(ctx, "CREATE INDEX idx_parquet_imports_subset ON parquet_imports(subset)")

	return nil
}

func duckQuoteString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func duckQuotedList(paths []string) string {
	parts := make([]string, 0, len(paths))
	for _, p := range paths {
		parts = append(parts, duckQuoteString(filepath.ToSlash(p)))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func localParquetPathToRemote(parquetPath string, cfg Config) string {
	rel, err := filepath.Rel(cfg.IndexDir(), parquetPath)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "..") {
		return ""
	}
	if strings.HasPrefix(rel, "crawl=") {
		return ccIndexWARCPathPrefix + rel
	}
	return rel
}
