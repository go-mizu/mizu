package validation

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/go-mizu/blueprints/cms/config"
)

// ValidatorService implements document and field validation.
type ValidatorService struct {
	uniqueChecker UniqueChecker
}

// NewValidator creates a new validator service.
func NewValidator(uniqueChecker UniqueChecker) *ValidatorService {
	return &ValidatorService{
		uniqueChecker: uniqueChecker,
	}
}

// ValidateDocument validates an entire document.
func (v *ValidatorService) ValidateDocument(ctx *ValidationContext, data map[string]any, fields []config.Field) *ValidationResult {
	result := &ValidationResult{Valid: true}

	for i := range fields {
		field := &fields[i]

		// Skip non-data fields
		if !field.IsDataField() {
			continue
		}

		// Check condition
		if field.Condition != nil && !field.Condition(data, data) {
			continue
		}

		value, exists := data[field.Name]
		fieldResult := v.validateFieldValue(ctx, value, exists, field, field.Name)

		if !fieldResult.IsValid() {
			result.Valid = false
			result.Errors = append(result.Errors, fieldResult.Errors...)
		}
	}

	return result
}

func (v *ValidatorService) validateFieldValue(ctx *ValidationContext, value any, exists bool, field *config.Field, path string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Check required
	if field.Required {
		if !exists || !v.ValidateRequired(value) {
			result.AddError(path, fmt.Sprintf("%s is required", field.Label), value)
			return result // Stop validation if required field is missing
		}
	}

	// Skip further validation if value is nil/empty and not required
	if !exists || value == nil {
		return result
	}

	// Type-specific validation
	switch field.Type {
	case config.FieldTypeText, config.FieldTypeTextarea:
		v.validateString(result, value, field, path)
	case config.FieldTypeEmail:
		v.validateEmail(result, value, field, path)
	case config.FieldTypeNumber:
		v.validateNumber(result, value, field, path)
	case config.FieldTypeSelect, config.FieldTypeRadio:
		v.validateSelect(result, value, field, path)
	case config.FieldTypeArray:
		v.validateArray(ctx, result, value, field, path)
	case config.FieldTypeBlocks:
		v.validateBlocks(ctx, result, value, field, path)
	case config.FieldTypeGroup:
		v.validateGroup(ctx, result, value, field, path)
	case config.FieldTypeTabs:
		v.validateTabs(ctx, result, value, field, path)
	case config.FieldTypeRelationship:
		v.validateRelationship(result, value, field, path)
	}

	// Run custom validation function
	if field.Validate != nil {
		valCtx := &config.ValidationContext{
			Data:        ctx.Data,
			SiblingData: ctx.SiblingData,
			Operation:   ctx.Operation,
			User:        ctx.User,
		}
		if err := field.Validate(value, valCtx); err != nil {
			result.AddError(path, err.Error(), value)
		}
	}

	// Check uniqueness
	if field.Unique && ctx.Collection != "" {
		isUnique, err := v.ValidateUnique(ctx.Ctx, ctx.Collection, field.Name, value, ctx.ID)
		if err == nil && !isUnique {
			result.AddError(path, fmt.Sprintf("%s must be unique", field.Label), value)
		}
	}

	return result
}

func (v *ValidatorService) validateString(result *ValidationResult, value any, field *config.Field, path string) {
	str, ok := value.(string)
	if !ok {
		return
	}

	if field.MinLength != nil && !v.ValidateMinLength(str, *field.MinLength) {
		result.AddError(path, fmt.Sprintf("%s must be at least %d characters", field.Label, *field.MinLength), value)
	}

	if field.MaxLength != nil && !v.ValidateMaxLength(str, *field.MaxLength) {
		result.AddError(path, fmt.Sprintf("%s must be at most %d characters", field.Label, *field.MaxLength), value)
	}
}

func (v *ValidatorService) validateEmail(result *ValidationResult, value any, field *config.Field, path string) {
	str, ok := value.(string)
	if !ok {
		return
	}

	if str != "" && !v.ValidateEmail(str) {
		result.AddError(path, fmt.Sprintf("%s must be a valid email address", field.Label), value)
	}
}

func (v *ValidatorService) validateNumber(result *ValidationResult, value any, field *config.Field, path string) {
	var num float64

	switch n := value.(type) {
	case float64:
		num = n
	case float32:
		num = float64(n)
	case int:
		num = float64(n)
	case int64:
		num = float64(n)
	default:
		return
	}

	if field.Min != nil && !v.ValidateMin(num, *field.Min) {
		result.AddError(path, fmt.Sprintf("%s must be at least %v", field.Label, *field.Min), value)
	}

	if field.Max != nil && !v.ValidateMax(num, *field.Max) {
		result.AddError(path, fmt.Sprintf("%s must be at most %v", field.Label, *field.Max), value)
	}
}

func (v *ValidatorService) validateSelect(result *ValidationResult, value any, field *config.Field, path string) {
	if len(field.Options) == 0 {
		return
	}

	validValues := make(map[string]bool)
	for _, opt := range field.Options {
		validValues[opt.Value] = true
	}

	if field.HasMany {
		// Validate array of values
		arr, ok := value.([]any)
		if !ok {
			if strArr, ok := value.([]string); ok {
				arr = make([]any, len(strArr))
				for i, s := range strArr {
					arr[i] = s
				}
			} else {
				return
			}
		}

		for _, item := range arr {
			if str, ok := item.(string); ok && !validValues[str] {
				result.AddError(path, fmt.Sprintf("Invalid option: %s", str), value)
			}
		}
	} else {
		// Validate single value
		if str, ok := value.(string); ok && !validValues[str] {
			result.AddError(path, fmt.Sprintf("Invalid option: %s", str), value)
		}
	}
}

func (v *ValidatorService) validateArray(ctx *ValidationContext, result *ValidationResult, value any, field *config.Field, path string) {
	arr, ok := value.([]any)
	if !ok {
		return
	}

	// Check min/max rows
	if field.MinRows != nil && len(arr) < *field.MinRows {
		result.AddError(path, fmt.Sprintf("%s must have at least %d items", field.Label, *field.MinRows), value)
	}

	if field.MaxRows != nil && len(arr) > *field.MaxRows {
		result.AddError(path, fmt.Sprintf("%s must have at most %d items", field.Label, *field.MaxRows), value)
	}

	// Validate each item
	for i, item := range arr {
		if itemMap, ok := item.(map[string]any); ok {
			itemPath := fmt.Sprintf("%s.%d", path, i)
			itemCtx := &ValidationContext{
				Ctx:         ctx.Ctx,
				Data:        ctx.Data,
				SiblingData: itemMap,
				Operation:   ctx.Operation,
				User:        ctx.User,
				ID:          ctx.ID,
				Collection:  ctx.Collection,
			}
			itemResult := v.ValidateDocument(itemCtx, itemMap, field.Fields)
			if !itemResult.IsValid() {
				result.Valid = false
				for _, err := range itemResult.Errors {
					err.Field = itemPath + "." + err.Field
					result.Errors = append(result.Errors, err)
				}
			}
		}
	}
}

func (v *ValidatorService) validateBlocks(ctx *ValidationContext, result *ValidationResult, value any, field *config.Field, path string) {
	arr, ok := value.([]any)
	if !ok {
		return
	}

	// Check min/max rows
	if field.MinRows != nil && len(arr) < *field.MinRows {
		result.AddError(path, fmt.Sprintf("%s must have at least %d blocks", field.Label, *field.MinRows), value)
	}

	if field.MaxRows != nil && len(arr) > *field.MaxRows {
		result.AddError(path, fmt.Sprintf("%s must have at most %d blocks", field.Label, *field.MaxRows), value)
	}

	for i, item := range arr {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		blockType, _ := itemMap["blockType"].(string)
		if blockType == "" {
			result.AddError(fmt.Sprintf("%s.%d", path, i), "Block type is required", item)
			continue
		}

		// Find block config
		var blockFields []config.Field
		validBlock := false
		for _, block := range field.Blocks {
			if block.Slug == blockType {
				blockFields = block.Fields
				validBlock = true
				break
			}
		}

		if !validBlock {
			result.AddError(fmt.Sprintf("%s.%d", path, i), fmt.Sprintf("Invalid block type: %s", blockType), item)
			continue
		}

		// Validate block fields
		itemPath := fmt.Sprintf("%s.%d", path, i)
		itemCtx := &ValidationContext{
			Ctx:         ctx.Ctx,
			Data:        ctx.Data,
			SiblingData: itemMap,
			Operation:   ctx.Operation,
			User:        ctx.User,
			ID:          ctx.ID,
			Collection:  ctx.Collection,
		}
		itemResult := v.ValidateDocument(itemCtx, itemMap, blockFields)
		if !itemResult.IsValid() {
			result.Valid = false
			for _, err := range itemResult.Errors {
				err.Field = itemPath + "." + err.Field
				result.Errors = append(result.Errors, err)
			}
		}
	}
}

func (v *ValidatorService) validateGroup(ctx *ValidationContext, result *ValidationResult, value any, field *config.Field, path string) {
	data, ok := value.(map[string]any)
	if !ok {
		return
	}

	groupCtx := &ValidationContext{
		Ctx:         ctx.Ctx,
		Data:        ctx.Data,
		SiblingData: data,
		Operation:   ctx.Operation,
		User:        ctx.User,
		ID:          ctx.ID,
		Collection:  ctx.Collection,
	}

	groupResult := v.ValidateDocument(groupCtx, data, field.Fields)
	if !groupResult.IsValid() {
		result.Valid = false
		for _, err := range groupResult.Errors {
			err.Field = path + "." + err.Field
			result.Errors = append(result.Errors, err)
		}
	}
}

func (v *ValidatorService) validateTabs(ctx *ValidationContext, result *ValidationResult, value any, field *config.Field, path string) {
	data, ok := value.(map[string]any)
	if !ok {
		return
	}

	for _, tab := range field.Tabs {
		var tabData map[string]any
		var tabPath string

		if tab.Name == "" {
			// Unnamed tab - fields at root level
			tabData = data
			tabPath = path
		} else {
			// Named tab
			if td, ok := data[tab.Name].(map[string]any); ok {
				tabData = td
				tabPath = path + "." + tab.Name
			} else {
				continue
			}
		}

		tabCtx := &ValidationContext{
			Ctx:         ctx.Ctx,
			Data:        ctx.Data,
			SiblingData: tabData,
			Operation:   ctx.Operation,
			User:        ctx.User,
			ID:          ctx.ID,
			Collection:  ctx.Collection,
		}

		tabResult := v.ValidateDocument(tabCtx, tabData, tab.Fields)
		if !tabResult.IsValid() {
			result.Valid = false
			for _, err := range tabResult.Errors {
				if tab.Name != "" {
					err.Field = tabPath + "." + err.Field
				}
				result.Errors = append(result.Errors, err)
			}
		}
	}
}

func (v *ValidatorService) validateRelationship(result *ValidationResult, value any, field *config.Field, path string) {
	if len(field.RelationTo) == 0 {
		return
	}

	// Just verify the format, not the existence
	// Existence would be checked on population or via hooks
}

// ValidateField validates a single field.
func (v *ValidatorService) ValidateField(ctx *ValidationContext, value any, field *config.Field, path string) *ValidationResult {
	return v.validateFieldValue(ctx, value, value != nil, field, path)
}

// ValidateRequired checks if a required field has a value.
func (v *ValidatorService) ValidateRequired(value any) bool {
	if value == nil {
		return false
	}

	switch val := value.(type) {
	case string:
		return strings.TrimSpace(val) != ""
	case []any:
		return len(val) > 0
	case map[string]any:
		return len(val) > 0
	default:
		return true
	}
}

// ValidateMinLength checks minimum string length.
func (v *ValidatorService) ValidateMinLength(value string, min int) bool {
	return len(value) >= min
}

// ValidateMaxLength checks maximum string length.
func (v *ValidatorService) ValidateMaxLength(value string, max int) bool {
	return len(value) <= max
}

// ValidateMin checks minimum numeric value.
func (v *ValidatorService) ValidateMin(value float64, min float64) bool {
	return value >= min
}

// ValidateMax checks maximum numeric value.
func (v *ValidatorService) ValidateMax(value float64, max float64) bool {
	return value <= max
}

// ValidateEmail validates email format.
func (v *ValidatorService) ValidateEmail(value string) bool {
	_, err := mail.ParseAddress(value)
	return err == nil
}

// ValidateUnique checks if a value is unique in the collection.
func (v *ValidatorService) ValidateUnique(ctx context.Context, collection, field string, value any, excludeID string) (bool, error) {
	if v.uniqueChecker == nil {
		return true, nil // Skip if no checker configured
	}
	return v.uniqueChecker.IsUnique(ctx, collection, field, value, excludeID)
}
