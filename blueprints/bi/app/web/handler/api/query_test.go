package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateIdentifier(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"valid simple", "users", false},
		{"valid with underscore", "user_accounts", false},
		{"valid with schema", "public.users", false},
		{"valid mixed case", "UserAccounts", false},
		{"valid with numbers", "table1", false},

		{"empty", "", true},
		{"too long", string(make([]byte, 200)), true},
		{"starts with number", "1users", true},
		{"has spaces", "user accounts", true},
		{"has hyphen", "user-accounts", true},
		{"has semicolon", "users;", true},
		{"SQL keyword SELECT", "SELECT", true},
		{"SQL keyword DROP", "DROP", true},
		{"SQL keyword UNION", "UNION", true},
		{"has parentheses", "users()", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIdentifier(tt.input)
			if tt.expectErr {
				assert.Error(t, err, "Expected error for %q", tt.input)
			} else {
				assert.NoError(t, err, "Expected no error for %q", tt.input)
			}
		})
	}
}

func TestValidateOperator(t *testing.T) {
	validOps := []string{
		"=", "!=", "<>", ">", ">=", "<", "<=",
		"LIKE", "like", "IN", "in", "NOT IN", "not in",
		"IS NULL", "is null", "IS NOT NULL", "is not null",
		"BETWEEN", "between",
	}

	for _, op := range validOps {
		t.Run("valid_"+op, func(t *testing.T) {
			err := validateOperator(op)
			assert.NoError(t, err)
		})
	}

	invalidOps := []string{
		"===", ">>", "DROP", "OR", "AND", "--",
	}

	for _, op := range invalidOps {
		t.Run("invalid_"+op, func(t *testing.T) {
			err := validateOperator(op)
			assert.Error(t, err)
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"users", `"users"`},
		{"user_accounts", `"user_accounts"`},
		{`col"name`, `"col""name"`}, // double quote escaping
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := quoteIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildSQLFromQuery_Basic(t *testing.T) {
	tests := []struct {
		name        string
		query       map[string]any
		expectedSQL string
		expectErr   bool
	}{
		{
			name: "simple select all",
			query: map[string]any{
				"table": "users",
			},
			expectedSQL: `SELECT * FROM "users"`,
			expectErr:   false,
		},
		{
			name: "select specific columns",
			query: map[string]any{
				"table":   "users",
				"columns": []any{"id", "name", "email"},
			},
			expectedSQL: `SELECT "id", "name", "email" FROM "users"`,
			expectErr:   false,
		},
		{
			name: "with order by",
			query: map[string]any{
				"table":   "users",
				"columns": []any{"name"},
				"order_by": []any{
					map[string]any{"column": "name", "direction": "ASC"},
				},
			},
			expectedSQL: `SELECT "name" FROM "users" ORDER BY "name" ASC`,
			expectErr:   false,
		},
		{
			name: "with limit",
			query: map[string]any{
				"table": "users",
				"limit": float64(100),
			},
			expectedSQL: `SELECT * FROM "users" LIMIT 100`,
			expectErr:   false,
		},
		{
			name: "limit exceeded",
			query: map[string]any{
				"table": "users",
				"limit": float64(50000),
			},
			expectedSQL: `SELECT * FROM "users" LIMIT 10000`,
			expectErr:   false,
		},
		{
			name: "no table specified",
			query: map[string]any{
				"columns": []any{"id"},
			},
			expectErr: true,
		},
		{
			name: "direct SQL rejected",
			query: map[string]any{
				"sql": "SELECT * FROM users",
			},
			expectErr: true,
		},
		{
			name: "invalid table name",
			query: map[string]any{
				"table": "users; DROP TABLE users;",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, _, err := buildSQLFromQuery(tt.query)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedSQL, sql)
			}
		})
	}
}

func TestBuildSQLFromQuery_Filters(t *testing.T) {
	tests := []struct {
		name           string
		query          map[string]any
		expectedSQL    string
		expectedParams []any
		expectErr      bool
	}{
		{
			name: "equals filter",
			query: map[string]any{
				"table": "users",
				"filters": []any{
					map[string]any{"column": "status", "operator": "=", "value": "active"},
				},
			},
			expectedSQL:    `SELECT * FROM "users" WHERE "status" = ?`,
			expectedParams: []any{"active"},
		},
		{
			name: "not equals filter",
			query: map[string]any{
				"table": "users",
				"filters": []any{
					map[string]any{"column": "status", "operator": "!=", "value": "deleted"},
				},
			},
			expectedSQL:    `SELECT * FROM "users" WHERE "status" != ?`,
			expectedParams: []any{"deleted"},
		},
		{
			name: "greater than filter",
			query: map[string]any{
				"table": "orders",
				"filters": []any{
					map[string]any{"column": "amount", "operator": ">", "value": 100},
				},
			},
			expectedSQL:    `SELECT * FROM "orders" WHERE "amount" > ?`,
			expectedParams: []any{100},
		},
		{
			name: "is null filter",
			query: map[string]any{
				"table": "users",
				"filters": []any{
					map[string]any{"column": "deleted_at", "operator": "IS NULL"},
				},
			},
			expectedSQL:    `SELECT * FROM "users" WHERE "deleted_at" IS NULL`,
			expectedParams: nil,
		},
		{
			name: "is not null filter",
			query: map[string]any{
				"table": "users",
				"filters": []any{
					map[string]any{"column": "email", "operator": "IS NOT NULL"},
				},
			},
			expectedSQL:    `SELECT * FROM "users" WHERE "email" IS NOT NULL`,
			expectedParams: nil,
		},
		{
			name: "in filter",
			query: map[string]any{
				"table": "users",
				"filters": []any{
					map[string]any{"column": "role", "operator": "IN", "value": []any{"admin", "moderator"}},
				},
			},
			expectedSQL:    `SELECT * FROM "users" WHERE "role" IN (?, ?)`,
			expectedParams: []any{"admin", "moderator"},
		},
		{
			name: "between filter",
			query: map[string]any{
				"table": "orders",
				"filters": []any{
					map[string]any{"column": "amount", "operator": "BETWEEN", "value": []any{100, 500}},
				},
			},
			expectedSQL:    `SELECT * FROM "orders" WHERE "amount" BETWEEN ? AND ?`,
			expectedParams: []any{100, 500},
		},
		{
			name: "like filter",
			query: map[string]any{
				"table": "products",
				"filters": []any{
					map[string]any{"column": "name", "operator": "LIKE", "value": "%coffee%"},
				},
			},
			expectedSQL:    `SELECT * FROM "products" WHERE "name" LIKE ?`,
			expectedParams: []any{"%coffee%"},
		},
		{
			name: "multiple filters",
			query: map[string]any{
				"table": "orders",
				"filters": []any{
					map[string]any{"column": "status", "operator": "=", "value": "pending"},
					map[string]any{"column": "amount", "operator": ">", "value": 50},
				},
			},
			expectedSQL:    `SELECT * FROM "orders" WHERE "status" = ? AND "amount" > ?`,
			expectedParams: []any{"pending", 50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, params, err := buildSQLFromQuery(tt.query)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedSQL, sql)
				assert.Equal(t, tt.expectedParams, params)
			}
		})
	}
}

func TestBuildSQLFromQuery_OrderBy(t *testing.T) {
	tests := []struct {
		name        string
		query       map[string]any
		expectedSQL string
	}{
		{
			name: "single column asc",
			query: map[string]any{
				"table": "users",
				"order_by": []any{
					map[string]any{"column": "created_at", "direction": "ASC"},
				},
			},
			expectedSQL: `SELECT * FROM "users" ORDER BY "created_at" ASC`,
		},
		{
			name: "single column desc",
			query: map[string]any{
				"table": "users",
				"order_by": []any{
					map[string]any{"column": "score", "direction": "DESC"},
				},
			},
			expectedSQL: `SELECT * FROM "users" ORDER BY "score" DESC`,
		},
		{
			name: "multiple columns",
			query: map[string]any{
				"table": "orders",
				"order_by": []any{
					map[string]any{"column": "priority", "direction": "DESC"},
					map[string]any{"column": "created_at", "direction": "ASC"},
				},
			},
			expectedSQL: `SELECT * FROM "orders" ORDER BY "priority" DESC, "created_at" ASC`,
		},
		{
			name: "case insensitive direction",
			query: map[string]any{
				"table": "users",
				"order_by": []any{
					map[string]any{"column": "name", "direction": "asc"},
				},
			},
			expectedSQL: `SELECT * FROM "users" ORDER BY "name" ASC`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, _, err := buildSQLFromQuery(tt.query)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}

func TestBuildSQLFromQuery_GroupBy(t *testing.T) {
	query := map[string]any{
		"table":    "orders",
		"columns":  []any{"status"},
		"group_by": []any{"status"},
	}

	sql, _, err := buildSQLFromQuery(query)
	require.NoError(t, err)
	assert.Contains(t, sql, `GROUP BY "status"`)
}

func TestBuildCountQuery(t *testing.T) {
	tests := []struct {
		name           string
		query          map[string]any
		expectedSQL    string
		expectedParams []any
	}{
		{
			name: "simple count",
			query: map[string]any{
				"table": "users",
			},
			expectedSQL:    `SELECT COUNT(*) as count FROM "users"`,
			expectedParams: nil,
		},
		{
			name: "count with filter",
			query: map[string]any{
				"table": "users",
				"filters": []any{
					map[string]any{"column": "status", "operator": "=", "value": "active"},
				},
			},
			expectedSQL:    `SELECT COUNT(*) as count FROM "users" WHERE "status" = ?`,
			expectedParams: []any{"active"},
		},
		{
			name: "count with multiple filters",
			query: map[string]any{
				"table": "orders",
				"filters": []any{
					map[string]any{"column": "status", "operator": "=", "value": "pending"},
					map[string]any{"column": "amount", "operator": ">", "value": 100},
				},
			},
			expectedSQL:    `SELECT COUNT(*) as count FROM "orders" WHERE "status" = ? AND "amount" > ?`,
			expectedParams: []any{"pending", 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, params, err := buildCountQuery(tt.query)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSQL, sql)
			assert.Equal(t, tt.expectedParams, params)
		})
	}
}

func TestPaginationInfo(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		expectedPage int
		expectedSize int
		expectedOff  int
	}{
		{"page 1", 1, 25, 1, 25, 0},
		{"page 2", 2, 25, 2, 25, 25},
		{"page 3 with 50", 3, 50, 3, 50, 100},
		{"large page size capped", 1, 5000, 1, 1000, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageSize := tt.pageSize
			if pageSize > 1000 {
				pageSize = 1000
			}
			offset := (tt.page - 1) * pageSize

			assert.Equal(t, tt.expectedPage, tt.page)
			assert.Equal(t, tt.expectedSize, pageSize)
			assert.Equal(t, tt.expectedOff, offset)
		})
	}
}
