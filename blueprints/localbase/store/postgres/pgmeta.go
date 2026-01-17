package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PGMetaStore implements postgres-meta API operations.
type PGMetaStore struct {
	pool *pgxpool.Pool
}

// =============================================================================
// Types
// =============================================================================

// Version represents PostgreSQL version info.
type Version struct {
	Version           string `json:"version"`
	VersionNumber     int    `json:"version_number"`
	ActiveConnections int    `json:"active_connections"`
	MaxConnections    int    `json:"max_connections"`
}

// Index represents a database index.
type Index struct {
	ID              int      `json:"id"`
	Schema          string   `json:"schema"`
	Table           string   `json:"table"`
	Name            string   `json:"name"`
	Columns         []string `json:"columns"`
	IsUnique        bool     `json:"is_unique"`
	IsPrimary       bool     `json:"is_primary"`
	IsExclusion     bool     `json:"is_exclusion"`
	IsValid         bool     `json:"is_valid"`
	IndexDefinition string   `json:"index_definition"`
}

// View represents a database view.
type View struct {
	ID          int    `json:"id"`
	Schema      string `json:"schema"`
	Name        string `json:"name"`
	IsUpdatable bool   `json:"is_updatable"`
	Comment     string `json:"comment"`
	Definition  string `json:"definition"`
}

// MaterializedView represents a materialized view.
type MaterializedView struct {
	ID          int    `json:"id"`
	Schema      string `json:"schema"`
	Name        string `json:"name"`
	IsPopulated bool   `json:"is_populated"`
	Definition  string `json:"definition"`
}

// ForeignTable represents a foreign table.
type ForeignTable struct {
	ID      int             `json:"id"`
	Schema  string          `json:"schema"`
	Name    string          `json:"name"`
	Server  string          `json:"server"`
	Columns []ForeignColumn `json:"columns"`
}

// ForeignColumn represents a column in a foreign table.
type ForeignColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Trigger represents a database trigger.
type Trigger struct {
	ID             int      `json:"id"`
	Name           string   `json:"name"`
	Schema         string   `json:"schema"`
	Table          string   `json:"table"`
	FunctionSchema string   `json:"function_schema"`
	FunctionName   string   `json:"function_name"`
	Events         []string `json:"events"`
	Orientation    string   `json:"orientation"`
	Timing         string   `json:"timing"`
	Condition      string   `json:"condition"`
	Enabled        bool     `json:"enabled"`
}

// Type represents a custom database type.
type Type struct {
	ID      int      `json:"id"`
	Schema  string   `json:"schema"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Enums   []string `json:"enums,omitempty"`
	Comment string   `json:"comment"`
}

// Role represents a database role.
type Role struct {
	ID                int               `json:"id"`
	Name              string            `json:"name"`
	IsSuperuser       bool              `json:"is_superuser"`
	CanCreateRole     bool              `json:"can_create_role"`
	CanCreateDB       bool              `json:"can_create_db"`
	CanLogin          bool              `json:"can_login"`
	IsReplicationRole bool              `json:"is_replication_role"`
	InheritRole       bool              `json:"inherit_role"`
	Config            map[string]string `json:"config"`
}

// Publication represents a logical replication publication.
type Publication struct {
	ID        int               `json:"id"`
	Name      string            `json:"name"`
	Owner     string            `json:"owner"`
	Tables    []PublicationTable `json:"tables"`
	AllTables bool              `json:"all_tables"`
	Insert    bool              `json:"insert"`
	Update    bool              `json:"update"`
	Delete    bool              `json:"delete"`
	Truncate  bool              `json:"truncate"`
}

// PublicationTable represents a table in a publication.
type PublicationTable struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
}

// TablePrivilege represents table-level privileges.
type TablePrivilege struct {
	Schema      string   `json:"schema"`
	Table       string   `json:"table"`
	Grantee     string   `json:"grantee"`
	Privileges  []string `json:"privileges"`
	IsGrantable bool     `json:"is_grantable"`
}

// ColumnPrivilege represents column-level privileges.
type ColumnPrivilege struct {
	Schema        string `json:"schema"`
	Table         string `json:"table"`
	Column        string `json:"column"`
	Grantee       string `json:"grantee"`
	PrivilegeType string `json:"privilege_type"`
}

// Constraint represents a database constraint.
type Constraint struct {
	ID         int      `json:"id"`
	Schema     string   `json:"schema"`
	Table      string   `json:"table"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Definition string   `json:"definition"`
	Columns    []string `json:"columns"`
	RefSchema  string   `json:"ref_schema,omitempty"`
	RefTable   string   `json:"ref_table,omitempty"`
	RefColumns []string `json:"ref_columns,omitempty"`
	OnUpdate   string   `json:"on_update,omitempty"`
	OnDelete   string   `json:"on_delete,omitempty"`
}

// PrimaryKey represents a primary key.
type PrimaryKey struct {
	Schema  string   `json:"schema"`
	Table   string   `json:"table"`
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}

// ForeignKeyInfo represents a foreign key relationship.
type ForeignKeyInfo struct {
	ID            int      `json:"id"`
	Schema        string   `json:"schema"`
	Table         string   `json:"table"`
	Name          string   `json:"name"`
	Columns       []string `json:"columns"`
	TargetSchema  string   `json:"target_schema"`
	TargetTable   string   `json:"target_table"`
	TargetColumns []string `json:"target_columns"`
	OnUpdate      string   `json:"on_update"`
	OnDelete      string   `json:"on_delete"`
}

// Relationship represents a table relationship.
type Relationship struct {
	ID             int      `json:"id"`
	SourceSchema   string   `json:"source_schema"`
	SourceTable    string   `json:"source_table"`
	SourceColumns  []string `json:"source_columns"`
	TargetSchema   string   `json:"target_schema"`
	TargetTable    string   `json:"target_table"`
	TargetColumns  []string `json:"target_columns"`
	ConstraintName string   `json:"constraint_name"`
}

// DatabaseFunction represents a database function.
type DatabaseFunction struct {
	ID         int    `json:"id"`
	Schema     string `json:"schema"`
	Name       string `json:"name"`
	Language   string `json:"language"`
	Definition string `json:"definition"`
	ReturnType string `json:"return_type"`
	Arguments  string `json:"arguments"`
	IsStrict   bool   `json:"is_strict"`
	Volatility string `json:"volatility"`
}

// =============================================================================
// Implementation
// =============================================================================

// GetVersion returns PostgreSQL version information.
func (s *PGMetaStore) GetVersion(ctx context.Context) (*Version, error) {
	var version Version

	// Get version string
	err := s.pool.QueryRow(ctx, "SELECT version()").Scan(&version.Version)
	if err != nil {
		return nil, err
	}

	// Get version number
	err = s.pool.QueryRow(ctx, "SHOW server_version_num").Scan(&version.VersionNumber)
	if err != nil {
		version.VersionNumber = 0
	}

	// Get connection stats
	err = s.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM pg_stat_activity WHERE state IS NOT NULL),
			(SELECT setting::int FROM pg_settings WHERE name = 'max_connections')
	`).Scan(&version.ActiveConnections, &version.MaxConnections)
	if err != nil {
		version.ActiveConnections = 0
		version.MaxConnections = 100
	}

	return &version, nil
}

// ListIndexes lists all indexes in the specified schemas.
func (s *PGMetaStore) ListIndexes(ctx context.Context, schemas []string) ([]*Index, error) {
	sql := `
	SELECT
		i.oid::int AS id,
		n.nspname AS schema,
		t.relname AS table,
		i.relname AS name,
		array_agg(a.attname ORDER BY k.n) AS columns,
		ix.indisunique AS is_unique,
		ix.indisprimary AS is_primary,
		ix.indisexclusion AS is_exclusion,
		ix.indisvalid AS is_valid,
		pg_get_indexdef(i.oid) AS index_definition
	FROM pg_index ix
	JOIN pg_class i ON i.oid = ix.indexrelid
	JOIN pg_class t ON t.oid = ix.indrelid
	JOIN pg_namespace n ON n.oid = t.relnamespace
	JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS k(attnum, n) ON true
	JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum
	WHERE n.nspname = ANY($1)
	GROUP BY i.oid, n.nspname, t.relname, i.relname, ix.indisunique, ix.indisprimary, ix.indisexclusion, ix.indisvalid
	ORDER BY n.nspname, t.relname, i.relname
	`

	rows, err := s.pool.Query(ctx, sql, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []*Index
	for rows.Next() {
		idx := &Index{}
		err := rows.Scan(
			&idx.ID, &idx.Schema, &idx.Table, &idx.Name, &idx.Columns,
			&idx.IsUnique, &idx.IsPrimary, &idx.IsExclusion, &idx.IsValid, &idx.IndexDefinition,
		)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, idx)
	}

	return indexes, nil
}

// ListViews lists all views in the specified schemas.
func (s *PGMetaStore) ListViews(ctx context.Context, schemas []string) ([]*View, error) {
	sql := `
	SELECT
		c.oid::int AS id,
		n.nspname AS schema,
		c.relname AS name,
		pg_catalog.pg_relation_is_updatable(c.oid, true)::int > 0 AS is_updatable,
		COALESCE(obj_description(c.oid), '') AS comment,
		pg_get_viewdef(c.oid, true) AS definition
	FROM pg_class c
	JOIN pg_namespace n ON n.oid = c.relnamespace
	WHERE c.relkind = 'v'
		AND n.nspname = ANY($1)
	ORDER BY n.nspname, c.relname
	`

	rows, err := s.pool.Query(ctx, sql, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []*View
	for rows.Next() {
		v := &View{}
		err := rows.Scan(&v.ID, &v.Schema, &v.Name, &v.IsUpdatable, &v.Comment, &v.Definition)
		if err != nil {
			return nil, err
		}
		views = append(views, v)
	}

	return views, nil
}

// CreateView creates a new view.
func (s *PGMetaStore) CreateView(ctx context.Context, schema, name, definition, checkOption string) (*View, error) {
	sql := fmt.Sprintf("CREATE VIEW %s.%s AS %s", quoteIdent(schema), quoteIdent(name), definition)
	if checkOption != "" {
		sql += " WITH " + checkOption + " CHECK OPTION"
	}

	_, err := s.pool.Exec(ctx, sql)
	if err != nil {
		return nil, err
	}

	// Retrieve the created view
	views, err := s.ListViews(ctx, []string{schema})
	if err != nil {
		return nil, err
	}
	for _, v := range views {
		if v.Name == name {
			return v, nil
		}
	}
	return &View{Schema: schema, Name: name, Definition: definition}, nil
}

// UpdateView updates a view.
func (s *PGMetaStore) UpdateView(ctx context.Context, id, definition, checkOption string) (*View, error) {
	// Parse id as schema.name
	schema := "public"
	name := id
	if strings.Contains(id, ".") {
		parts := strings.SplitN(id, ".", 2)
		schema, name = parts[0], parts[1]
	}

	sql := fmt.Sprintf("CREATE OR REPLACE VIEW %s.%s AS %s", quoteIdent(schema), quoteIdent(name), definition)
	if checkOption != "" {
		sql += " WITH " + checkOption + " CHECK OPTION"
	}

	_, err := s.pool.Exec(ctx, sql)
	if err != nil {
		return nil, err
	}

	return &View{Schema: schema, Name: name, Definition: definition}, nil
}

// DropView drops a view.
func (s *PGMetaStore) DropView(ctx context.Context, id string) error {
	schema := "public"
	name := id
	if strings.Contains(id, ".") {
		parts := strings.SplitN(id, ".", 2)
		schema, name = parts[0], parts[1]
	}

	sql := fmt.Sprintf("DROP VIEW IF EXISTS %s.%s CASCADE", quoteIdent(schema), quoteIdent(name))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// ListMaterializedViews lists all materialized views.
func (s *PGMetaStore) ListMaterializedViews(ctx context.Context, schemas []string) ([]*MaterializedView, error) {
	sql := `
	SELECT
		c.oid::int AS id,
		n.nspname AS schema,
		c.relname AS name,
		c.relispopulated AS is_populated,
		pg_get_viewdef(c.oid, true) AS definition
	FROM pg_class c
	JOIN pg_namespace n ON n.oid = c.relnamespace
	WHERE c.relkind = 'm'
		AND n.nspname = ANY($1)
	ORDER BY n.nspname, c.relname
	`

	rows, err := s.pool.Query(ctx, sql, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mvs []*MaterializedView
	for rows.Next() {
		mv := &MaterializedView{}
		err := rows.Scan(&mv.ID, &mv.Schema, &mv.Name, &mv.IsPopulated, &mv.Definition)
		if err != nil {
			return nil, err
		}
		mvs = append(mvs, mv)
	}

	return mvs, nil
}

// CreateMaterializedView creates a new materialized view.
func (s *PGMetaStore) CreateMaterializedView(ctx context.Context, schema, name, definition string) (*MaterializedView, error) {
	sql := fmt.Sprintf("CREATE MATERIALIZED VIEW %s.%s AS %s", quoteIdent(schema), quoteIdent(name), definition)

	_, err := s.pool.Exec(ctx, sql)
	if err != nil {
		return nil, err
	}

	return &MaterializedView{Schema: schema, Name: name, Definition: definition, IsPopulated: true}, nil
}

// RefreshMaterializedView refreshes a materialized view.
func (s *PGMetaStore) RefreshMaterializedView(ctx context.Context, id string) error {
	schema := "public"
	name := id
	if strings.Contains(id, ".") {
		parts := strings.SplitN(id, ".", 2)
		schema, name = parts[0], parts[1]
	}

	sql := fmt.Sprintf("REFRESH MATERIALIZED VIEW %s.%s", quoteIdent(schema), quoteIdent(name))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// DropMaterializedView drops a materialized view.
func (s *PGMetaStore) DropMaterializedView(ctx context.Context, id string) error {
	schema := "public"
	name := id
	if strings.Contains(id, ".") {
		parts := strings.SplitN(id, ".", 2)
		schema, name = parts[0], parts[1]
	}

	sql := fmt.Sprintf("DROP MATERIALIZED VIEW IF EXISTS %s.%s CASCADE", quoteIdent(schema), quoteIdent(name))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// ListForeignTables lists all foreign tables.
func (s *PGMetaStore) ListForeignTables(ctx context.Context, schemas []string) ([]*ForeignTable, error) {
	sql := `
	SELECT
		c.oid::int AS id,
		n.nspname AS schema,
		c.relname AS name,
		COALESCE(fs.srvname, '') AS server
	FROM pg_class c
	JOIN pg_namespace n ON n.oid = c.relnamespace
	LEFT JOIN pg_foreign_table ft ON ft.ftrelid = c.oid
	LEFT JOIN pg_foreign_server fs ON fs.oid = ft.ftserver
	WHERE c.relkind = 'f'
		AND n.nspname = ANY($1)
	ORDER BY n.nspname, c.relname
	`

	rows, err := s.pool.Query(ctx, sql, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []*ForeignTable
	for rows.Next() {
		ft := &ForeignTable{Columns: []ForeignColumn{}}
		err := rows.Scan(&ft.ID, &ft.Schema, &ft.Name, &ft.Server)
		if err != nil {
			return nil, err
		}
		tables = append(tables, ft)
	}

	return tables, nil
}

// ListTriggers lists all triggers.
func (s *PGMetaStore) ListTriggers(ctx context.Context, schemas []string) ([]*Trigger, error) {
	sql := `
	SELECT
		t.oid::int AS id,
		t.tgname AS name,
		n.nspname AS schema,
		c.relname AS table,
		np.nspname AS function_schema,
		p.proname AS function_name,
		ARRAY(
			SELECT unnest(ARRAY['INSERT', 'UPDATE', 'DELETE', 'TRUNCATE'])
			WHERE (t.tgtype & 4) > 0 AND 'INSERT' = 'INSERT'
			   OR (t.tgtype & 8) > 0 AND 'INSERT' = 'DELETE'
			   OR (t.tgtype & 16) > 0 AND 'INSERT' = 'UPDATE'
			   OR (t.tgtype & 32) > 0 AND 'INSERT' = 'TRUNCATE'
		) AS events,
		CASE WHEN t.tgtype & 1 > 0 THEN 'ROW' ELSE 'STATEMENT' END AS orientation,
		CASE
			WHEN t.tgtype & 2 > 0 THEN 'BEFORE'
			WHEN t.tgtype & 64 > 0 THEN 'INSTEAD OF'
			ELSE 'AFTER'
		END AS timing,
		COALESCE(pg_get_triggerdef(t.oid), '') AS condition,
		t.tgenabled != 'D' AS enabled
	FROM pg_trigger t
	JOIN pg_class c ON c.oid = t.tgrelid
	JOIN pg_namespace n ON n.oid = c.relnamespace
	JOIN pg_proc p ON p.oid = t.tgfoid
	JOIN pg_namespace np ON np.oid = p.pronamespace
	WHERE NOT t.tgisinternal
		AND n.nspname = ANY($1)
	ORDER BY n.nspname, c.relname, t.tgname
	`

	rows, err := s.pool.Query(ctx, sql, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []*Trigger
	for rows.Next() {
		t := &Trigger{}
		err := rows.Scan(
			&t.ID, &t.Name, &t.Schema, &t.Table, &t.FunctionSchema, &t.FunctionName,
			&t.Events, &t.Orientation, &t.Timing, &t.Condition, &t.Enabled,
		)
		if err != nil {
			return nil, err
		}
		triggers = append(triggers, t)
	}

	return triggers, nil
}

// CreateTrigger creates a new trigger.
func (s *PGMetaStore) CreateTrigger(ctx context.Context, name, schema, table, funcSchema, funcName string,
	events []string, timing, orientation, condition string) (*Trigger, error) {

	eventStr := strings.Join(events, " OR ")
	sql := fmt.Sprintf(
		"CREATE TRIGGER %s %s %s ON %s.%s FOR EACH %s EXECUTE FUNCTION %s.%s()",
		quoteIdent(name), timing, eventStr, quoteIdent(schema), quoteIdent(table),
		orientation, quoteIdent(funcSchema), quoteIdent(funcName),
	)

	_, err := s.pool.Exec(ctx, sql)
	if err != nil {
		return nil, err
	}

	return &Trigger{
		Name: name, Schema: schema, Table: table,
		FunctionSchema: funcSchema, FunctionName: funcName,
		Events: events, Timing: timing, Orientation: orientation,
		Enabled: true,
	}, nil
}

// DropTrigger drops a trigger.
func (s *PGMetaStore) DropTrigger(ctx context.Context, id string) error {
	// ID format: schema.table.trigger_name
	parts := strings.Split(id, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid trigger id: %s", id)
	}

	var schema, table, name string
	if len(parts) == 3 {
		schema, table, name = parts[0], parts[1], parts[2]
	} else {
		schema, table, name = "public", parts[0], parts[1]
	}

	sql := fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s.%s CASCADE", quoteIdent(name), quoteIdent(schema), quoteIdent(table))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// ListTypes lists all custom types.
func (s *PGMetaStore) ListTypes(ctx context.Context, schemas []string) ([]*Type, error) {
	sql := `
	SELECT
		t.oid::int AS id,
		n.nspname AS schema,
		t.typname AS name,
		CASE t.typtype
			WHEN 'e' THEN 'enum'
			WHEN 'c' THEN 'composite'
			WHEN 'd' THEN 'domain'
			WHEN 'r' THEN 'range'
			ELSE 'other'
		END AS type,
		COALESCE(
			ARRAY(SELECT e.enumlabel FROM pg_enum e WHERE e.enumtypid = t.oid ORDER BY e.enumsortorder),
			ARRAY[]::text[]
		) AS enums,
		COALESCE(obj_description(t.oid), '') AS comment
	FROM pg_type t
	JOIN pg_namespace n ON n.oid = t.typnamespace
	WHERE t.typtype IN ('e', 'c', 'd', 'r')
		AND n.nspname = ANY($1)
		AND NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.reltype = t.oid)
	ORDER BY n.nspname, t.typname
	`

	rows, err := s.pool.Query(ctx, sql, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []*Type
	for rows.Next() {
		t := &Type{}
		err := rows.Scan(&t.ID, &t.Schema, &t.Name, &t.Type, &t.Enums, &t.Comment)
		if err != nil {
			return nil, err
		}
		types = append(types, t)
	}

	return types, nil
}

// CreateType creates a new type.
func (s *PGMetaStore) CreateType(ctx context.Context, schema, name, typType string, values []string) (*Type, error) {
	var sql string
	switch typType {
	case "enum":
		quotedValues := make([]string, len(values))
		for i, v := range values {
			quotedValues[i] = "'" + strings.ReplaceAll(v, "'", "''") + "'"
		}
		sql = fmt.Sprintf("CREATE TYPE %s.%s AS ENUM (%s)",
			quoteIdent(schema), quoteIdent(name), strings.Join(quotedValues, ", "))
	default:
		return nil, fmt.Errorf("unsupported type: %s", typType)
	}

	_, err := s.pool.Exec(ctx, sql)
	if err != nil {
		return nil, err
	}

	return &Type{Schema: schema, Name: name, Type: typType, Enums: values}, nil
}

// DropType drops a type.
func (s *PGMetaStore) DropType(ctx context.Context, id string) error {
	schema := "public"
	name := id
	if strings.Contains(id, ".") {
		parts := strings.SplitN(id, ".", 2)
		schema, name = parts[0], parts[1]
	}

	sql := fmt.Sprintf("DROP TYPE IF EXISTS %s.%s CASCADE", quoteIdent(schema), quoteIdent(name))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// ListRoles lists all database roles.
func (s *PGMetaStore) ListRoles(ctx context.Context) ([]*Role, error) {
	sql := `
	SELECT
		r.oid::int AS id,
		r.rolname AS name,
		r.rolsuper AS is_superuser,
		r.rolcreaterole AS can_create_role,
		r.rolcreatedb AS can_create_db,
		r.rolcanlogin AS can_login,
		r.rolreplication AS is_replication_role,
		r.rolinherit AS inherit_role
	FROM pg_roles r
	WHERE r.rolname NOT LIKE 'pg_%'
	ORDER BY r.rolname
	`

	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*Role
	for rows.Next() {
		r := &Role{Config: make(map[string]string)}
		err := rows.Scan(
			&r.ID, &r.Name, &r.IsSuperuser, &r.CanCreateRole, &r.CanCreateDB,
			&r.CanLogin, &r.IsReplicationRole, &r.InheritRole,
		)
		if err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}

	return roles, nil
}

// CreateRole creates a new role.
func (s *PGMetaStore) CreateRole(ctx context.Context, name string, isSuperuser, canLogin bool, password string, inheritRole bool) (*Role, error) {
	var opts []string
	if isSuperuser {
		opts = append(opts, "SUPERUSER")
	} else {
		opts = append(opts, "NOSUPERUSER")
	}
	if canLogin {
		opts = append(opts, "LOGIN")
	} else {
		opts = append(opts, "NOLOGIN")
	}
	if inheritRole {
		opts = append(opts, "INHERIT")
	} else {
		opts = append(opts, "NOINHERIT")
	}
	if password != "" {
		opts = append(opts, fmt.Sprintf("PASSWORD '%s'", strings.ReplaceAll(password, "'", "''")))
	}

	sql := fmt.Sprintf("CREATE ROLE %s %s", quoteIdent(name), strings.Join(opts, " "))
	_, err := s.pool.Exec(ctx, sql)
	if err != nil {
		return nil, err
	}

	return &Role{Name: name, IsSuperuser: isSuperuser, CanLogin: canLogin, InheritRole: inheritRole}, nil
}

// UpdateRole updates a role.
func (s *PGMetaStore) UpdateRole(ctx context.Context, id int, isSuperuser, canLogin *bool, password *string, inheritRole *bool) (*Role, error) {
	// Get role name by ID
	var name string
	err := s.pool.QueryRow(ctx, "SELECT rolname FROM pg_roles WHERE oid = $1", id).Scan(&name)
	if err != nil {
		return nil, err
	}

	var opts []string
	if isSuperuser != nil {
		if *isSuperuser {
			opts = append(opts, "SUPERUSER")
		} else {
			opts = append(opts, "NOSUPERUSER")
		}
	}
	if canLogin != nil {
		if *canLogin {
			opts = append(opts, "LOGIN")
		} else {
			opts = append(opts, "NOLOGIN")
		}
	}
	if inheritRole != nil {
		if *inheritRole {
			opts = append(opts, "INHERIT")
		} else {
			opts = append(opts, "NOINHERIT")
		}
	}
	if password != nil {
		opts = append(opts, fmt.Sprintf("PASSWORD '%s'", strings.ReplaceAll(*password, "'", "''")))
	}

	if len(opts) > 0 {
		sql := fmt.Sprintf("ALTER ROLE %s %s", quoteIdent(name), strings.Join(opts, " "))
		_, err = s.pool.Exec(ctx, sql)
		if err != nil {
			return nil, err
		}
	}

	return &Role{ID: id, Name: name}, nil
}

// DropRole drops a role.
func (s *PGMetaStore) DropRole(ctx context.Context, id int) error {
	var name string
	err := s.pool.QueryRow(ctx, "SELECT rolname FROM pg_roles WHERE oid = $1", id).Scan(&name)
	if err != nil {
		return err
	}

	sql := fmt.Sprintf("DROP ROLE IF EXISTS %s", quoteIdent(name))
	_, err = s.pool.Exec(ctx, sql)
	return err
}

// ListPublications lists all publications.
func (s *PGMetaStore) ListPublications(ctx context.Context) ([]*Publication, error) {
	sql := `
	SELECT
		p.oid::int AS id,
		p.pubname AS name,
		r.rolname AS owner,
		p.puballtables AS all_tables,
		p.pubinsert AS insert,
		p.pubupdate AS update,
		p.pubdelete AS delete,
		p.pubtruncate AS truncate
	FROM pg_publication p
	JOIN pg_roles r ON r.oid = p.pubowner
	ORDER BY p.pubname
	`

	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pubs []*Publication
	for rows.Next() {
		p := &Publication{Tables: []PublicationTable{}}
		err := rows.Scan(
			&p.ID, &p.Name, &p.Owner, &p.AllTables,
			&p.Insert, &p.Update, &p.Delete, &p.Truncate,
		)
		if err != nil {
			return nil, err
		}
		pubs = append(pubs, p)
	}

	return pubs, nil
}

// CreatePublication creates a new publication.
func (s *PGMetaStore) CreatePublication(ctx context.Context, name string, allTables bool, tables []string,
	insert, update, delete, truncate bool) (*Publication, error) {

	var sql string
	if allTables {
		sql = fmt.Sprintf("CREATE PUBLICATION %s FOR ALL TABLES", quoteIdent(name))
	} else if len(tables) > 0 {
		sql = fmt.Sprintf("CREATE PUBLICATION %s FOR TABLE %s", quoteIdent(name), strings.Join(tables, ", "))
	} else {
		sql = fmt.Sprintf("CREATE PUBLICATION %s", quoteIdent(name))
	}

	// Add WITH options
	var opts []string
	if insert {
		opts = append(opts, "publish = 'insert'")
	}
	if update {
		opts = append(opts, "publish = 'update'")
	}
	if delete {
		opts = append(opts, "publish = 'delete'")
	}
	if truncate {
		opts = append(opts, "publish = 'truncate'")
	}

	_, err := s.pool.Exec(ctx, sql)
	if err != nil {
		return nil, err
	}

	return &Publication{Name: name, AllTables: allTables, Insert: insert, Update: update, Delete: delete, Truncate: truncate}, nil
}

// DropPublication drops a publication.
func (s *PGMetaStore) DropPublication(ctx context.Context, id string) error {
	sql := fmt.Sprintf("DROP PUBLICATION IF EXISTS %s", quoteIdent(id))
	_, err := s.pool.Exec(ctx, sql)
	return err
}

// ListTablePrivileges lists table-level privileges.
func (s *PGMetaStore) ListTablePrivileges(ctx context.Context, schemas []string) ([]*TablePrivilege, error) {
	sql := `
	SELECT
		n.nspname AS schema,
		c.relname AS table,
		grantee.rolname AS grantee,
		array_agg(DISTINCT acl.privilege_type) AS privileges,
		bool_or(acl.is_grantable) AS is_grantable
	FROM pg_class c
	JOIN pg_namespace n ON n.oid = c.relnamespace
	CROSS JOIN LATERAL aclexplode(c.relacl) AS acl
	JOIN pg_roles grantee ON grantee.oid = acl.grantee
	WHERE c.relkind IN ('r', 'v', 'm')
		AND n.nspname = ANY($1)
	GROUP BY n.nspname, c.relname, grantee.rolname
	ORDER BY n.nspname, c.relname, grantee.rolname
	`

	rows, err := s.pool.Query(ctx, sql, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var privs []*TablePrivilege
	for rows.Next() {
		p := &TablePrivilege{}
		err := rows.Scan(&p.Schema, &p.Table, &p.Grantee, &p.Privileges, &p.IsGrantable)
		if err != nil {
			return nil, err
		}
		privs = append(privs, p)
	}

	return privs, nil
}

// ListColumnPrivileges lists column-level privileges.
func (s *PGMetaStore) ListColumnPrivileges(ctx context.Context, schemas []string) ([]*ColumnPrivilege, error) {
	sql := `
	SELECT
		n.nspname AS schema,
		c.relname AS table,
		a.attname AS column,
		grantee.rolname AS grantee,
		acl.privilege_type
	FROM pg_class c
	JOIN pg_namespace n ON n.oid = c.relnamespace
	JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
	CROSS JOIN LATERAL aclexplode(a.attacl) AS acl
	JOIN pg_roles grantee ON grantee.oid = acl.grantee
	WHERE c.relkind IN ('r', 'v', 'm')
		AND n.nspname = ANY($1)
		AND a.attacl IS NOT NULL
	ORDER BY n.nspname, c.relname, a.attname, grantee.rolname
	`

	rows, err := s.pool.Query(ctx, sql, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var privs []*ColumnPrivilege
	for rows.Next() {
		p := &ColumnPrivilege{}
		err := rows.Scan(&p.Schema, &p.Table, &p.Column, &p.Grantee, &p.PrivilegeType)
		if err != nil {
			return nil, err
		}
		privs = append(privs, p)
	}

	return privs, nil
}

// ListConstraints lists all constraints.
func (s *PGMetaStore) ListConstraints(ctx context.Context, schemas []string) ([]*Constraint, error) {
	sql := `
	SELECT
		c.oid::int AS id,
		n.nspname AS schema,
		t.relname AS table,
		c.conname AS name,
		CASE c.contype
			WHEN 'p' THEN 'PRIMARY KEY'
			WHEN 'f' THEN 'FOREIGN KEY'
			WHEN 'u' THEN 'UNIQUE'
			WHEN 'c' THEN 'CHECK'
			WHEN 'x' THEN 'EXCLUSION'
		END AS type,
		pg_get_constraintdef(c.oid) AS definition
	FROM pg_constraint c
	JOIN pg_class t ON t.oid = c.conrelid
	JOIN pg_namespace n ON n.oid = t.relnamespace
	WHERE n.nspname = ANY($1)
	ORDER BY n.nspname, t.relname, c.conname
	`

	rows, err := s.pool.Query(ctx, sql, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var constraints []*Constraint
	for rows.Next() {
		c := &Constraint{}
		err := rows.Scan(&c.ID, &c.Schema, &c.Table, &c.Name, &c.Type, &c.Definition)
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, c)
	}

	return constraints, nil
}

// ListPrimaryKeys lists all primary keys.
func (s *PGMetaStore) ListPrimaryKeys(ctx context.Context, schemas []string) ([]*PrimaryKey, error) {
	sql := `
	SELECT
		n.nspname AS schema,
		t.relname AS table,
		c.conname AS name,
		array_agg(a.attname ORDER BY k.n) AS columns
	FROM pg_constraint c
	JOIN pg_class t ON t.oid = c.conrelid
	JOIN pg_namespace n ON n.oid = t.relnamespace
	JOIN LATERAL unnest(c.conkey) WITH ORDINALITY AS k(attnum, n) ON true
	JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum
	WHERE c.contype = 'p'
		AND n.nspname = ANY($1)
	GROUP BY n.nspname, t.relname, c.conname
	ORDER BY n.nspname, t.relname
	`

	rows, err := s.pool.Query(ctx, sql, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pks []*PrimaryKey
	for rows.Next() {
		pk := &PrimaryKey{}
		err := rows.Scan(&pk.Schema, &pk.Table, &pk.Name, &pk.Columns)
		if err != nil {
			return nil, err
		}
		pks = append(pks, pk)
	}

	return pks, nil
}

// ListForeignKeysAll lists all foreign keys.
func (s *PGMetaStore) ListForeignKeysAll(ctx context.Context, schemas []string) ([]*ForeignKeyInfo, error) {
	sql := `
	SELECT
		c.oid::int AS id,
		n1.nspname AS schema,
		t1.relname AS table,
		c.conname AS name,
		array_agg(a1.attname ORDER BY k1.n) AS columns,
		n2.nspname AS target_schema,
		t2.relname AS target_table,
		array_agg(a2.attname ORDER BY k2.n) AS target_columns,
		CASE c.confupdtype
			WHEN 'a' THEN 'NO ACTION'
			WHEN 'r' THEN 'RESTRICT'
			WHEN 'c' THEN 'CASCADE'
			WHEN 'n' THEN 'SET NULL'
			WHEN 'd' THEN 'SET DEFAULT'
		END AS on_update,
		CASE c.confdeltype
			WHEN 'a' THEN 'NO ACTION'
			WHEN 'r' THEN 'RESTRICT'
			WHEN 'c' THEN 'CASCADE'
			WHEN 'n' THEN 'SET NULL'
			WHEN 'd' THEN 'SET DEFAULT'
		END AS on_delete
	FROM pg_constraint c
	JOIN pg_class t1 ON t1.oid = c.conrelid
	JOIN pg_namespace n1 ON n1.oid = t1.relnamespace
	JOIN pg_class t2 ON t2.oid = c.confrelid
	JOIN pg_namespace n2 ON n2.oid = t2.relnamespace
	JOIN LATERAL unnest(c.conkey) WITH ORDINALITY AS k1(attnum, n) ON true
	JOIN pg_attribute a1 ON a1.attrelid = t1.oid AND a1.attnum = k1.attnum
	JOIN LATERAL unnest(c.confkey) WITH ORDINALITY AS k2(attnum, n) ON true
	JOIN pg_attribute a2 ON a2.attrelid = t2.oid AND a2.attnum = k2.attnum AND k1.n = k2.n
	WHERE c.contype = 'f'
		AND n1.nspname = ANY($1)
	GROUP BY c.oid, n1.nspname, t1.relname, c.conname, n2.nspname, t2.relname, c.confupdtype, c.confdeltype
	ORDER BY n1.nspname, t1.relname, c.conname
	`

	rows, err := s.pool.Query(ctx, sql, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fks []*ForeignKeyInfo
	for rows.Next() {
		fk := &ForeignKeyInfo{}
		err := rows.Scan(
			&fk.ID, &fk.Schema, &fk.Table, &fk.Name, &fk.Columns,
			&fk.TargetSchema, &fk.TargetTable, &fk.TargetColumns,
			&fk.OnUpdate, &fk.OnDelete,
		)
		if err != nil {
			return nil, err
		}
		fks = append(fks, fk)
	}

	return fks, nil
}

// ListRelationships lists table relationships.
func (s *PGMetaStore) ListRelationships(ctx context.Context, schemas []string) ([]*Relationship, error) {
	fks, err := s.ListForeignKeysAll(ctx, schemas)
	if err != nil {
		return nil, err
	}

	rels := make([]*Relationship, len(fks))
	for i, fk := range fks {
		rels[i] = &Relationship{
			ID:             fk.ID,
			SourceSchema:   fk.Schema,
			SourceTable:    fk.Table,
			SourceColumns:  fk.Columns,
			TargetSchema:   fk.TargetSchema,
			TargetTable:    fk.TargetTable,
			TargetColumns:  fk.TargetColumns,
			ConstraintName: fk.Name,
		}
	}

	return rels, nil
}

// ExplainQuery explains a query execution plan.
func (s *PGMetaStore) ExplainQuery(ctx context.Context, query string, analyze, buffers bool, format string) (interface{}, error) {
	explainOpts := []string{fmt.Sprintf("FORMAT %s", format)}
	if analyze {
		explainOpts = append(explainOpts, "ANALYZE")
	}
	if buffers {
		explainOpts = append(explainOpts, "BUFFERS")
	}

	sql := fmt.Sprintf("EXPLAIN (%s) %s", strings.Join(explainOpts, ", "), query)

	if format == "json" {
		var plan interface{}
		err := s.pool.QueryRow(ctx, sql).Scan(&plan)
		if err != nil {
			return nil, err
		}
		return plan, nil
	}

	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n"), nil
}

// GenerateTypescript generates TypeScript types from database schema.
func (s *PGMetaStore) GenerateTypescript(ctx context.Context, schemas []string) (string, error) {
	// Get tables and columns
	var sb strings.Builder
	sb.WriteString("export type Json = string | number | boolean | null | { [key: string]: Json } | Json[];\n\n")
	sb.WriteString("export interface Database {\n")

	for _, schema := range schemas {
		sb.WriteString(fmt.Sprintf("  %s: {\n", schema))
		sb.WriteString("    Tables: {\n")

		// Get tables for this schema
		tables, err := s.getTablesForSchema(ctx, schema)
		if err != nil {
			return "", err
		}

		for _, table := range tables {
			sb.WriteString(fmt.Sprintf("      %s: {\n", table.Name))
			sb.WriteString("        Row: {\n")

			for _, col := range table.Columns {
				tsType := pgTypeToTS(col.Type)
				nullable := ""
				if col.IsNullable {
					nullable = " | null"
				}
				sb.WriteString(fmt.Sprintf("          %s: %s%s;\n", col.Name, tsType, nullable))
			}

			sb.WriteString("        };\n")
			sb.WriteString("        Insert: {\n")

			for _, col := range table.Columns {
				tsType := pgTypeToTS(col.Type)
				optional := ""
				if col.HasDefault || col.IsNullable {
					optional = "?"
				}
				sb.WriteString(fmt.Sprintf("          %s%s: %s;\n", col.Name, optional, tsType))
			}

			sb.WriteString("        };\n")
			sb.WriteString("        Update: {\n")

			for _, col := range table.Columns {
				tsType := pgTypeToTS(col.Type)
				sb.WriteString(fmt.Sprintf("          %s?: %s;\n", col.Name, tsType))
			}

			sb.WriteString("        };\n")
			sb.WriteString("      };\n")
		}

		sb.WriteString("    };\n")
		sb.WriteString("  };\n")
	}

	sb.WriteString("}\n")

	return sb.String(), nil
}

type tableWithColumns struct {
	Name    string
	Columns []columnInfo
}

type columnInfo struct {
	Name       string
	Type       string
	IsNullable bool
	HasDefault bool
}

func (s *PGMetaStore) getTablesForSchema(ctx context.Context, schema string) ([]tableWithColumns, error) {
	sql := `
	SELECT
		c.relname AS table_name,
		a.attname AS column_name,
		pg_catalog.format_type(a.atttypid, a.atttypmod) AS type,
		NOT a.attnotnull AS is_nullable,
		a.atthasdef AS has_default
	FROM pg_class c
	JOIN pg_namespace n ON n.oid = c.relnamespace
	JOIN pg_attribute a ON a.attrelid = c.oid
	WHERE n.nspname = $1
		AND c.relkind = 'r'
		AND a.attnum > 0
		AND NOT a.attisdropped
	ORDER BY c.relname, a.attnum
	`

	rows, err := s.pool.Query(ctx, sql, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tableMap := make(map[string]*tableWithColumns)
	var tableOrder []string

	for rows.Next() {
		var tableName, colName, colType string
		var isNullable, hasDefault bool
		err := rows.Scan(&tableName, &colName, &colType, &isNullable, &hasDefault)
		if err != nil {
			return nil, err
		}

		if _, ok := tableMap[tableName]; !ok {
			tableMap[tableName] = &tableWithColumns{Name: tableName}
			tableOrder = append(tableOrder, tableName)
		}
		tableMap[tableName].Columns = append(tableMap[tableName].Columns, columnInfo{
			Name:       colName,
			Type:       colType,
			IsNullable: isNullable,
			HasDefault: hasDefault,
		})
	}

	result := make([]tableWithColumns, len(tableOrder))
	for i, name := range tableOrder {
		result[i] = *tableMap[name]
	}

	return result, nil
}

func pgTypeToTS(pgType string) string {
	pgType = strings.ToLower(pgType)
	switch {
	case strings.HasPrefix(pgType, "int"), strings.HasPrefix(pgType, "smallint"),
		strings.HasPrefix(pgType, "bigint"), strings.HasPrefix(pgType, "numeric"),
		strings.HasPrefix(pgType, "decimal"), strings.HasPrefix(pgType, "real"),
		strings.HasPrefix(pgType, "double"), strings.HasPrefix(pgType, "serial"):
		return "number"
	case strings.HasPrefix(pgType, "bool"):
		return "boolean"
	case strings.HasPrefix(pgType, "json"):
		return "Json"
	case strings.HasPrefix(pgType, "uuid"), strings.HasPrefix(pgType, "text"),
		strings.HasPrefix(pgType, "char"), strings.HasPrefix(pgType, "varchar"),
		strings.HasPrefix(pgType, "citext"), strings.HasPrefix(pgType, "date"),
		strings.HasPrefix(pgType, "time"), strings.HasPrefix(pgType, "timestamp"):
		return "string"
	case strings.Contains(pgType, "[]"):
		elemType := strings.TrimSuffix(pgType, "[]")
		return pgTypeToTS(elemType) + "[]"
	default:
		return "unknown"
	}
}

// GenerateOpenAPI generates an OpenAPI specification.
func (s *PGMetaStore) GenerateOpenAPI(ctx context.Context, schemas []string) (map[string]interface{}, error) {
	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":   "Localbase API",
			"version": "1.0.0",
		},
		"paths": map[string]interface{}{},
	}

	paths := spec["paths"].(map[string]interface{})

	for _, schema := range schemas {
		tables, err := s.getTablesForSchema(ctx, schema)
		if err != nil {
			return nil, err
		}

		for _, table := range tables {
			path := "/" + table.Name
			paths[path] = map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     fmt.Sprintf("Get %s", table.Name),
					"operationId": fmt.Sprintf("get_%s", table.Name),
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful response",
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     fmt.Sprintf("Create %s", table.Name),
					"operationId": fmt.Sprintf("create_%s", table.Name),
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Created",
						},
					},
				},
			}
		}
	}

	return spec, nil
}

// GenerateGo generates Go struct types from database schema.
func (s *PGMetaStore) GenerateGo(ctx context.Context, schemas []string, packageName string) (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	sb.WriteString("import (\n")
	sb.WriteString("\t\"time\"\n")
	sb.WriteString("\t\"encoding/json\"\n")
	sb.WriteString(")\n\n")
	sb.WriteString("// Json represents a JSON value\n")
	sb.WriteString("type Json = json.RawMessage\n\n")

	for _, schema := range schemas {
		tables, err := s.getTablesForSchema(ctx, schema)
		if err != nil {
			return "", err
		}

		for _, table := range tables {
			structName := toPascalCase(table.Name)
			sb.WriteString(fmt.Sprintf("// %s represents a row from %s.%s\n", structName, schema, table.Name))
			sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))

			for _, col := range table.Columns {
				goType := pgTypeToGo(col.Type, col.IsNullable)
				fieldName := toPascalCase(col.Name)
				jsonTag := col.Name
				sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, goType, jsonTag))
			}

			sb.WriteString("}\n\n")
		}
	}

	return sb.String(), nil
}

func pgTypeToGo(pgType string, nullable bool) string {
	pgType = strings.ToLower(pgType)
	var goType string

	switch {
	case strings.HasPrefix(pgType, "int2"), strings.HasPrefix(pgType, "smallint"):
		goType = "int16"
	case strings.HasPrefix(pgType, "int4"), strings.HasPrefix(pgType, "integer"):
		goType = "int32"
	case strings.HasPrefix(pgType, "int8"), strings.HasPrefix(pgType, "bigint"):
		goType = "int64"
	case strings.HasPrefix(pgType, "serial"):
		goType = "int32"
	case strings.HasPrefix(pgType, "bigserial"):
		goType = "int64"
	case strings.HasPrefix(pgType, "numeric"), strings.HasPrefix(pgType, "decimal"):
		goType = "float64"
	case strings.HasPrefix(pgType, "real"), strings.HasPrefix(pgType, "float4"):
		goType = "float32"
	case strings.HasPrefix(pgType, "double"), strings.HasPrefix(pgType, "float8"):
		goType = "float64"
	case strings.HasPrefix(pgType, "bool"):
		goType = "bool"
	case strings.HasPrefix(pgType, "json"):
		goType = "Json"
	case strings.HasPrefix(pgType, "uuid"):
		goType = "string"
	case strings.HasPrefix(pgType, "text"), strings.HasPrefix(pgType, "char"),
		strings.HasPrefix(pgType, "varchar"), strings.HasPrefix(pgType, "citext"):
		goType = "string"
	case strings.HasPrefix(pgType, "timestamp"), strings.HasPrefix(pgType, "date"):
		goType = "time.Time"
	case strings.HasPrefix(pgType, "time"):
		goType = "string"
	case strings.HasPrefix(pgType, "bytea"):
		goType = "[]byte"
	case strings.Contains(pgType, "[]"):
		elemType := strings.TrimSuffix(pgType, "[]")
		return "[]" + pgTypeToGo(elemType, false)
	default:
		goType = "interface{}"
	}

	if nullable && goType != "interface{}" && !strings.HasPrefix(goType, "[]") {
		goType = "*" + goType
	}

	return goType
}

func toPascalCase(s string) string {
	words := strings.Split(s, "_")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, "")
}

// GenerateSwift generates Swift struct types from database schema.
func (s *PGMetaStore) GenerateSwift(ctx context.Context, schemas []string) (string, error) {
	var sb strings.Builder
	sb.WriteString("import Foundation\n\n")
	sb.WriteString("// MARK: - Database Types\n\n")
	sb.WriteString("typealias Json = [String: Any]\n\n")

	for _, schema := range schemas {
		tables, err := s.getTablesForSchema(ctx, schema)
		if err != nil {
			return "", err
		}

		for _, table := range tables {
			structName := toPascalCase(table.Name)
			sb.WriteString(fmt.Sprintf("// MARK: - %s\n\n", structName))
			sb.WriteString(fmt.Sprintf("struct %s: Codable {\n", structName))

			for _, col := range table.Columns {
				swiftType := pgTypeToSwift(col.Type, col.IsNullable)
				fieldName := toCamelCase(col.Name)
				sb.WriteString(fmt.Sprintf("    let %s: %s\n", fieldName, swiftType))
			}

			sb.WriteString("\n    enum CodingKeys: String, CodingKey {\n")
			for _, col := range table.Columns {
				fieldName := toCamelCase(col.Name)
				if fieldName != col.Name {
					sb.WriteString(fmt.Sprintf("        case %s = \"%s\"\n", fieldName, col.Name))
				} else {
					sb.WriteString(fmt.Sprintf("        case %s\n", fieldName))
				}
			}
			sb.WriteString("    }\n")
			sb.WriteString("}\n\n")
		}
	}

	return sb.String(), nil
}

func pgTypeToSwift(pgType string, nullable bool) string {
	pgType = strings.ToLower(pgType)
	var swiftType string

	switch {
	case strings.HasPrefix(pgType, "int2"), strings.HasPrefix(pgType, "smallint"):
		swiftType = "Int16"
	case strings.HasPrefix(pgType, "int4"), strings.HasPrefix(pgType, "integer"), strings.HasPrefix(pgType, "serial"):
		swiftType = "Int"
	case strings.HasPrefix(pgType, "int8"), strings.HasPrefix(pgType, "bigint"), strings.HasPrefix(pgType, "bigserial"):
		swiftType = "Int64"
	case strings.HasPrefix(pgType, "numeric"), strings.HasPrefix(pgType, "decimal"),
		strings.HasPrefix(pgType, "real"), strings.HasPrefix(pgType, "float"),
		strings.HasPrefix(pgType, "double"):
		swiftType = "Double"
	case strings.HasPrefix(pgType, "bool"):
		swiftType = "Bool"
	case strings.HasPrefix(pgType, "json"):
		swiftType = "Json"
	case strings.HasPrefix(pgType, "uuid"):
		swiftType = "UUID"
	case strings.HasPrefix(pgType, "text"), strings.HasPrefix(pgType, "char"),
		strings.HasPrefix(pgType, "varchar"), strings.HasPrefix(pgType, "citext"):
		swiftType = "String"
	case strings.HasPrefix(pgType, "timestamp"), strings.HasPrefix(pgType, "date"):
		swiftType = "Date"
	case strings.HasPrefix(pgType, "time"):
		swiftType = "String"
	case strings.HasPrefix(pgType, "bytea"):
		swiftType = "Data"
	case strings.Contains(pgType, "[]"):
		elemType := strings.TrimSuffix(pgType, "[]")
		return "[" + pgTypeToSwift(elemType, false) + "]"
	default:
		swiftType = "Any"
	}

	if nullable {
		swiftType += "?"
	}

	return swiftType
}

func toCamelCase(s string) string {
	words := strings.Split(s, "_")
	for i, w := range words {
		if len(w) > 0 {
			if i == 0 {
				words[i] = strings.ToLower(w)
			} else {
				words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
			}
		}
	}
	return strings.Join(words, "")
}

// GeneratePython generates Python dataclass types from database schema.
func (s *PGMetaStore) GeneratePython(ctx context.Context, schemas []string) (string, error) {
	var sb strings.Builder
	sb.WriteString("from dataclasses import dataclass\n")
	sb.WriteString("from typing import Optional, List, Any, Dict\n")
	sb.WriteString("from datetime import datetime, date, time\n")
	sb.WriteString("from uuid import UUID\n\n")
	sb.WriteString("# Type aliases\n")
	sb.WriteString("Json = Dict[str, Any]\n\n")

	for _, schema := range schemas {
		tables, err := s.getTablesForSchema(ctx, schema)
		if err != nil {
			return "", err
		}

		for _, table := range tables {
			className := toPascalCase(table.Name)
			sb.WriteString("@dataclass\n")
			sb.WriteString(fmt.Sprintf("class %s:\n", className))
			sb.WriteString(fmt.Sprintf("    \"\"\"Represents a row from %s.%s\"\"\"\n", schema, table.Name))

			if len(table.Columns) == 0 {
				sb.WriteString("    pass\n")
			} else {
				for _, col := range table.Columns {
					pyType := pgTypeToPython(col.Type, col.IsNullable)
					sb.WriteString(fmt.Sprintf("    %s: %s\n", col.Name, pyType))
				}
			}
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

func pgTypeToPython(pgType string, nullable bool) string {
	pgType = strings.ToLower(pgType)
	var pyType string

	switch {
	case strings.HasPrefix(pgType, "int"), strings.HasPrefix(pgType, "smallint"),
		strings.HasPrefix(pgType, "bigint"), strings.HasPrefix(pgType, "serial"),
		strings.HasPrefix(pgType, "bigserial"):
		pyType = "int"
	case strings.HasPrefix(pgType, "numeric"), strings.HasPrefix(pgType, "decimal"),
		strings.HasPrefix(pgType, "real"), strings.HasPrefix(pgType, "float"),
		strings.HasPrefix(pgType, "double"):
		pyType = "float"
	case strings.HasPrefix(pgType, "bool"):
		pyType = "bool"
	case strings.HasPrefix(pgType, "json"):
		pyType = "Json"
	case strings.HasPrefix(pgType, "uuid"):
		pyType = "UUID"
	case strings.HasPrefix(pgType, "text"), strings.HasPrefix(pgType, "char"),
		strings.HasPrefix(pgType, "varchar"), strings.HasPrefix(pgType, "citext"):
		pyType = "str"
	case strings.HasPrefix(pgType, "timestamp"):
		pyType = "datetime"
	case strings.HasPrefix(pgType, "date"):
		pyType = "date"
	case strings.HasPrefix(pgType, "time"):
		pyType = "time"
	case strings.HasPrefix(pgType, "bytea"):
		pyType = "bytes"
	case strings.Contains(pgType, "[]"):
		elemType := strings.TrimSuffix(pgType, "[]")
		return "List[" + pgTypeToPython(elemType, false) + "]"
	default:
		pyType = "Any"
	}

	if nullable {
		pyType = "Optional[" + pyType + "]"
	}

	return pyType
}

// ListDatabaseFunctions lists all database functions.
func (s *PGMetaStore) ListDatabaseFunctions(ctx context.Context, schemas []string) ([]*DatabaseFunction, error) {
	sql := `
	SELECT
		p.oid::int AS id,
		n.nspname AS schema,
		p.proname AS name,
		l.lanname AS language,
		pg_get_functiondef(p.oid) AS definition,
		pg_get_function_result(p.oid) AS return_type,
		pg_get_function_arguments(p.oid) AS arguments,
		p.proisstrict AS is_strict,
		CASE p.provolatile
			WHEN 'i' THEN 'IMMUTABLE'
			WHEN 's' THEN 'STABLE'
			WHEN 'v' THEN 'VOLATILE'
		END AS volatility
	FROM pg_proc p
	JOIN pg_namespace n ON n.oid = p.pronamespace
	JOIN pg_language l ON l.oid = p.prolang
	WHERE n.nspname = ANY($1)
		AND p.prokind = 'f'
	ORDER BY n.nspname, p.proname
	`

	rows, err := s.pool.Query(ctx, sql, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []*DatabaseFunction
	for rows.Next() {
		f := &DatabaseFunction{}
		err := rows.Scan(
			&f.ID, &f.Schema, &f.Name, &f.Language, &f.Definition,
			&f.ReturnType, &f.Arguments, &f.IsStrict, &f.Volatility,
		)
		if err != nil {
			return nil, err
		}
		functions = append(functions, f)
	}

	return functions, nil
}
