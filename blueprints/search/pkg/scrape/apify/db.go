package apify

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// DB wraps DuckDB access for Apify crawl output.
type DB struct {
	db   *sql.DB
	path string
	mu   sync.Mutex
}

func OpenDB(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	d := &DB{db: db, path: path}
	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) initSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS crawl_runs (
			run_id             VARCHAR PRIMARY KEY,
			started_at         TIMESTAMP,
			finished_at        TIMESTAMP,
			store_url          VARCHAR,
			expected_total      BIGINT,
			index_pages         BIGINT,
			indexed_total       BIGINT,
			detail_queued       BIGINT,
			detail_done         BIGINT,
			detail_success      BIGINT,
			detail_failed       BIGINT,
			status             VARCHAR,
			notes              VARCHAR
		)`,
		`CREATE TABLE IF NOT EXISTS actors_index (
			object_id          VARCHAR PRIMARY KEY,
			username           VARCHAR,
			name               VARCHAR,
			title              VARCHAR,
			description        VARCHAR,
			categories_json    VARCHAR,
			modified_at_epoch  BIGINT,
			created_at_epoch   BIGINT,
			picture_url        VARCHAR,
			raw_json           VARCHAR,
			indexed_at         TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS actors_detail (
			object_id                 VARCHAR PRIMARY KEY,
			username                  VARCHAR,
			name                      VARCHAR,
			title                     VARCHAR,
			description               VARCHAR,
			notice                    VARCHAR,
			readme_summary            VARCHAR,
			actor_permission_level    VARCHAR,
			deployment_key            VARCHAR,
			standby_url               VARCHAR,
			picture_url               VARCHAR,
			seo_title                 VARCHAR,
			seo_description           VARCHAR,
			is_public                 BOOLEAN,
			is_deprecated             BOOLEAN,
			is_generic                BOOLEAN,
			is_critical               BOOLEAN,
			is_source_code_hidden     BOOLEAN,
			has_no_dataset            BOOLEAN,
			created_at                TIMESTAMP,
			modified_at               TIMESTAMP,
			categories_json           VARCHAR,
			stats_json                VARCHAR,
			pricing_infos_json        VARCHAR,
			versions_json             VARCHAR,
			versions_all_json         VARCHAR,
			default_run_options_json  VARCHAR,
			example_run_input_json    VARCHAR,
			tagged_builds_json        VARCHAR,
			input_schema_json         VARCHAR,
			output_schema_json        VARCHAR,
			readme                    VARCHAR,
			readme_markdown           VARCHAR,
			latest_build_id           VARCHAR,
			latest_build_status       VARCHAR,
			latest_build_started_at   TIMESTAMP,
			latest_build_finished_at  TIMESTAMP,
			latest_build_stats_json   VARCHAR,
			latest_build_options_json VARCHAR,
			latest_build_meta_json    VARCHAR,
			latest_build_env_json     VARCHAR,
			latest_build_storages_json VARCHAR,
			latest_build_input_schema VARCHAR,
			latest_build_readme       VARCHAR,
			latest_build_changelog    VARCHAR,
			latest_build_dockerfile   VARCHAR,
			latest_build_raw_json     VARCHAR,
			enrichment_error          VARCHAR,
			status_code               INTEGER,
			error                     VARCHAR,
			raw_json                  VARCHAR,
			fetched_at                TIMESTAMP
		)`,
	}
	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	// Forward-compatible schema migration for existing DB files.
	migrations := []string{
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS readme_summary VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS actor_permission_level VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS deployment_key VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS standby_url VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS is_generic BOOLEAN`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS is_critical BOOLEAN`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS is_source_code_hidden BOOLEAN`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS has_no_dataset BOOLEAN`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS versions_all_json VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS latest_build_id VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS latest_build_status VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS latest_build_started_at TIMESTAMP`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS latest_build_finished_at TIMESTAMP`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS latest_build_stats_json VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS latest_build_options_json VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS latest_build_meta_json VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS latest_build_env_json VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS latest_build_storages_json VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS latest_build_input_schema VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS latest_build_readme VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS latest_build_changelog VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS latest_build_dockerfile VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS latest_build_raw_json VARCHAR`,
		`ALTER TABLE actors_detail ADD COLUMN IF NOT EXISTS enrichment_error VARCHAR`,
	}
	for _, stmt := range migrations {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema migration: %w", err)
		}
	}
	return nil
}

func (d *DB) UpsertIndex(hit StoreActorHit, rawJSON string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	cats, _ := json.Marshal(hit.Categories)
	modifiedEpoch := epochFromAny(hit.ModifiedAt)
	createdEpoch := epochFromAny(hit.CreatedAt)
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO actors_index (
			object_id, username, name, title, description, categories_json,
			modified_at_epoch, created_at_epoch, picture_url, raw_json, indexed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		hit.ObjectID,
		nullStr(hit.Username),
		nullStr(hit.Name),
		nullStr(hit.Title),
		nullStr(hit.Description),
		nullStr(string(cats)),
		modifiedEpoch,
		createdEpoch,
		nullStr(hit.PictureURL),
		nullStr(rawJSON),
		time.Now(),
	)
	return err
}

func (d *DB) UpsertDetailFailure(objectID string, statusCode int, errMsg string, rawJSON string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO actors_detail (
			object_id, status_code, error, raw_json, fetched_at
		) VALUES (?, ?, ?, ?, ?)
	`, objectID, statusCode, nullStr(errMsg), nullStr(rawJSON), time.Now())
	return err
}

func (d *DB) UpsertDetail(detail *ActorDetail, statusCode int, rawJSON string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	cats, _ := json.Marshal(detail.Categories)
	stats, _ := json.Marshal(detail.Stats)
	pricing, _ := json.Marshal(detail.PricingInfos)
	versions, _ := json.Marshal(detail.Versions)
	versionsAll, _ := json.Marshal(detail.VersionsAll)
	dro, _ := json.Marshal(detail.DefaultRunOptions)
	eri, _ := json.Marshal(detail.ExampleRunInput)
	tagged, _ := json.Marshal(detail.TaggedBuilds)
	inSchema, _ := json.Marshal(detail.InputSchema)
	outSchema, _ := json.Marshal(detail.OutputSchema)
	latestBuildID := latestBuildIDFromTagged(detail.TaggedBuilds)
	buildStatus := buildString(detail.LatestBuild, "status")
	buildStarted := buildTime(detail.LatestBuild, "startedAt")
	buildFinished := buildTime(detail.LatestBuild, "finishedAt")
	buildStats, _ := json.Marshal(buildMap(detail.LatestBuild, "stats"))
	buildOptions, _ := json.Marshal(buildMap(detail.LatestBuild, "options"))
	buildMeta, _ := json.Marshal(buildMap(detail.LatestBuild, "meta"))
	buildEnv, _ := json.Marshal(buildMap(detail.LatestBuild, "environmentVariables"))
	buildStorages, _ := json.Marshal(buildMap(detail.LatestBuild, "storages"))
	buildInputSchema := buildString(detail.LatestBuild, "inputSchema")
	buildReadme := buildString(detail.LatestBuild, "readme")
	buildChangelog := buildString(detail.LatestBuild, "changeLog")
	buildDockerfile := buildString(detail.LatestBuild, "dockerfile")
	buildRaw, _ := json.Marshal(detail.LatestBuild)

	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO actors_detail (
			object_id, username, name, title, description, notice,
			readme_summary, actor_permission_level, deployment_key, standby_url,
			picture_url, seo_title, seo_description,
			is_public, is_deprecated, is_generic, is_critical, is_source_code_hidden, has_no_dataset, created_at, modified_at,
			categories_json, stats_json, pricing_infos_json, versions_json, versions_all_json,
			default_run_options_json, example_run_input_json, tagged_builds_json,
			input_schema_json, output_schema_json, readme, readme_markdown,
			latest_build_id, latest_build_status, latest_build_started_at, latest_build_finished_at,
			latest_build_stats_json, latest_build_options_json, latest_build_meta_json,
			latest_build_env_json, latest_build_storages_json, latest_build_input_schema,
			latest_build_readme, latest_build_changelog, latest_build_dockerfile, latest_build_raw_json,
			enrichment_error,
			status_code, error, raw_json, fetched_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		detail.ID,
		nullStr(detail.Username),
		nullStr(detail.Name),
		nullStr(detail.Title),
		nullStr(detail.Description),
		nullStr(detail.Notice),
		nullStr(detail.ReadmeSummary),
		nullStr(detail.ActorPermission),
		nullStr(detail.DeploymentKey),
		nullStr(detail.StandbyURL),
		nullStr(detail.PictureURL),
		nullStr(detail.SEOTitle),
		nullStr(detail.SEODescription),
		detail.IsPublic,
		detail.IsDeprecated,
		detail.IsGeneric,
		detail.IsCritical,
		detail.IsSourceCodeHidden,
		detail.HasNoDataset,
		nullTime(detail.CreatedAt),
		nullTime(detail.ModifiedAt),
		nullStr(string(cats)),
		nullStr(string(stats)),
		nullStr(string(pricing)),
		nullStr(string(versions)),
		nullStr(string(versionsAll)),
		nullStr(string(dro)),
		nullStr(string(eri)),
		nullStr(string(tagged)),
		nullStr(string(inSchema)),
		nullStr(string(outSchema)),
		nullStr(detail.Readme),
		nullStr(detail.ReadmeMarkdown),
		nullStr(latestBuildID),
		nullStr(buildStatus),
		nullTime(buildStarted),
		nullTime(buildFinished),
		nullStr(string(buildStats)),
		nullStr(string(buildOptions)),
		nullStr(string(buildMeta)),
		nullStr(string(buildEnv)),
		nullStr(string(buildStorages)),
		nullStr(buildInputSchema),
		nullStr(buildReadme),
		nullStr(buildChangelog),
		nullStr(buildDockerfile),
		nullStr(string(buildRaw)),
		nullStr(detail.EnrichmentError),
		statusCode,
		nil,
		nullStr(rawJSON),
		time.Now(),
	)
	return err
}

func (d *DB) PendingDetailIDs(max int, refresh bool) ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	query := `
		SELECT i.object_id
		FROM actors_index i
		LEFT JOIN actors_detail d ON d.object_id = i.object_id
	`
	if !refresh {
		query += ` WHERE d.object_id IS NULL OR d.status_code < 200 OR d.status_code >= 300`
	}
	query += ` ORDER BY i.object_id`
	if max > 0 {
		query += fmt.Sprintf(" LIMIT %d", max)
	}

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (d *DB) SaveRunStart(runID string, cfg Config) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO crawl_runs (
			run_id, started_at, store_url, status, notes
		) VALUES (?, ?, ?, ?, ?)
	`, runID, time.Now(), cfg.StoreURL, "running", "")
	return err
}

func (d *DB) SaveRunFinish(runID string, s CrawlStats, status, notes string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(`
		UPDATE crawl_runs
		SET finished_at = ?, expected_total = ?, index_pages = ?, indexed_total = ?,
		    detail_queued = ?, detail_done = ?, detail_success = ?, detail_failed = ?,
		    status = ?, notes = ?
		WHERE run_id = ?
	`,
		time.Now(),
		s.ExpectedTotal,
		s.IndexPages,
		s.IndexedTotal,
		s.DetailQueued,
		s.DetailDone,
		s.DetailSuccess,
		s.DetailFailed,
		status,
		notes,
		runID,
	)
	return err
}

func (d *DB) Counts() (indexed int64, detailed int64, failed int64, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	err = d.db.QueryRow(`SELECT COUNT(*) FROM actors_index`).Scan(&indexed)
	if err != nil {
		return
	}
	err = d.db.QueryRow(`SELECT COUNT(*) FROM actors_detail WHERE status_code >= 200 AND status_code < 300`).Scan(&detailed)
	if err != nil {
		return
	}
	err = d.db.QueryRow(`SELECT COUNT(*) FROM actors_detail WHERE status_code < 200 OR status_code >= 300`).Scan(&failed)
	return
}

func (d *DB) IndexCount() (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var n int64
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM actors_index`).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (d *DB) Close() error { return d.db.Close() }
func (d *DB) Path() string { return d.path }

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func epochFromAny(v any) int64 {
	switch x := v.(type) {
	case nil:
		return 0
	case int64:
		return x
	case int32:
		return int64(x)
	case int:
		return int64(x)
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return 0
		}
		return int64(x)
	case json.Number:
		n, err := x.Int64()
		if err != nil {
			return 0
		}
		return n
	case string:
		if x == "" {
			return 0
		}
		if n, err := strconv.ParseInt(x, 10, 64); err == nil {
			return n
		}
		// ISO timestamp fallback.
		if ts, err := time.Parse(time.RFC3339, x); err == nil {
			return ts.Unix()
		}
		if ts, err := time.Parse("2006-01-02T15:04:05.000Z", x); err == nil {
			return ts.Unix()
		}
		return 0
	default:
		return 0
	}
}

func latestBuildIDFromTagged(tagged map[string]any) string {
	latest, ok := tagged["latest"].(map[string]any)
	if !ok {
		return ""
	}
	v, _ := latest["buildId"].(string)
	return v
}

func buildMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, ok := m[key].(map[string]any)
	if !ok {
		return nil
	}
	return v
}

func buildString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func buildTime(m map[string]any, key string) time.Time {
	if m == nil {
		return time.Time{}
	}
	s, _ := m[key].(string)
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
