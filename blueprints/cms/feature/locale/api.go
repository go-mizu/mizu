// Package locale provides field-level localization support.
package locale

import (
	"context"

	"github.com/go-mizu/blueprints/cms/config"
)

// LocaleOptions holds options for localization operations.
type LocaleOptions struct {
	Locale         string // Target locale
	FallbackLocale string // Fallback if target not available
}

// Service defines the localization service interface.
type Service interface {
	// LocalizeDocument processes a document for the target locale.
	// Localized fields are flattened from {locale: value} to just value.
	LocalizeDocument(ctx context.Context, doc map[string]any, fields []config.Field, opts *LocaleOptions) (map[string]any, error)

	// LocalizeDocs processes multiple documents for the target locale.
	LocalizeDocs(ctx context.Context, docs []map[string]any, fields []config.Field, opts *LocaleOptions) ([]map[string]any, error)

	// ExpandLocalizedField expands a localized field value to its storage format.
	// Example: "Hello" with locale "en" -> {"en": "Hello"}
	ExpandLocalizedField(value any, locale string, existingLocales map[string]any) map[string]any

	// PrepareDataForStorage prepares document data for storage by expanding localized fields.
	PrepareDataForStorage(data map[string]any, fields []config.Field, locale string) (map[string]any, error)

	// GetAvailableLocales returns all locales for which a document has content.
	GetAvailableLocales(doc map[string]any, fields []config.Field) []string
}

// Store defines the locale storage interface.
type Store interface {
	// GetLocalizedValue retrieves a localized value for a field.
	GetLocalizedValue(ctx context.Context, collection, docID, fieldPath, locale string) (any, error)

	// SetLocalizedValue stores a localized value for a field.
	SetLocalizedValue(ctx context.Context, collection, docID, fieldPath, locale string, value any) error

	// GetAllLocales retrieves all localized values for a field.
	GetAllLocales(ctx context.Context, collection, docID, fieldPath string) (map[string]any, error)

	// DeleteLocale removes a locale value for a field.
	DeleteLocale(ctx context.Context, collection, docID, fieldPath, locale string) error
}
