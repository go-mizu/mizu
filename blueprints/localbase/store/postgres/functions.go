package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FunctionsStore implements store.FunctionsStore using PostgreSQL.
type FunctionsStore struct {
	pool *pgxpool.Pool
}

// CreateFunction creates a new edge function.
func (s *FunctionsStore) CreateFunction(ctx context.Context, fn *store.Function) error {
	sql := `
	INSERT INTO functions.functions (id, name, slug, version, status, entrypoint, import_map, verify_jwt)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := s.pool.Exec(ctx, sql,
		fn.ID,
		fn.Name,
		fn.Slug,
		fn.Version,
		fn.Status,
		fn.Entrypoint,
		nullIfEmpty(fn.ImportMap),
		fn.VerifyJWT,
	)

	return err
}

// GetFunction retrieves a function by ID.
func (s *FunctionsStore) GetFunction(ctx context.Context, id string) (*store.Function, error) {
	sql := `
	SELECT id, name, slug, version, status, entrypoint, import_map, verify_jwt, created_at, updated_at
	FROM functions.functions
	WHERE id = $1
	`

	fn := &store.Function{}
	var importMap *string

	err := s.pool.QueryRow(ctx, sql, id).Scan(
		&fn.ID,
		&fn.Name,
		&fn.Slug,
		&fn.Version,
		&fn.Status,
		&fn.Entrypoint,
		&importMap,
		&fn.VerifyJWT,
		&fn.CreatedAt,
		&fn.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("function not found")
	}
	if err != nil {
		return nil, err
	}

	if importMap != nil {
		fn.ImportMap = *importMap
	}

	return fn, nil
}

// GetFunctionByName retrieves a function by name or slug.
// This supports both the display name and the URL-friendly slug.
func (s *FunctionsStore) GetFunctionByName(ctx context.Context, name string) (*store.Function, error) {
	// Try to find by name first, then by slug
	sql := `SELECT id FROM functions.functions WHERE name = $1 OR slug = $1`

	var id string
	err := s.pool.QueryRow(ctx, sql, name).Scan(&id)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("function not found")
	}
	if err != nil {
		return nil, err
	}

	return s.GetFunction(ctx, id)
}

// ListFunctions lists all functions.
func (s *FunctionsStore) ListFunctions(ctx context.Context) ([]*store.Function, error) {
	sql := `
	SELECT id, name, slug, version, status, entrypoint, import_map, verify_jwt, created_at, updated_at
	FROM functions.functions
	ORDER BY name
	`

	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialize with empty slice instead of nil to return [] instead of null in JSON
	functions := make([]*store.Function, 0)
	for rows.Next() {
		fn := &store.Function{}
		var importMap *string

		err := rows.Scan(
			&fn.ID,
			&fn.Name,
			&fn.Slug,
			&fn.Version,
			&fn.Status,
			&fn.Entrypoint,
			&importMap,
			&fn.VerifyJWT,
			&fn.CreatedAt,
			&fn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if importMap != nil {
			fn.ImportMap = *importMap
		}

		functions = append(functions, fn)
	}

	return functions, nil
}

// UpdateFunction updates a function.
func (s *FunctionsStore) UpdateFunction(ctx context.Context, fn *store.Function) error {
	sql := `
	UPDATE functions.functions
	SET name = $2, slug = $3, version = $4, status = $5, entrypoint = $6, import_map = $7, verify_jwt = $8, updated_at = NOW()
	WHERE id = $1
	`

	_, err := s.pool.Exec(ctx, sql,
		fn.ID,
		fn.Name,
		fn.Slug,
		fn.Version,
		fn.Status,
		fn.Entrypoint,
		nullIfEmpty(fn.ImportMap),
		fn.VerifyJWT,
	)

	return err
}

// DeleteFunction deletes a function.
func (s *FunctionsStore) DeleteFunction(ctx context.Context, id string) error {
	sql := `DELETE FROM functions.functions WHERE id = $1`
	_, err := s.pool.Exec(ctx, sql, id)
	return err
}

// CreateDeployment creates a new deployment.
func (s *FunctionsStore) CreateDeployment(ctx context.Context, deployment *store.Deployment) error {
	sql := `
	INSERT INTO functions.deployments (id, function_id, version, source_code, bundle_path, status)
	VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.pool.Exec(ctx, sql,
		deployment.ID,
		deployment.FunctionID,
		deployment.Version,
		deployment.SourceCode,
		nullIfEmpty(deployment.BundlePath),
		deployment.Status,
	)

	return err
}

// GetDeployment retrieves a deployment by ID.
func (s *FunctionsStore) GetDeployment(ctx context.Context, id string) (*store.Deployment, error) {
	sql := `
	SELECT id, function_id, version, source_code, bundle_path, status, deployed_at
	FROM functions.deployments
	WHERE id = $1
	`

	deployment := &store.Deployment{}
	var bundlePath *string

	err := s.pool.QueryRow(ctx, sql, id).Scan(
		&deployment.ID,
		&deployment.FunctionID,
		&deployment.Version,
		&deployment.SourceCode,
		&bundlePath,
		&deployment.Status,
		&deployment.DeployedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("deployment not found")
	}
	if err != nil {
		return nil, err
	}

	if bundlePath != nil {
		deployment.BundlePath = *bundlePath
	}

	return deployment, nil
}

// GetLatestDeployment retrieves the latest deployment for a function.
func (s *FunctionsStore) GetLatestDeployment(ctx context.Context, functionID string) (*store.Deployment, error) {
	sql := `
	SELECT id, function_id, version, source_code, bundle_path, status, deployed_at
	FROM functions.deployments
	WHERE function_id = $1
	ORDER BY version DESC
	LIMIT 1
	`

	deployment := &store.Deployment{}
	var bundlePath *string

	err := s.pool.QueryRow(ctx, sql, functionID).Scan(
		&deployment.ID,
		&deployment.FunctionID,
		&deployment.Version,
		&deployment.SourceCode,
		&bundlePath,
		&deployment.Status,
		&deployment.DeployedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("no deployments found")
	}
	if err != nil {
		return nil, err
	}

	if bundlePath != nil {
		deployment.BundlePath = *bundlePath
	}

	return deployment, nil
}

// ListDeployments lists deployments for a function.
func (s *FunctionsStore) ListDeployments(ctx context.Context, functionID string, limit int) ([]*store.Deployment, error) {
	sql := `
	SELECT id, function_id, version, source_code, bundle_path, status, deployed_at
	FROM functions.deployments
	WHERE function_id = $1
	ORDER BY version DESC
	LIMIT $2
	`

	rows, err := s.pool.Query(ctx, sql, functionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialize with empty slice instead of nil to return [] instead of null in JSON
	deployments := make([]*store.Deployment, 0)
	for rows.Next() {
		deployment := &store.Deployment{}
		var bundlePath *string

		err := rows.Scan(
			&deployment.ID,
			&deployment.FunctionID,
			&deployment.Version,
			&deployment.SourceCode,
			&bundlePath,
			&deployment.Status,
			&deployment.DeployedAt,
		)
		if err != nil {
			return nil, err
		}

		if bundlePath != nil {
			deployment.BundlePath = *bundlePath
		}

		deployments = append(deployments, deployment)
	}

	return deployments, nil
}

// UpdateDeployment updates a deployment.
func (s *FunctionsStore) UpdateDeployment(ctx context.Context, deployment *store.Deployment) error {
	sql := `
	UPDATE functions.deployments
	SET status = $2, bundle_path = $3
	WHERE id = $1
	`

	_, err := s.pool.Exec(ctx, sql,
		deployment.ID,
		deployment.Status,
		nullIfEmpty(deployment.BundlePath),
	)

	return err
}

// CreateSecret creates a new secret.
func (s *FunctionsStore) CreateSecret(ctx context.Context, secret *store.Secret) error {
	sql := `
	INSERT INTO functions.secrets (id, name, value)
	VALUES ($1, $2, $3)
	ON CONFLICT (name) DO UPDATE SET value = $3
	`

	_, err := s.pool.Exec(ctx, sql,
		secret.ID,
		secret.Name,
		secret.Value,
	)

	return err
}

// GetSecret retrieves a secret by name.
func (s *FunctionsStore) GetSecret(ctx context.Context, name string) (*store.Secret, error) {
	sql := `
	SELECT id, name, value, created_at
	FROM functions.secrets
	WHERE name = $1
	`

	secret := &store.Secret{}

	err := s.pool.QueryRow(ctx, sql, name).Scan(
		&secret.ID,
		&secret.Name,
		&secret.Value,
		&secret.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("secret not found")
	}
	if err != nil {
		return nil, err
	}

	return secret, nil
}

// ListSecrets lists all secrets (without values).
func (s *FunctionsStore) ListSecrets(ctx context.Context) ([]*store.Secret, error) {
	sql := `
	SELECT id, name, created_at
	FROM functions.secrets
	ORDER BY name
	`

	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialize with empty slice instead of nil to return [] instead of null in JSON
	secrets := make([]*store.Secret, 0)
	for rows.Next() {
		secret := &store.Secret{}

		err := rows.Scan(
			&secret.ID,
			&secret.Name,
			&secret.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		secrets = append(secrets, secret)
	}

	return secrets, nil
}

// UpdateSecret updates a secret.
func (s *FunctionsStore) UpdateSecret(ctx context.Context, secret *store.Secret) error {
	sql := `UPDATE functions.secrets SET value = $2 WHERE name = $1`
	_, err := s.pool.Exec(ctx, sql, secret.Name, secret.Value)
	return err
}

// DeleteSecret deletes a secret.
func (s *FunctionsStore) DeleteSecret(ctx context.Context, name string) error {
	sql := `DELETE FROM functions.secrets WHERE name = $1`
	_, err := s.pool.Exec(ctx, sql, name)
	return err
}

// BulkUpsertSecrets creates or updates multiple secrets at once.
func (s *FunctionsStore) BulkUpsertSecrets(ctx context.Context, secrets []*store.Secret) (created, updated int, err error) {
	for _, secret := range secrets {
		// Check if exists
		var exists bool
		err = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM functions.secrets WHERE name = $1)`, secret.Name).Scan(&exists)
		if err != nil {
			return created, updated, err
		}

		if exists {
			_, err = s.pool.Exec(ctx, `UPDATE functions.secrets SET value = $2 WHERE name = $1`, secret.Name, secret.Value)
			if err != nil {
				return created, updated, err
			}
			updated++
		} else {
			_, err = s.pool.Exec(ctx, `INSERT INTO functions.secrets (id, name, value) VALUES ($1, $2, $3)`,
				secret.ID, secret.Name, secret.Value)
			if err != nil {
				return created, updated, err
			}
			created++
		}
	}
	return created, updated, nil
}

// ListFunctionsWithLatestDeployment lists all functions with their latest deployment.
func (s *FunctionsStore) ListFunctionsWithLatestDeployment(ctx context.Context) ([]*store.Function, error) {
	sql := `
	SELECT
		f.id, f.name, f.slug, f.version, f.status, f.entrypoint, f.import_map, f.verify_jwt,
		f.draft_source, f.draft_import_map, f.created_at, f.updated_at,
		d.id as deploy_id, d.version as deploy_version, d.status as deploy_status, d.deployed_at
	FROM functions.functions f
	LEFT JOIN LATERAL (
		SELECT id, version, status, deployed_at
		FROM functions.deployments
		WHERE function_id = f.id
		ORDER BY version DESC
		LIMIT 1
	) d ON true
	ORDER BY f.name
	`

	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	functions := make([]*store.Function, 0)
	for rows.Next() {
		fn := &store.Function{}
		var importMap, draftSource, draftImportMap *string
		var deployID, deployStatus *string
		var deployVersion *int
		var deployedAt *time.Time

		err := rows.Scan(
			&fn.ID, &fn.Name, &fn.Slug, &fn.Version, &fn.Status, &fn.Entrypoint,
			&importMap, &fn.VerifyJWT, &draftSource, &draftImportMap,
			&fn.CreatedAt, &fn.UpdatedAt,
			&deployID, &deployVersion, &deployStatus, &deployedAt,
		)
		if err != nil {
			return nil, err
		}

		if importMap != nil {
			fn.ImportMap = *importMap
		}
		if draftSource != nil {
			fn.DraftSource = *draftSource
		}
		if draftImportMap != nil {
			fn.DraftImportMap = *draftImportMap
		}

		// Attach latest deployment if exists
		if deployID != nil {
			fn.LatestDeployment = &store.Deployment{
				ID:         *deployID,
				FunctionID: fn.ID,
				Version:    *deployVersion,
				Status:     *deployStatus,
				DeployedAt: *deployedAt,
			}
		}

		functions = append(functions, fn)
	}

	return functions, nil
}

// SaveDraftSource saves draft source code without deploying.
func (s *FunctionsStore) SaveDraftSource(ctx context.Context, functionID, source, importMap string) error {
	sql := `UPDATE functions.functions SET draft_source = $2, draft_import_map = $3, updated_at = NOW() WHERE id = $1`
	_, err := s.pool.Exec(ctx, sql, functionID, nullIfEmpty(source), nullIfEmpty(importMap))
	return err
}

// ClearDraftSource clears the draft source code (usually after deploy).
func (s *FunctionsStore) ClearDraftSource(ctx context.Context, functionID string) error {
	sql := `UPDATE functions.functions SET draft_source = NULL, draft_import_map = NULL WHERE id = $1`
	_, err := s.pool.Exec(ctx, sql, functionID)
	return err
}

// CreateFunctionLog creates a new function execution log entry.
func (s *FunctionsStore) CreateFunctionLog(ctx context.Context, log *store.FunctionLog) error {
	sql := `
	INSERT INTO functions.logs (id, function_id, request_id, timestamp, level, message, duration_ms, status_code, region, metadata)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	var requestID any
	if log.RequestID != "" {
		requestID = log.RequestID
	}

	_, err := s.pool.Exec(ctx, sql,
		log.ID, log.FunctionID, requestID, log.Timestamp, log.Level, log.Message,
		log.DurationMs, log.StatusCode, log.Region, log.Metadata,
	)
	return err
}

// ListFunctionLogs lists function execution logs with optional filtering.
func (s *FunctionsStore) ListFunctionLogs(ctx context.Context, functionID string, limit int, level string, since *time.Time) ([]*store.FunctionLog, error) {
	sql := `
	SELECT id, function_id, request_id, timestamp, level, message, duration_ms, status_code, region, metadata
	FROM functions.logs
	WHERE function_id = $1
	`
	args := []any{functionID}
	argNum := 2

	if level != "" {
		sql += fmt.Sprintf(" AND level = $%d", argNum)
		args = append(args, level)
		argNum++
	}

	if since != nil {
		sql += fmt.Sprintf(" AND timestamp >= $%d", argNum)
		args = append(args, *since)
		argNum++
	}

	sql += " ORDER BY timestamp DESC"

	if limit > 0 {
		sql += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, limit)
	}

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]*store.FunctionLog, 0)
	for rows.Next() {
		log := &store.FunctionLog{}
		var requestID *string
		var durationMs, statusCode *int
		var region *string
		var metadata map[string]any

		err := rows.Scan(
			&log.ID, &log.FunctionID, &requestID, &log.Timestamp, &log.Level, &log.Message,
			&durationMs, &statusCode, &region, &metadata,
		)
		if err != nil {
			return nil, err
		}

		if requestID != nil {
			log.RequestID = *requestID
		}
		if durationMs != nil {
			log.DurationMs = *durationMs
		}
		if statusCode != nil {
			log.StatusCode = *statusCode
		}
		if region != nil {
			log.Region = *region
		}
		log.Metadata = metadata

		logs = append(logs, log)
	}

	return logs, nil
}

// RecordFunctionInvocation records a function invocation for metrics aggregation.
func (s *FunctionsStore) RecordFunctionInvocation(ctx context.Context, functionID string, durationMs int, success bool) error {
	// Round to current hour
	now := time.Now().UTC().Truncate(time.Hour)

	sql := `
	INSERT INTO functions.metrics (id, function_id, hour, invocations, successes, errors, total_duration_ms)
	VALUES (gen_random_uuid(), $1, $2, 1, $3, $4, $5)
	ON CONFLICT (function_id, hour) DO UPDATE SET
		invocations = functions.metrics.invocations + 1,
		successes = functions.metrics.successes + $3,
		errors = functions.metrics.errors + $4,
		total_duration_ms = functions.metrics.total_duration_ms + $5
	`

	successInc := 0
	errorInc := 0
	if success {
		successInc = 1
	} else {
		errorInc = 1
	}

	_, err := s.pool.Exec(ctx, sql, functionID, now, successInc, errorInc, durationMs)
	return err
}

// GetFunctionMetrics retrieves function metrics for a time range.
func (s *FunctionsStore) GetFunctionMetrics(ctx context.Context, functionID string, from, to time.Time) ([]*store.FunctionMetrics, error) {
	sql := `
	SELECT function_id, hour, invocations, successes, errors, total_duration_ms, p50_latency, p95_latency, p99_latency
	FROM functions.metrics
	WHERE function_id = $1 AND hour >= $2 AND hour <= $3
	ORDER BY hour
	`

	rows, err := s.pool.Query(ctx, sql, functionID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics := make([]*store.FunctionMetrics, 0)
	for rows.Next() {
		m := &store.FunctionMetrics{}
		var p50, p95, p99 *int

		err := rows.Scan(
			&m.FunctionID, &m.Hour, &m.Invocations, &m.Successes, &m.Errors,
			&m.TotalDurationMs, &p50, &p95, &p99,
		)
		if err != nil {
			return nil, err
		}

		if p50 != nil {
			m.P50Latency = *p50
		}
		if p95 != nil {
			m.P95Latency = *p95
		}
		if p99 != nil {
			m.P99Latency = *p99
		}

		metrics = append(metrics, m)
	}

	return metrics, nil
}
