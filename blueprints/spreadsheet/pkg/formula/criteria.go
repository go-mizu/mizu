package formula

import (
	"regexp"
	"strconv"
	"strings"
)

// CriteriaOperator represents the comparison operator in a criteria.
type CriteriaOperator int

const (
	OpEqual CriteriaOperator = iota
	OpNotEqual
	OpGreater
	OpGreaterEqual
	OpLess
	OpLessEqual
	OpContains   // wildcard match
	OpStartsWith // wildcard at end
	OpEndsWith   // wildcard at start
	OpRegex      // for future regex support
)

// Criteria represents a parsed criteria expression.
type Criteria struct {
	Operator CriteriaOperator
	Value    any
	Pattern  *regexp.Regexp // for wildcard/regex matching
}

// ParseCriteria parses a criteria string like ">10", "=text", "<>0", "abc*".
func ParseCriteria(criteriaStr any) *Criteria {
	if criteriaStr == nil {
		return &Criteria{Operator: OpEqual, Value: nil}
	}

	// If it's a number, treat as equality
	if n, ok := toNumber(criteriaStr); ok {
		return &Criteria{Operator: OpEqual, Value: n}
	}

	// If it's a bool, treat as equality
	if b, ok := criteriaStr.(bool); ok {
		return &Criteria{Operator: OpEqual, Value: b}
	}

	str := toString(criteriaStr)
	if str == "" {
		return &Criteria{Operator: OpEqual, Value: ""}
	}

	// Parse operator prefix
	if strings.HasPrefix(str, "<>") {
		return parseCriteriaValue(str[2:], OpNotEqual)
	}
	if strings.HasPrefix(str, ">=") {
		return parseCriteriaValue(str[2:], OpGreaterEqual)
	}
	if strings.HasPrefix(str, "<=") {
		return parseCriteriaValue(str[2:], OpLessEqual)
	}
	if strings.HasPrefix(str, "<") {
		return parseCriteriaValue(str[1:], OpLess)
	}
	if strings.HasPrefix(str, ">") {
		return parseCriteriaValue(str[1:], OpGreater)
	}
	if strings.HasPrefix(str, "=") {
		return parseCriteriaValue(str[1:], OpEqual)
	}

	// Check for wildcards
	hasWildcard := strings.Contains(str, "*") || strings.Contains(str, "?")
	if hasWildcard {
		return parseWildcardCriteria(str)
	}

	// Plain text - equality match
	if n, err := strconv.ParseFloat(str, 64); err == nil {
		return &Criteria{Operator: OpEqual, Value: n}
	}
	return &Criteria{Operator: OpEqual, Value: str}
}

// parseCriteriaValue parses the value part of a criteria with given operator.
func parseCriteriaValue(valueStr string, op CriteriaOperator) *Criteria {
	valueStr = strings.TrimSpace(valueStr)

	// Try to parse as number
	if n, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return &Criteria{Operator: op, Value: n}
	}

	// Try to parse as bool
	lower := strings.ToLower(valueStr)
	if lower == "true" {
		return &Criteria{Operator: op, Value: true}
	}
	if lower == "false" {
		return &Criteria{Operator: op, Value: false}
	}

	// Text value
	return &Criteria{Operator: op, Value: valueStr}
}

// parseWildcardCriteria parses a wildcard pattern (* and ? supported).
func parseWildcardCriteria(pattern string) *Criteria {
	// Convert wildcard to regex
	// * matches any sequence, ? matches single character
	var regexStr strings.Builder
	regexStr.WriteString("^")
	for _, r := range pattern {
		switch r {
		case '*':
			regexStr.WriteString(".*")
		case '?':
			regexStr.WriteString(".")
		case '.', '+', '^', '$', '(', ')', '[', ']', '{', '}', '|', '\\':
			regexStr.WriteString("\\" + string(r))
		default:
			regexStr.WriteString(string(r))
		}
	}
	regexStr.WriteString("$")

	re, err := regexp.Compile("(?i)" + regexStr.String()) // case insensitive
	if err != nil {
		// Fallback to exact match
		return &Criteria{Operator: OpEqual, Value: pattern}
	}

	return &Criteria{Operator: OpContains, Pattern: re}
}

// Matches checks if a value matches the criteria.
func (c *Criteria) Matches(value any) bool {
	if c == nil {
		return true
	}

	// Handle nil values
	if value == nil {
		if c.Value == nil {
			return c.Operator == OpEqual
		}
		// nil compared to non-nil
		switch c.Operator {
		case OpNotEqual:
			return true
		case OpEqual:
			return c.Value == "" // empty string matches nil
		default:
			return false
		}
	}

	// Handle pattern matching
	if c.Pattern != nil {
		return c.Pattern.MatchString(toString(value))
	}

	// Get numeric values if possible
	valueNum, valueIsNum := toNumber(value)
	criteriaNum, criteriaIsNum := toNumber(c.Value)

	switch c.Operator {
	case OpEqual:
		if valueIsNum && criteriaIsNum {
			return valueNum == criteriaNum
		}
		return strings.EqualFold(toString(value), toString(c.Value))

	case OpNotEqual:
		if valueIsNum && criteriaIsNum {
			return valueNum != criteriaNum
		}
		return !strings.EqualFold(toString(value), toString(c.Value))

	case OpGreater:
		if valueIsNum && criteriaIsNum {
			return valueNum > criteriaNum
		}
		return toString(value) > toString(c.Value)

	case OpGreaterEqual:
		if valueIsNum && criteriaIsNum {
			return valueNum >= criteriaNum
		}
		return toString(value) >= toString(c.Value)

	case OpLess:
		if valueIsNum && criteriaIsNum {
			return valueNum < criteriaNum
		}
		return toString(value) < toString(c.Value)

	case OpLessEqual:
		if valueIsNum && criteriaIsNum {
			return valueNum <= criteriaNum
		}
		return toString(value) <= toString(c.Value)
	}

	return false
}

// evalCriteriaRange evaluates a range against a criteria and returns matching indices.
func evalCriteriaRange(criteriaRange any, criteria *Criteria) []int {
	values := flattenValues(criteriaRange)
	indices := make([]int, 0)

	for i, v := range values {
		if criteria.Matches(v) {
			indices = append(indices, i)
		}
	}

	return indices
}

// getValuesByIndices gets values from a range by indices.
func getValuesByIndices(sumRange any, indices []int) []any {
	values := flattenValues(sumRange)
	result := make([]any, 0, len(indices))

	for _, idx := range indices {
		if idx >= 0 && idx < len(values) {
			result = append(result, values[idx])
		}
	}

	return result
}
