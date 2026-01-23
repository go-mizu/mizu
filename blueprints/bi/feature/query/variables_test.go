package query

import (
	"testing"
)

func TestParseVariables(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected []string
	}{
		{
			name:     "single variable",
			sql:      "SELECT * FROM products WHERE category = {{category}}",
			expected: []string{"category"},
		},
		{
			name:     "multiple variables",
			sql:      "SELECT * FROM orders WHERE order_date >= {{start_date}} AND order_date <= {{end_date}}",
			expected: []string{"start_date", "end_date"},
		},
		{
			name:     "duplicate variables",
			sql:      "SELECT * FROM orders WHERE {{category}} = 1 OR {{category}} = 2",
			expected: []string{"category"},
		},
		{
			name:     "no variables",
			sql:      "SELECT * FROM products LIMIT 10",
			expected: nil,
		},
		{
			name:     "complex query",
			sql:      "SELECT p.name, c.name FROM products p JOIN categories c ON p.category_id = c.id WHERE p.unit_price > {{min_price}} AND c.name = {{category}} LIMIT {{limit}}",
			expected: []string{"min_price", "category", "limit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := ParseVariables(tt.sql)

			if len(vars) != len(tt.expected) {
				t.Errorf("expected %d variables, got %d", len(tt.expected), len(vars))
				return
			}

			for i, v := range vars {
				if v.Name != tt.expected[i] {
					t.Errorf("expected variable %q at position %d, got %q", tt.expected[i], i, v.Name)
				}
			}
		})
	}
}

func TestInferVariableType(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		varName  string
		expected VariableType
	}{
		{
			name:     "date in variable name",
			sql:      "SELECT * FROM orders WHERE order_date = {{start_date}}",
			varName:  "start_date",
			expected: VariableTypeDate,
		},
		{
			name:     "timestamp in variable name",
			sql:      "SELECT * FROM events WHERE created_at = {{event_timestamp}}",
			varName:  "event_timestamp",
			expected: VariableTypeDate,
		},
		{
			name:     "price in variable name",
			sql:      "SELECT * FROM products WHERE unit_price > {{min_price}}",
			varName:  "min_price",
			expected: VariableTypeNumber,
		},
		{
			name:     "id in variable name",
			sql:      "SELECT * FROM orders WHERE customer_id = {{customer_id}}",
			varName:  "customer_id",
			expected: VariableTypeNumber,
		},
		{
			name:     "limit variable",
			sql:      "SELECT * FROM products LIMIT {{limit}}",
			varName:  "limit",
			expected: VariableTypeNumber,
		},
		{
			name:     "comparison operator context",
			sql:      "SELECT * FROM products WHERE {{amount}} > 100",
			varName:  "amount",
			expected: VariableTypeNumber,
		},
		{
			name:     "text variable (default)",
			sql:      "SELECT * FROM products WHERE name = {{search}}",
			varName:  "search",
			expected: VariableTypeText,
		},
		{
			name:     "category variable",
			sql:      "SELECT * FROM products WHERE category = {{category}}",
			varName:  "category",
			expected: VariableTypeText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferVariableType(tt.sql, tt.varName)
			if result != tt.expected {
				t.Errorf("expected type %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSubstituteVariables(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		values      map[string]VariableValue
		expectedSQL string
		expectedLen int
		expectError bool
	}{
		{
			name: "single text variable",
			sql:  "SELECT * FROM products WHERE category = {{category}}",
			values: map[string]VariableValue{
				"category": {Type: VariableTypeText, Value: "Beverages"},
			},
			expectedSQL: "SELECT * FROM products WHERE category = ?",
			expectedLen: 1,
		},
		{
			name: "multiple variables",
			sql:  "SELECT * FROM products WHERE unit_price >= {{min_price}} AND unit_price <= {{max_price}}",
			values: map[string]VariableValue{
				"min_price": {Type: VariableTypeNumber, Value: 10.0},
				"max_price": {Type: VariableTypeNumber, Value: 50.0},
			},
			expectedSQL: "SELECT * FROM products WHERE unit_price >= ? AND unit_price <= ?",
			expectedLen: 2,
		},
		{
			name: "date variable",
			sql:  "SELECT * FROM orders WHERE order_date >= {{start_date}}",
			values: map[string]VariableValue{
				"start_date": {Type: VariableTypeDate, Value: "2024-01-01"},
			},
			expectedSQL: "SELECT * FROM orders WHERE order_date >= ?",
			expectedLen: 1,
		},
		{
			name:        "missing variable value",
			sql:         "SELECT * FROM products WHERE category = {{category}}",
			values:      map[string]VariableValue{},
			expectError: true,
		},
		{
			name: "duplicate variables get same placeholder",
			sql:  "SELECT * FROM orders WHERE {{category}} = 1 OR name LIKE {{category}}",
			values: map[string]VariableValue{
				"category": {Type: VariableTypeText, Value: "Test"},
			},
			expectedSQL: "SELECT * FROM orders WHERE ? = 1 OR name LIKE ?",
			expectedLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, params, err := SubstituteVariables(tt.sql, tt.values)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expectedSQL {
				t.Errorf("expected SQL %q, got %q", tt.expectedSQL, result)
			}

			if len(params) != tt.expectedLen {
				t.Errorf("expected %d params, got %d", tt.expectedLen, len(params))
			}
		})
	}
}

func TestValidateSQL(t *testing.T) {
	tests := []struct {
		name         string
		sql          string
		expectWarn   bool
		warnContains string
	}{
		{
			name:       "safe query with limit",
			sql:        "SELECT name, price FROM products LIMIT 100",
			expectWarn: false,
		},
		{
			name:         "no limit warning",
			sql:          "SELECT * FROM products",
			expectWarn:   true,
			warnContains: "no LIMIT",
		},
		{
			name:         "select star warning",
			sql:          "SELECT * FROM products LIMIT 100",
			expectWarn:   true,
			warnContains: "SELECT *",
		},
		{
			name:         "drop statement",
			sql:          "DROP TABLE products",
			expectWarn:   true,
			warnContains: "DROP",
		},
		{
			name:         "delete statement",
			sql:          "DELETE FROM products WHERE id = 1",
			expectWarn:   true,
			warnContains: "DELETE",
		},
		{
			name:         "insert statement",
			sql:          "INSERT INTO products VALUES (1, 'test')",
			expectWarn:   true,
			warnContains: "INSERT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := ValidateSQL(tt.sql)

			if tt.expectWarn {
				if len(warnings) == 0 {
					t.Error("expected warnings but got none")
					return
				}

				found := false
				for _, w := range warnings {
					if contains(w, tt.warnContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning containing %q, got %v", tt.warnContains, warnings)
				}
			}
		})
	}
}

func TestHasVariables(t *testing.T) {
	tests := []struct {
		sql      string
		expected bool
	}{
		{"SELECT * FROM products WHERE category = {{category}}", true},
		{"SELECT * FROM products LIMIT 10", false},
		{"SELECT * FROM products WHERE id = 1", false},
		{"SELECT {{col}} FROM products", true},
	}

	for _, tt := range tests {
		t.Run(tt.sql, func(t *testing.T) {
			if result := HasVariables(tt.sql); result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExtractVariableNames(t *testing.T) {
	sql := "SELECT * FROM orders WHERE date >= {{start_date}} AND date <= {{end_date}} AND status = {{status}}"
	names := ExtractVariableNames(sql)

	expected := []string{"start_date", "end_date", "status"}
	if len(names) != len(expected) {
		t.Errorf("expected %d names, got %d", len(expected), len(names))
		return
	}

	for i, name := range names {
		if name != expected[i] {
			t.Errorf("expected %q at position %d, got %q", expected[i], i, name)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
