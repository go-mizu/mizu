// Package query provides utilities for building database queries.
package query

import (
	"fmt"
	"strings"

	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/records"
)

// FilterBuilder builds SQL WHERE clauses from filter specifications.
type FilterBuilder struct {
	fields    map[string]*fields.Field
	argOffset int
}

// NewFilterBuilder creates a new filter builder.
func NewFilterBuilder(fieldList []*fields.Field) *FilterBuilder {
	fieldMap := make(map[string]*fields.Field)
	for _, f := range fieldList {
		fieldMap[f.ID] = f
	}
	return &FilterBuilder{
		fields:    fieldMap,
		argOffset: 0,
	}
}

// SetArgOffset sets the starting offset for placeholder arguments.
func (b *FilterBuilder) SetArgOffset(offset int) {
	b.argOffset = offset
}

// Build generates a SQL WHERE clause and arguments from filters.
func (b *FilterBuilder) Build(filters []records.Filter, logic string) (string, []any) {
	if len(filters) == 0 {
		return "", nil
	}

	if logic == "" {
		logic = "and"
	}

	var conditions []string
	var args []any

	for _, filter := range filters {
		field := b.fields[filter.FieldID]
		if field == nil {
			continue
		}

		condition, filterArgs := b.buildCondition(filter, field, len(args)+b.argOffset+1)
		if condition != "" {
			conditions = append(conditions, condition)
			args = append(args, filterArgs...)
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	joiner := " AND "
	if strings.ToLower(logic) == "or" {
		joiner = " OR "
	}

	return "(" + strings.Join(conditions, joiner) + ")", args
}

// BuildSort generates ORDER BY clause from sort specifications.
func (b *FilterBuilder) BuildSort(sorts []records.SortSpec) string {
	if len(sorts) == 0 {
		return ""
	}

	var clauses []string
	for _, sort := range sorts {
		field := b.fields[sort.FieldID]
		if field == nil {
			continue
		}

		col := b.cellColumn(sort.FieldID, field)
		dir := "ASC"
		if strings.ToLower(sort.Direction) == "desc" {
			dir = "DESC"
		}
		clauses = append(clauses, fmt.Sprintf("%s %s NULLS LAST", col, dir))
	}

	if len(clauses) == 0 {
		return ""
	}

	return "ORDER BY " + strings.Join(clauses, ", ")
}

// BuildSearch generates a search condition across all text fields.
func (b *FilterBuilder) BuildSearch(query string, argOffset int) (string, []any) {
	if query == "" {
		return "", nil
	}

	var conditions []string
	searchPattern := "%" + strings.ToLower(query) + "%"

	// Search in all text-like fields
	for _, field := range b.fields {
		if isTextType(field.Type) {
			col := b.cellColumn(field.ID, field)
			conditions = append(conditions, fmt.Sprintf("LOWER(%s) LIKE $%d", col, argOffset))
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return "(" + strings.Join(conditions, " OR ") + ")", []any{searchPattern}
}

// buildCondition builds a single filter condition.
func (b *FilterBuilder) buildCondition(filter records.Filter, field *fields.Field, argNum int) (string, []any) {
	col := b.cellColumn(filter.FieldID, field)

	switch filter.Operator {
	case records.OpEquals:
		return fmt.Sprintf("%s = $%d", col, argNum), []any{filter.Value}

	case records.OpNotEquals:
		return fmt.Sprintf("(%s IS NULL OR %s != $%d)", col, col, argNum), []any{filter.Value}

	case records.OpContains:
		return fmt.Sprintf("LOWER(%s) LIKE $%d", col, argNum), []any{"%" + strings.ToLower(toString(filter.Value)) + "%"}

	case records.OpNotContains:
		return fmt.Sprintf("LOWER(%s) NOT LIKE $%d", col, argNum), []any{"%" + strings.ToLower(toString(filter.Value)) + "%"}

	case records.OpStartsWith:
		return fmt.Sprintf("LOWER(%s) LIKE $%d", col, argNum), []any{strings.ToLower(toString(filter.Value)) + "%"}

	case records.OpEndsWith:
		return fmt.Sprintf("LOWER(%s) LIKE $%d", col, argNum), []any{"%" + strings.ToLower(toString(filter.Value))}

	case records.OpIsEmpty:
		return fmt.Sprintf("(%s IS NULL OR %s = '')", col, col), nil

	case records.OpIsNotEmpty:
		return fmt.Sprintf("(%s IS NOT NULL AND %s != '')", col, col), nil

	case records.OpGreaterThan:
		numCol := b.numericColumn(filter.FieldID, field)
		return fmt.Sprintf("%s > $%d", numCol, argNum), []any{toFloat(filter.Value)}

	case records.OpGreaterThanOrEqual:
		numCol := b.numericColumn(filter.FieldID, field)
		return fmt.Sprintf("%s >= $%d", numCol, argNum), []any{toFloat(filter.Value)}

	case records.OpLessThan:
		numCol := b.numericColumn(filter.FieldID, field)
		return fmt.Sprintf("%s < $%d", numCol, argNum), []any{toFloat(filter.Value)}

	case records.OpLessThanOrEqual:
		numCol := b.numericColumn(filter.FieldID, field)
		return fmt.Sprintf("%s <= $%d", numCol, argNum), []any{toFloat(filter.Value)}

	case records.OpBetween:
		numCol := b.numericColumn(filter.FieldID, field)
		values, ok := filter.Value.([]any)
		if !ok || len(values) < 2 {
			return "", nil
		}
		return fmt.Sprintf("%s BETWEEN $%d AND $%d", numCol, argNum, argNum+1), []any{toFloat(values[0]), toFloat(values[1])}

	case records.OpIn, records.OpIsAnyOf:
		values := toSlice(filter.Value)
		if len(values) == 0 {
			return "", nil
		}
		placeholders := make([]string, len(values))
		args := make([]any, len(values))
		for i, v := range values {
			placeholders[i] = fmt.Sprintf("$%d", argNum+i)
			args[i] = v
		}
		return fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", ")), args

	case records.OpNotIn, records.OpIsNoneOf:
		values := toSlice(filter.Value)
		if len(values) == 0 {
			return "", nil
		}
		placeholders := make([]string, len(values))
		args := make([]any, len(values))
		for i, v := range values {
			placeholders[i] = fmt.Sprintf("$%d", argNum+i)
			args[i] = v
		}
		return fmt.Sprintf("(%s IS NULL OR %s NOT IN (%s))", col, col, strings.Join(placeholders, ", ")), args

	case records.OpIsBefore:
		return fmt.Sprintf("%s < $%d", col, argNum), []any{filter.Value}

	case records.OpIsAfter:
		return fmt.Sprintf("%s > $%d", col, argNum), []any{filter.Value}

	case records.OpIsChecked:
		return fmt.Sprintf("%s = 'true'", col), nil

	case records.OpIsUnchecked:
		return fmt.Sprintf("(%s IS NULL OR %s = 'false' OR %s = '')", col, col, col), nil

	default:
		return "", nil
	}
}

// cellColumn returns the SQL expression to access a cell value.
func (b *FilterBuilder) cellColumn(fieldID string, field *fields.Field) string {
	// DuckDB JSON extraction: cells->>'field_id'
	return fmt.Sprintf("cells->>'%s'", fieldID)
}

// numericColumn returns a SQL expression for numeric comparison.
func (b *FilterBuilder) numericColumn(fieldID string, field *fields.Field) string {
	// Cast to DOUBLE for numeric comparisons
	return fmt.Sprintf("TRY_CAST(cells->>'%s' AS DOUBLE)", fieldID)
}

// isTextType returns true if the field type is text-based.
func isTextType(fieldType string) bool {
	switch fieldType {
	case "text", "single_line_text", "long_text", "rich_text", "email", "url", "phone":
		return true
	default:
		return false
	}
}

// toString converts a value to string.
func toString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// toFloat converts a value to float64.
func toFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	default:
		return 0
	}
}

// toSlice converts a value to a slice.
func toSlice(v any) []any {
	switch val := v.(type) {
	case []any:
		return val
	case []string:
		result := make([]any, len(val))
		for i, s := range val {
			result[i] = s
		}
		return result
	default:
		return nil
	}
}
