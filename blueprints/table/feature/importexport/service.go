package importexport

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/table/feature/bases"
	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/feature/tables"
	"github.com/go-mizu/blueprints/table/feature/views"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

// Service implements the import/export API.
type Service struct {
	bases   bases.API
	tables  tables.API
	fields  fields.API
	records records.API
	views   views.API
}

// NewService creates a new import/export service.
func NewService(
	bases bases.API,
	tables tables.API,
	fields fields.API,
	records records.API,
	views views.API,
) *Service {
	return &Service{
		bases:   bases,
		tables:  tables,
		fields:  fields,
		records: records,
		views:   views,
	}
}

// Export exports a base to a directory.
func (s *Service) Export(ctx context.Context, baseID, dir string) error {
	// Export metadata
	meta, err := s.ExportMeta(ctx, baseID)
	if err != nil {
		return err
	}

	// Create directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Write base.json
	metaJSON, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "base.json"), metaJSON, 0644); err != nil {
		return fmt.Errorf("write base.json: %w", err)
	}

	// Create data directory
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	// Export each table's data
	for _, tm := range meta.Tables {
		if err := s.exportTableData(ctx, &tm, dataDir); err != nil {
			return fmt.Errorf("export table %s: %w", tm.Table.Name, err)
		}
	}

	return nil
}

// Import imports a base from a directory.
func (s *Service) Import(ctx context.Context, workspaceID, userID, dir string) (*bases.Base, error) {
	// Check directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, ErrDirNotExist
	}

	// Read base.json
	metaJSON, err := os.ReadFile(filepath.Join(dir, "base.json"))
	if err != nil {
		return nil, fmt.Errorf("read base.json: %w", err)
	}

	var meta Meta
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		return nil, fmt.Errorf("parse base.json: %w", err)
	}

	// Import metadata
	base, idMaps, err := s.importMetaWithMaps(ctx, workspaceID, userID, &meta)
	if err != nil {
		return nil, err
	}

	// Import data for each table
	dataDir := filepath.Join(dir, "data")
	for _, tm := range meta.Tables {
		newTableID := idMaps.tableIDs[tm.Table.ID]
		if err := s.importTableData(ctx, &tm, newTableID, idMaps, userID, dataDir); err != nil {
			return nil, fmt.Errorf("import table %s: %w", tm.Table.Name, err)
		}
	}

	return base, nil
}

// ExportMeta exports only the metadata (no data).
func (s *Service) ExportMeta(ctx context.Context, baseID string) (*Meta, error) {
	// Get base
	base, err := s.bases.GetByID(ctx, baseID)
	if err != nil {
		return nil, ErrBaseNotFound
	}

	// Get tables
	tbls, err := s.tables.ListByBase(ctx, baseID)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}

	meta := &Meta{
		Version:    Version,
		ExportedAt: time.Now().UTC(),
		Base:       *base,
		Tables:     make([]TableMeta, 0, len(tbls)),
	}

	// Get metadata for each table
	for _, tbl := range tbls {
		tm := TableMeta{
			Table:   *tbl,
			Choices: make(map[string][]*fields.SelectChoice),
		}

		// Get fields
		flds, err := s.fields.ListByTable(ctx, tbl.ID)
		if err != nil {
			return nil, fmt.Errorf("list fields for table %s: %w", tbl.Name, err)
		}
		tm.Fields = flds

		// Get select choices for select fields
		for _, fld := range flds {
			if fld.Type == fields.TypeSingleSelect || fld.Type == fields.TypeMultiSelect {
				choices, err := s.fields.ListSelectChoices(ctx, fld.ID)
				if err != nil {
					return nil, fmt.Errorf("list choices for field %s: %w", fld.Name, err)
				}
				tm.Choices[fld.ID] = choices
			}
		}

		// Get views
		vws, err := s.views.ListByTable(ctx, tbl.ID)
		if err != nil {
			return nil, fmt.Errorf("list views for table %s: %w", tbl.Name, err)
		}
		tm.Views = vws

		meta.Tables = append(meta.Tables, tm)
	}

	return meta, nil
}

// ImportMeta imports only the metadata (no data).
func (s *Service) ImportMeta(ctx context.Context, workspaceID, userID string, meta *Meta) (*bases.Base, error) {
	base, _, err := s.importMetaWithMaps(ctx, workspaceID, userID, meta)
	return base, err
}

// idMaps holds mappings from old IDs to new IDs during import.
type idMaps struct {
	tableIDs  map[string]string
	fieldIDs  map[string]string
	choiceIDs map[string]string
	viewIDs   map[string]string
}

// importMetaWithMaps imports metadata and returns the ID mappings.
func (s *Service) importMetaWithMaps(ctx context.Context, workspaceID, userID string, meta *Meta) (*bases.Base, *idMaps, error) {
	maps := &idMaps{
		tableIDs:  make(map[string]string),
		fieldIDs:  make(map[string]string),
		choiceIDs: make(map[string]string),
		viewIDs:   make(map[string]string),
	}

	// Create base
	base, err := s.bases.Create(ctx, userID, bases.CreateIn{
		WorkspaceID: workspaceID,
		Name:        meta.Base.Name,
		Description: meta.Base.Description,
		Icon:        meta.Base.Icon,
		Color:       meta.Base.Color,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create base: %w", err)
	}

	// Create tables
	for _, tm := range meta.Tables {
		tbl, err := s.tables.Create(ctx, userID, tables.CreateIn{
			BaseID:      base.ID,
			Name:        tm.Table.Name,
			Description: tm.Table.Description,
			Icon:        tm.Table.Icon,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("create table %s: %w", tm.Table.Name, err)
		}
		maps.tableIDs[tm.Table.ID] = tbl.ID

		// Create fields
		var primaryFieldNewID string
		for _, fld := range tm.Fields {
			// Prepare options with remapped choice IDs
			var options json.RawMessage
			if len(fld.Options) > 0 && (fld.Type == fields.TypeSingleSelect || fld.Type == fields.TypeMultiSelect) {
				options = s.remapChoiceIDsInOptions(fld.Options, fld.ID, tm.Choices, maps)
			} else {
				options = fld.Options
			}

			newFld, err := s.fields.Create(ctx, userID, fields.CreateIn{
				TableID:     tbl.ID,
				Name:        fld.Name,
				Type:        fld.Type,
				Description: fld.Description,
				Options:     options,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("create field %s: %w", fld.Name, err)
			}
			maps.fieldIDs[fld.ID] = newFld.ID

			if fld.IsPrimary {
				primaryFieldNewID = newFld.ID
			}

			// Map choice IDs from options
			if fld.Type == fields.TypeSingleSelect || fld.Type == fields.TypeMultiSelect {
				choices := tm.Choices[fld.ID]
				newChoices, _ := s.fields.ListSelectChoices(ctx, newFld.ID)
				// Map by position (since we created them in order via options)
				for i, oldChoice := range choices {
					if i < len(newChoices) {
						maps.choiceIDs[oldChoice.ID] = newChoices[i].ID
					}
				}
			}
		}

		// Set primary field if it was tracked
		if primaryFieldNewID != "" {
			s.tables.SetPrimaryField(ctx, tbl.ID, primaryFieldNewID)
		}

		// Create views with remapped field IDs
		for _, vw := range tm.Views {
			newView, err := s.views.Create(ctx, userID, views.CreateIn{
				TableID:   tbl.ID,
				Name:      vw.Name,
				Type:      vw.Type,
				IsDefault: vw.IsDefault,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("create view %s: %w", vw.Name, err)
			}
			maps.viewIDs[vw.ID] = newView.ID

			// Set view configuration with remapped field IDs
			if len(vw.Filters) > 0 {
				remappedFilters := s.remapFilters(vw.Filters, maps)
				s.views.SetFilters(ctx, newView.ID, remappedFilters)
			}
			if len(vw.Sorts) > 0 {
				remappedSorts := s.remapSorts(vw.Sorts, maps)
				s.views.SetSorts(ctx, newView.ID, remappedSorts)
			}
			if len(vw.Groups) > 0 {
				remappedGroups := s.remapGroups(vw.Groups, maps)
				s.views.SetGroups(ctx, newView.ID, remappedGroups)
			}
			if len(vw.FieldConfig) > 0 {
				remappedFieldConfig := s.remapFieldConfig(vw.FieldConfig, maps)
				s.views.SetFieldConfig(ctx, newView.ID, remappedFieldConfig)
			}
			if len(vw.Config) > 0 {
				remappedConfig := s.remapViewConfig(vw.Config, maps)
				if remappedConfig != nil {
					s.views.SetConfig(ctx, newView.ID, remappedConfig)
				}
			}
		}
	}

	return base, maps, nil
}

// remapChoiceIDsInOptions remaps choice IDs in field options and updates the maps.
func (s *Service) remapChoiceIDsInOptions(options json.RawMessage, fieldID string, allChoices map[string][]*fields.SelectChoice, maps *idMaps) json.RawMessage {
	var opts map[string]interface{}
	if err := json.Unmarshal(options, &opts); err != nil {
		return options
	}

	choices, ok := opts["choices"].([]interface{})
	if !ok {
		return options
	}

	// Get original choices to maintain order
	originalChoices := allChoices[fieldID]
	newChoices := make([]interface{}, 0, len(choices))

	for i, c := range choices {
		choice, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		oldID, _ := choice["id"].(string)
		newID := ulid.New()

		// Track the mapping
		if oldID != "" {
			maps.choiceIDs[oldID] = newID
		} else if i < len(originalChoices) {
			maps.choiceIDs[originalChoices[i].ID] = newID
		}

		newChoice := map[string]interface{}{
			"id":    newID,
			"name":  choice["name"],
			"color": choice["color"],
		}
		newChoices = append(newChoices, newChoice)
	}

	opts["choices"] = newChoices
	result, _ := json.Marshal(opts)
	return result
}

// remapFilters remaps field IDs in filters.
func (s *Service) remapFilters(filters []views.Filter, maps *idMaps) []views.Filter {
	result := make([]views.Filter, len(filters))
	for i, f := range filters {
		result[i] = views.Filter{
			FieldID:  maps.fieldIDs[f.FieldID],
			Operator: f.Operator,
			Value:    s.remapValueChoiceIDs(f.Value, maps),
		}
		if result[i].FieldID == "" {
			result[i].FieldID = f.FieldID // Keep original if not mapped
		}
	}
	return result
}

// remapSorts remaps field IDs in sorts.
func (s *Service) remapSorts(sorts []views.SortSpec, maps *idMaps) []views.SortSpec {
	result := make([]views.SortSpec, len(sorts))
	for i, s := range sorts {
		result[i] = views.SortSpec{
			FieldID:   maps.fieldIDs[s.FieldID],
			Direction: s.Direction,
		}
		if result[i].FieldID == "" {
			result[i].FieldID = s.FieldID
		}
	}
	return result
}

// remapGroups remaps field IDs in groups.
func (s *Service) remapGroups(groups []views.GroupSpec, maps *idMaps) []views.GroupSpec {
	result := make([]views.GroupSpec, len(groups))
	for i, g := range groups {
		result[i] = views.GroupSpec{
			FieldID:   maps.fieldIDs[g.FieldID],
			Direction: g.Direction,
			Collapsed: g.Collapsed,
		}
		if result[i].FieldID == "" {
			result[i].FieldID = g.FieldID
		}
	}
	return result
}

// remapFieldConfig remaps field IDs in field config.
func (s *Service) remapFieldConfig(config []views.FieldViewConfig, maps *idMaps) []views.FieldViewConfig {
	result := make([]views.FieldViewConfig, len(config))
	for i, c := range config {
		result[i] = views.FieldViewConfig{
			FieldID:  maps.fieldIDs[c.FieldID],
			Visible:  c.Visible,
			Width:    c.Width,
			Position: c.Position,
		}
		if result[i].FieldID == "" {
			result[i].FieldID = c.FieldID
		}
	}
	return result
}

// remapViewConfig remaps field IDs in view config JSON.
func (s *Service) remapViewConfig(config json.RawMessage, maps *idMaps) map[string]interface{} {
	var cfg map[string]interface{}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil
	}

	// Common field references in config
	fieldKeys := []string{"groupBy", "dateField", "endDateField", "cover_field_id", "title_field_id"}
	for _, key := range fieldKeys {
		if val, ok := cfg[key].(string); ok && val != "" {
			if newID, exists := maps.fieldIDs[val]; exists {
				cfg[key] = newID
			}
		}
	}

	return cfg
}

// remapValueChoiceIDs remaps choice IDs in filter values.
func (s *Service) remapValueChoiceIDs(value interface{}, maps *idMaps) interface{} {
	switch v := value.(type) {
	case string:
		if newID, ok := maps.choiceIDs[v]; ok {
			return newID
		}
		return v
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = s.remapValueChoiceIDs(item, maps)
		}
		return result
	default:
		return v
	}
}

// exportTableData exports table records to CSV.
func (s *Service) exportTableData(ctx context.Context, tm *TableMeta, dataDir string) error {
	// Get all records
	recordList, err := s.records.List(ctx, tm.Table.ID, records.ListOpts{Limit: 100000})
	if err != nil {
		return fmt.Errorf("list records: %w", err)
	}

	if len(recordList.Records) == 0 {
		return nil // No data to export
	}

	// Create CSV file
	filename := sanitizeFilename(tm.Table.Name) + ".csv"
	f, err := os.Create(filepath.Join(dataDir, filename))
	if err != nil {
		return fmt.Errorf("create CSV file: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Build field map for O(1) lookups
	fieldMap := buildFieldMap(tm.Fields)

	// Build header row - _id + field names
	headers := []string{"_id"}
	fieldIndex := make(map[string]int) // fieldID -> column index
	for i, fld := range tm.Fields {
		// Skip computed fields
		if fld.IsComputed {
			continue
		}
		headers = append(headers, fld.Name)
		fieldIndex[fld.ID] = i + 1 // +1 for _id column
	}

	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Build choice name maps for select fields
	choiceNames := make(map[string]map[string]string) // fieldID -> choiceID -> name
	for fieldID, choices := range tm.Choices {
		choiceNames[fieldID] = make(map[string]string)
		for _, c := range choices {
			choiceNames[fieldID][c.ID] = c.Name
		}
	}

	// Write records
	for _, rec := range recordList.Records {
		row := make([]string, len(headers))
		row[0] = rec.ID

		for fieldID, colIdx := range fieldIndex {
			if colIdx >= len(row) {
				continue
			}
			value := rec.Cells[fieldID]
			row[colIdx] = s.formatCellValue(value, fieldMap[fieldID], choiceNames[fieldID])
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}

	return nil
}

// importTableData imports table records from CSV.
func (s *Service) importTableData(ctx context.Context, tm *TableMeta, newTableID string, maps *idMaps, userID, dataDir string) error {
	filename := sanitizeFilename(tm.Table.Name) + ".csv"
	filepath := filepath.Join(dataDir, filename)

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil // No data file, skip
	}

	f, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("open CSV file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1 // Allow variable fields

	// Read header
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read header: %w", err)
	}

	// Build field name to new ID mapping
	fieldByName := make(map[string]*fields.Field)
	for _, fld := range tm.Fields {
		fieldByName[fld.Name] = fld
	}

	// Build choice name to new ID mapping for select fields
	choiceIDByName := make(map[string]map[string]string) // fieldID -> name -> newChoiceID
	for oldFieldID, choices := range tm.Choices {
		newFieldID := maps.fieldIDs[oldFieldID]
		choiceIDByName[newFieldID] = make(map[string]string)
		for _, c := range choices {
			newChoiceID := maps.choiceIDs[c.ID]
			choiceIDByName[newFieldID][c.Name] = newChoiceID
		}
	}

	// Read and import records
	var recordsData []map[string]interface{}
	for {
		row, err := reader.Read()
		if err != nil {
			break // EOF or error
		}

		cells := make(map[string]interface{})
		for i, header := range headers {
			if i >= len(row) {
				continue
			}
			if header == "_id" {
				continue // Skip ID column
			}

			fld, ok := fieldByName[header]
			if !ok {
				continue // Unknown field
			}

			newFieldID := maps.fieldIDs[fld.ID]
			if newFieldID == "" {
				continue
			}

			value := s.parseCellValue(row[i], fld, choiceIDByName[newFieldID])
			if value != nil {
				cells[newFieldID] = value
			}
		}

		if len(cells) > 0 {
			recordsData = append(recordsData, cells)
		}
	}

	// Batch create records
	if len(recordsData) > 0 {
		_, err := s.records.CreateBatch(ctx, newTableID, recordsData, userID)
		if err != nil {
			return fmt.Errorf("create records: %w", err)
		}
	}

	return nil
}

// formatCellValue converts a cell value to CSV string format.
func (s *Service) formatCellValue(value interface{}, fld *fields.Field, choiceNames map[string]string) string {
	if value == nil {
		return ""
	}

	if fld == nil {
		return fmt.Sprintf("%v", value)
	}

	switch fld.Type {
	case fields.TypeSingleSelect:
		// Convert choice ID to name
		if id, ok := value.(string); ok {
			if name, exists := choiceNames[id]; exists {
				return name
			}
			return id
		}
	case fields.TypeMultiSelect:
		// Convert choice IDs to names
		if ids, ok := value.([]interface{}); ok {
			names := make([]string, 0, len(ids))
			for _, id := range ids {
				if idStr, ok := id.(string); ok {
					if name, exists := choiceNames[idStr]; exists {
						names = append(names, name)
					} else {
						names = append(names, idStr)
					}
				}
			}
			return strings.Join(names, ",")
		}
	case fields.TypeCheckbox:
		if b, ok := value.(bool); ok {
			return strconv.FormatBool(b)
		}
	case fields.TypeNumber, fields.TypeCurrency, fields.TypePercent, fields.TypeRating, fields.TypeDuration:
		switch v := value.(type) {
		case float64:
			if v == float64(int(v)) {
				return strconv.Itoa(int(v))
			}
			return strconv.FormatFloat(v, 'f', -1, 64)
		case int:
			return strconv.Itoa(v)
		case int64:
			return strconv.FormatInt(v, 10)
		}
	case fields.TypeAttachment:
		// Serialize attachments as JSON
		if data, err := json.Marshal(value); err == nil {
			return string(data)
		}
	case fields.TypeCollaborators:
		// Serialize as comma-separated
		if ids, ok := value.([]interface{}); ok {
			strs := make([]string, len(ids))
			for i, id := range ids {
				strs[i] = fmt.Sprintf("%v", id)
			}
			return strings.Join(strs, ",")
		}
	}

	return fmt.Sprintf("%v", value)
}

// parseCellValue converts a CSV string to the appropriate type.
func (s *Service) parseCellValue(str string, fld *fields.Field, choiceIDByName map[string]string) interface{} {
	if str == "" {
		return nil
	}

	switch fld.Type {
	case fields.TypeSingleSelect:
		// Convert name to choice ID
		if id, ok := choiceIDByName[str]; ok {
			return id
		}
		return str
	case fields.TypeMultiSelect:
		// Convert names to choice IDs
		names := strings.Split(str, ",")
		ids := make([]string, 0, len(names))
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if id, ok := choiceIDByName[name]; ok {
				ids = append(ids, id)
			} else {
				ids = append(ids, name)
			}
		}
		return ids
	case fields.TypeCheckbox:
		return str == "true" || str == "1" || str == "yes"
	case fields.TypeNumber, fields.TypeCurrency, fields.TypePercent, fields.TypeRating:
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			return f
		}
		return nil
	case fields.TypeDuration:
		if i, err := strconv.ParseInt(str, 10, 64); err == nil {
			return i
		}
		return nil
	case fields.TypeAttachment:
		// Parse JSON attachments
		var attachments []map[string]interface{}
		if err := json.Unmarshal([]byte(str), &attachments); err == nil {
			return attachments
		}
		return nil
	case fields.TypeCollaborators:
		// Parse comma-separated IDs
		parts := strings.Split(str, ",")
		ids := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				ids = append(ids, p)
			}
		}
		return ids
	default:
		return str
	}
}

// sanitizeFilename converts a name to a safe filename.
func sanitizeFilename(name string) string {
	// Convert to lowercase and replace spaces with underscores
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	// Remove any characters that aren't alphanumeric or underscore
	reg := regexp.MustCompile(`[^a-z0-9_]`)
	name = reg.ReplaceAllString(name, "")
	return name
}

// buildFieldMap creates a map for O(1) field lookups by ID.
func buildFieldMap(fieldList []*fields.Field) map[string]*fields.Field {
	m := make(map[string]*fields.Field, len(fieldList))
	for _, f := range fieldList {
		m[f.ID] = f
	}
	return m
}

