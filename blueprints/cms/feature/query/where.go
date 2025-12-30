package query

import (
	"fmt"
	"strings"
)

// WhereBuilderService implements WHERE clause building with AND/OR support.
type WhereBuilderService struct{}

// NewWhereBuilder creates a new WHERE builder service.
func NewWhereBuilder() *WhereBuilderService {
	return &WhereBuilderService{}
}

// Build builds a WHERE clause from a query object.
// Supports nested AND/OR operations.
func (w *WhereBuilderService) Build(where map[string]any) (string, []any) {
	if len(where) == 0 {
		return "", nil
	}

	clause, args := w.buildConditions(where)
	if clause == "" {
		return "", nil
	}

	return "WHERE " + clause, args
}

func (w *WhereBuilderService) buildConditions(where map[string]any) (string, []any) {
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

func (w *WhereBuilderService) buildAnd(value any) (string, []any) {
	clauses, ok := value.([]any)
	if !ok {
		// Try []map[string]any
		if mapClauses, ok := value.([]map[string]any); ok {
			clauses = make([]any, len(mapClauses))
			for i, m := range mapClauses {
				clauses[i] = m
			}
		} else {
			return "", nil
		}
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

func (w *WhereBuilderService) buildOr(value any) (string, []any) {
	clauses, ok := value.([]any)
	if !ok {
		// Try []map[string]any
		if mapClauses, ok := value.([]map[string]any); ok {
			clauses = make([]any, len(mapClauses))
			for i, m := range mapClauses {
				clauses[i] = m
			}
		} else {
			return "", nil
		}
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

func (w *WhereBuilderService) buildFieldCondition(field string, value any) (string, []any) {
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

func (w *WhereBuilderService) buildOperatorClause(col, op string, value any) (string, []any) {
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
		// Case-insensitive like with wildcards
		return "LOWER(" + col + ") LIKE LOWER(?)", []any{"%" + fmt.Sprint(value) + "%"}

	case "contains":
		return col + " LIKE ?", []any{"%" + fmt.Sprint(value) + "%"}

	case "not_contains":
		return col + " NOT LIKE ?", []any{"%" + fmt.Sprint(value) + "%"}

	case "in":
		return w.buildInClause(col, value, false)

	case "not_in":
		return w.buildInClause(col, value, true)

	case "all":
		// For arrays that must contain all specified values
		// This is database-specific; simplified implementation
		return w.buildAllClause(col, value)

	case "exists":
		if value == true || value == "true" {
			return col + " IS NOT NULL", nil
		}
		return col + " IS NULL", nil

	case "near":
		// For point fields - requires special handling
		return w.buildNearClause(col, value)

	default:
		return col + " = ?", []any{value}
	}
}

func (w *WhereBuilderService) buildInClause(col string, value any, negate bool) (string, []any) {
	var vals []any

	switch v := value.(type) {
	case []any:
		vals = v
	case []string:
		vals = make([]any, len(v))
		for i, s := range v {
			vals[i] = s
		}
	case []int:
		vals = make([]any, len(v))
		for i, n := range v {
			vals[i] = n
		}
	case []float64:
		vals = make([]any, len(v))
		for i, n := range v {
			vals[i] = n
		}
	default:
		// Single value
		if negate {
			return col + " != ?", []any{value}
		}
		return col + " = ?", []any{value}
	}

	if len(vals) == 0 {
		if negate {
			return "1=1", nil // NOT IN empty list = always true
		}
		return "1=0", nil // IN empty list = always false
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

func (w *WhereBuilderService) buildAllClause(col string, value any) (string, []any) {
	// "all" operator requires all values to be present
	// This is typically used for array fields
	// Simplified implementation - may need database-specific handling
	vals, ok := value.([]any)
	if !ok {
		return col + " = ?", []any{value}
	}

	var conditions []string
	var args []any

	for _, v := range vals {
		conditions = append(conditions, col+" LIKE ?")
		args = append(args, "%"+fmt.Sprint(v)+"%")
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return "(" + strings.Join(conditions, " AND ") + ")", args
}

func (w *WhereBuilderService) buildNearClause(col string, value any) (string, []any) {
	// For geospatial queries
	// Expected format: {coordinates: [lng, lat], maxDistance: meters}
	nearOpts, ok := value.(map[string]any)
	if !ok {
		return "", nil
	}

	coords, ok := nearOpts["coordinates"].([]any)
	if !ok || len(coords) != 2 {
		return "", nil
	}

	// Simplified - actual implementation would use spatial functions
	// This is a placeholder that just checks for non-null
	return col + " IS NOT NULL", nil
}

// ParseWhereParam parses a WHERE clause from URL query parameters.
// Format: where[field][operator]=value
func ParseWhereParam(params map[string][]string) map[string]any {
	result := make(map[string]any)

	for key, values := range params {
		if !strings.HasPrefix(key, "where[") {
			continue
		}

		// Parse key: where[field][operator] or where[field]
		parts := parseWhereKey(key)
		if len(parts) < 1 {
			continue
		}

		field := parts[0]
		value := values[0]

		if len(parts) == 1 {
			// Simple equality: where[field]=value
			result[field] = value
		} else {
			// Operator: where[field][operator]=value
			operator := parts[1]
			if existing, ok := result[field].(map[string]any); ok {
				existing[operator] = parseValue(value)
			} else {
				result[field] = map[string]any{
					operator: parseValue(value),
				}
			}
		}
	}

	return result
}

func parseWhereKey(key string) []string {
	// Remove "where[" prefix
	key = strings.TrimPrefix(key, "where[")
	// Remove trailing "]"
	key = strings.TrimSuffix(key, "]")

	// Split by "][" to get parts
	return strings.Split(key, "][")
}

func parseValue(s string) any {
	// Try to parse as bool
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if s == "null" {
		return nil
	}

	// Check for comma-separated values (for in/not_in)
	if strings.Contains(s, ",") {
		parts := strings.Split(s, ",")
		result := make([]any, len(parts))
		for i, p := range parts {
			result[i] = strings.TrimSpace(p)
		}
		return result
	}

	return s
}
