package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
)

// CSVConverter converts databases to CSV format.
type CSVConverter struct{}

// NewCSVConverter creates a new CSV converter.
func NewCSVConverter() *CSVConverter {
	return &CSVConverter{}
}

// Convert converts an exported database to CSV.
func (c *CSVConverter) Convert(db *ExportedDatabase) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header row
	headers := make([]string, len(db.Properties))
	for i, prop := range db.Properties {
		headers[i] = prop.Name
	}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, row := range db.Rows {
		record := make([]string, len(db.Properties))
		for i, prop := range db.Properties {
			record[i] = c.formatPropertyValue(prop, row.Properties)
		}
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

// ContentType returns the MIME type.
func (c *CSVConverter) ContentType() string {
	return "text/csv; charset=utf-8"
}

// Extension returns the file extension.
func (c *CSVConverter) Extension() string {
	return ".csv"
}

// formatPropertyValue formats a property value for CSV output.
func (c *CSVConverter) formatPropertyValue(prop databases.Property, properties pages.Properties) string {
	if properties == nil {
		return ""
	}

	propValue, ok := properties[prop.ID]
	if !ok {
		// Try by name as fallback
		propValue, ok = properties[prop.Name]
		if !ok {
			return ""
		}
	}

	if propValue.Value == nil {
		return ""
	}

	switch prop.Type {
	case databases.PropTitle:
		return c.formatTextValue(propValue.Value)

	case databases.PropRichText:
		return c.formatTextValue(propValue.Value)

	case databases.PropNumber:
		return c.formatNumberValue(propValue.Value)

	case databases.PropSelect:
		return c.formatSelectValue(propValue.Value)

	case databases.PropMultiSelect:
		return c.formatMultiSelectValue(propValue.Value)

	case databases.PropDate:
		return c.formatDateValue(propValue.Value)

	case databases.PropCheckbox:
		return c.formatCheckboxValue(propValue.Value)

	case databases.PropURL:
		return c.formatStringValue(propValue.Value)

	case databases.PropEmail:
		return c.formatStringValue(propValue.Value)

	case databases.PropPhone:
		return c.formatStringValue(propValue.Value)

	case databases.PropPerson:
		return c.formatPersonValue(propValue.Value)

	case databases.PropFiles:
		return c.formatFilesValue(propValue.Value)

	case databases.PropCreatedTime, databases.PropLastEditTime:
		return c.formatTimestampValue(propValue.Value)

	case databases.PropCreatedBy, databases.PropLastEditBy:
		return c.formatPersonValue(propValue.Value)

	case databases.PropFormula:
		return c.formatFormulaValue(propValue.Value)

	case databases.PropRelation:
		return c.formatRelationValue(propValue.Value)

	case databases.PropRollup:
		return c.formatRollupValue(propValue.Value)

	case databases.PropStatus:
		return c.formatStatusValue(propValue.Value)

	default:
		return fmt.Sprintf("%v", propValue.Value)
	}
}

// formatTextValue formats a text/rich text value.
func (c *CSVConverter) formatTextValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []interface{}:
		// Rich text array
		var parts []string
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "")
	default:
		return fmt.Sprintf("%v", value)
	}
}

// formatNumberValue formats a number value.
func (c *CSVConverter) formatNumberValue(value interface{}) string {
	switch v := value.(type) {
	case float64:
		// Check if it's a whole number
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// formatSelectValue formats a select value.
func (c *CSVConverter) formatSelectValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]interface{}:
		if name, ok := v["name"].(string); ok {
			return name
		}
	}
	return fmt.Sprintf("%v", value)
}

// formatMultiSelectValue formats a multi-select value.
func (c *CSVConverter) formatMultiSelectValue(value interface{}) string {
	switch v := value.(type) {
	case []interface{}:
		var names []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				names = append(names, s)
			} else if m, ok := item.(map[string]interface{}); ok {
				if name, ok := m["name"].(string); ok {
					names = append(names, name)
				}
			}
		}
		return strings.Join(names, ", ")
	case []string:
		return strings.Join(v, ", ")
	}
	return fmt.Sprintf("%v", value)
}

// formatDateValue formats a date value.
func (c *CSVConverter) formatDateValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case time.Time:
		return v.Format("2006-01-02")
	case map[string]interface{}:
		var parts []string
		if start, ok := v["start"].(string); ok {
			parts = append(parts, start)
		}
		if end, ok := v["end"].(string); ok && end != "" {
			parts = append(parts, end)
		}
		return strings.Join(parts, " → ")
	}
	return fmt.Sprintf("%v", value)
}

// formatCheckboxValue formats a checkbox value.
func (c *CSVConverter) formatCheckboxValue(value interface{}) string {
	switch v := value.(type) {
	case bool:
		if v {
			return "true"
		}
		return "false"
	}
	return fmt.Sprintf("%v", value)
}

// formatStringValue formats a simple string value.
func (c *CSVConverter) formatStringValue(value interface{}) string {
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

// formatPersonValue formats a person/user value.
func (c *CSVConverter) formatPersonValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []interface{}:
		var names []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				names = append(names, s)
			} else if m, ok := item.(map[string]interface{}); ok {
				if name, ok := m["name"].(string); ok {
					names = append(names, name)
				} else if email, ok := m["email"].(string); ok {
					names = append(names, email)
				}
			}
		}
		return strings.Join(names, ", ")
	case map[string]interface{}:
		if name, ok := v["name"].(string); ok {
			return name
		}
		if email, ok := v["email"].(string); ok {
			return email
		}
	}
	return fmt.Sprintf("%v", value)
}

// formatFilesValue formats a files value.
func (c *CSVConverter) formatFilesValue(value interface{}) string {
	switch v := value.(type) {
	case []interface{}:
		var urls []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				urls = append(urls, s)
			} else if m, ok := item.(map[string]interface{}); ok {
				if url, ok := m["url"].(string); ok {
					urls = append(urls, url)
				} else if name, ok := m["name"].(string); ok {
					urls = append(urls, name)
				}
			}
		}
		return strings.Join(urls, ", ")
	case []string:
		return strings.Join(v, ", ")
	}
	return fmt.Sprintf("%v", value)
}

// formatTimestampValue formats a timestamp value.
func (c *CSVConverter) formatTimestampValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case time.Time:
		return v.Format(time.RFC3339)
	case float64:
		// Unix timestamp in milliseconds
		t := time.UnixMilli(int64(v))
		return t.Format(time.RFC3339)
	}
	return fmt.Sprintf("%v", value)
}

// formatFormulaValue formats a formula result value.
func (c *CSVConverter) formatFormulaValue(value interface{}) string {
	switch v := value.(type) {
	case map[string]interface{}:
		if result, ok := v["result"]; ok {
			return fmt.Sprintf("%v", result)
		}
	}
	return fmt.Sprintf("%v", value)
}

// formatRelationValue formats a relation value.
func (c *CSVConverter) formatRelationValue(value interface{}) string {
	switch v := value.(type) {
	case []interface{}:
		var titles []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				titles = append(titles, s)
			} else if m, ok := item.(map[string]interface{}); ok {
				if title, ok := m["title"].(string); ok {
					titles = append(titles, title)
				} else if id, ok := m["id"].(string); ok {
					titles = append(titles, id)
				}
			}
		}
		return strings.Join(titles, ", ")
	}
	return fmt.Sprintf("%v", value)
}

// formatRollupValue formats a rollup value.
func (c *CSVConverter) formatRollupValue(value interface{}) string {
	switch v := value.(type) {
	case map[string]interface{}:
		if result, ok := v["result"]; ok {
			return fmt.Sprintf("%v", result)
		}
	}
	return fmt.Sprintf("%v", value)
}

// formatStatusValue formats a status value.
func (c *CSVConverter) formatStatusValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]interface{}:
		if name, ok := v["name"].(string); ok {
			return name
		}
	}
	return fmt.Sprintf("%v", value)
}
