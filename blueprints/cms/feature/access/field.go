package access

import (
	"github.com/go-mizu/blueprints/cms/config"
)

// FieldService implements field-level access control.
type FieldService struct{}

// NewFieldService creates a new field access service.
func NewFieldService() *FieldService {
	return &FieldService{}
}

// CanCreateField checks if the user can create this field.
func (s *FieldService) CanCreateField(ctx *AccessContext, field *config.Field) (bool, error) {
	if field.Access == nil || field.Access.Create == nil {
		// Default: allow if user is authenticated
		return ctx.User != nil, nil
	}

	configCtx := toConfigAccessContext(ctx)
	result, err := field.Access.Create(configCtx)
	if err != nil {
		return false, err
	}

	return result != nil && result.Allowed, nil
}

// CanReadField checks if the user can read this field.
func (s *FieldService) CanReadField(ctx *AccessContext, field *config.Field) (bool, error) {
	if field.Access == nil || field.Access.Read == nil {
		// Default: allow read
		return true, nil
	}

	configCtx := toConfigAccessContext(ctx)
	result, err := field.Access.Read(configCtx)
	if err != nil {
		return false, err
	}

	return result != nil && result.Allowed, nil
}

// CanUpdateField checks if the user can update this field.
func (s *FieldService) CanUpdateField(ctx *AccessContext, field *config.Field) (bool, error) {
	if field.Access == nil || field.Access.Update == nil {
		// Default: allow if user is authenticated
		return ctx.User != nil, nil
	}

	configCtx := toConfigAccessContext(ctx)
	result, err := field.Access.Update(configCtx)
	if err != nil {
		return false, err
	}

	return result != nil && result.Allowed, nil
}

// FilterReadableFields removes fields the user cannot read.
func (s *FieldService) FilterReadableFields(doc map[string]any, fields []config.Field, ctx *AccessContext) (map[string]any, error) {
	if doc == nil {
		return nil, nil
	}

	result := make(map[string]any)

	// Copy all non-field data (id, timestamps, etc.)
	fieldNames := make(map[string]bool)
	for _, f := range fields {
		fieldNames[f.Name] = true
	}
	for k, v := range doc {
		if !fieldNames[k] {
			result[k] = v
		}
	}

	// Filter fields based on access
	for i := range fields {
		field := &fields[i]

		value, exists := doc[field.Name]
		if !exists {
			continue
		}

		canRead, err := s.CanReadField(ctx, field)
		if err != nil {
			return nil, err
		}

		if !canRead {
			continue
		}

		// Recursively filter nested fields
		if field.HasNestedFields() {
			filtered, err := s.filterNestedReadable(value, field, ctx)
			if err != nil {
				return nil, err
			}
			result[field.Name] = filtered
		} else {
			result[field.Name] = value
		}
	}

	return result, nil
}

func (s *FieldService) filterNestedReadable(value any, field *config.Field, ctx *AccessContext) (any, error) {
	switch field.Type {
	case config.FieldTypeArray:
		arr, ok := value.([]any)
		if !ok {
			return value, nil
		}
		result := make([]any, 0, len(arr))
		for _, item := range arr {
			if itemMap, ok := item.(map[string]any); ok {
				filtered, err := s.FilterReadableFields(itemMap, field.Fields, ctx)
				if err != nil {
					return nil, err
				}
				result = append(result, filtered)
			} else {
				result = append(result, item)
			}
		}
		return result, nil

	case config.FieldTypeGroup:
		if data, ok := value.(map[string]any); ok {
			return s.FilterReadableFields(data, field.Fields, ctx)
		}

	case config.FieldTypeBlocks:
		arr, ok := value.([]any)
		if !ok {
			return value, nil
		}
		result := make([]any, 0, len(arr))
		for _, item := range arr {
			itemMap, ok := item.(map[string]any)
			if !ok {
				result = append(result, item)
				continue
			}
			blockType, _ := itemMap["blockType"].(string)
			var blockFields []config.Field
			for _, block := range field.Blocks {
				if block.Slug == blockType {
					blockFields = block.Fields
					break
				}
			}
			if len(blockFields) > 0 {
				filtered, err := s.FilterReadableFields(itemMap, blockFields, ctx)
				if err != nil {
					return nil, err
				}
				result = append(result, filtered)
			} else {
				result = append(result, item)
			}
		}
		return result, nil

	case config.FieldTypeTabs:
		data, ok := value.(map[string]any)
		if !ok {
			return value, nil
		}
		result := make(map[string]any)
		for _, tab := range field.Tabs {
			if tab.Name == "" {
				// Unnamed tab
				filtered, err := s.FilterReadableFields(data, tab.Fields, ctx)
				if err != nil {
					return nil, err
				}
				for k, v := range filtered {
					result[k] = v
				}
			} else if tabData, ok := data[tab.Name].(map[string]any); ok {
				filtered, err := s.FilterReadableFields(tabData, tab.Fields, ctx)
				if err != nil {
					return nil, err
				}
				result[tab.Name] = filtered
			}
		}
		return result, nil
	}

	return value, nil
}

// FilterWritableFields removes fields the user cannot create/update.
func (s *FieldService) FilterWritableFields(data map[string]any, fields []config.Field, ctx *AccessContext, isCreate bool) (map[string]any, error) {
	if data == nil {
		return nil, nil
	}

	result := make(map[string]any)

	for i := range fields {
		field := &fields[i]

		value, exists := data[field.Name]
		if !exists {
			continue
		}

		var canWrite bool
		var err error

		if isCreate {
			canWrite, err = s.CanCreateField(ctx, field)
		} else {
			canWrite, err = s.CanUpdateField(ctx, field)
		}

		if err != nil {
			return nil, err
		}

		if !canWrite {
			continue
		}

		// Recursively filter nested fields
		if field.HasNestedFields() {
			filtered, err := s.filterNestedWritable(value, field, ctx, isCreate)
			if err != nil {
				return nil, err
			}
			result[field.Name] = filtered
		} else {
			result[field.Name] = value
		}
	}

	return result, nil
}

func (s *FieldService) filterNestedWritable(value any, field *config.Field, ctx *AccessContext, isCreate bool) (any, error) {
	switch field.Type {
	case config.FieldTypeArray:
		arr, ok := value.([]any)
		if !ok {
			return value, nil
		}
		result := make([]any, 0, len(arr))
		for _, item := range arr {
			if itemMap, ok := item.(map[string]any); ok {
				filtered, err := s.FilterWritableFields(itemMap, field.Fields, ctx, isCreate)
				if err != nil {
					return nil, err
				}
				result = append(result, filtered)
			} else {
				result = append(result, item)
			}
		}
		return result, nil

	case config.FieldTypeGroup:
		if data, ok := value.(map[string]any); ok {
			return s.FilterWritableFields(data, field.Fields, ctx, isCreate)
		}

	case config.FieldTypeBlocks:
		arr, ok := value.([]any)
		if !ok {
			return value, nil
		}
		result := make([]any, 0, len(arr))
		for _, item := range arr {
			itemMap, ok := item.(map[string]any)
			if !ok {
				result = append(result, item)
				continue
			}
			blockType, _ := itemMap["blockType"].(string)
			var blockFields []config.Field
			for _, block := range field.Blocks {
				if block.Slug == blockType {
					blockFields = block.Fields
					break
				}
			}
			if len(blockFields) > 0 {
				filtered, err := s.FilterWritableFields(itemMap, blockFields, ctx, isCreate)
				if err != nil {
					return nil, err
				}
				// Preserve blockType
				filtered["blockType"] = blockType
				result = append(result, filtered)
			} else {
				result = append(result, item)
			}
		}
		return result, nil

	case config.FieldTypeTabs:
		data, ok := value.(map[string]any)
		if !ok {
			return value, nil
		}
		result := make(map[string]any)
		for _, tab := range field.Tabs {
			if tab.Name == "" {
				filtered, err := s.FilterWritableFields(data, tab.Fields, ctx, isCreate)
				if err != nil {
					return nil, err
				}
				for k, v := range filtered {
					result[k] = v
				}
			} else if tabData, ok := data[tab.Name].(map[string]any); ok {
				filtered, err := s.FilterWritableFields(tabData, tab.Fields, ctx, isCreate)
				if err != nil {
					return nil, err
				}
				result[tab.Name] = filtered
			}
		}
		return result, nil
	}

	return value, nil
}
