package locale

import (
	"context"

	"github.com/go-mizu/blueprints/cms/config"
)

// LocaleService implements field-level localization.
type LocaleService struct {
	defaultLocale string
	locales       []config.Locale
}

// NewService creates a new locale service.
func NewService(cfg *config.LocalizationConfig) *LocaleService {
	s := &LocaleService{
		defaultLocale: "en",
	}

	if cfg != nil {
		s.defaultLocale = cfg.DefaultLocale
		s.locales = cfg.Locales
	}

	return s
}

// LocalizeDocument processes a document for the target locale.
func (s *LocaleService) LocalizeDocument(ctx context.Context, doc map[string]any, fields []config.Field, opts *LocaleOptions) (map[string]any, error) {
	if doc == nil {
		return nil, nil
	}

	if opts == nil {
		opts = &LocaleOptions{
			Locale:         s.defaultLocale,
			FallbackLocale: s.defaultLocale,
		}
	}
	if opts.Locale == "" {
		opts.Locale = s.defaultLocale
	}
	if opts.FallbackLocale == "" {
		opts.FallbackLocale = s.defaultLocale
	}

	result := make(map[string]any)

	// Copy non-field data
	fieldNames := make(map[string]bool)
	for _, f := range fields {
		fieldNames[f.Name] = true
	}
	for k, v := range doc {
		if !fieldNames[k] {
			result[k] = v
		}
	}

	// Process fields
	for i := range fields {
		field := &fields[i]
		value, exists := doc[field.Name]
		if !exists {
			continue
		}

		if field.Localized {
			// Extract value for target locale
			localized := s.extractLocalizedValue(value, opts.Locale, opts.FallbackLocale)
			result[field.Name] = localized
		} else if field.HasNestedFields() {
			// Recursively process nested fields
			nested := s.localizeNested(ctx, value, field, opts)
			result[field.Name] = nested
		} else {
			result[field.Name] = value
		}
	}

	return result, nil
}

func (s *LocaleService) extractLocalizedValue(value any, locale, fallback string) any {
	locales, ok := value.(map[string]any)
	if !ok {
		// Not a localized value, return as-is
		return value
	}

	// Try target locale
	if v, ok := locales[locale]; ok && v != nil {
		return v
	}

	// Try fallback
	if v, ok := locales[fallback]; ok && v != nil {
		return v
	}

	// Try default
	if v, ok := locales[s.defaultLocale]; ok && v != nil {
		return v
	}

	// Return first available
	for _, v := range locales {
		if v != nil {
			return v
		}
	}

	return nil
}

func (s *LocaleService) localizeNested(ctx context.Context, value any, field *config.Field, opts *LocaleOptions) any {
	switch field.Type {
	case config.FieldTypeArray:
		return s.localizeArray(ctx, value, field, opts)
	case config.FieldTypeGroup:
		if data, ok := value.(map[string]any); ok {
			result, _ := s.LocalizeDocument(ctx, data, field.Fields, opts)
			return result
		}
	case config.FieldTypeBlocks:
		return s.localizeBlocks(ctx, value, field, opts)
	case config.FieldTypeTabs:
		return s.localizeTabs(ctx, value, field, opts)
	}
	return value
}

func (s *LocaleService) localizeArray(ctx context.Context, value any, field *config.Field, opts *LocaleOptions) any {
	arr, ok := value.([]any)
	if !ok {
		return value
	}

	result := make([]any, len(arr))
	for i, item := range arr {
		if itemMap, ok := item.(map[string]any); ok {
			localized, _ := s.LocalizeDocument(ctx, itemMap, field.Fields, opts)
			result[i] = localized
		} else {
			result[i] = item
		}
	}
	return result
}

func (s *LocaleService) localizeBlocks(ctx context.Context, value any, field *config.Field, opts *LocaleOptions) any {
	arr, ok := value.([]any)
	if !ok {
		return value
	}

	result := make([]any, len(arr))
	for i, item := range arr {
		itemMap, ok := item.(map[string]any)
		if !ok {
			result[i] = item
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
			localized, _ := s.LocalizeDocument(ctx, itemMap, blockFields, opts)
			result[i] = localized
		} else {
			result[i] = item
		}
	}
	return result
}

func (s *LocaleService) localizeTabs(ctx context.Context, value any, field *config.Field, opts *LocaleOptions) any {
	data, ok := value.(map[string]any)
	if !ok {
		return value
	}

	result := make(map[string]any)
	for _, tab := range field.Tabs {
		if tab.Name == "" {
			// Unnamed tab - fields at root level
			localized, _ := s.LocalizeDocument(ctx, data, tab.Fields, opts)
			for k, v := range localized {
				result[k] = v
			}
		} else if tabData, ok := data[tab.Name].(map[string]any); ok {
			localized, _ := s.LocalizeDocument(ctx, tabData, tab.Fields, opts)
			result[tab.Name] = localized
		}
	}
	return result
}

// LocalizeDocs processes multiple documents for the target locale.
func (s *LocaleService) LocalizeDocs(ctx context.Context, docs []map[string]any, fields []config.Field, opts *LocaleOptions) ([]map[string]any, error) {
	result := make([]map[string]any, len(docs))
	for i, doc := range docs {
		localized, err := s.LocalizeDocument(ctx, doc, fields, opts)
		if err != nil {
			return nil, err
		}
		result[i] = localized
	}
	return result, nil
}

// ExpandLocalizedField expands a value to its localized storage format.
func (s *LocaleService) ExpandLocalizedField(value any, locale string, existingLocales map[string]any) map[string]any {
	result := make(map[string]any)

	// Preserve existing locales
	for k, v := range existingLocales {
		result[k] = v
	}

	// Set the new value for the target locale
	result[locale] = value

	return result
}

// PrepareDataForStorage prepares document data by expanding localized fields.
func (s *LocaleService) PrepareDataForStorage(data map[string]any, fields []config.Field, locale string) (map[string]any, error) {
	if locale == "" {
		locale = s.defaultLocale
	}

	result := make(map[string]any)
	for k, v := range data {
		result[k] = v
	}

	for i := range fields {
		field := &fields[i]
		value, exists := data[field.Name]
		if !exists {
			continue
		}

		if field.Localized {
			// Check if already in localized format
			if locales, ok := value.(map[string]any); ok {
				// Already localized, use as-is
				result[field.Name] = locales
			} else {
				// Wrap in locale map
				result[field.Name] = map[string]any{locale: value}
			}
		} else if field.HasNestedFields() {
			// Recursively process nested fields
			nested := s.prepareNestedForStorage(value, field, locale)
			result[field.Name] = nested
		}
	}

	return result, nil
}

func (s *LocaleService) prepareNestedForStorage(value any, field *config.Field, locale string) any {
	switch field.Type {
	case config.FieldTypeArray:
		if arr, ok := value.([]any); ok {
			result := make([]any, len(arr))
			for i, item := range arr {
				if itemMap, ok := item.(map[string]any); ok {
					prepared, _ := s.PrepareDataForStorage(itemMap, field.Fields, locale)
					result[i] = prepared
				} else {
					result[i] = item
				}
			}
			return result
		}
	case config.FieldTypeGroup:
		if data, ok := value.(map[string]any); ok {
			prepared, _ := s.PrepareDataForStorage(data, field.Fields, locale)
			return prepared
		}
	case config.FieldTypeBlocks:
		if arr, ok := value.([]any); ok {
			result := make([]any, len(arr))
			for i, item := range arr {
				itemMap, ok := item.(map[string]any)
				if !ok {
					result[i] = item
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
					prepared, _ := s.PrepareDataForStorage(itemMap, blockFields, locale)
					result[i] = prepared
				} else {
					result[i] = item
				}
			}
			return result
		}
	case config.FieldTypeTabs:
		if data, ok := value.(map[string]any); ok {
			result := make(map[string]any)
			for k, v := range data {
				result[k] = v
			}
			for _, tab := range field.Tabs {
				if tab.Name == "" {
					prepared, _ := s.PrepareDataForStorage(data, tab.Fields, locale)
					for k, v := range prepared {
						result[k] = v
					}
				} else if tabData, ok := data[tab.Name].(map[string]any); ok {
					prepared, _ := s.PrepareDataForStorage(tabData, tab.Fields, locale)
					result[tab.Name] = prepared
				}
			}
			return result
		}
	}
	return value
}

// GetAvailableLocales returns all locales for which a document has content.
func (s *LocaleService) GetAvailableLocales(doc map[string]any, fields []config.Field) []string {
	localeSet := make(map[string]bool)

	for i := range fields {
		field := &fields[i]
		if !field.Localized {
			continue
		}

		value, ok := doc[field.Name]
		if !ok {
			continue
		}

		if locales, ok := value.(map[string]any); ok {
			for locale := range locales {
				localeSet[locale] = true
			}
		}
	}

	result := make([]string, 0, len(localeSet))
	for locale := range localeSet {
		result = append(result, locale)
	}

	return result
}
