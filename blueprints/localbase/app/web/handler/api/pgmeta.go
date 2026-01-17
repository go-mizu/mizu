package api

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
)

// PGMetaHandler handles postgres-meta API endpoints for Supabase Dashboard compatibility.
type PGMetaHandler struct {
	store *postgres.Store
}

// NewPGMetaHandler creates a new postgres-meta handler.
func NewPGMetaHandler(store *postgres.Store) *PGMetaHandler {
	return &PGMetaHandler{store: store}
}

// =============================================================================
// Config & Version
// =============================================================================

// GetVersion returns PostgreSQL version information.
func (h *PGMetaHandler) GetVersion(c *mizu.Ctx) error {
	version, err := h.store.PGMeta().GetVersion(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to get version: " + err.Error()})
	}
	return c.JSON(200, version)
}

// =============================================================================
// Indexes
// =============================================================================

// ListIndexes lists all indexes with optional schema filter.
func (h *PGMetaHandler) ListIndexes(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	schemaList := parseSchemas(schemas)

	indexes, err := h.store.PGMeta().ListIndexes(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list indexes: " + err.Error()})
	}
	return c.JSON(200, indexes)
}

// CreateIndex creates a new index.
func (h *PGMetaHandler) CreateIndex(c *mizu.Ctx) error {
	var req struct {
		Schema  string   `json:"schema"`
		Table   string   `json:"table"`
		Name    string   `json:"name"`
		Columns []string `json:"columns"`
		Unique  bool     `json:"unique"`
		Using   string   `json:"using"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Schema == "" {
		req.Schema = "public"
	}
	if req.Using == "" {
		req.Using = "btree"
	}

	idx := &store.Index{
		Schema:   req.Schema,
		Table:    req.Table,
		Name:     req.Name,
		Columns:  req.Columns,
		IsUnique: req.Unique,
		Type:     req.Using,
	}

	if err := h.store.Database().CreateIndex(c.Context(), idx); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create index: " + err.Error()})
	}

	return c.JSON(201, idx)
}

// DropIndex drops an index by name.
func (h *PGMetaHandler) DropIndex(c *mizu.Ctx) error {
	id := c.Param("id")
	// ID format: schema.name or just name (defaults to public)
	schema := "public"
	name := id
	if strings.Contains(id, ".") {
		parts := strings.SplitN(id, ".", 2)
		schema, name = parts[0], parts[1]
	}

	if err := h.store.Database().DropIndex(c.Context(), schema, name); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to drop index: " + err.Error()})
	}

	return c.NoContent()
}

// =============================================================================
// Views
// =============================================================================

// ListViews lists all views.
func (h *PGMetaHandler) ListViews(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	schemaList := parseSchemas(schemas)

	views, err := h.store.PGMeta().ListViews(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list views: " + err.Error()})
	}
	return c.JSON(200, views)
}

// CreateView creates a new view.
func (h *PGMetaHandler) CreateView(c *mizu.Ctx) error {
	var req struct {
		Schema      string `json:"schema"`
		Name        string `json:"name"`
		Definition  string `json:"definition"`
		CheckOption string `json:"check_option"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Schema == "" {
		req.Schema = "public"
	}

	view, err := h.store.PGMeta().CreateView(c.Context(), req.Schema, req.Name, req.Definition, req.CheckOption)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create view: " + err.Error()})
	}

	return c.JSON(201, view)
}

// UpdateView updates a view.
func (h *PGMetaHandler) UpdateView(c *mizu.Ctx) error {
	id := c.Param("id")

	var req struct {
		Definition  string `json:"definition"`
		CheckOption string `json:"check_option"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	view, err := h.store.PGMeta().UpdateView(c.Context(), id, req.Definition, req.CheckOption)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update view: " + err.Error()})
	}

	return c.JSON(200, view)
}

// DropView drops a view.
func (h *PGMetaHandler) DropView(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.store.PGMeta().DropView(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to drop view: " + err.Error()})
	}

	return c.NoContent()
}

// =============================================================================
// Materialized Views
// =============================================================================

// ListMaterializedViews lists all materialized views.
func (h *PGMetaHandler) ListMaterializedViews(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	schemaList := parseSchemas(schemas)

	mvs, err := h.store.PGMeta().ListMaterializedViews(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list materialized views: " + err.Error()})
	}
	return c.JSON(200, mvs)
}

// CreateMaterializedView creates a new materialized view.
func (h *PGMetaHandler) CreateMaterializedView(c *mizu.Ctx) error {
	var req struct {
		Schema     string `json:"schema"`
		Name       string `json:"name"`
		Definition string `json:"definition"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Schema == "" {
		req.Schema = "public"
	}

	mv, err := h.store.PGMeta().CreateMaterializedView(c.Context(), req.Schema, req.Name, req.Definition)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create materialized view: " + err.Error()})
	}

	return c.JSON(201, mv)
}

// RefreshMaterializedView refreshes a materialized view.
func (h *PGMetaHandler) RefreshMaterializedView(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.store.PGMeta().RefreshMaterializedView(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to refresh materialized view: " + err.Error()})
	}

	return c.NoContent()
}

// DropMaterializedView drops a materialized view.
func (h *PGMetaHandler) DropMaterializedView(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.store.PGMeta().DropMaterializedView(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to drop materialized view: " + err.Error()})
	}

	return c.NoContent()
}

// =============================================================================
// Foreign Tables
// =============================================================================

// ListForeignTables lists all foreign tables.
func (h *PGMetaHandler) ListForeignTables(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	schemaList := parseSchemas(schemas)

	tables, err := h.store.PGMeta().ListForeignTables(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list foreign tables: " + err.Error()})
	}
	return c.JSON(200, tables)
}

// =============================================================================
// Triggers
// =============================================================================

// ListTriggers lists all triggers.
func (h *PGMetaHandler) ListTriggers(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	schemaList := parseSchemas(schemas)

	triggers, err := h.store.PGMeta().ListTriggers(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list triggers: " + err.Error()})
	}
	return c.JSON(200, triggers)
}

// CreateTrigger creates a new trigger.
func (h *PGMetaHandler) CreateTrigger(c *mizu.Ctx) error {
	var req struct {
		Name           string   `json:"name"`
		Schema         string   `json:"schema"`
		Table          string   `json:"table"`
		FunctionSchema string   `json:"function_schema"`
		FunctionName   string   `json:"function_name"`
		Events         []string `json:"events"`
		Timing         string   `json:"timing"`
		Orientation    string   `json:"orientation"`
		Condition      string   `json:"condition"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Schema == "" {
		req.Schema = "public"
	}
	if req.FunctionSchema == "" {
		req.FunctionSchema = "public"
	}
	if req.Orientation == "" {
		req.Orientation = "ROW"
	}
	if req.Timing == "" {
		req.Timing = "AFTER"
	}

	trigger, err := h.store.PGMeta().CreateTrigger(c.Context(), req.Name, req.Schema, req.Table,
		req.FunctionSchema, req.FunctionName, req.Events, req.Timing, req.Orientation, req.Condition)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create trigger: " + err.Error()})
	}

	return c.JSON(201, trigger)
}

// DropTrigger drops a trigger.
func (h *PGMetaHandler) DropTrigger(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.store.PGMeta().DropTrigger(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to drop trigger: " + err.Error()})
	}

	return c.NoContent()
}

// =============================================================================
// Types (Custom Types / Enums)
// =============================================================================

// ListTypes lists all custom types.
func (h *PGMetaHandler) ListTypes(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	schemaList := parseSchemas(schemas)

	types, err := h.store.PGMeta().ListTypes(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list types: " + err.Error()})
	}
	return c.JSON(200, types)
}

// CreateType creates a new type.
func (h *PGMetaHandler) CreateType(c *mizu.Ctx) error {
	var req struct {
		Schema string   `json:"schema"`
		Name   string   `json:"name"`
		Type   string   `json:"type"`   // enum, composite
		Values []string `json:"values"` // for enum
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Schema == "" {
		req.Schema = "public"
	}

	typ, err := h.store.PGMeta().CreateType(c.Context(), req.Schema, req.Name, req.Type, req.Values)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create type: " + err.Error()})
	}

	return c.JSON(201, typ)
}

// DropType drops a type.
func (h *PGMetaHandler) DropType(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.store.PGMeta().DropType(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to drop type: " + err.Error()})
	}

	return c.NoContent()
}

// =============================================================================
// Roles
// =============================================================================

// ListRoles lists all database roles.
func (h *PGMetaHandler) ListRoles(c *mizu.Ctx) error {
	roles, err := h.store.PGMeta().ListRoles(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list roles: " + err.Error()})
	}
	return c.JSON(200, roles)
}

// CreateRole creates a new role.
func (h *PGMetaHandler) CreateRole(c *mizu.Ctx) error {
	var req struct {
		Name        string `json:"name"`
		IsSuperuser bool   `json:"is_superuser"`
		CanLogin    bool   `json:"can_login"`
		Password    string `json:"password"`
		InheritRole bool   `json:"inherit_role"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	role, err := h.store.PGMeta().CreateRole(c.Context(), req.Name, req.IsSuperuser, req.CanLogin, req.Password, req.InheritRole)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create role: " + err.Error()})
	}

	return c.JSON(201, role)
}

// UpdateRole updates a role.
func (h *PGMetaHandler) UpdateRole(c *mizu.Ctx) error {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid role id"})
	}

	var req struct {
		IsSuperuser *bool   `json:"is_superuser"`
		CanLogin    *bool   `json:"can_login"`
		Password    *string `json:"password"`
		InheritRole *bool   `json:"inherit_role"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	role, err := h.store.PGMeta().UpdateRole(c.Context(), idInt, req.IsSuperuser, req.CanLogin, req.Password, req.InheritRole)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update role: " + err.Error()})
	}

	return c.JSON(200, role)
}

// DropRole drops a role.
func (h *PGMetaHandler) DropRole(c *mizu.Ctx) error {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid role id"})
	}

	if err := h.store.PGMeta().DropRole(c.Context(), idInt); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to drop role: " + err.Error()})
	}

	return c.NoContent()
}

// =============================================================================
// Publications
// =============================================================================

// ListPublications lists all publications.
func (h *PGMetaHandler) ListPublications(c *mizu.Ctx) error {
	pubs, err := h.store.PGMeta().ListPublications(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list publications: " + err.Error()})
	}
	return c.JSON(200, pubs)
}

// CreatePublication creates a new publication.
func (h *PGMetaHandler) CreatePublication(c *mizu.Ctx) error {
	var req struct {
		Name      string `json:"name"`
		AllTables bool   `json:"all_tables"`
		Tables    []struct {
			Schema string `json:"schema"`
			Name   string `json:"name"`
		} `json:"tables"`
		Insert   bool `json:"insert"`
		Update   bool `json:"update"`
		Delete   bool `json:"delete"`
		Truncate bool `json:"truncate"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	tables := make([]string, len(req.Tables))
	for i, t := range req.Tables {
		if t.Schema == "" {
			t.Schema = "public"
		}
		tables[i] = fmt.Sprintf("%s.%s", t.Schema, t.Name)
	}

	pub, err := h.store.PGMeta().CreatePublication(c.Context(), req.Name, req.AllTables, tables,
		req.Insert, req.Update, req.Delete, req.Truncate)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create publication: " + err.Error()})
	}

	return c.JSON(201, pub)
}

// DropPublication drops a publication.
func (h *PGMetaHandler) DropPublication(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.store.PGMeta().DropPublication(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to drop publication: " + err.Error()})
	}

	return c.NoContent()
}

// =============================================================================
// Privileges
// =============================================================================

// ListTablePrivileges lists table-level privileges.
func (h *PGMetaHandler) ListTablePrivileges(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	schemaList := parseSchemas(schemas)

	privs, err := h.store.PGMeta().ListTablePrivileges(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list table privileges: " + err.Error()})
	}
	return c.JSON(200, privs)
}

// ListColumnPrivileges lists column-level privileges.
func (h *PGMetaHandler) ListColumnPrivileges(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	schemaList := parseSchemas(schemas)

	privs, err := h.store.PGMeta().ListColumnPrivileges(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list column privileges: " + err.Error()})
	}
	return c.JSON(200, privs)
}

// =============================================================================
// Constraints
// =============================================================================

// ListConstraints lists all constraints.
func (h *PGMetaHandler) ListConstraints(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	schemaList := parseSchemas(schemas)

	constraints, err := h.store.PGMeta().ListConstraints(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list constraints: " + err.Error()})
	}
	return c.JSON(200, constraints)
}

// ListPrimaryKeys lists all primary keys.
func (h *PGMetaHandler) ListPrimaryKeys(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	schemaList := parseSchemas(schemas)

	pks, err := h.store.PGMeta().ListPrimaryKeys(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list primary keys: " + err.Error()})
	}
	return c.JSON(200, pks)
}

// ListForeignKeysAll lists all foreign keys.
func (h *PGMetaHandler) ListForeignKeysAll(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	schemaList := parseSchemas(schemas)

	fks, err := h.store.PGMeta().ListForeignKeysAll(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list foreign keys: " + err.Error()})
	}
	return c.JSON(200, fks)
}

// ListRelationships lists table relationships.
func (h *PGMetaHandler) ListRelationships(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	schemaList := parseSchemas(schemas)

	rels, err := h.store.PGMeta().ListRelationships(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list relationships: " + err.Error()})
	}
	return c.JSON(200, rels)
}

// =============================================================================
// SQL Utilities
// =============================================================================

// FormatSQL formats a SQL query.
func (h *PGMetaHandler) FormatSQL(c *mizu.Ctx) error {
	var req struct {
		Query string `json:"query"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	// Basic SQL formatting (simplified)
	formatted := formatSQL(req.Query)

	return c.JSON(200, map[string]string{"formatted": formatted})
}

// ExplainQuery explains a query execution plan.
func (h *PGMetaHandler) ExplainQuery(c *mizu.Ctx) error {
	var req struct {
		Query   string `json:"query"`
		Analyze bool   `json:"analyze"`
		Buffers bool   `json:"buffers"`
		Format  string `json:"format"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Format == "" {
		req.Format = "json"
	}

	plan, err := h.store.PGMeta().ExplainQuery(c.Context(), req.Query, req.Analyze, req.Buffers, req.Format)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to explain query: " + err.Error()})
	}

	return c.JSON(200, plan)
}

// =============================================================================
// Type Generators
// =============================================================================

// GenerateTypescript generates TypeScript types from database schema.
func (h *PGMetaHandler) GenerateTypescript(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	if schemas == "" {
		schemas = "public"
	}
	schemaList := parseSchemas(schemas)

	typescript, err := h.store.PGMeta().GenerateTypescript(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to generate typescript: " + err.Error()})
	}

	c.Writer().Header().Set("Content-Type", "text/plain; charset=utf-8")
	return c.Text(200, typescript)
}

// GenerateOpenAPI generates OpenAPI specification from database schema.
func (h *PGMetaHandler) GenerateOpenAPI(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	if schemas == "" {
		schemas = "public"
	}
	schemaList := parseSchemas(schemas)

	spec, err := h.store.PGMeta().GenerateOpenAPI(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to generate openapi: " + err.Error()})
	}

	return c.JSON(200, spec)
}

// GenerateGo generates Go struct types from database schema.
func (h *PGMetaHandler) GenerateGo(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	if schemas == "" {
		schemas = "public"
	}
	schemaList := parseSchemas(schemas)

	packageName := c.Query("package")
	if packageName == "" {
		packageName = "models"
	}

	goCode, err := h.store.PGMeta().GenerateGo(c.Context(), schemaList, packageName)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to generate go: " + err.Error()})
	}

	c.Writer().Header().Set("Content-Type", "text/plain; charset=utf-8")
	return c.Text(200, goCode)
}

// GenerateSwift generates Swift struct types from database schema.
func (h *PGMetaHandler) GenerateSwift(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	if schemas == "" {
		schemas = "public"
	}
	schemaList := parseSchemas(schemas)

	swiftCode, err := h.store.PGMeta().GenerateSwift(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to generate swift: " + err.Error()})
	}

	c.Writer().Header().Set("Content-Type", "text/plain; charset=utf-8")
	return c.Text(200, swiftCode)
}

// GeneratePython generates Python dataclass types from database schema.
func (h *PGMetaHandler) GeneratePython(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	if schemas == "" {
		schemas = "public"
	}
	schemaList := parseSchemas(schemas)

	pythonCode, err := h.store.PGMeta().GeneratePython(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to generate python: " + err.Error()})
	}

	c.Writer().Header().Set("Content-Type", "text/plain; charset=utf-8")
	return c.Text(200, pythonCode)
}

// =============================================================================
// Functions (Database Functions)
// =============================================================================

// ListDatabaseFunctions lists all database functions.
func (h *PGMetaHandler) ListDatabaseFunctions(c *mizu.Ctx) error {
	schemas := c.Query("included_schemas")
	schemaList := parseSchemas(schemas)

	functions, err := h.store.PGMeta().ListDatabaseFunctions(c.Context(), schemaList)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list functions: " + err.Error()})
	}
	return c.JSON(200, functions)
}

// =============================================================================
// Helpers
// =============================================================================

func parseSchemas(schemas string) []string {
	if schemas == "" {
		return []string{"public"}
	}
	parts := strings.Split(schemas, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{"public"}
	}
	return result
}

func formatSQL(sql string) string {
	// Basic SQL formatting - uppercase keywords
	keywords := []string{
		"SELECT", "FROM", "WHERE", "AND", "OR", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER",
		"ON", "GROUP BY", "ORDER BY", "LIMIT", "OFFSET", "INSERT", "INTO", "VALUES",
		"UPDATE", "SET", "DELETE", "CREATE", "TABLE", "INDEX", "VIEW", "DROP", "ALTER",
		"ADD", "COLUMN", "PRIMARY KEY", "FOREIGN KEY", "REFERENCES", "UNIQUE", "NOT NULL",
		"DEFAULT", "CASCADE", "RESTRICT", "NULL", "AS", "DISTINCT", "ALL", "HAVING",
		"UNION", "INTERSECT", "EXCEPT", "IN", "EXISTS", "BETWEEN", "LIKE", "IS", "CASE",
		"WHEN", "THEN", "ELSE", "END", "GRANT", "REVOKE", "TO", "WITH",
	}

	result := sql
	for _, kw := range keywords {
		// Case-insensitive replace
		lower := strings.ToLower(kw)
		result = strings.ReplaceAll(result, lower, kw)
	}

	// Add newlines after major keywords
	result = strings.ReplaceAll(result, " FROM ", "\nFROM ")
	result = strings.ReplaceAll(result, " WHERE ", "\nWHERE ")
	result = strings.ReplaceAll(result, " AND ", "\n  AND ")
	result = strings.ReplaceAll(result, " OR ", "\n  OR ")
	result = strings.ReplaceAll(result, " JOIN ", "\nJOIN ")
	result = strings.ReplaceAll(result, " LEFT JOIN ", "\nLEFT JOIN ")
	result = strings.ReplaceAll(result, " RIGHT JOIN ", "\nRIGHT JOIN ")
	result = strings.ReplaceAll(result, " ORDER BY ", "\nORDER BY ")
	result = strings.ReplaceAll(result, " GROUP BY ", "\nGROUP BY ")
	result = strings.ReplaceAll(result, " LIMIT ", "\nLIMIT ")

	return result
}
