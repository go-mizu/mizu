package postgrest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Filter represents a parsed filter condition.
type Filter struct {
	Column   string
	Operator string
	Value    interface{}
	Negated  bool
	IsJSON   bool
	JSONPath string
}

// OrderClause represents a parsed ORDER BY clause.
type OrderClause struct {
	Column     string
	Descending bool
	NullsFirst *bool
}

// SelectColumn represents a parsed select column.
type SelectColumn struct {
	Name       string
	Alias      string
	Cast       string
	JSONPath   string
	Embedded   *EmbeddedSelect
	IsSpread   bool
}

// EmbeddedSelect represents an embedded resource in select.
type EmbeddedSelect struct {
	Table      string
	Columns    []SelectColumn
	Hint       string // FK hint for disambiguation
	InnerJoin  bool   // !inner modifier
	Filters    []Filter
	Order      []OrderClause
	Limit      *int
	Offset     *int
}

// ParseSelect parses a PostgREST select string.
// Format: col1,col2,alias:col3,table(col1,col2),...table(*)
func ParseSelect(sel string) ([]SelectColumn, error) {
	if sel == "" || sel == "*" {
		return []SelectColumn{{Name: "*"}}, nil
	}

	var columns []SelectColumn
	// Split by comma, but respect parentheses for embedded resources
	parts := splitSelectParts(sel)

	for _, part := range parts {
		col, err := parseSelectColumn(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}

	return columns, nil
}

func splitSelectParts(sel string) []string {
	var parts []string
	var current strings.Builder
	depth := 0

	for _, r := range sel {
		switch r {
		case '(':
			depth++
			current.WriteRune(r)
		case ')':
			depth--
			current.WriteRune(r)
		case ',':
			if depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func parseSelectColumn(part string) (SelectColumn, error) {
	col := SelectColumn{}

	// Check for spread operator ...table(cols)
	if strings.HasPrefix(part, "...") {
		col.IsSpread = true
		part = part[3:]
	}

	// Check for embedded resource: table(cols) or table!hint(cols)
	if idx := strings.Index(part, "("); idx > 0 {
		tablePart := part[:idx]
		colsPart := part[idx+1 : len(part)-1]

		embedded := &EmbeddedSelect{}

		// Check for hints like !inner or !fk_name
		if bangIdx := strings.Index(tablePart, "!"); bangIdx > 0 {
			embedded.Table = tablePart[:bangIdx]
			hint := tablePart[bangIdx+1:]
			if hint == "inner" {
				embedded.InnerJoin = true
			} else if hint == "left" {
				// Default, do nothing
			} else {
				embedded.Hint = hint
			}
		} else {
			embedded.Table = tablePart
		}

		// Check for alias: alias:table
		if colonIdx := strings.Index(embedded.Table, ":"); colonIdx > 0 {
			col.Alias = embedded.Table[:colonIdx]
			embedded.Table = embedded.Table[colonIdx+1:]
		}

		// Parse embedded columns
		if colsPart != "" && colsPart != "*" {
			subCols, err := ParseSelect(colsPart)
			if err != nil {
				return col, err
			}
			embedded.Columns = subCols
		} else {
			embedded.Columns = []SelectColumn{{Name: "*"}}
		}

		col.Embedded = embedded
		return col, nil
	}

	// Check for alias: alias:column
	if colonIdx := strings.Index(part, ":"); colonIdx > 0 {
		col.Alias = part[:colonIdx]
		part = part[colonIdx+1:]
	}

	// Check for type cast: column::type
	if castIdx := strings.Index(part, "::"); castIdx > 0 {
		col.Cast = part[castIdx+2:]
		part = part[:castIdx]
	}

	// Check for JSON path: column->key or column->>key
	if jsonIdx := strings.Index(part, "->>"); jsonIdx > 0 {
		col.Name = part[:jsonIdx]
		col.JSONPath = "->>" + part[jsonIdx+3:]
	} else if jsonIdx := strings.Index(part, "->"); jsonIdx > 0 {
		col.Name = part[:jsonIdx]
		col.JSONPath = "->" + part[jsonIdx+2:]
	} else {
		col.Name = part
	}

	return col, nil
}

// ParseFilter parses a PostgREST filter value.
// Format: operator.value or not.operator.value
func ParseFilter(column, value string) (*Filter, error) {
	filter := &Filter{Column: column}

	// Check for JSON path in column
	if jsonIdx := strings.Index(column, "->>"); jsonIdx > 0 {
		filter.Column = column[:jsonIdx]
		filter.JSONPath = "->>" + column[jsonIdx+3:]
		filter.IsJSON = true
	} else if jsonIdx := strings.Index(column, "->"); jsonIdx > 0 {
		filter.Column = column[:jsonIdx]
		filter.JSONPath = "->" + column[jsonIdx+2:]
		filter.IsJSON = true
	}

	// Check for 'not.' prefix
	if strings.HasPrefix(value, "not.") {
		filter.Negated = true
		value = value[4:]
	}

	// Parse operator and value
	parts := strings.SplitN(value, ".", 2)
	if len(parts) < 2 {
		// No operator, treat as equality
		filter.Operator = "eq"
		filter.Value = value
		return filter, nil
	}

	filter.Operator = parts[0]
	filter.Value = parts[1]

	return filter, nil
}

// FilterToSQL converts a filter to SQL.
func FilterToSQL(f *Filter, paramIdx *int) (string, []interface{}, error) {
	var params []interface{}
	col := QuoteIdent(f.Column)

	// Handle JSON path
	if f.IsJSON && f.JSONPath != "" {
		col = col + f.JSONPath
	}

	var sql string
	var err error

	switch f.Operator {
	case "eq":
		sql = fmt.Sprintf("%s = $%d", col, *paramIdx)
		params = append(params, f.Value)
		*paramIdx++

	case "neq":
		sql = fmt.Sprintf("%s != $%d", col, *paramIdx)
		params = append(params, f.Value)
		*paramIdx++

	case "gt":
		sql = fmt.Sprintf("%s > $%d", col, *paramIdx)
		params = append(params, f.Value)
		*paramIdx++

	case "gte":
		sql = fmt.Sprintf("%s >= $%d", col, *paramIdx)
		params = append(params, f.Value)
		*paramIdx++

	case "lt":
		sql = fmt.Sprintf("%s < $%d", col, *paramIdx)
		params = append(params, f.Value)
		*paramIdx++

	case "lte":
		sql = fmt.Sprintf("%s <= $%d", col, *paramIdx)
		params = append(params, f.Value)
		*paramIdx++

	case "like":
		// Convert * to % for SQL LIKE
		val := strings.ReplaceAll(f.Value.(string), "*", "%")
		sql = fmt.Sprintf("%s LIKE $%d", col, *paramIdx)
		params = append(params, val)
		*paramIdx++

	case "ilike":
		val := strings.ReplaceAll(f.Value.(string), "*", "%")
		sql = fmt.Sprintf("%s ILIKE $%d", col, *paramIdx)
		params = append(params, val)
		*paramIdx++

	case "match":
		sql = fmt.Sprintf("%s ~ $%d", col, *paramIdx)
		params = append(params, f.Value)
		*paramIdx++

	case "imatch":
		sql = fmt.Sprintf("%s ~* $%d", col, *paramIdx)
		params = append(params, f.Value)
		*paramIdx++

	case "is":
		val := strings.ToUpper(f.Value.(string))
		switch val {
		case "NULL":
			sql = fmt.Sprintf("%s IS NULL", col)
		case "TRUE":
			sql = fmt.Sprintf("%s IS TRUE", col)
		case "FALSE":
			sql = fmt.Sprintf("%s IS FALSE", col)
		case "UNKNOWN":
			sql = fmt.Sprintf("%s IS UNKNOWN", col)
		default:
			return "", nil, fmt.Errorf("invalid value for 'is' operator: %s", f.Value)
		}

	case "isdistinct":
		sql = fmt.Sprintf("%s IS DISTINCT FROM $%d", col, *paramIdx)
		params = append(params, f.Value)
		*paramIdx++

	case "in":
		// Parse array: (val1,val2,val3)
		vals, err := parseArrayValue(f.Value.(string))
		if err != nil {
			return "", nil, err
		}
		placeholders := make([]string, len(vals))
		for i, v := range vals {
			placeholders[i] = fmt.Sprintf("$%d", *paramIdx)
			params = append(params, v)
			*paramIdx++
		}
		sql = fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", "))

	case "cs":
		// Contains (array or JSON)
		sql = fmt.Sprintf("%s @> $%d", col, *paramIdx)
		val, err := parseArrayOrJSON(f.Value.(string))
		if err != nil {
			return "", nil, err
		}
		params = append(params, val)
		*paramIdx++

	case "cd":
		// Contained by
		sql = fmt.Sprintf("%s <@ $%d", col, *paramIdx)
		val, err := parseArrayOrJSON(f.Value.(string))
		if err != nil {
			return "", nil, err
		}
		params = append(params, val)
		*paramIdx++

	case "ov":
		// Overlap
		sql = fmt.Sprintf("%s && $%d", col, *paramIdx)
		val, err := parseArrayOrJSON(f.Value.(string))
		if err != nil {
			return "", nil, err
		}
		params = append(params, val)
		*paramIdx++

	case "sl":
		// Strictly left
		sql = fmt.Sprintf("%s << $%d", col, *paramIdx)
		params = append(params, f.Value)
		*paramIdx++

	case "sr":
		// Strictly right
		sql = fmt.Sprintf("%s >> $%d", col, *paramIdx)
		params = append(params, f.Value)
		*paramIdx++

	case "nxl":
		// Does not extend left
		sql = fmt.Sprintf("%s &> $%d", col, *paramIdx)
		params = append(params, f.Value)
		*paramIdx++

	case "nxr":
		// Does not extend right
		sql = fmt.Sprintf("%s &< $%d", col, *paramIdx)
		params = append(params, f.Value)
		*paramIdx++

	case "adj":
		// Adjacent
		sql = fmt.Sprintf("%s -|- $%d", col, *paramIdx)
		params = append(params, f.Value)
		*paramIdx++

	case "fts", "plfts", "phfts", "wfts":
		// Full-text search
		sql, params, err = parseFTSFilter(f, col, paramIdx)
		if err != nil {
			return "", nil, err
		}

	default:
		return "", nil, fmt.Errorf("unknown operator: %s", f.Operator)
	}

	if f.Negated {
		sql = "NOT (" + sql + ")"
	}

	return sql, params, nil
}

func parseFTSFilter(f *Filter, col string, paramIdx *int) (string, []interface{}, error) {
	var params []interface{}
	val := f.Value.(string)

	// Check for language config: fts(english).query
	var config string
	if strings.Contains(f.Operator, "(") {
		re := regexp.MustCompile(`(\w+)\((\w+)\)`)
		matches := re.FindStringSubmatch(f.Operator)
		if len(matches) == 3 {
			f.Operator = matches[1]
			config = matches[2]
		}
	}

	var tsFunc string
	switch f.Operator {
	case "fts":
		tsFunc = "to_tsquery"
	case "plfts":
		tsFunc = "plainto_tsquery"
	case "phfts":
		tsFunc = "phraseto_tsquery"
	case "wfts":
		tsFunc = "websearch_to_tsquery"
	}

	var sql string
	if config != "" {
		sql = fmt.Sprintf("%s @@ %s('%s', $%d)", col, tsFunc, config, *paramIdx)
	} else {
		sql = fmt.Sprintf("%s @@ %s($%d)", col, tsFunc, *paramIdx)
	}
	params = append(params, val)
	*paramIdx++

	return sql, params, nil
}

func parseArrayValue(val string) ([]string, error) {
	// Format: (val1,val2,val3) or {val1,val2,val3}
	val = strings.TrimSpace(val)
	if strings.HasPrefix(val, "(") && strings.HasSuffix(val, ")") {
		val = val[1 : len(val)-1]
	} else if strings.HasPrefix(val, "{") && strings.HasSuffix(val, "}") {
		val = val[1 : len(val)-1]
	}

	if val == "" {
		return []string{}, nil
	}

	// Split by comma, but handle quoted values
	var values []string
	var current strings.Builder
	inQuote := false

	for _, r := range val {
		switch r {
		case '"':
			inQuote = !inQuote
		case ',':
			if !inQuote {
				values = append(values, strings.TrimSpace(current.String()))
				current.Reset()
				continue
			}
		}
		current.WriteRune(r)
	}

	if current.Len() > 0 {
		values = append(values, strings.TrimSpace(current.String()))
	}

	return values, nil
}

func parseArrayOrJSON(val string) (interface{}, error) {
	val = strings.TrimSpace(val)

	// JSON object
	if strings.HasPrefix(val, "{") && strings.HasSuffix(val, "}") {
		// Check if it's a PostgreSQL array or JSON
		if strings.Contains(val, ":") || strings.Contains(val, `"`) {
			// Likely JSON
			return val, nil
		}
		// PostgreSQL array format
		return val, nil
	}

	return val, nil
}

// ParseOrder parses a PostgREST order string.
// Format: col1.asc,col2.desc.nullsfirst
func ParseOrder(order string) ([]OrderClause, error) {
	if order == "" {
		return nil, nil
	}

	var clauses []OrderClause
	parts := strings.Split(order, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		clause := OrderClause{}
		segments := strings.Split(part, ".")

		if len(segments) == 0 {
			continue
		}

		clause.Column = segments[0]

		for i := 1; i < len(segments); i++ {
			seg := strings.ToLower(segments[i])
			switch seg {
			case "asc":
				clause.Descending = false
			case "desc":
				clause.Descending = true
			case "nullsfirst":
				t := true
				clause.NullsFirst = &t
			case "nullslast":
				f := false
				clause.NullsFirst = &f
			}
		}

		clauses = append(clauses, clause)
	}

	return clauses, nil
}

// OrderToSQL converts order clauses to SQL.
func OrderToSQL(clauses []OrderClause) string {
	if len(clauses) == 0 {
		return ""
	}

	var parts []string
	for _, c := range clauses {
		part := QuoteIdent(c.Column)
		if c.Descending {
			part += " DESC"
		} else {
			part += " ASC"
		}
		if c.NullsFirst != nil {
			if *c.NullsFirst {
				part += " NULLS FIRST"
			} else {
				part += " NULLS LAST"
			}
		}
		parts = append(parts, part)
	}

	return "ORDER BY " + strings.Join(parts, ", ")
}

// ParseLogicalFilter parses and() and or() filter expressions.
// Format: and(filter1,filter2) or or(filter1,filter2)
func ParseLogicalFilter(value string) (string, []Filter, error) {
	value = strings.TrimSpace(value)

	if strings.HasPrefix(value, "and(") && strings.HasSuffix(value, ")") {
		inner := value[4 : len(value)-1]
		filters, err := parseLogicalInner(inner)
		return "AND", filters, err
	}

	if strings.HasPrefix(value, "or(") && strings.HasSuffix(value, ")") {
		inner := value[3 : len(value)-1]
		filters, err := parseLogicalInner(inner)
		return "OR", filters, err
	}

	return "", nil, fmt.Errorf("not a logical filter")
}

func parseLogicalInner(inner string) ([]Filter, error) {
	// Split by comma at depth 0
	var filters []Filter
	var current strings.Builder
	depth := 0

	for _, r := range inner {
		switch r {
		case '(':
			depth++
			current.WriteRune(r)
		case ')':
			depth--
			current.WriteRune(r)
		case ',':
			if depth == 0 {
				filter, err := parseLogicalPart(current.String())
				if err != nil {
					return nil, err
				}
				filters = append(filters, *filter)
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		filter, err := parseLogicalPart(current.String())
		if err != nil {
			return nil, err
		}
		filters = append(filters, *filter)
	}

	return filters, nil
}

func parseLogicalPart(part string) (*Filter, error) {
	part = strings.TrimSpace(part)

	// Format: column.operator.value
	dotIdx := strings.Index(part, ".")
	if dotIdx < 0 {
		return nil, fmt.Errorf("invalid filter format: %s", part)
	}

	column := part[:dotIdx]
	value := part[dotIdx+1:]

	return ParseFilter(column, value)
}

// QuoteIdent quotes an identifier for PostgreSQL.
func QuoteIdent(s string) string {
	// Don't quote if it's already quoted or contains special characters that shouldn't be quoted
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// ParseInt parses an integer with a default value.
func ParseInt(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// BuildSelectColumns builds the SELECT column list from parsed select.
func BuildSelectColumns(cols []SelectColumn, tableAlias string) string {
	if len(cols) == 0 || (len(cols) == 1 && cols[0].Name == "*") {
		if tableAlias != "" {
			return tableAlias + ".*"
		}
		return "*"
	}

	var parts []string
	for _, col := range cols {
		if col.Embedded != nil {
			continue // Skip embedded for now, handled separately
		}

		var part string
		if tableAlias != "" {
			part = tableAlias + "." + QuoteIdent(col.Name)
		} else {
			part = QuoteIdent(col.Name)
		}

		if col.JSONPath != "" {
			part += col.JSONPath
		}

		if col.Cast != "" {
			part = "(" + part + ")::" + col.Cast
		}

		if col.Alias != "" {
			part += " AS " + QuoteIdent(col.Alias)
		}

		parts = append(parts, part)
	}

	return strings.Join(parts, ", ")
}
