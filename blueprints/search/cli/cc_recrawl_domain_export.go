package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"golang.org/x/sync/errgroup"

	_ "github.com/duckdb/duckdb-go/v2"
)

func ccRecrawlDomainRootDir(cfg cc.Config) string {
	return filepath.Join(cfg.RecrawlDir(), "domains")
}

type ccDomainExportSummary struct {
	ParquetFolder  string
	RootDir        string
	DomainsUpdated int
	FilesExported  int
	FilesSkipped   int
	TasksTotal     int
}

type ccDomainExportSource struct {
	DBPath      string
	Table       string
	SourceName  string // results_000 / failed_domains / failed_urls
	StageDir    string
	DomainCount int
}

type ccDomainFinalizeTask struct {
	Domain     string
	SourceName string
	StagePath  string
	OutPath    string
}

type ccDomainFinalizeTaskResult struct {
	Skipped bool
}

func ccExportPerDomainRecrawlArtifacts(ctx context.Context, cfg cc.Config, sourceParquetPath, resultDir, failedDBPath string) (ccDomainExportSummary, error) {
	parquetFolder := ccRecrawlParquetFolderName(sourceParquetPath)
	rootDir := ccRecrawlDomainRootDir(cfg)
	stageRoot := filepath.Join(filepath.Dir(resultDir), ".domain-export-stage-"+parquetFolder)

	fmt.Println(infoStyle.Render("Domain parquet export (per-domain parquet files)..."))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Layout: %s/<tld>/<domain>/%s/<PART>.parquet", rootDir, parquetFolder)))
	fmt.Println(labelStyle.Render("  Mechanism: single partitioned DuckDB COPY per source shard/table (preserves column types), then high-parallel file finalize"))

	start := time.Now()
	sources, domainTaskCounts, estimatedTasks, err := ccBuildDomainParquetExportPlan(ctx, stageRoot, resultDir, failedDBPath)
	if err != nil {
		return ccDomainExportSummary{}, err
	}
	totalDomains := len(domainTaskCounts)
	if totalDomains == 0 {
		fmt.Println(labelStyle.Render("  No domains found to export"))
		return ccDomainExportSummary{ParquetFolder: parquetFolder, RootDir: rootDir}, nil
	}

	copyWorkers := min(max(runtime.NumCPU()/2, 2), 8)
	finalizeWorkers := min(1000, max(1, estimatedTasks))

	fmt.Println(infoStyle.Render("Export estimate"))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Domains: %s", ccFmtInt64(int64(totalDomains)))))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Parquet tasks (estimated): %s", ccFmtInt64(int64(estimatedTasks)))))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Source scans: %d (partitioned COPY workers=%d)", len(sources), copyWorkers)))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Finalize workers: %d (rename/copy workers)", finalizeWorkers)))

	_ = os.RemoveAll(stageRoot)
	if err := os.MkdirAll(stageRoot, 0755); err != nil {
		return ccDomainExportSummary{}, fmt.Errorf("creating stage root: %w", err)
	}

	// Phase 1: one partitioned COPY per source shard/table. This is the main speedup:
	// scan each source once instead of 1 SQL query per domain.
	fmt.Println(infoStyle.Render("Stage 1: Partitioned source export (single scan per source)..."))
	var sourcesDone atomic.Int64
	var stageFiles atomic.Int64
	copyStart := time.Now()

	copyProgressDone := make(chan struct{})
	go func() {
		defer close(copyProgressDone)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				done := sourcesDone.Load()
				elapsed := time.Since(copyStart)
				eta := "---"
				if done > 0 && elapsed > 0 {
					rate := float64(done) / elapsed.Seconds()
					if rate > 0 {
						remain := float64(len(sources)) - float64(done)
						etaDur := time.Duration(float64(time.Second) * remain / rate)
						eta = etaDur.Truncate(time.Second).String()
					}
				}
				fmt.Println(labelStyle.Render(fmt.Sprintf(
					"  Source export progress: %s/%s sources  │  staged files %s  │  ETA %s",
					ccFmtInt64(done), ccFmtInt64(int64(len(sources))), ccFmtInt64(stageFiles.Load()), eta,
				)))
				if done >= int64(len(sources)) {
					return
				}
			}
		}
	}()

	gCopy, gctxCopy := errgroup.WithContext(ctx)
	gCopy.SetLimit(copyWorkers)
	for _, src := range sources {
		src := src
		gCopy.Go(func() error {
			fmt.Println(labelStyle.Render(fmt.Sprintf("    [%s] %s (%s domains) → partitioned export...",
				src.SourceName, filepath.Base(src.DBPath), ccFmtInt64(int64(src.DomainCount)))))
			srcStart := time.Now()
			n, err := ccRunDomainPartitionedSourceExport(gctxCopy, src)
			if err != nil {
				return err
			}
			stageFiles.Add(int64(n))
			sourcesDone.Add(1)
			fmt.Println(successStyle.Render(fmt.Sprintf("    [%s] staged %s parquet file(s) in %s",
				src.SourceName, ccFmtInt64(int64(n)), time.Since(srcStart).Truncate(time.Second))))
			return nil
		})
	}
	copyErr := gCopy.Wait()
	if copyErr != nil {
		<-copyProgressDone
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Stage files kept for debugging: %s", stageRoot)))
		return ccDomainExportSummary{}, copyErr
	}
	<-copyProgressDone
	copyDuration := time.Since(copyStart)
	fmt.Println(successStyle.Render(fmt.Sprintf("  Stage 1 complete: %s source scans, %s staged files (%s)",
		ccFmtInt64(int64(len(sources))), ccFmtInt64(stageFiles.Load()), copyDuration.Truncate(time.Second))))

	// Phase 2: build finalize task list from staged partition directories.
	fmt.Println(infoStyle.Render("Stage 2: Finalize staged parquet files into domain layout..."))
	finalizeTasks, duplicateSplits, err := ccBuildDomainParquetFinalizeTasks(stageRoot, rootDir, parquetFolder, sources)
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Stage files kept for debugging: %s", stageRoot)))
		return ccDomainExportSummary{}, err
	}
	if duplicateSplits > 0 {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %d domain/source partitions emitted multiple parquet files; extra files were suffixed", duplicateSplits)))
	}
	if len(finalizeTasks) == 0 {
		_ = os.RemoveAll(stageRoot)
		fmt.Println(labelStyle.Render("  No staged parquet files found to finalize"))
		return ccDomainExportSummary{ParquetFolder: parquetFolder, RootDir: rootDir}, nil
	}
	if len(finalizeTasks) != estimatedTasks {
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Finalize tasks: %s actual (estimate %s)", ccFmtInt64(int64(len(finalizeTasks))), ccFmtInt64(int64(estimatedTasks)))))
	}

	pending := make(map[string]*atomic.Int32, len(domainTaskCounts))
	for d, n := range domainTaskCounts {
		var c atomic.Int32
		c.Store(int32(n))
		pending[d] = &c
	}

	var tasksDone atomic.Int64
	var filesExported atomic.Int64
	var filesSkipped atomic.Int64
	var domainsDone atomic.Int64
	finalizeStart := time.Now()

	progressDone := make(chan struct{})
	go func() {
		defer close(progressDone)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				doneD := domainsDone.Load()
				doneT := tasksDone.Load()
				elapsed := time.Since(finalizeStart)
				eta := "---"
				if doneD > 0 && elapsed > 0 {
					rate := float64(doneD) / elapsed.Seconds()
					if rate > 0 {
						remain := float64(totalDomains) - float64(doneD)
						etaDur := time.Duration(float64(time.Second) * remain / rate)
						eta = etaDur.Truncate(time.Second).String()
					}
				}
				fmt.Println(labelStyle.Render(fmt.Sprintf(
					"  Export progress: domains %s/%s (%.1f%%)  files %s/%s  exported=%s skipped=%s  ETA %s",
					ccFmtInt64(doneD), ccFmtInt64(int64(totalDomains)), 100*float64(doneD)/float64(totalDomains),
					ccFmtInt64(doneT), ccFmtInt64(int64(len(finalizeTasks))),
					ccFmtInt64(filesExported.Load()), ccFmtInt64(filesSkipped.Load()), eta,
				)))
				if doneT >= int64(len(finalizeTasks)) {
					return
				}
			}
		}
	}()

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(finalizeWorkers)
	for _, task := range finalizeTasks {
		task := task
		g.Go(func() error {
			res, err := ccRunDomainParquetFinalizeTask(gctx, task)
			if err != nil {
				return err
			}
			if res.Skipped {
				filesSkipped.Add(1)
			} else {
				filesExported.Add(1)
			}
			tasksDone.Add(1)
			if p := pending[task.Domain]; p != nil {
				if left := p.Add(-1); left == 0 {
					domainsDone.Add(1)
				}
			}
			return nil
		})
	}
	finalizeErr := g.Wait()
	if finalizeErr != nil {
		<-progressDone
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Stage files kept for debugging: %s", stageRoot)))
		return ccDomainExportSummary{}, finalizeErr
	}
	<-progressDone
	finalizeDuration := time.Since(finalizeStart)

	summary := ccDomainExportSummary{
		ParquetFolder:  parquetFolder,
		RootDir:        rootDir,
		DomainsUpdated: int(domainsDone.Load()),
		FilesExported:  int(filesExported.Load()),
		FilesSkipped:   int(filesSkipped.Load()),
		TasksTotal:     len(finalizeTasks),
	}
	_ = os.RemoveAll(stageRoot)
	fmt.Println(successStyle.Render(fmt.Sprintf(
		"  Domain parquet export complete: %s domains, %s files (%s exported, %s skipped) in %s",
		ccFmtInt64(int64(summary.DomainsUpdated)),
		ccFmtInt64(int64(summary.TasksTotal)),
		ccFmtInt64(int64(summary.FilesExported)),
		ccFmtInt64(int64(summary.FilesSkipped)),
		time.Since(start).Truncate(time.Second),
	)))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Stage timings: source-copy=%s, finalize=%s",
		copyDuration.Truncate(time.Second), finalizeDuration.Truncate(time.Second))))
	return summary, nil
}

func ccBuildDomainParquetExportPlan(ctx context.Context, stageRoot, resultDir, failedDBPath string) ([]ccDomainExportSource, map[string]int, int, error) {
	sources := make([]ccDomainExportSource, 0, 32)
	domainTaskCounts := make(map[string]int, 8192)
	estimatedTasks := 0

	addSource := func(domains []string, dbPath, table, sourceName string) {
		stageDir := filepath.Join(stageRoot, sanitizePathToken(sourceName))
		sources = append(sources, ccDomainExportSource{
			DBPath:      dbPath,
			Table:       table,
			SourceName:  sourceName,
			StageDir:    stageDir,
			DomainCount: len(domains),
		})
		estimatedTasks += len(domains)
		for _, d := range domains {
			d = strings.ToLower(strings.TrimSpace(d))
			if d == "" {
				continue
			}
			domainTaskCounts[d]++
		}
	}

	entries, err := os.ReadDir(resultDir)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("reading results dir: %w", err)
	}
	var shardPaths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), "results_") && strings.HasSuffix(e.Name(), ".duckdb") {
			shardPaths = append(shardPaths, filepath.Join(resultDir, e.Name()))
		}
	}
	sort.Strings(shardPaths)

	fmt.Println(labelStyle.Render(fmt.Sprintf("  Planning exports from %d result shard(s)...", len(shardPaths))))
	for i, shardPath := range shardPaths {
		domains, err := ccDistinctDomainsFromTable(ctx, shardPath, "results")
		if err != nil {
			return nil, nil, 0, err
		}
		partName := strings.TrimSuffix(filepath.Base(shardPath), ".duckdb")
		addSource(domains, shardPath, "results", partName)
		fmt.Println(labelStyle.Render(fmt.Sprintf("    [%d/%d] %s → %s domains", i+1, len(shardPaths), filepath.Base(shardPath), ccFmtInt64(int64(len(domains))))))
	}

	fmt.Println(labelStyle.Render("  Planning exports from failed.duckdb..."))
	for _, table := range []string{"failed_domains", "failed_urls"} {
		domains, err := ccDistinctDomainsFromTable(ctx, failedDBPath, table)
		if err != nil {
			return nil, nil, 0, err
		}
		addSource(domains, failedDBPath, table, table)
		fmt.Println(labelStyle.Render(fmt.Sprintf("    %s → %s domains", table, ccFmtInt64(int64(len(domains))))))
	}

	return sources, domainTaskCounts, estimatedTasks, nil
}

func ccDistinctDomainsFromTable(ctx context.Context, dbPath, table string) ([]string, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("opening %s for %s domain scan: %w", dbPath, table, err)
	}
	defer db.Close()

	query := fmt.Sprintf("SELECT DISTINCT domain FROM %s WHERE COALESCE(domain,'') <> '' ORDER BY domain", table)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying distinct domains from %s (%s): %w", table, filepath.Base(dbPath), err)
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, fmt.Errorf("scanning domain from %s: %w", table, err)
		}
		d = strings.ToLower(strings.TrimSpace(d))
		if d != "" {
			domains = append(domains, d)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating domains from %s: %w", table, err)
	}
	return domains, nil
}

func ccRunDomainPartitionedSourceExport(ctx context.Context, src ccDomainExportSource) (int, error) {
	_ = os.RemoveAll(src.StageDir)
	if err := os.MkdirAll(filepath.Dir(src.StageDir), 0755); err != nil {
		return 0, fmt.Errorf("creating source stage parent: %w", err)
	}

	db, err := sql.Open("duckdb", src.DBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return 0, fmt.Errorf("opening source db %s: %w", filepath.Base(src.DBPath), err)
	}
	defer db.Close()

	copySQL := fmt.Sprintf(
		"COPY (SELECT * FROM %s WHERE COALESCE(TRIM(domain), '') <> '') TO %s (FORMAT PARQUET, COMPRESSION ZSTD, PARTITION_BY (domain), PER_THREAD_OUTPUT FALSE)",
		src.Table,
		ccDuckSQLString(src.StageDir),
	)
	if _, err := db.ExecContext(ctx, copySQL); err != nil {
		return 0, fmt.Errorf("partitioned export %s (%s.%s): %w", src.SourceName, filepath.Base(src.DBPath), src.Table, err)
	}

	n, err := ccCountParquetFilesUnder(src.StageDir)
	if err != nil {
		return 0, fmt.Errorf("counting staged parquet files for %s: %w", src.SourceName, err)
	}
	return n, nil
}

func ccBuildDomainParquetFinalizeTasks(stageRoot, rootDir, parquetFolder string, sources []ccDomainExportSource) ([]ccDomainFinalizeTask, int, error) {
	tasks := make([]ccDomainFinalizeTask, 0, 1024)
	dupCounts := make(map[string]int, 1024)
	duplicateSplits := 0

	fmt.Println(labelStyle.Render(fmt.Sprintf("  Scanning staged files from %d source exports...", len(sources))))
	for i, src := range sources {
		localCount := 0
		err := filepath.WalkDir(src.StageDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".parquet") {
				return nil
			}
			domain, perr := ccDomainFromPartitionedStagePath(src.StageDir, path)
			if perr != nil {
				return perr
			}
			outDir := ccDomainParquetOutputDir(rootDir, domain, parquetFolder)
			baseName := src.SourceName + ".parquet"
			key := domain + "\x00" + src.SourceName
			if dupCounts[key] > 0 {
				duplicateSplits++
				baseName = fmt.Sprintf("%s__%03d.parquet", src.SourceName, dupCounts[key]+1)
			}
			dupCounts[key]++
			tasks = append(tasks, ccDomainFinalizeTask{
				Domain:     domain,
				SourceName: src.SourceName,
				StagePath:  path,
				OutPath:    filepath.Join(outDir, baseName),
			})
			localCount++
			return nil
		})
		if err != nil {
			return nil, 0, fmt.Errorf("walking staged files for %s: %w", src.SourceName, err)
		}
		fmt.Println(labelStyle.Render(fmt.Sprintf("    [%d/%d] %s → %s staged parquet file(s)", i+1, len(sources), src.SourceName, ccFmtInt64(int64(localCount)))))
	}

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Domain == tasks[j].Domain {
			if tasks[i].SourceName == tasks[j].SourceName {
				return tasks[i].OutPath < tasks[j].OutPath
			}
			return tasks[i].SourceName < tasks[j].SourceName
		}
		return tasks[i].Domain < tasks[j].Domain
	})
	return tasks, duplicateSplits, nil
}

func ccDomainFromPartitionedStagePath(stageDir, parquetPath string) (string, error) {
	rel, err := filepath.Rel(stageDir, parquetPath)
	if err != nil {
		return "", fmt.Errorf("relative stage path: %w", err)
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	for _, p := range parts[:max(0, len(parts)-1)] {
		if strings.HasPrefix(p, "domain=") {
			raw := strings.TrimPrefix(p, "domain=")
			if dec, err := url.PathUnescape(raw); err == nil {
				raw = dec
			}
			raw = strings.ToLower(strings.TrimSpace(raw))
			if raw == "" {
				break
			}
			return raw, nil
		}
	}
	return "", fmt.Errorf("could not parse domain partition from staged path: %s", parquetPath)
}

func ccRunDomainParquetFinalizeTask(ctx context.Context, task ccDomainFinalizeTask) (ccDomainFinalizeTaskResult, error) {
	select {
	case <-ctx.Done():
		return ccDomainFinalizeTaskResult{}, ctx.Err()
	default:
	}

	if fi, err := os.Stat(task.OutPath); err == nil && fi.Size() > 0 {
		_ = os.Remove(task.StagePath)
		return ccDomainFinalizeTaskResult{Skipped: true}, nil
	}
	if err := os.MkdirAll(filepath.Dir(task.OutPath), 0755); err != nil {
		return ccDomainFinalizeTaskResult{}, fmt.Errorf("creating domain parquet dir: %w", err)
	}
	if err := ccMoveFile(task.StagePath, task.OutPath); err != nil {
		return ccDomainFinalizeTaskResult{}, fmt.Errorf("finalizing %s for %s: %w", task.SourceName, task.Domain, err)
	}
	return ccDomainFinalizeTaskResult{}, nil
}

func ccMoveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".tmp"
	_ = os.Remove(tmp)
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Remove(src)
	return nil
}

func ccCountParquetFilesUnder(root string) (int, error) {
	count := 0
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".parquet") {
			count++
		}
		return nil
	})
	return count, err
}

func ccDomainParquetOutputDir(rootDir, domain, parquetFolder string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	tld := domain
	if i := strings.LastIndex(domain, "."); i >= 0 && i+1 < len(domain) {
		tld = domain[i+1:]
	}
	return filepath.Join(rootDir, sanitizePathToken(tld), sanitizePathToken(domain), parquetFolder)
}

func ccRecrawlParquetFolderName(parquetPath string) string {
	base := filepath.Base(parquetPath)
	if strings.HasPrefix(base, "part-") {
		rest := strings.TrimPrefix(base, "part-")
		if dash := strings.IndexByte(rest, '-'); dash > 0 {
			num := rest[:dash]
			if _, err := strconv.Atoi(num); err == nil {
				return num
			}
		}
	}
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.TrimSuffix(base, ".parquet")
	return sanitizePathToken(base)
}

func ccDuckSQLString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
