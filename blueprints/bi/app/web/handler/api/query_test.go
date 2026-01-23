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
		// New operators for Metabase parity
		"contains", "CONTAINS", "not-contains", "NOT-CONTAINS",
		"starts-with", "STARTS-WITH", "ends-with", "ENDS-WITH",
		"is-empty", "IS-EMPTY", "is-not-empty", "IS-NOT-EMPTY",
		"relative", "RELATIVE",
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

func TestValidateJoinType(t *testing.T) {
	validTypes := []string{"left", "LEFT", "right", "RIGHT", "inner", "INNER", "full", "FULL"}

	for _, jt := range validTypes {
		t.Run("valid_"+jt, func(t *testing.T) {
			err := validateJoinType(jt)
			assert.NoError(t, err)
		})
	}

	invalidTypes := []string{"outer", "cross", "natural", ""}

	for _, jt := range invalidTypes {
		t.Run("invalid_"+jt, func(t *testing.T) {
			err := validateJoinType(jt)
			assert.Error(t, err)
		})
	}
}

func TestValidateAggregationFunction(t *testing.T) {
	validFns := []string{"count", "COUNT", "sum", "SUM", "avg", "AVG", "min", "MIN", "max", "MAX", "distinct", "DISTINCT"}

	for _, fn := range validFns {
		t.Run("valid_"+fn, func(t *testing.T) {
			err := validateAggregationFunction(fn)
			assert.NoError(t, err)
		})
	}

	invalidFns := []string{"median", "percentile", "variance", "stddev", ""}

	for _, fn := range invalidFns {
		t.Run("invalid_"+fn, func(t *testing.T) {
			err := validateAggregationFunction(fn)
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

// ============================================================================
// Aggregation Tests
// ============================================================================

func TestBuildSQLFromQuery_Aggregations(t *testing.T) {
	tests := []struct {
		name        string
		query       map[string]any
		expectedSQL string
		expectErr   bool
	}{
		{
			name: "count all",
			query: map[string]any{
				"table": "orders",
				"aggregations": []any{
					map[string]any{"function": "count"},
				},
			},
			expectedSQL: `SELECT COUNT(*) FROM "orders"`,
		},
		{
			name: "count with alias",
			query: map[string]any{
				"table": "orders",
				"aggregations": []any{
					map[string]any{"function": "count", "alias": "total_orders"},
				},
			},
			expectedSQL: `SELECT COUNT(*) AS "total_orders" FROM "orders"`,
		},
		{
			name: "count column",
			query: map[string]any{
				"table": "products",
				"aggregations": []any{
					map[string]any{"function": "count", "column": "category_id"},
				},
			},
			expectedSQL: `SELECT COUNT("category_id") FROM "products"`,
		},
		{
			name: "distinct count",
			query: map[string]any{
				"table": "products",
				"aggregations": []any{
					map[string]any{"function": "distinct", "column": "category_id", "alias": "unique_categories"},
				},
			},
			expectedSQL: `SELECT COUNT(DISTINCT "category_id") AS "unique_categories" FROM "products"`,
		},
		{
			name: "sum",
			query: map[string]any{
				"table": "order_details",
				"aggregations": []any{
					map[string]any{"function": "sum", "column": "quantity", "alias": "total_qty"},
				},
			},
			expectedSQL: `SELECT SUM("quantity") AS "total_qty" FROM "order_details"`,
		},
		{
			name: "avg",
			query: map[string]any{
				"table": "products",
				"aggregations": []any{
					map[string]any{"function": "avg", "column": "unit_price", "alias": "avg_price"},
				},
			},
			expectedSQL: `SELECT AVG("unit_price") AS "avg_price" FROM "products"`,
		},
		{
			name: "min max",
			query: map[string]any{
				"table": "products",
				"aggregations": []any{
					map[string]any{"function": "min", "column": "unit_price", "alias": "min_price"},
					map[string]any{"function": "max", "column": "unit_price", "alias": "max_price"},
				},
			},
			expectedSQL: `SELECT MIN("unit_price") AS "min_price", MAX("unit_price") AS "max_price" FROM "products"`,
		},
		{
			name: "aggregation with group by",
			query: map[string]any{
				"table": "products",
				"aggregations": []any{
					map[string]any{"function": "count", "alias": "count"},
				},
				"group_by": []any{"category_id"},
			},
			expectedSQL: `SELECT COUNT(*) AS "count", "category_id" FROM "products" GROUP BY "category_id"`,
		},
		{
			name: "distinct without column should fail",
			query: map[string]any{
				"table": "products",
				"aggregations": []any{
					map[string]any{"function": "distinct"},
				},
			},
			expectErr: true,
		},
		{
			name: "sum without column should fail",
			query: map[string]any{
				"table": "products",
				"aggregations": []any{
					map[string]any{"function": "sum"},
				},
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

// ============================================================================
// JOIN Tests
// ============================================================================

func TestBuildSQLFromQuery_Joins(t *testing.T) {
	tests := []struct {
		name        string
		query       map[string]any
		expectedSQL string
		expectErr   bool
	}{
		{
			name: "simple left join",
			query: map[string]any{
				"table": "products",
				"joins": []any{
					map[string]any{
						"type":         "left",
						"target_table": "categories",
						"conditions": []any{
							map[string]any{"source_column": "category_id", "target_column": "id"},
						},
					},
				},
			},
			expectedSQL: `SELECT * FROM "products" LEFT JOIN "categories" ON "products"."category_id" = "categories"."id"`,
		},
		{
			name: "inner join",
			query: map[string]any{
				"table": "orders",
				"joins": []any{
					map[string]any{
						"type":         "inner",
						"target_table": "customers",
						"conditions": []any{
							map[string]any{"source_column": "customer_id", "target_column": "id"},
						},
					},
				},
			},
			expectedSQL: `SELECT * FROM "orders" INNER JOIN "customers" ON "orders"."customer_id" = "customers"."id"`,
		},
		{
			name: "right join",
			query: map[string]any{
				"table": "order_details",
				"joins": []any{
					map[string]any{
						"type":         "right",
						"target_table": "products",
						"conditions": []any{
							map[string]any{"source_column": "product_id", "target_column": "id"},
						},
					},
				},
			},
			expectedSQL: `SELECT * FROM "order_details" RIGHT JOIN "products" ON "order_details"."product_id" = "products"."id"`,
		},
		{
			name: "multiple joins",
			query: map[string]any{
				"table": "order_details",
				"joins": []any{
					map[string]any{
						"type":         "left",
						"target_table": "orders",
						"conditions": []any{
							map[string]any{"source_column": "order_id", "target_column": "id"},
						},
					},
					map[string]any{
						"type":         "left",
						"target_table": "products",
						"conditions": []any{
							map[string]any{"source_column": "product_id", "target_column": "id"},
						},
					},
				},
			},
			expectedSQL: `SELECT * FROM "order_details" LEFT JOIN "orders" ON "order_details"."order_id" = "orders"."id" LEFT JOIN "products" ON "order_details"."product_id" = "products"."id"`,
		},
		{
			name: "join without conditions should fail",
			query: map[string]any{
				"table": "products",
				"joins": []any{
					map[string]any{
						"type":         "left",
						"target_table": "categories",
					},
				},
			},
			expectErr: true,
		},
		{
			name: "join without target_table should fail",
			query: map[string]any{
				"table": "products",
				"joins": []any{
					map[string]any{
						"type": "left",
						"conditions": []any{
							map[string]any{"source_column": "category_id", "target_column": "id"},
						},
					},
				},
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

// ============================================================================
// HAVING Tests
// ============================================================================

func TestBuildSQLFromQuery_Having(t *testing.T) {
	tests := []struct {
		name           string
		query          map[string]any
		expectedSQL    string
		expectedParams []any
		expectErr      bool
	}{
		{
			name: "having with count",
			query: map[string]any{
				"table": "products",
				"aggregations": []any{
					map[string]any{"function": "count", "alias": "product_count"},
				},
				"group_by": []any{"category_id"},
				"having": []any{
					map[string]any{"function": "count", "operator": ">", "value": 5},
				},
			},
			expectedSQL:    `SELECT COUNT(*) AS "product_count", "category_id" FROM "products" GROUP BY "category_id" HAVING COUNT(*) > ?`,
			expectedParams: []any{5},
		},
		{
			name: "having with sum",
			query: map[string]any{
				"table": "order_details",
				"aggregations": []any{
					map[string]any{"function": "sum", "column": "quantity", "alias": "total_qty"},
				},
				"group_by": []any{"product_id"},
				"having": []any{
					map[string]any{"function": "sum", "column": "quantity", "operator": ">=", "value": 100},
				},
			},
			expectedSQL:    `SELECT SUM("quantity") AS "total_qty", "product_id" FROM "order_details" GROUP BY "product_id" HAVING SUM("quantity") >= ?`,
			expectedParams: []any{100},
		},
		{
			name: "multiple having conditions",
			query: map[string]any{
				"table": "products",
				"aggregations": []any{
					map[string]any{"function": "count", "alias": "cnt"},
					map[string]any{"function": "avg", "column": "unit_price", "alias": "avg_price"},
				},
				"group_by": []any{"category_id"},
				"having": []any{
					map[string]any{"function": "count", "operator": ">", "value": 3},
					map[string]any{"function": "avg", "column": "unit_price", "operator": "<", "value": 50.0},
				},
			},
			expectedSQL:    `SELECT COUNT(*) AS "cnt", AVG("unit_price") AS "avg_price", "category_id" FROM "products" GROUP BY "category_id" HAVING COUNT(*) > ? AND AVG("unit_price") < ?`,
			expectedParams: []any{3, 50.0},
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

// ============================================================================
// String Filter Tests (contains, starts-with, ends-with)
// ============================================================================

func TestBuildSQLFromQuery_StringFilters(t *testing.T) {
	tests := []struct {
		name           string
		query          map[string]any
		expectedSQL    string
		expectedParams []any
	}{
		{
			name: "contains filter",
			query: map[string]any{
				"table": "products",
				"filters": []any{
					map[string]any{"column": "name", "operator": "contains", "value": "Sauce"},
				},
			},
			expectedSQL:    `SELECT * FROM "products" WHERE "name" LIKE ?`,
			expectedParams: []any{"%Sauce%"},
		},
		{
			name: "not-contains filter",
			query: map[string]any{
				"table": "products",
				"filters": []any{
					map[string]any{"column": "name", "operator": "not-contains", "value": "Sauce"},
				},
			},
			expectedSQL:    `SELECT * FROM "products" WHERE "name" NOT LIKE ?`,
			expectedParams: []any{"%Sauce%"},
		},
		{
			name: "starts-with filter",
			query: map[string]any{
				"table": "customers",
				"filters": []any{
					map[string]any{"column": "company_name", "operator": "starts-with", "value": "Alpha"},
				},
			},
			expectedSQL:    `SELECT * FROM "customers" WHERE "company_name" LIKE ?`,
			expectedParams: []any{"Alpha%"},
		},
		{
			name: "ends-with filter",
			query: map[string]any{
				"table": "customers",
				"filters": []any{
					map[string]any{"column": "company_name", "operator": "ends-with", "value": "Inc"},
				},
			},
			expectedSQL:    `SELECT * FROM "customers" WHERE "company_name" LIKE ?`,
			expectedParams: []any{"%Inc"},
		},
		{
			name: "is-empty filter",
			query: map[string]any{
				"table": "customers",
				"filters": []any{
					map[string]any{"column": "region", "operator": "is-empty"},
				},
			},
			expectedSQL:    `SELECT * FROM "customers" WHERE ("region" IS NULL OR "region" = '')`,
			expectedParams: nil,
		},
		{
			name: "is-not-empty filter",
			query: map[string]any{
				"table": "customers",
				"filters": []any{
					map[string]any{"column": "region", "operator": "is-not-empty"},
				},
			},
			expectedSQL:    `SELECT * FROM "customers" WHERE ("region" IS NOT NULL AND "region" != '')`,
			expectedParams: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, params, err := buildSQLFromQuery(tt.query)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSQL, sql)
			assert.Equal(t, tt.expectedParams, params)
		})
	}
}

// ============================================================================
// Relative Date Filter Tests
// ============================================================================

func TestBuildSQLFromQuery_RelativeDateFilters(t *testing.T) {
	tests := []struct {
		name         string
		query        map[string]any
		sqlContains  string
		paramsCount  int
		expectErr    bool
	}{
		{
			name: "last 30 days",
			query: map[string]any{
				"table": "orders",
				"filters": []any{
					map[string]any{
						"column":   "order_date",
						"operator": "relative",
						"value":    map[string]any{"type": "last", "amount": float64(30), "unit": "day"},
					},
				},
			},
			sqlContains: `>= date('now', ?)`,
			paramsCount: 1,
		},
		{
			name: "last 7 days (week)",
			query: map[string]any{
				"table": "orders",
				"filters": []any{
					map[string]any{
						"column":   "order_date",
						"operator": "relative",
						"value":    map[string]any{"type": "last", "amount": float64(1), "unit": "week"},
					},
				},
			},
			sqlContains: `>= date('now', ?)`,
			paramsCount: 1,
		},
		{
			name: "this month",
			query: map[string]any{
				"table": "orders",
				"filters": []any{
					map[string]any{
						"column":   "order_date",
						"operator": "relative",
						"value":    map[string]any{"type": "this", "unit": "month"},
					},
				},
			},
			sqlContains: `start of month`,
			paramsCount: 0,
		},
		{
			name: "this year",
			query: map[string]any{
				"table": "orders",
				"filters": []any{
					map[string]any{
						"column":   "order_date",
						"operator": "relative",
						"value":    map[string]any{"type": "this", "unit": "year"},
					},
				},
			},
			sqlContains: `start of year`,
			paramsCount: 0,
		},
		{
			name: "previous month",
			query: map[string]any{
				"table": "orders",
				"filters": []any{
					map[string]any{
						"column":   "order_date",
						"operator": "relative",
						"value":    map[string]any{"type": "previous", "unit": "month"},
					},
				},
			},
			sqlContains: `-1 month`,
			paramsCount: 0,
		},
		{
			name: "invalid unit should fail",
			query: map[string]any{
				"table": "orders",
				"filters": []any{
					map[string]any{
						"column":   "order_date",
						"operator": "relative",
						"value":    map[string]any{"type": "last", "amount": float64(30), "unit": "invalid"},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "invalid type should fail",
			query: map[string]any{
				"table": "orders",
				"filters": []any{
					map[string]any{
						"column":   "order_date",
						"operator": "relative",
						"value":    map[string]any{"type": "invalid", "amount": float64(30), "unit": "day"},
					},
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, params, err := buildSQLFromQuery(tt.query)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Contains(t, sql, tt.sqlContains)
				assert.Len(t, params, tt.paramsCount)
			}
		})
	}
}

// ============================================================================
// Complex Query Tests
// ============================================================================

func TestBuildSQLFromQuery_ComplexQuery(t *testing.T) {
	// Test a complex query with multiple features
	query := map[string]any{
		"table":   "products",
		"columns": []any{"name", "unit_price"},
		"joins": []any{
			map[string]any{
				"type":         "left",
				"target_table": "categories",
				"conditions": []any{
					map[string]any{"source_column": "category_id", "target_column": "id"},
				},
			},
		},
		"filters": []any{
			map[string]any{"column": "discontinued", "operator": "=", "value": 0},
			map[string]any{"column": "unit_price", "operator": ">", "value": 10.0},
		},
		"order_by": []any{
			map[string]any{"column": "unit_price", "direction": "desc"},
		},
		"limit": float64(50),
	}

	sql, params, err := buildSQLFromQuery(query)
	require.NoError(t, err)

	// Verify all parts are present
	assert.Contains(t, sql, `SELECT "name", "unit_price"`)
	assert.Contains(t, sql, `FROM "products"`)
	assert.Contains(t, sql, `LEFT JOIN "categories"`)
	assert.Contains(t, sql, `WHERE "discontinued" = ?`)
	assert.Contains(t, sql, `AND "unit_price" > ?`)
	assert.Contains(t, sql, `ORDER BY "unit_price" DESC`)
	assert.Contains(t, sql, `LIMIT 50`)

	// Verify params
	assert.Len(t, params, 2)
	assert.Equal(t, 0, params[0])
	assert.Equal(t, 10.0, params[1])
}
