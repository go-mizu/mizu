package duckdb

import (
	"fmt"
	"strings"
)

// WhereBuilder builds SQL WHERE clauses from query objects with AND/OR support.
type WhereBuilder struct{}

// NewWhereBuilder creates a new WHERE builder.
func NewWhereBuilder() *WhereBuilder {
	return &WhereBuilder{}
}

// Build builds a WHERE clause from a query object.
func (w *WhereBuilder) Build(where map[string]any) (string, []any) {
	if len(where) == 0 {
		return "", nil
	}

	clause, args := w.buildConditions(where)
	if clause == "" {
		return "", nil
	}

	return "WHERE " + clause, args
}

func (w *WhereBuilder) buildConditions(where map[string]any) (string, []any) {
	if len(where) == 0 {
		return "", nil
	}

	var conditions []string
	var args []any

	for key, value := range where {
		switch key {
		case "and":
			clause, clauseArgs := w.buildAnd(value)
			if clause != "" {
				conditions = append(conditions, clause)
				args = append(args, clauseArgs...)
			}
		case "or":
			clause, clauseArgs := w.buildOr(value)
			if clause != "" {
				conditions = append(conditions, clause)
				args = append(args, clauseArgs...)
			}
		default:
			clause, clauseArgs := w.buildFieldCondition(key, value)
			if clause != "" {
				conditions = append(conditions, clause)
				args = append(args, clauseArgs...)
			}
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return strings.Join(conditions, " AND "), args
}

func (w *WhereBuilder) buildAnd(value any) (string, []any) {
	clauses := toAnySlice(value)
	if len(clauses) == 0 {
		return "", nil
	}

	var conditions []string
	var args []any

	for _, clause := range clauses {
		if clauseMap, ok := clause.(map[string]any); ok {
			cond, condArgs := w.buildConditions(clauseMap)
			if cond != "" {
				conditions = append(conditions, "("+cond+")")
				args = append(args, condArgs...)
			}
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return "(" + strings.Join(conditions, " AND ") + ")", args
}

func (w *WhereBuilder) buildOr(value any) (string, []any) {
	clauses := toAnySlice(value)
	if len(clauses) == 0 {
		return "", nil
	}

	var conditions []string
	var args []any

	for _, clause := range clauses {
		if clauseMap, ok := clause.(map[string]any); ok {
			cond, condArgs := w.buildConditions(clauseMap)
			if cond != "" {
				conditions = append(conditions, "("+cond+")")
				args = append(args, condArgs...)
			}
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return "(" + strings.Join(conditions, " OR ") + ")", args
}

func (w *WhereBuilder) buildFieldCondition(field string, value any) (string, []any) {
	col := toSnakeCase(field)

	// Check if value is an operator map
	if opMap, ok := value.(map[string]any); ok {
		var conditions []string
		var args []any

		for op, opValue := range opMap {
			clause, opArgs := w.buildOperatorClause(col, op, opValue)
			if clause != "" {
				conditions = append(conditions, clause)
				args = append(args, opArgs...)
			}
		}

		if len(conditions) == 0 {
			return "", nil
		}

		return strings.Join(conditions, " AND "), args
	}

	// Simple equality
	return col + " = ?", []any{value}
}

func (w *WhereBuilder) buildOperatorClause(col, op string, value any) (string, []any) {
	switch op {
	case "equals":
		if value == nil {
			return col + " IS NULL", nil
		}
		return col + " = ?", []any{value}

	case "not_equals":
		if value == nil {
			return col + " IS NOT NULL", nil
		}
		return col + " != ?", []any{value}

	case "greater_than":
		return col + " > ?", []any{value}

	case "greater_than_equal":
		return col + " >= ?", []any{value}

	case "less_than":
		return col + " < ?", []any{value}

	case "less_than_equal":
		return col + " <= ?", []any{value}

	case "like":
		return "LOWER(" + col + ") LIKE LOWER(?)", []any{"%" + fmt.Sprint(value) + "%"}

	case "contains":
		return col + " LIKE ?", []any{"%" + fmt.Sprint(value) + "%"}

	case "not_contains":
		return col + " NOT LIKE ?", []any{"%" + fmt.Sprint(value) + "%"}

	case "in":
		return w.buildInClause(col, value, false)

	case "not_in":
		return w.buildInClause(col, value, true)

	case "exists":
		if value == true || value == "true" {
			return col + " IS NOT NULL", nil
		}
		return col + " IS NULL", nil

	default:
		return col + " = ?", []any{value}
	}
}

func (w *WhereBuilder) buildInClause(col string, value any, negate bool) (string, []any) {
	vals := toAnySlice(value)
	if len(vals) == 0 {
		if negate {
			return "1=1", nil
		}
		return "1=0", nil
	}

	placeholders := make([]string, len(vals))
	for i := range vals {
		placeholders[i] = "?"
	}

	operator := "IN"
	if negate {
		operator = "NOT IN"
	}

	return col + " " + operator + " (" + strings.Join(placeholders, ", ") + ")", vals
}

func toAnySlice(value any) []any {
	switch v := value.(type) {
	case []any:
		return v
	case []map[string]any:
		result := make([]any, len(v))
		for i, m := range v {
			result[i] = m
		}
		return result
	case []string:
		result := make([]any, len(v))
		for i, s := range v {
			result[i] = s
		}
		return result
	case []int:
		result := make([]any, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	case []float64:
		result := make([]any, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	default:
		return nil
	}
}

// BuildWhereClauseEnhanced builds WHERE clause with AND/OR support.
// This replaces the simple buildWhereClause function.
func BuildWhereClauseEnhanced(where map[string]any) (string, []any) {
	builder := NewWhereBuilder()
	return builder.Build(where)
}
