package query

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// VariableType represents the type of a query variable
type VariableType string

const (
	VariableTypeText   VariableType = "text"
	VariableTypeNumber VariableType = "number"
	VariableTypeDate   VariableType = "date"
)

// Variable represents a parsed variable from a SQL query
type Variable struct {
	Name     string       `json:"name"`
	Type     VariableType `json:"type"`
	Start    int          `json:"start"`
	End      int          `json:"end"`
	Required bool         `json:"required"`
}

// VariableValue represents a value for a variable
type VariableValue struct {
	Type  VariableType `json:"type"`
	Value any          `json:"value"`
}

var variableRegex = regexp.MustCompile(`\{\{(\w+)\}\}`)

// ParseVariables extracts {{variable}} patterns from SQL
func ParseVariables(sql string) []Variable {
	matches := variableRegex.FindAllStringSubmatchIndex(sql, -1)
	seen := make(map[string]bool)
	var variables []Variable

	for _, match := range matches {
		name := sql[match[2]:match[3]]
		if seen[name] {
			continue
		}
		seen[name] = true

		variables = append(variables, Variable{
			Name:     name,
			Type:     InferVariableType(sql, name),
			Start:    match[0],
			End:      match[1],
			Required: true,
		})
	}

	return variables
}

// InferVariableType attempts to infer the type of a variable from its context
func InferVariableType(sql string, varName string) VariableType {
	lowerVar := strings.ToLower(varName)

	// Date patterns
	datePatterns := []string{
		"date", "time", "created", "updated", "_at", "timestamp",
	}
	for _, pattern := range datePatterns {
		if strings.Contains(lowerVar, pattern) {
			return VariableTypeDate
		}
	}

	// Check for BETWEEN context (often dates)
	betweenRegex := regexp.MustCompile(`(?i)between\s+\{\{` + varName + `\}\}`)
	if betweenRegex.MatchString(sql) {
		return VariableTypeDate
	}

	// Number patterns
	numberPatterns := []string{
		"price", "amount", "count", "qty", "quantity", "id", "_id",
		"limit", "offset", "min", "max", "total",
	}
	for _, pattern := range numberPatterns {
		if strings.Contains(lowerVar, pattern) {
			return VariableTypeNumber
		}
	}

	// Check for comparison operators (often numbers)
	comparisonRegex := regexp.MustCompile(`(?i)\{\{` + varName + `\}\}\s*[<>]`)
	if comparisonRegex.MatchString(sql) {
		return VariableTypeNumber
	}

	// Check for LIMIT context
	limitRegex := regexp.MustCompile(`(?i)limit\s+\{\{` + varName + `\}\}`)
	if limitRegex.MatchString(sql) {
		return VariableTypeNumber
	}

	// Default to text
	return VariableTypeText
}

// SubstituteVariables replaces {{var}} placeholders with parameterized placeholders
// Returns the modified SQL, ordered parameters, and any error
func SubstituteVariables(sql string, values map[string]VariableValue) (string, []any, error) {
	var params []any
	var lastEnd int
	var result strings.Builder

	matches := variableRegex.FindAllStringSubmatchIndex(sql, -1)

	for _, match := range matches {
		// Add the text before this match
		result.WriteString(sql[lastEnd:match[0]])

		varName := sql[match[2]:match[3]]
		value, ok := values[varName]
		if !ok {
			return "", nil, fmt.Errorf("missing value for variable: %s", varName)
		}

		// Convert value to appropriate type
		param, err := convertValue(value)
		if err != nil {
			return "", nil, fmt.Errorf("invalid value for %s: %w", varName, err)
		}

		params = append(params, param)
		result.WriteString("?")

		lastEnd = match[1]
	}

	// Add remaining text
	result.WriteString(sql[lastEnd:])

	return result.String(), params, nil
}

// convertValue converts a VariableValue to its appropriate Go type
func convertValue(v VariableValue) (any, error) {
	if v.Value == nil {
		return nil, nil
	}

	switch v.Type {
	case VariableTypeNumber:
		switch val := v.Value.(type) {
		case float64:
			return val, nil
		case int:
			return val, nil
		case int64:
			return val, nil
		case string:
			// Try to parse as number
			var f float64
			if _, err := fmt.Sscanf(val, "%f", &f); err == nil {
				return f, nil
			}
			return nil, fmt.Errorf("cannot convert %q to number", val)
		default:
			return v.Value, nil
		}

	case VariableTypeDate:
		switch val := v.Value.(type) {
		case string:
			// Try common date formats
			formats := []string{
				"2006-01-02",
				"2006-01-02T15:04:05Z",
				"2006-01-02T15:04:05-07:00",
				"2006-01-02 15:04:05",
				time.RFC3339,
			}
			for _, format := range formats {
				if t, err := time.Parse(format, val); err == nil {
					return t.Format("2006-01-02"), nil
				}
			}
			// Return as-is if no format matches
			return val, nil
		case time.Time:
			return val.Format("2006-01-02"), nil
		default:
			return v.Value, nil
		}

	default:
		// Text or unknown - return as string
		return fmt.Sprintf("%v", v.Value), nil
	}
}

// ValidateSQL performs basic SQL validation
func ValidateSQL(sql string) []string {
	var warnings []string

	// Check for dangerous patterns
	dangerousPatterns := []struct {
		pattern string
		message string
	}{
		{`(?i)\bDROP\b`, "Query contains DROP statement"},
		{`(?i)\bDELETE\s+FROM\b`, "Query contains DELETE statement"},
		{`(?i)\bTRUNCATE\b`, "Query contains TRUNCATE statement"},
		{`(?i)\bALTER\b`, "Query contains ALTER statement"},
		{`(?i)\bCREATE\b`, "Query contains CREATE statement"},
		{`(?i)\bINSERT\b`, "Query contains INSERT statement"},
		{`(?i)\bUPDATE\b`, "Query contains UPDATE statement"},
	}

	for _, dp := range dangerousPatterns {
		if matched, _ := regexp.MatchString(dp.pattern, sql); matched {
			warnings = append(warnings, dp.message)
		}
	}

	// Check for potential issues
	if !strings.Contains(strings.ToUpper(sql), "LIMIT") {
		warnings = append(warnings, "Query has no LIMIT clause - may return many rows")
	}

	// Check for SELECT *
	if matched, _ := regexp.MatchString(`(?i)SELECT\s+\*`, sql); matched {
		warnings = append(warnings, "Query uses SELECT * - consider specifying columns")
	}

	return warnings
}

// HasVariables checks if the SQL contains any variables
func HasVariables(sql string) bool {
	return variableRegex.MatchString(sql)
}

// ExtractVariableNames returns just the names of variables in the SQL
func ExtractVariableNames(sql string) []string {
	matches := variableRegex.FindAllStringSubmatch(sql, -1)
	seen := make(map[string]bool)
	var names []string

	for _, match := range matches {
		name := match[1]
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}

	return names
}
