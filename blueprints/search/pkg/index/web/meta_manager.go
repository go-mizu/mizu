package web

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/api"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/metastore"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/web/metastore/drivers/duckdb"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/web/metastore/drivers/sqlite"
)

const (
	defaultMetaDriver     = "sqlite"
	defaultMetaRefreshTTL = 30 * time.Second
)

// MetaConfig configures the dashboard metadata cache manager.
type MetaConfig struct {
	Driver      string
	DSN         string
	RefreshTTL  time.Duration
	Prewarm     bool
	BusyTimeout time.Duration
	JournalMode string
	ActiveCrawl string
	ActiveDir   string
	CommonCrawl string // parent dir containing crawl directories
}

// DataSummaryWithMeta is the /api/overview payload with cache metadata.
type DataSummaryWithMeta = api.DataSummaryWithMeta

// MetaStatus is returned by /api/meta/status and /api/meta/refresh.
type MetaStatus = api.MetaStatus

// MetaManager handles cache read/refresh policy around a metastore.Store.
type MetaManager struct {
	store       metastore.Store
	backend     string
	refreshTTL  time.Duration
	activeCrawl string
	activeDir   string
	commonCrawl string

	mu              sync.Mutex
	refreshing      map[string]bool
	lastRefreshAt   map[string]time.Time // in-memory cache of last successful refresh
	lastScanDur     map[string]time.Duration
	manifestMu      sync.Mutex
	manifestCache map[string]metaManifestCacheEntry
}

type metaManifestCacheEntry struct {
	paths     []string
	fetchedAt time.Time
}

// NewMetaManager creates a metadata manager. Driver "none" disables persistent
// metastore and falls back to direct scan mode.
func NewMetaManager(ctx context.Context, cfg MetaConfig) (*MetaManager, error) {
	driver := cfg.Driver
	if driver == "" {
		driver = defaultMetaDriver
	}
	refreshTTL := cfg.RefreshTTL
	if refreshTTL <= 0 {
		refreshTTL = defaultMetaRefreshTTL
	}
	commonCrawl := cfg.CommonCrawl
	if commonCrawl == "" {
		commonCrawl = filepath.Dir(cfg.ActiveDir)
	}

	m := &MetaManager{
		backend:       driver,
		refreshTTL:    refreshTTL,
		activeCrawl:   cfg.ActiveCrawl,
		activeDir:     cfg.ActiveDir,
		commonCrawl:   commonCrawl,
		refreshing:    make(map[string]bool),
		lastRefreshAt: make(map[string]time.Time),
		lastScanDur:   make(map[string]time.Duration),
		manifestCache: make(map[string]metaManifestCacheEntry),
	}

	if driver == "none" {
		m.backend = "scan-fallback"
		logInfof("meta manager configured backend=scan-fallback")
		return m, nil
	}

	dsn := cfg.DSN
	if dsn == "" {
		dsn = defaultMetaDSN(commonCrawl, driver)
	}

	store, err := metastore.Open(driver, dsn, metastore.Options{
		BusyTimeout: cfg.BusyTimeout,
		JournalMode: cfg.JournalMode,
	})
	if err != nil {
		return nil, fmt.Errorf("open metastore: %w", err)
	}
	if err := store.Init(ctx); err != nil {
		store.Close()
		return nil, fmt.Errorf("init metastore: %w", err)
	}
	m.store = store
	logInfof("meta manager configured backend=%s dsn=%s ttl=%s", driver, dsn, refreshTTL)

	if cfg.Prewarm && cfg.ActiveCrawl != "" {
		m.TriggerRefresh(cfg.ActiveCrawl, cfg.ActiveDir, false)
	}
	return m, nil
}

func defaultMetaDSN(commonCrawl, driver string) string {
	metaDir := filepath.Join(commonCrawl, ".meta")
	switch driver {
	case "duckdb":
		return filepath.Join(metaDir, "dashboard_meta.duckdb")
	case "sqlite":
		fallthrough
	default:
		return filepath.Join(metaDir, "dashboard_meta.sqlite")
	}
}

// Close closes the underlying metastore if enabled.
func (m *MetaManager) Close() error {
	if m == nil || m.store == nil {
		return nil
	}
	logInfof("meta manager close backend=%s", m.backend)
	return m.store.Close()
}

// GetSummary returns cached summary if available and schedules refresh when
// stale. On cache miss it performs a synchronous refresh once.
func (m *MetaManager) GetSummary(ctx context.Context, crawlID, crawlDir string) DataSummaryWithMeta {
	crawlID, crawlDir = m.resolveCrawl(crawlID, crawlDir)

	// No store configured: direct scan mode.
	if m.store == nil {
		return m.scanFallback(crawlID, crawlDir, "")
	}

	rec, ok, err := m.store.GetSummary(ctx, crawlID)
	if err != nil {
		return m.scanFallback(crawlID, crawlDir, err.Error())
	}
	st, hasState, stErr := m.store.GetRefreshState(ctx, crawlID)
	if stErr != nil {
		// State lookup failure is non-fatal; continue with summary.
		st = metastore.RefreshState{}
		hasState = false
	}

	if ok {
		stale := m.isStale(rec.GeneratedAt, rec.ScanDuration)
		refreshing := hasState && st.Status == "refreshing"
		if stale && !refreshing {
			m.TriggerRefresh(crawlID, crawlDir, false)
		}
		return m.summaryFromRecord(rec, stale, refreshing, st.LastError)
	}

	// Cache miss: refresh once synchronously so the first response is populated.
	resp, rErr := m.refreshSync(ctx, crawlID, crawlDir)
	if rErr != nil {
		return m.scanFallback(crawlID, crawlDir, rErr.Error())
	}
	return resp
}

// TriggerRefresh starts an async refresh if one is not already running.
func (m *MetaManager) TriggerRefresh(crawlID, crawlDir string, force bool) bool {
	if m == nil || m.store == nil {
		return false
	}
	crawlID, crawlDir = m.resolveCrawl(crawlID, crawlDir)
	if !m.acquireRefresh(crawlID, force) {
		return false
	}

	go func() {
		defer m.releaseRefresh(crawlID)
		_, _ = m.refreshSync(context.Background(), crawlID, crawlDir)
	}()
	return true
}

// Backend returns the configured backend name without hitting the database.
func (m *MetaManager) Backend() string {
	return m.backend
}

// Store returns the underlying metastore, or nil if not configured.
func (m *MetaManager) Store() metastore.Store {
	return m.store
}

// IsStale returns true if the cached data for the given crawl is stale.
// This reads only in-memory state and never touches the database.
func (m *MetaManager) IsStale(crawlID string) bool {
	crawlID, _ = m.resolveCrawl(crawlID, "")
	m.mu.Lock()
	t, ok := m.lastRefreshAt[crawlID]
	dur := m.lastScanDur[crawlID]
	m.mu.Unlock()
	if !ok {
		return true // never refreshed
	}
	return m.isStale(t, dur)
}

// IsRefreshing returns true if a refresh goroutine is active for the given crawl.
// This reads only in-memory state and never touches the database.
func (m *MetaManager) IsRefreshing(crawlID string) bool {
	crawlID, _ = m.resolveCrawl(crawlID, "")
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.refreshing[crawlID]
}

// Status returns the current refresh state for a crawl.
func (m *MetaManager) Status(ctx context.Context, crawlID string) MetaStatus {
	crawlID, _ = m.resolveCrawl(crawlID, "")
	status := MetaStatus{
		CrawlID:      crawlID,
		Backend:      m.backend,
		Enabled:      m.store != nil,
		Status:       "idle",
		Refreshing:   false,
		RefreshTTLMS: m.refreshTTL.Milliseconds(),
	}
	if m.store == nil {
		return status
	}
	st, ok, err := m.store.GetRefreshState(ctx, crawlID)
	if err != nil || !ok {
		return status
	}
	status.Status = st.Status
	status.Refreshing = st.Status == "refreshing"
	status.Generation = st.Generation
	status.LastError = st.LastError
	if st.StartedAt != nil {
		status.StartedAt = st.StartedAt.UTC().Format(time.RFC3339)
	}
	if st.FinishedAt != nil {
		status.FinishedAt = st.FinishedAt.UTC().Format(time.RFC3339)
	}
	return status
}

// ListWARCs returns cached per-WARC metadata rows for a crawl.
func (m *MetaManager) ListWARCs(ctx context.Context, crawlID, crawlDir string) ([]metastore.WARCRecord, DataSummaryWithMeta, error) {
	crawlID, crawlDir = m.resolveCrawl(crawlID, crawlDir)
	summary := m.GetSummary(ctx, crawlID, crawlDir)

	if m.store == nil {
		return buildWARCRecords(crawlID, crawlDir, nil, time.Now().UTC()), summary, nil
	}

	recs, err := m.store.ListWARCs(ctx, crawlID)
	if err != nil {
		return nil, summary, err
	}
	if len(recs) == 0 && summary.WARCCount > 0 && !summary.MetaRefreshing {
		m.TriggerRefresh(crawlID, crawlDir, true)
	}
	return recs, summary, nil
}

// GetWARC returns cached per-WARC metadata row for a crawl/index.
func (m *MetaManager) GetWARC(ctx context.Context, crawlID, crawlDir, warcIndex string) (metastore.WARCRecord, bool, DataSummaryWithMeta, error) {
	crawlID, crawlDir = m.resolveCrawl(crawlID, crawlDir)
	summary := m.GetSummary(ctx, crawlID, crawlDir)

	if m.store == nil {
		recs := buildWARCRecords(crawlID, crawlDir, nil, time.Now().UTC())
		for _, rec := range recs {
			if rec.WARCIndex == warcIndex {
				return rec, true, summary, nil
			}
		}
		return metastore.WARCRecord{}, false, summary, nil
	}

	rec, ok, err := m.store.GetWARC(ctx, crawlID, warcIndex)
	if err != nil {
		return metastore.WARCRecord{}, false, summary, err
	}
	if !ok && !summary.MetaRefreshing {
		m.TriggerRefresh(crawlID, crawlDir, false)
	}
	return rec, ok, summary, nil
}

func (m *MetaManager) refreshSync(ctx context.Context, crawlID, crawlDir string) (DataSummaryWithMeta, error) {
	if m.store == nil {
		return m.scanFallback(crawlID, crawlDir, ""), nil
	}

	now := time.Now().UTC()
	prev, _, _ := m.store.GetRefreshState(ctx, crawlID)
	nextGen := prev.Generation + 1

	start := now
	if err := m.store.SetRefreshState(ctx, metastore.RefreshState{
		CrawlID:    crawlID,
		Status:     "refreshing",
		StartedAt:  &start,
		FinishedAt: nil,
		LastError:  "",
		Generation: nextGen,
	}); err != nil {
		return DataSummaryWithMeta{}, fmt.Errorf("set refreshing state: %w", err)
	}

	scanStart := time.Now()
	ds := ScanDataDir(crawlDir)
	manifestPaths, mErr := m.getManifestPaths(ctx, crawlID)
	if mErr != nil {
		logErrorf("meta refresh manifest-fetch failed crawl=%s err=%v", crawlID, mErr)
	}
	warcs := buildWARCRecords(crawlID, crawlDir, manifestPaths, now)
	scanDuration := time.Since(scanStart)
	ds.CrawlID = crawlID

	fin := time.Now().UTC()
	rec := summaryToRecord(ds, warcs, fin, scanDuration)
	if err := m.store.PutSummary(ctx, rec); err != nil {
		logErrorf("meta refresh write-failed crawl=%s generation=%d err=%v", crawlID, nextGen, err)
		finErr := time.Now().UTC()
		_ = m.store.SetRefreshState(ctx, metastore.RefreshState{
			CrawlID:    crawlID,
			Status:     "error",
			StartedAt:  &start,
			FinishedAt: &finErr,
			LastError:  err.Error(),
			Generation: nextGen,
		})
		return DataSummaryWithMeta{}, fmt.Errorf("put summary: %w", err)
	}

	if err := m.store.SetRefreshState(ctx, metastore.RefreshState{
		CrawlID:    crawlID,
		Status:     "idle",
		StartedAt:  &start,
		FinishedAt: &fin,
		LastError:  "",
		Generation: nextGen,
	}); err != nil {
		logErrorf("meta refresh state-update-failed crawl=%s generation=%d err=%v", crawlID, nextGen, err)
		return DataSummaryWithMeta{}, fmt.Errorf("set idle state: %w", err)
	}

	// Cache refresh timestamp in memory so handleOverview can check staleness
	// without touching the database.
	m.mu.Lock()
	m.lastRefreshAt[crawlID] = fin
	m.lastScanDur[crawlID] = scanDuration
	m.mu.Unlock()

	return m.summaryFromRecord(rec, false, false, ""), nil
}

func (m *MetaManager) scanFallback(crawlID, crawlDir, lastErr string) DataSummaryWithMeta {
	ds := ScanDataDir(crawlDir)
	ds.CrawlID = crawlID
	resp := DataSummaryWithMeta{
		DataSummary:     ds,
		MetaBackend:     "scan-fallback",
		MetaGeneratedAt: time.Now().UTC().Format(time.RFC3339),
		MetaStale:       false,
		MetaRefreshing:  false,
		MetaLastError:   lastErr,
	}
	return resp
}

func (m *MetaManager) summaryFromRecord(rec metastore.SummaryRecord, stale, refreshing bool, lastErr string) DataSummaryWithMeta {
	ds := DataSummary{
		CrawlID:       rec.CrawlID,
		WARCCount:     int(rec.WARCCount),
		WARCTotalSize: rec.WARCTotalSize,
		MDShards:      int(rec.MDShards),
		MDTotalSize:   rec.MDTotalSize,
		MDDocEstimate: int(rec.MDDocEstimate),
		PackFormats:   make(map[string]int64),
		FTSEngines:    make(map[string]int64),
		FTSShardCount: make(map[string]int),
	}
	for k, v := range rec.PackFormats {
		ds.PackFormats[k] = v
	}
	for k, v := range rec.FTSEngines {
		ds.FTSEngines[k] = v
	}
	for k, v := range rec.FTSShardCount {
		ds.FTSShardCount[k] = int(v)
	}

	return DataSummaryWithMeta{
		DataSummary:     ds,
		MetaBackend:     m.backend,
		MetaGeneratedAt: rec.GeneratedAt.UTC().Format(time.RFC3339),
		MetaStale:       stale,
		MetaRefreshing:  refreshing,
		MetaLastError:   lastErr,
	}
}

func summaryToRecord(ds DataSummary, warcs []metastore.WARCRecord, generatedAt time.Time, scanDuration time.Duration) metastore.SummaryRecord {
	rec := metastore.SummaryRecord{
		CrawlID:       ds.CrawlID,
		WARCCount:     int64(ds.WARCCount),
		WARCTotalSize: ds.WARCTotalSize,
		MDShards:      int64(ds.MDShards),
		MDTotalSize:   ds.MDTotalSize,
		MDDocEstimate: int64(ds.MDDocEstimate),
		PackFormats:   make(map[string]int64),
		FTSEngines:    make(map[string]int64),
		FTSShardCount: make(map[string]int64),
		WARCs:         warcs,
		GeneratedAt:   generatedAt.UTC(),
		ScanDuration:  scanDuration,
	}
	for k, v := range ds.PackFormats {
		rec.PackFormats[k] = v
	}
	for k, v := range ds.FTSEngines {
		rec.FTSEngines[k] = v
	}
	for k, v := range ds.FTSShardCount {
		rec.FTSShardCount[k] = int64(v)
	}
	return rec
}

func (m *MetaManager) getManifestPaths(ctx context.Context, crawlID string) ([]string, error) {
	if crawlID == "" {
		return nil, nil
	}
	const manifestTTL = 30 * time.Minute
	now := time.Now()

	m.manifestMu.Lock()
	if entry, ok := m.manifestCache[crawlID]; ok && now.Sub(entry.fetchedAt) < manifestTTL && len(entry.paths) > 0 {
		cached := append([]string(nil), entry.paths...)
		m.manifestMu.Unlock()
		return cached, nil
	}
	m.manifestMu.Unlock()

	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	m.manifestMu.Lock()
	m.manifestCache[crawlID] = metaManifestCacheEntry{
		paths:     append([]string(nil), paths...),
		fetchedAt: now,
	}
	m.manifestMu.Unlock()
	return paths, nil
}

func (m *MetaManager) isStale(generatedAt time.Time, scanDuration time.Duration) bool {
	if generatedAt.IsZero() {
		return true
	}
	threshold := m.refreshTTL
	if scanDuration > 0 {
		// Avoid auto-refresh loops when scans are expensive.
		scanAware := m.refreshTTL + scanDuration
		if scanAware > threshold {
			threshold = scanAware
		}
	}
	return time.Since(generatedAt) > threshold
}

func (m *MetaManager) acquireRefresh(crawlID string, force bool) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !force && m.refreshing[crawlID] {
		return false
	}
	m.refreshing[crawlID] = true
	return true
}

func (m *MetaManager) releaseRefresh(crawlID string) {
	m.mu.Lock()
	delete(m.refreshing, crawlID)
	m.mu.Unlock()
}

func (m *MetaManager) resolveCrawl(crawlID, crawlDir string) (string, string) {
	if crawlID == "" {
		crawlID = m.activeCrawl
	}
	if crawlDir == "" {
		if crawlID == m.activeCrawl && m.activeDir != "" {
			crawlDir = m.activeDir
		} else {
			crawlDir = filepath.Join(m.commonCrawl, crawlID)
		}
	}
	return crawlID, crawlDir
}
