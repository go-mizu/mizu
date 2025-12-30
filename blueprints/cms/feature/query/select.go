package query

import (
	"github.com/go-mizu/blueprints/cms/config"
)

// SelectorService implements field selection.
type SelectorService struct{}

// NewSelector creates a new selector service.
func NewSelector() *SelectorService {
	return &SelectorService{}
}

// ApplySelect filters document fields based on selection options.
func (s *SelectorService) ApplySelect(doc map[string]any, opts *SelectOptions) map[string]any {
	if doc == nil {
		return nil
	}

	if opts == nil || (len(opts.Include) == 0 && len(opts.Exclude) == 0) {
		return doc
	}

	result := make(map[string]any)

	// If include list is provided, only include those fields
	if len(opts.Include) > 0 {
		// Always include system fields
		systemFields := []string{"id", "createdAt", "updatedAt", "_status", "_version"}
		for _, f := range systemFields {
			if v, ok := doc[f]; ok {
				result[f] = v
			}
		}

		for field, include := range opts.Include {
			if include {
				if v, ok := doc[field]; ok {
					result[field] = v
				}
			}
		}
		return result
	}

	// If exclude list is provided, exclude those fields
	if len(opts.Exclude) > 0 {
		for k, v := range doc {
			if !opts.Exclude[k] {
				result[k] = v
			}
		}
		return result
	}

	return doc
}

// ApplySelectDocs filters fields for multiple documents.
func (s *SelectorService) ApplySelectDocs(docs []map[string]any, opts *SelectOptions) []map[string]any {
	if opts == nil || (len(opts.Include) == 0 && len(opts.Exclude) == 0) {
		return docs
	}

	result := make([]map[string]any, len(docs))
	for i, doc := range docs {
		result[i] = s.ApplySelect(doc, opts)
	}
	return result
}

// BuildSelectColumns builds SQL column list from selection options.
func (s *SelectorService) BuildSelectColumns(fields []config.Field, opts *SelectOptions) []string {
	if opts == nil || (len(opts.Include) == 0 && len(opts.Exclude) == 0) {
		return []string{"*"}
	}

	// Always include system columns
	columns := []string{"id", "created_at", "updated_at"}

	if len(opts.Include) > 0 {
		// Build column list from include set
		for _, field := range fields {
			if opts.Include[field.Name] {
				columns = append(columns, toSnakeCase(field.Name))
			}
		}
	} else if len(opts.Exclude) > 0 {
		// Build column list excluding certain fields
		for _, field := range fields {
			if !opts.Exclude[field.Name] {
				columns = append(columns, toSnakeCase(field.Name))
			}
		}
	}

	return columns
}

// ParseSelectParam parses select parameter from query.
// Format: "field1,field2" for include or "-field1,-field2" for exclude
func ParseSelectParam(param string) *SelectOptions {
	if param == "" {
		return nil
	}

	opts := &SelectOptions{
		Include: make(map[string]bool),
		Exclude: make(map[string]bool),
	}

	// Split by comma and process each field
	fields := splitFields(param)
	for _, field := range fields {
		if field == "" {
			continue
		}
		if field[0] == '-' {
			opts.Exclude[field[1:]] = true
		} else {
			opts.Include[field] = true
		}
	}

	// Clear include if any exclusions were found
	if len(opts.Exclude) > 0 {
		opts.Include = nil
	}

	return opts
}

func splitFields(s string) []string {
	var result []string
	var current string
	for _, c := range s {
		if c == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else if c != ' ' {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func toSnakeCase(s string) string {
	var result []byte
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(r+32))
		} else {
			result = append(result, byte(r))
		}
	}
	return string(result)
}
