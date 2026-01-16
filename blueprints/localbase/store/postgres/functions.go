package postgres

import (
	"context"
	"fmt"

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

// GetFunctionByName retrieves a function by name.
func (s *FunctionsStore) GetFunctionByName(ctx context.Context, name string) (*store.Function, error) {
	sql := `SELECT id FROM functions.functions WHERE name = $1`

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

	var functions []*store.Function
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

	var deployments []*store.Deployment
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

	var secrets []*store.Secret
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
