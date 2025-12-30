package hooks

import (
	"fmt"

	"github.com/go-mizu/blueprints/cms/config"
)

// FieldService implements field-level hook execution.
type FieldService struct{}

// NewFieldService creates a new field hooks service.
func NewFieldService() *FieldService {
	return &FieldService{}
}

// toConfigFieldHookContext converts to config.FieldHookContext.
func toConfigFieldHookContext(ctx *FieldHookContext) *config.FieldHookContext {
	return &config.FieldHookContext{
		Value:       ctx.Value,
		OriginalDoc: ctx.OriginalDoc,
		Data:        ctx.Data,
		SiblingData: ctx.SiblingData,
		Field:       ctx.Field,
		Collection:  ctx.Collection,
		Operation:   string(ctx.Operation),
		Req:         ctx.Req,
		User:        ctx.User,
	}
}

// ExecuteFieldBeforeValidate executes field beforeValidate hooks.
func (s *FieldService) ExecuteFieldBeforeValidate(ctx *FieldHookContext, hooks *config.FieldHooks) (any, error) {
	if hooks == nil || len(hooks.BeforeValidate) == 0 {
		return ctx.Value, nil
	}

	configCtx := toConfigFieldHookContext(ctx)
	value := ctx.Value

	for _, hook := range hooks.BeforeValidate {
		result, err := hook(configCtx)
		if err != nil {
			return nil, err
		}
		if result != nil {
			value = result
			configCtx.Value = result
		}
	}

	return value, nil
}

// ExecuteFieldBeforeChange executes field beforeChange hooks.
func (s *FieldService) ExecuteFieldBeforeChange(ctx *FieldHookContext, hooks *config.FieldHooks) (any, error) {
	if hooks == nil || len(hooks.BeforeChange) == 0 {
		return ctx.Value, nil
	}

	configCtx := toConfigFieldHookContext(ctx)
	value := ctx.Value

	for _, hook := range hooks.BeforeChange {
		result, err := hook(configCtx)
		if err != nil {
			return nil, err
		}
		if result != nil {
			value = result
			configCtx.Value = result
		}
	}

	return value, nil
}

// ExecuteFieldAfterChange executes field afterChange hooks.
func (s *FieldService) ExecuteFieldAfterChange(ctx *FieldHookContext, hooks *config.FieldHooks) (any, error) {
	if hooks == nil || len(hooks.AfterChange) == 0 {
		return ctx.Value, nil
	}

	configCtx := toConfigFieldHookContext(ctx)
	value := ctx.Value

	for _, hook := range hooks.AfterChange {
		result, err := hook(configCtx)
		if err != nil {
			return nil, err
		}
		if result != nil {
			value = result
			configCtx.Value = result
		}
	}

	return value, nil
}

// ExecuteFieldAfterRead executes field afterRead hooks.
func (s *FieldService) ExecuteFieldAfterRead(ctx *FieldHookContext, hooks *config.FieldHooks) (any, error) {
	if hooks == nil || len(hooks.AfterRead) == 0 {
		return ctx.Value, nil
	}

	configCtx := toConfigFieldHookContext(ctx)
	value := ctx.Value

	for _, hook := range hooks.AfterRead {
		result, err := hook(configCtx)
		if err != nil {
			return nil, err
		}
		if result != nil {
			value = result
			configCtx.Value = result
		}
	}

	return value, nil
}

// ProcessFieldsBeforeValidate processes all fields with beforeValidate hooks.
func (s *FieldService) ProcessFieldsBeforeValidate(ctx *HookContext, fields []config.Field, data map[string]any) (map[string]any, error) {
	return s.processFields(ctx, fields, data, "", func(fctx *FieldHookContext, hooks *config.FieldHooks) (any, error) {
		return s.ExecuteFieldBeforeValidate(fctx, hooks)
	})
}

// ProcessFieldsBeforeChange processes all fields with beforeChange hooks.
func (s *FieldService) ProcessFieldsBeforeChange(ctx *HookContext, fields []config.Field, data map[string]any) (map[string]any, error) {
	return s.processFields(ctx, fields, data, "", func(fctx *FieldHookContext, hooks *config.FieldHooks) (any, error) {
		return s.ExecuteFieldBeforeChange(fctx, hooks)
	})
}

// ProcessFieldsAfterChange processes all fields with afterChange hooks.
func (s *FieldService) ProcessFieldsAfterChange(ctx *HookContext, fields []config.Field, data map[string]any) (map[string]any, error) {
	return s.processFields(ctx, fields, data, "", func(fctx *FieldHookContext, hooks *config.FieldHooks) (any, error) {
		return s.ExecuteFieldAfterChange(fctx, hooks)
	})
}

// ProcessFieldsAfterRead processes all fields with afterRead hooks.
func (s *FieldService) ProcessFieldsAfterRead(ctx *HookContext, fields []config.Field, data map[string]any) (map[string]any, error) {
	return s.processFields(ctx, fields, data, "", func(fctx *FieldHookContext, hooks *config.FieldHooks) (any, error) {
		return s.ExecuteFieldAfterRead(fctx, hooks)
	})
}

type fieldHookFn func(ctx *FieldHookContext, hooks *config.FieldHooks) (any, error)

func (s *FieldService) processFields(
	ctx *HookContext,
	fields []config.Field,
	data map[string]any,
	pathPrefix string,
	hookFn fieldHookFn,
) (map[string]any, error) {
	if data == nil {
		return nil, nil
	}

	result := make(map[string]any)
	for k, v := range data {
		result[k] = v
	}

	for i := range fields {
		field := &fields[i]

		// Skip non-data fields
		if !field.IsDataField() {
			continue
		}

		value, exists := data[field.Name]
		if !exists {
			continue
		}

		path := field.Name
		if pathPrefix != "" {
			path = pathPrefix + "." + field.Name
		}

		// Create field hook context
		fctx := &FieldHookContext{
			Ctx:         ctx.Ctx,
			Value:       value,
			OriginalDoc: ctx.OriginalDoc,
			Data:        ctx.Data,
			SiblingData: data,
			Field:       field,
			Collection:  ctx.Collection,
			Operation:   ctx.Operation,
			Req:         ctx.Req,
			User:        ctx.User,
			Path:        path,
		}

		// Execute field hook if defined
		if field.Hooks != nil {
			newValue, err := hookFn(fctx, field.Hooks)
			if err != nil {
				return nil, fmt.Errorf("field hook error at %s: %w", path, err)
			}
			value = newValue
			result[field.Name] = value
		}

		// Process nested fields
		if field.HasNestedFields() {
			processedValue, err := s.processNestedFields(ctx, field, value, path, hookFn)
			if err != nil {
				return nil, err
			}
			result[field.Name] = processedValue
		}
	}

	return result, nil
}

func (s *FieldService) processNestedFields(
	ctx *HookContext,
	field *config.Field,
	value any,
	path string,
	hookFn fieldHookFn,
) (any, error) {
	switch field.Type {
	case config.FieldTypeArray:
		return s.processArrayField(ctx, field, value, path, hookFn)
	case config.FieldTypeGroup:
		if data, ok := value.(map[string]any); ok {
			return s.processFields(ctx, field.Fields, data, path, hookFn)
		}
	case config.FieldTypeBlocks:
		return s.processBlocksField(ctx, field, value, path, hookFn)
	case config.FieldTypeTabs:
		return s.processTabsField(ctx, field, value, path, hookFn)
	case config.FieldTypeRow, config.FieldTypeCollapsible:
		if data, ok := value.(map[string]any); ok {
			return s.processFields(ctx, field.Fields, data, path, hookFn)
		}
	}

	return value, nil
}

func (s *FieldService) processArrayField(
	ctx *HookContext,
	field *config.Field,
	value any,
	path string,
	hookFn fieldHookFn,
) (any, error) {
	arr, ok := value.([]any)
	if !ok {
		return value, nil
	}

	result := make([]any, len(arr))
	for i, item := range arr {
		itemData, ok := item.(map[string]any)
		if !ok {
			result[i] = item
			continue
		}

		itemPath := fmt.Sprintf("%s.%d", path, i)
		processed, err := s.processFields(ctx, field.Fields, itemData, itemPath, hookFn)
		if err != nil {
			return nil, err
		}
		result[i] = processed
	}

	return result, nil
}

func (s *FieldService) processBlocksField(
	ctx *HookContext,
	field *config.Field,
	value any,
	path string,
	hookFn fieldHookFn,
) (any, error) {
	arr, ok := value.([]any)
	if !ok {
		return value, nil
	}

	result := make([]any, len(arr))
	for i, item := range arr {
		itemData, ok := item.(map[string]any)
		if !ok {
			result[i] = item
			continue
		}

		// Get block type
		blockType, _ := itemData["blockType"].(string)
		if blockType == "" {
			result[i] = item
			continue
		}

		// Find matching block config
		var blockFields []config.Field
		for _, block := range field.Blocks {
			if block.Slug == blockType {
				blockFields = block.Fields
				break
			}
		}

		if len(blockFields) == 0 {
			result[i] = item
			continue
		}

		itemPath := fmt.Sprintf("%s.%d", path, i)
		processed, err := s.processFields(ctx, blockFields, itemData, itemPath, hookFn)
		if err != nil {
			return nil, err
		}
		result[i] = processed
	}

	return result, nil
}

func (s *FieldService) processTabsField(
	ctx *HookContext,
	field *config.Field,
	value any,
	path string,
	hookFn fieldHookFn,
) (any, error) {
	data, ok := value.(map[string]any)
	if !ok {
		return value, nil
	}

	result := make(map[string]any)
	for k, v := range data {
		result[k] = v
	}

	for _, tab := range field.Tabs {
		if tab.Name == "" {
			// Unnamed tab - fields are at root level
			processed, err := s.processFields(ctx, tab.Fields, data, path, hookFn)
			if err != nil {
				return nil, err
			}
			for k, v := range processed {
				result[k] = v
			}
		} else {
			// Named tab - fields are nested under tab name
			tabData, ok := data[tab.Name].(map[string]any)
			if !ok {
				continue
			}
			tabPath := path + "." + tab.Name
			processed, err := s.processFields(ctx, tab.Fields, tabData, tabPath, hookFn)
			if err != nil {
				return nil, err
			}
			result[tab.Name] = processed
		}
	}

	return result, nil
}
