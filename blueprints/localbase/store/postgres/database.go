package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseStore implements store.DatabaseStore using PostgreSQL.
type DatabaseStore struct {
	pool *pgxpool.Pool
}

// ListTables lists all tables in a schema.
func (s *DatabaseStore) ListTables(ctx context.Context, schema string) ([]*store.Table, error) {
	if schema == "" {
		schema = "public"
	}

	sql := `
	SELECT
		c.oid::int AS id,
		n.nspname AS schema,
		c.relname AS name,
		pg_stat_get_live_tuples(c.oid) AS row_count,
		pg_total_relation_size(c.oid) AS size_bytes,
		COALESCE(obj_description(c.oid), '') AS comment,
		c.relrowsecurity AS rls_enabled
	FROM pg_class c
	JOIN pg_namespace n ON n.oid = c.relnamespace
	WHERE n.nspname = $1
		AND c.relkind = 'r'
	ORDER BY c.relname
	`

	rows, err := s.pool.Query(ctx, sql, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []*store.Table
	for rows.Next() {
		table := &store.Table{}

		err := rows.Scan(
			&table.ID,
			&table.Schema,
			&table.Name,
			&table.RowCount,
			&table.SizeBytes,
			&table.Comment,
			&table.RLSEnabled,
		)
		if err != nil {
			return nil, err
		}

		tables = append(tables, table)
	}

	return tables, nil
}

// GetTable retrieves a table with its columns.
func (s *DatabaseStore) GetTable(ctx context.Context, schema, name string) (*store.Table, error) {
	sql := `
	SELECT
		c.oid::int AS id,
		n.nspname AS schema,
		c.relname AS name,
		pg_stat_get_live_tuples(c.oid) AS row_count,
		pg_total_relation_size(c.oid) AS size_bytes,
		COALESCE(obj_description(c.oid), '') AS comment,
		c.relrowsecurity AS rls_enabled
	FROM pg_class c
	JOIN pg_namespace n ON n.oid = c.relnamespace
	WHERE n.nspname = $1 AND c.relname = $2
	`

	table := &store.Table{}

	err := s.pool.QueryRow(ctx, sql, schema, name).Scan(
		&table.ID,
		&table.Schema,
		&table.Name,
		&table.RowCount,
		&table.SizeBytes,
		&table.Comment,
		&table.RLSEnabled,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("table not found")
	}
	if err != nil {
		return nil, err
	}

	// Get columns
	columns, err := s.ListColumns(ctx, schema, name)
	if err != nil {
		return nil, err
	}
	table.Columns = columns

	return table, nil
}

// CreateTable creates a new table.
func (s *DatabaseStore) CreateTable(ctx context.Context, schema, name string, columns []*store.Column) error {
	if schema == "" {
		schema = "public"
	}

	var colDefs []string
	for _, col := range columns {
		def := fmt.Sprintf("%s %s", quoteIdent(col.Name), col.Type)
		if !col.IsNullable {
			def += " NOT NULL"
		}
		if col.DefaultValue != "" {
			def += " DEFAULT " + col.DefaultValue
		}
		if col.IsPrimaryKey {
			def += " PRIMARY KEY"
		}
		if col.IsUnique && !col.IsPrimaryKey {
			def += " UNIQUE"
		}
		colDefs = append(colDefs, def)
	}

	sql := fmt.Sprintf("CREATE TABLE %s.%s (%s)",
		quoteIdent(schema),
		quoteIdent(name),
		strings.Join(colDefs, ", "),
	)

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// DropTable drops a table.
func (s *DatabaseStore) DropTable(ctx context.Context, schema, name string) error {
	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s.%s CASCADE", quoteIdent(schema), quoteIdent(name))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// ListColumns lists all columns in a table.
func (s *DatabaseStore) ListColumns(ctx context.Context, schema, table string) ([]*store.Column, error) {
	sql := `
	SELECT
		a.attname AS name,
		pg_catalog.format_type(a.atttypid, a.atttypmod) AS type,
		COALESCE(pg_get_expr(d.adbin, d.adrelid), '') AS default_value,
		NOT a.attnotnull AS is_nullable,
		COALESCE(pk.is_pk, false) AS is_primary_key,
		COALESCE(uq.is_unique, false) AS is_unique,
		COALESCE(col_description(c.oid, a.attnum), '') AS comment
	FROM pg_class c
	JOIN pg_namespace n ON n.oid = c.relnamespace
	JOIN pg_attribute a ON a.attrelid = c.oid
	LEFT JOIN pg_attrdef d ON d.adrelid = c.oid AND d.adnum = a.attnum
	LEFT JOIN (
		SELECT i.indrelid, unnest(i.indkey) AS attnum, true AS is_pk
		FROM pg_index i WHERE i.indisprimary
	) pk ON pk.indrelid = c.oid AND pk.attnum = a.attnum
	LEFT JOIN (
		SELECT i.indrelid, unnest(i.indkey) AS attnum, true AS is_unique
		FROM pg_index i WHERE i.indisunique AND NOT i.indisprimary
	) uq ON uq.indrelid = c.oid AND uq.attnum = a.attnum
	WHERE n.nspname = $1
		AND c.relname = $2
		AND a.attnum > 0
		AND NOT a.attisdropped
	ORDER BY a.attnum
	`

	rows, err := s.pool.Query(ctx, sql, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []*store.Column
	for rows.Next() {
		col := &store.Column{}

		err := rows.Scan(
			&col.Name,
			&col.Type,
			&col.DefaultValue,
			&col.IsNullable,
			&col.IsPrimaryKey,
			&col.IsUnique,
			&col.Comment,
		)
		if err != nil {
			return nil, err
		}

		columns = append(columns, col)
	}

	return columns, nil
}

// AddColumn adds a column to a table.
func (s *DatabaseStore) AddColumn(ctx context.Context, schema, table string, column *store.Column) error {
	sql := fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN %s %s",
		quoteIdent(schema),
		quoteIdent(table),
		quoteIdent(column.Name),
		column.Type,
	)

	if !column.IsNullable {
		sql += " NOT NULL"
	}
	if column.DefaultValue != "" {
		sql += " DEFAULT " + column.DefaultValue
	}

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// AlterColumn alters a column.
func (s *DatabaseStore) AlterColumn(ctx context.Context, schema, table string, column *store.Column) error {
	// Change type
	sql := fmt.Sprintf("ALTER TABLE %s.%s ALTER COLUMN %s TYPE %s",
		quoteIdent(schema),
		quoteIdent(table),
		quoteIdent(column.Name),
		column.Type,
	)

	if _, err := s.pool.Exec(ctx, sql); err != nil {
		return err
	}

	// Set/drop NOT NULL
	if column.IsNullable {
		sql = fmt.Sprintf("ALTER TABLE %s.%s ALTER COLUMN %s DROP NOT NULL",
			quoteIdent(schema), quoteIdent(table), quoteIdent(column.Name))
	} else {
		sql = fmt.Sprintf("ALTER TABLE %s.%s ALTER COLUMN %s SET NOT NULL",
			quoteIdent(schema), quoteIdent(table), quoteIdent(column.Name))
	}

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// DropColumn drops a column from a table.
func (s *DatabaseStore) DropColumn(ctx context.Context, schema, table, column string) error {
	sql := fmt.Sprintf("ALTER TABLE %s.%s DROP COLUMN %s",
		quoteIdent(schema),
		quoteIdent(table),
		quoteIdent(column),
	)

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// ListIndexes lists all indexes on a table.
func (s *DatabaseStore) ListIndexes(ctx context.Context, schema, table string) ([]*store.Index, error) {
	sql := `
	SELECT
		i.relname AS name,
		n.nspname AS schema,
		t.relname AS table,
		array_agg(a.attname ORDER BY k.n) AS columns,
		ix.indisunique AS is_unique,
		ix.indisprimary AS is_primary,
		am.amname AS type
	FROM pg_index ix
	JOIN pg_class i ON i.oid = ix.indexrelid
	JOIN pg_class t ON t.oid = ix.indrelid
	JOIN pg_namespace n ON n.oid = t.relnamespace
	JOIN pg_am am ON am.oid = i.relam
	JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS k(attnum, n) ON true
	JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum
	WHERE n.nspname = $1 AND t.relname = $2
	GROUP BY i.relname, n.nspname, t.relname, ix.indisunique, ix.indisprimary, am.amname
	ORDER BY i.relname
	`

	rows, err := s.pool.Query(ctx, sql, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []*store.Index
	for rows.Next() {
		idx := &store.Index{}

		err := rows.Scan(
			&idx.Name,
			&idx.Schema,
			&idx.Table,
			&idx.Columns,
			&idx.IsUnique,
			&idx.IsPrimary,
			&idx.Type,
		)
		if err != nil {
			return nil, err
		}

		indexes = append(indexes, idx)
	}

	return indexes, nil
}

// CreateIndex creates a new index.
func (s *DatabaseStore) CreateIndex(ctx context.Context, index *store.Index) error {
	unique := ""
	if index.IsUnique {
		unique = "UNIQUE "
	}

	var quotedCols []string
	for _, col := range index.Columns {
		quotedCols = append(quotedCols, quoteIdent(col))
	}

	sql := fmt.Sprintf("CREATE %sINDEX %s ON %s.%s USING %s (%s)",
		unique,
		quoteIdent(index.Name),
		quoteIdent(index.Schema),
		quoteIdent(index.Table),
		index.Type,
		strings.Join(quotedCols, ", "),
	)

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// DropIndex drops an index.
func (s *DatabaseStore) DropIndex(ctx context.Context, schema, name string) error {
	sql := fmt.Sprintf("DROP INDEX IF EXISTS %s.%s", quoteIdent(schema), quoteIdent(name))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// ListForeignKeys lists all foreign keys on a table.
func (s *DatabaseStore) ListForeignKeys(ctx context.Context, schema, table string) ([]*store.ForeignKey, error) {
	sql := `
	SELECT
		c.conname AS name,
		n1.nspname AS schema,
		t1.relname AS table,
		a1.attname AS column,
		n2.nspname AS target_schema,
		t2.relname AS target_table,
		a2.attname AS target_column,
		CASE c.confdeltype
			WHEN 'a' THEN 'NO ACTION'
			WHEN 'r' THEN 'RESTRICT'
			WHEN 'c' THEN 'CASCADE'
			WHEN 'n' THEN 'SET NULL'
			WHEN 'd' THEN 'SET DEFAULT'
		END AS on_delete,
		CASE c.confupdtype
			WHEN 'a' THEN 'NO ACTION'
			WHEN 'r' THEN 'RESTRICT'
			WHEN 'c' THEN 'CASCADE'
			WHEN 'n' THEN 'SET NULL'
			WHEN 'd' THEN 'SET DEFAULT'
		END AS on_update
	FROM pg_constraint c
	JOIN pg_class t1 ON t1.oid = c.conrelid
	JOIN pg_namespace n1 ON n1.oid = t1.relnamespace
	JOIN pg_class t2 ON t2.oid = c.confrelid
	JOIN pg_namespace n2 ON n2.oid = t2.relnamespace
	JOIN pg_attribute a1 ON a1.attrelid = c.conrelid AND a1.attnum = ANY(c.conkey)
	JOIN pg_attribute a2 ON a2.attrelid = c.confrelid AND a2.attnum = ANY(c.confkey)
	WHERE c.contype = 'f'
		AND n1.nspname = $1
		AND t1.relname = $2
	ORDER BY c.conname
	`

	rows, err := s.pool.Query(ctx, sql, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fks []*store.ForeignKey
	for rows.Next() {
		fk := &store.ForeignKey{}

		err := rows.Scan(
			&fk.Name,
			&fk.Schema,
			&fk.Table,
			&fk.Column,
			&fk.TargetSchema,
			&fk.TargetTable,
			&fk.TargetColumn,
			&fk.OnDelete,
			&fk.OnUpdate,
		)
		if err != nil {
			return nil, err
		}

		fks = append(fks, fk)
	}

	return fks, nil
}

// CreateForeignKey creates a new foreign key.
func (s *DatabaseStore) CreateForeignKey(ctx context.Context, fk *store.ForeignKey) error {
	sql := fmt.Sprintf(`
	ALTER TABLE %s.%s
	ADD CONSTRAINT %s
	FOREIGN KEY (%s) REFERENCES %s.%s(%s)
	ON DELETE %s ON UPDATE %s
	`,
		quoteIdent(fk.Schema),
		quoteIdent(fk.Table),
		quoteIdent(fk.Name),
		quoteIdent(fk.Column),
		quoteIdent(fk.TargetSchema),
		quoteIdent(fk.TargetTable),
		quoteIdent(fk.TargetColumn),
		fk.OnDelete,
		fk.OnUpdate,
	)

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// DropForeignKey drops a foreign key.
func (s *DatabaseStore) DropForeignKey(ctx context.Context, schema, table, name string) error {
	sql := fmt.Sprintf("ALTER TABLE %s.%s DROP CONSTRAINT %s",
		quoteIdent(schema), quoteIdent(table), quoteIdent(name))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// ListPolicies lists all RLS policies on a table.
func (s *DatabaseStore) ListPolicies(ctx context.Context, schema, table string) ([]*store.Policy, error) {
	sql := `
	SELECT
		p.oid::int AS id,
		p.polname AS name,
		n.nspname AS schema,
		c.relname AS table,
		CASE p.polcmd
			WHEN 'r' THEN 'SELECT'
			WHEN 'a' THEN 'INSERT'
			WHEN 'w' THEN 'UPDATE'
			WHEN 'd' THEN 'DELETE'
			WHEN '*' THEN 'ALL'
		END AS command,
		pg_get_expr(p.polqual, p.polrelid, true) AS definition,
		COALESCE(pg_get_expr(p.polwithcheck, p.polrelid, true), '') AS check_expression,
		COALESCE(array_agg(r.rolname) FILTER (WHERE r.rolname IS NOT NULL), ARRAY[]::text[]) AS roles
	FROM pg_policy p
	JOIN pg_class c ON c.oid = p.polrelid
	JOIN pg_namespace n ON n.oid = c.relnamespace
	LEFT JOIN pg_roles r ON r.oid = ANY(p.polroles)
	WHERE n.nspname = $1 AND c.relname = $2
	GROUP BY p.oid, p.polname, n.nspname, c.relname, p.polcmd, p.polqual, p.polrelid, p.polwithcheck
	ORDER BY p.polname
	`

	rows, err := s.pool.Query(ctx, sql, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*store.Policy
	for rows.Next() {
		policy := &store.Policy{}

		err := rows.Scan(
			&policy.ID,
			&policy.Name,
			&policy.Schema,
			&policy.Table,
			&policy.Command,
			&policy.Definition,
			&policy.CheckExpr,
			&policy.Roles,
		)
		if err != nil {
			return nil, err
		}

		policies = append(policies, policy)
	}

	return policies, nil
}

// CreatePolicy creates a new RLS policy.
func (s *DatabaseStore) CreatePolicy(ctx context.Context, policy *store.Policy) error {
	sql := fmt.Sprintf("CREATE POLICY %s ON %s.%s FOR %s",
		quoteIdent(policy.Name),
		quoteIdent(policy.Schema),
		quoteIdent(policy.Table),
		policy.Command,
	)

	if len(policy.Roles) > 0 {
		var quotedRoles []string
		for _, role := range policy.Roles {
			quotedRoles = append(quotedRoles, quoteIdent(role))
		}
		sql += " TO " + strings.Join(quotedRoles, ", ")
	}

	if policy.Definition != "" {
		sql += " USING (" + policy.Definition + ")"
	}

	if policy.CheckExpr != "" {
		sql += " WITH CHECK (" + policy.CheckExpr + ")"
	}

	_, err := s.pool.Exec(ctx, sql)
	return err
}

// DropPolicy drops an RLS policy.
func (s *DatabaseStore) DropPolicy(ctx context.Context, schema, table, name string) error {
	sql := fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s.%s",
		quoteIdent(name), quoteIdent(schema), quoteIdent(table))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// EnableRLS enables RLS on a table.
func (s *DatabaseStore) EnableRLS(ctx context.Context, schema, table string) error {
	sql := fmt.Sprintf("ALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY",
		quoteIdent(schema), quoteIdent(table))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// DisableRLS disables RLS on a table.
func (s *DatabaseStore) DisableRLS(ctx context.Context, schema, table string) error {
	sql := fmt.Sprintf("ALTER TABLE %s.%s DISABLE ROW LEVEL SECURITY",
		quoteIdent(schema), quoteIdent(table))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// ListExtensions lists all available and installed extensions.
func (s *DatabaseStore) ListExtensions(ctx context.Context) ([]*store.Extension, error) {
	sql := `
	SELECT
		e.name,
		COALESCE(x.extversion, '') AS installed_version,
		e.default_version,
		COALESCE(e.comment, '') AS comment
	FROM pg_available_extensions e
	LEFT JOIN pg_extension x ON x.extname = e.name
	ORDER BY e.name
	`

	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var extensions []*store.Extension
	for rows.Next() {
		ext := &store.Extension{}

		err := rows.Scan(
			&ext.Name,
			&ext.InstalledVersion,
			&ext.DefaultVersion,
			&ext.Comment,
		)
		if err != nil {
			return nil, err
		}

		extensions = append(extensions, ext)
	}

	return extensions, nil
}

// EnableExtension enables an extension.
func (s *DatabaseStore) EnableExtension(ctx context.Context, name string) error {
	sql := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", quoteIdent(name))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// DisableExtension disables an extension.
func (s *DatabaseStore) DisableExtension(ctx context.Context, name string) error {
	sql := fmt.Sprintf("DROP EXTENSION IF EXISTS %s CASCADE", quoteIdent(name))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// Query executes a SQL query and returns results.
func (s *DatabaseStore) Query(ctx context.Context, sql string, params ...interface{}) (*store.QueryResult, error) {
	start := time.Now()

	rows, err := s.pool.Query(ctx, sql, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column names
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}

	// Collect rows
	var results []map[string]interface{}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	// Check for errors that occurred during iteration
	// This is important for catching constraint violations in INSERT/UPDATE statements
	if err := rows.Err(); err != nil {
		return nil, err
	}

	duration := time.Since(start).Seconds() * 1000

	return &store.QueryResult{
		Columns:  columns,
		Rows:     results,
		RowCount: len(results),
		Duration: duration,
	}, nil
}

// Exec executes a SQL statement and returns affected rows.
func (s *DatabaseStore) Exec(ctx context.Context, sql string, params ...interface{}) (int64, error) {
	result, err := s.pool.Exec(ctx, sql, params...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// ListSchemas lists all schemas.
func (s *DatabaseStore) ListSchemas(ctx context.Context) ([]string, error) {
	sql := `
	SELECT nspname
	FROM pg_namespace
	WHERE nspname NOT LIKE 'pg_%' AND nspname != 'information_schema'
	ORDER BY nspname
	`

	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		schemas = append(schemas, name)
	}

	return schemas, nil
}

// CreateSchema creates a new schema.
func (s *DatabaseStore) CreateSchema(ctx context.Context, name string) error {
	sql := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", quoteIdent(name))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// DropSchema drops a schema.
func (s *DatabaseStore) DropSchema(ctx context.Context, name string) error {
	sql := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quoteIdent(name))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// TableExists checks if a table exists in the given schema.
func (s *DatabaseStore) TableExists(ctx context.Context, schema, table string) (bool, error) {
	if schema == "" {
		schema = "public"
	}

	sql := `
	SELECT EXISTS (
		SELECT 1
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1
			AND c.relname = $2
			AND c.relkind IN ('r', 'v', 'm')
	)
	`

	var exists bool
	err := s.pool.QueryRow(ctx, sql, schema, table).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetForeignKeysForEmbedding returns foreign key relationships for resource embedding.
func (s *DatabaseStore) GetForeignKeysForEmbedding(ctx context.Context, schema, table string) ([]store.ForeignKeyInfo, error) {
	sql := `
	SELECT
		c.conname AS constraint_name,
		a1.attname AS column_name,
		n2.nspname AS foreign_schema,
		t2.relname AS foreign_table,
		a2.attname AS foreign_column
	FROM pg_constraint c
	JOIN pg_class t1 ON t1.oid = c.conrelid
	JOIN pg_namespace n1 ON n1.oid = t1.relnamespace
	JOIN pg_class t2 ON t2.oid = c.confrelid
	JOIN pg_namespace n2 ON n2.oid = t2.relnamespace
	JOIN pg_attribute a1 ON a1.attrelid = c.conrelid AND a1.attnum = ANY(c.conkey)
	JOIN pg_attribute a2 ON a2.attrelid = c.confrelid AND a2.attnum = ANY(c.confkey)
	WHERE c.contype = 'f'
		AND n1.nspname = $1
		AND t1.relname = $2
	ORDER BY c.conname
	`

	rows, err := s.pool.Query(ctx, sql, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fks []store.ForeignKeyInfo
	for rows.Next() {
		var fk store.ForeignKeyInfo
		err := rows.Scan(
			&fk.ConstraintName,
			&fk.ColumnName,
			&fk.ForeignSchema,
			&fk.ForeignTable,
			&fk.ForeignColumn,
		)
		if err != nil {
			return nil, err
		}
		fks = append(fks, fk)
	}

	return fks, nil
}

// quoteIdent safely quotes an identifier.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// RLSContext holds the JWT claims and role for RLS enforcement.
type RLSContext struct {
	Role       string // anon, authenticated, service_role
	UserID     string // auth.uid() - the sub claim
	Email      string // auth.email() - the email claim
	ClaimsJSON string // Full JWT claims as JSON for request.jwt.claims
}

// QueryWithRLS executes a query with RLS context set from JWT claims.
// For service_role, RLS is bypassed.
func (s *DatabaseStore) QueryWithRLS(ctx context.Context, rlsCtx *RLSContext, sql string, params ...interface{}) (*store.QueryResult, error) {
	// service_role bypasses RLS - execute directly
	if rlsCtx == nil || rlsCtx.Role == "service_role" {
		return s.Query(ctx, sql, params...)
	}

	start := time.Now()

	// Use a transaction to set GUC variables that affect RLS
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set JWT claims for RLS policies
	if err := s.setRLSContext(ctx, tx, rlsCtx); err != nil {
		return nil, fmt.Errorf("failed to set RLS context: %w", err)
	}

	// Execute the query within the transaction
	rows, err := tx.Query(ctx, sql, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column names
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}

	// Collect rows
	var results []map[string]interface{}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	// Check for errors that occurred during iteration
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Commit the transaction (read-only, but completes the transaction)
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	duration := time.Since(start).Seconds() * 1000

	return &store.QueryResult{
		Columns:  columns,
		Rows:     results,
		RowCount: len(results),
		Duration: duration,
	}, nil
}

// ExecWithRLS executes a statement with RLS context set from JWT claims.
// For service_role, RLS is bypassed.
func (s *DatabaseStore) ExecWithRLS(ctx context.Context, rlsCtx *RLSContext, sql string, params ...interface{}) (int64, error) {
	// service_role bypasses RLS - execute directly
	if rlsCtx == nil || rlsCtx.Role == "service_role" {
		return s.Exec(ctx, sql, params...)
	}

	// Use a transaction to set GUC variables that affect RLS
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set JWT claims for RLS policies
	if err := s.setRLSContext(ctx, tx, rlsCtx); err != nil {
		return 0, fmt.Errorf("failed to set RLS context: %w", err)
	}

	// Execute the statement
	result, err := tx.Exec(ctx, sql, params...)
	if err != nil {
		return 0, err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result.RowsAffected(), nil
}

// setRLSContext sets the PostgreSQL GUC variables for RLS enforcement.
// This enables auth.uid(), auth.role(), auth.jwt() functions in RLS policies.
func (s *DatabaseStore) setRLSContext(ctx context.Context, tx pgx.Tx, rlsCtx *RLSContext) error {
	// Set the full JWT claims for request.jwt.claims
	// This is used by auth.jwt() and can be used in custom RLS policies
	if rlsCtx.ClaimsJSON != "" {
		_, err := tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, TRUE)", rlsCtx.ClaimsJSON)
		if err != nil {
			return fmt.Errorf("failed to set request.jwt.claims: %w", err)
		}
	}

	// Set individual claims for convenience functions
	if rlsCtx.UserID != "" {
		_, err := tx.Exec(ctx, "SELECT set_config('request.jwt.claim.sub', $1, TRUE)", rlsCtx.UserID)
		if err != nil {
			return fmt.Errorf("failed to set request.jwt.claim.sub: %w", err)
		}
	}

	if rlsCtx.Role != "" {
		_, err := tx.Exec(ctx, "SELECT set_config('request.jwt.claim.role', $1, TRUE)", rlsCtx.Role)
		if err != nil {
			return fmt.Errorf("failed to set request.jwt.claim.role: %w", err)
		}
	}

	if rlsCtx.Email != "" {
		_, err := tx.Exec(ctx, "SELECT set_config('request.jwt.claim.email', $1, TRUE)", rlsCtx.Email)
		if err != nil {
			return fmt.Errorf("failed to set request.jwt.claim.email: %w", err)
		}
	}

	return nil
}
