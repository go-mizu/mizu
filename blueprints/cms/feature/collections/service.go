package collections

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/config"
	"github.com/go-mizu/blueprints/cms/store/duckdb"
)

var (
	ErrNotFound          = errors.New("document not found")
	ErrCollectionNotFound = errors.New("collection not found")
	ErrValidation        = errors.New("validation error")
)

// Service implements the collections API.
type Service struct {
	store       *duckdb.CollectionsStore
	versions    *duckdb.VersionsStore
	collections map[string]*config.CollectionConfig
}

// NewService creates a new collections service.
func NewService(store *duckdb.CollectionsStore, versions *duckdb.VersionsStore, collections []config.CollectionConfig) *Service {
	collectionMap := make(map[string]*config.CollectionConfig)
	for i := range collections {
		collectionMap[collections[i].Slug] = &collections[i]
	}
	return &Service{
		store:       store,
		versions:    versions,
		collections: collectionMap,
	}
}

// GetConfig returns the collection configuration.
func (s *Service) GetConfig(collection string) *config.CollectionConfig {
	return s.collections[collection]
}

// Find finds documents matching the query.
func (s *Service) Find(ctx context.Context, collection string, input *FindInput) (*FindResult, error) {
	cfg := s.GetConfig(collection)
	if cfg == nil {
		return nil, ErrCollectionNotFound
	}

	if input == nil {
		input = &FindInput{}
	}

	// Parse sort string
	var sortFields []duckdb.SortField
	if input.Sort != "" {
		parts := strings.Split(input.Sort, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			desc := false
			if strings.HasPrefix(part, "-") {
				desc = true
				part = part[1:]
			}
			sortFields = append(sortFields, duckdb.SortField{
				Field: part,
				Desc:  desc,
			})
		}
	}

	opts := &duckdb.FindOptions{
		Where: input.Where,
		Sort:  sortFields,
		Limit: input.Limit,
		Page:  input.Page,
	}

	result, err := s.store.Find(ctx, collection, opts)
	if err != nil {
		return nil, fmt.Errorf("find documents: %w", err)
	}

	// Convert to output format
	docs := make([]map[string]any, len(result.Docs))
	for i, doc := range result.Docs {
		docs[i] = s.documentToMap(doc, input.Depth)
	}

	return &FindResult{
		Docs:          docs,
		TotalDocs:     result.TotalDocs,
		Limit:         result.Limit,
		TotalPages:    result.TotalPages,
		Page:          result.Page,
		PagingCounter: result.PagingCounter,
		HasPrevPage:   result.HasPrevPage,
		HasNextPage:   result.HasNextPage,
		PrevPage:      result.PrevPage,
		NextPage:      result.NextPage,
	}, nil
}

// FindByID finds a document by ID.
func (s *Service) FindByID(ctx context.Context, collection, id string, depth int, locale string) (map[string]any, error) {
	cfg := s.GetConfig(collection)
	if cfg == nil {
		return nil, ErrCollectionNotFound
	}

	doc, err := s.store.FindByID(ctx, collection, id)
	if err != nil {
		return nil, fmt.Errorf("find by id: %w", err)
	}
	if doc == nil {
		return nil, ErrNotFound
	}

	return s.documentToMap(*doc, depth), nil
}

// Count counts documents matching the query.
func (s *Service) Count(ctx context.Context, collection string, where map[string]any) (int, error) {
	cfg := s.GetConfig(collection)
	if cfg == nil {
		return 0, ErrCollectionNotFound
	}

	return s.store.Count(ctx, collection, where)
}

// Create creates a new document.
func (s *Service) Create(ctx context.Context, collection string, input *CreateInput) (map[string]any, error) {
	cfg := s.GetConfig(collection)
	if cfg == nil {
		return nil, ErrCollectionNotFound
	}

	// Validate required fields
	if err := s.validateFields(cfg.Fields, input.Data); err != nil {
		return nil, err
	}

	// Set status if versioning enabled
	if cfg.Versions != nil && cfg.Versions.Drafts {
		if input.Draft {
			input.Data["_status"] = "draft"
		} else {
			input.Data["_status"] = "published"
		}
	}

	doc, err := s.store.Create(ctx, collection, input.Data)
	if err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}

	// Create initial version if versioning enabled
	if cfg.Versions != nil {
		version := &duckdb.Version{
			Parent:    doc.ID,
			Version:   1,
			Snapshot:  input.Data,
			Published: !input.Draft,
		}
		if err := s.versions.Create(ctx, collection, version); err != nil {
			// Log but don't fail
			fmt.Printf("failed to create version: %v\n", err)
		}
	}

	return s.documentToMap(*doc, input.Depth), nil
}

// UpdateByID updates a document by ID.
func (s *Service) UpdateByID(ctx context.Context, collection, id string, input *UpdateInput) (map[string]any, error) {
	cfg := s.GetConfig(collection)
	if cfg == nil {
		return nil, ErrCollectionNotFound
	}

	// Get existing document for version tracking
	existing, err := s.store.FindByID(ctx, collection, id)
	if err != nil {
		return nil, fmt.Errorf("get existing: %w", err)
	}
	if existing == nil {
		return nil, ErrNotFound
	}

	// Validate fields
	if err := s.validateFields(cfg.Fields, input.Data); err != nil {
		return nil, err
	}

	// Update status if versioning enabled
	if cfg.Versions != nil && cfg.Versions.Drafts {
		if input.Draft {
			input.Data["_status"] = "draft"
		} else {
			input.Data["_status"] = "published"
		}
	}

	doc, err := s.store.UpdateByID(ctx, collection, id, input.Data)
	if err != nil {
		return nil, fmt.Errorf("update document: %w", err)
	}
	if doc == nil {
		return nil, ErrNotFound
	}

	// Create new version if versioning enabled
	if cfg.Versions != nil {
		latestVersion, _ := s.versions.GetLatestVersion(ctx, collection, id)
		version := &duckdb.Version{
			Parent:    id,
			Version:   latestVersion + 1,
			Snapshot:  input.Data,
			Published: !input.Draft,
			Autosave:  input.Autosave,
		}
		if err := s.versions.Create(ctx, collection, version); err != nil {
			fmt.Printf("failed to create version: %v\n", err)
		}

		// Clean up old versions if max exceeded
		if cfg.Versions.MaxPerDoc > 0 {
			s.versions.DeleteOldVersions(ctx, collection, id, cfg.Versions.MaxPerDoc)
		}
	}

	return s.documentToMap(*doc, input.Depth), nil
}

// Update updates documents matching the query.
func (s *Service) Update(ctx context.Context, collection string, where map[string]any, input *UpdateInput) ([]map[string]any, error) {
	cfg := s.GetConfig(collection)
	if cfg == nil {
		return nil, ErrCollectionNotFound
	}

	// First find all matching documents
	result, err := s.store.Find(ctx, collection, &duckdb.FindOptions{
		Where: where,
		Limit: 100, // Reasonable limit for bulk updates
	})
	if err != nil {
		return nil, fmt.Errorf("find documents: %w", err)
	}

	var updated []map[string]any
	for _, doc := range result.Docs {
		updatedDoc, err := s.UpdateByID(ctx, collection, doc.ID, input)
		if err != nil {
			continue // Skip failed updates
		}
		updated = append(updated, updatedDoc)
	}

	return updated, nil
}

// DeleteByID deletes a document by ID.
func (s *Service) DeleteByID(ctx context.Context, collection, id string) (*DeleteResult, error) {
	cfg := s.GetConfig(collection)
	if cfg == nil {
		return nil, ErrCollectionNotFound
	}

	// Get document before deletion
	doc, err := s.store.FindByID(ctx, collection, id)
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}
	if doc == nil {
		return nil, ErrNotFound
	}

	deleted, err := s.store.DeleteByID(ctx, collection, id)
	if err != nil {
		return nil, fmt.Errorf("delete document: %w", err)
	}

	return &DeleteResult{
		ID:      id,
		Deleted: deleted,
		Doc:     s.documentToMap(*doc, 0),
	}, nil
}

// Delete deletes documents matching the query.
func (s *Service) Delete(ctx context.Context, collection string, where map[string]any) ([]DeleteResult, error) {
	cfg := s.GetConfig(collection)
	if cfg == nil {
		return nil, ErrCollectionNotFound
	}

	// First find all matching documents
	result, err := s.store.Find(ctx, collection, &duckdb.FindOptions{
		Where: where,
		Limit: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("find documents: %w", err)
	}

	var results []DeleteResult
	for _, doc := range result.Docs {
		deleteResult, err := s.DeleteByID(ctx, collection, doc.ID)
		if err != nil {
			continue
		}
		results = append(results, *deleteResult)
	}

	return results, nil
}

// validateFields validates document data against field definitions.
func (s *Service) validateFields(fields []config.Field, data map[string]any) error {
	for _, field := range fields {
		if !field.IsDataField() {
			continue
		}

		value, exists := data[field.Name]

		// Check required
		if field.Required && (!exists || value == nil || value == "") {
			return fmt.Errorf("%w: %s is required", ErrValidation, field.Name)
		}

		if !exists || value == nil {
			continue
		}

		// Type-specific validation
		switch field.Type {
		case config.FieldTypeEmail:
			str, ok := value.(string)
			if ok && str != "" && !strings.Contains(str, "@") {
				return fmt.Errorf("%w: %s must be a valid email", ErrValidation, field.Name)
			}
		case config.FieldTypeNumber:
			// Check min/max
			var num float64
			switch v := value.(type) {
			case int:
				num = float64(v)
			case int64:
				num = float64(v)
			case float64:
				num = v
			}
			if field.Min != nil && num < *field.Min {
				return fmt.Errorf("%w: %s must be >= %f", ErrValidation, field.Name, *field.Min)
			}
			if field.Max != nil && num > *field.Max {
				return fmt.Errorf("%w: %s must be <= %f", ErrValidation, field.Name, *field.Max)
			}
		case config.FieldTypeText, config.FieldTypeTextarea:
			str, ok := value.(string)
			if ok {
				if field.MinLength != nil && len(str) < *field.MinLength {
					return fmt.Errorf("%w: %s must be at least %d characters", ErrValidation, field.Name, *field.MinLength)
				}
				if field.MaxLength != nil && len(str) > *field.MaxLength {
					return fmt.Errorf("%w: %s must be at most %d characters", ErrValidation, field.Name, *field.MaxLength)
				}
			}
		}

		// Custom validation
		if field.Validate != nil {
			if err := field.Validate(value, &config.ValidationContext{
				Data:      data,
				Operation: "create",
			}); err != nil {
				return fmt.Errorf("%w: %s: %v", ErrValidation, field.Name, err)
			}
		}

		// Validate nested fields
		if field.HasNestedFields() {
			// Handle arrays
			if field.Type == config.FieldTypeArray {
				if arr, ok := value.([]any); ok {
					for i, item := range arr {
						if itemMap, ok := item.(map[string]any); ok {
							if err := s.validateFields(field.Fields, itemMap); err != nil {
								return fmt.Errorf("%s[%d]: %w", field.Name, i, err)
							}
						}
					}
				}
			}
			// Handle groups
			if field.Type == config.FieldTypeGroup {
				if itemMap, ok := value.(map[string]any); ok {
					if err := s.validateFields(field.Fields, itemMap); err != nil {
						return fmt.Errorf("%s: %w", field.Name, err)
					}
				}
			}
		}
	}
	return nil
}

// documentToMap converts a store document to a response map.
func (s *Service) documentToMap(doc duckdb.Document, depth int) map[string]any {
	result := map[string]any{
		"id":        doc.ID,
		"createdAt": doc.CreatedAt.Format(time.RFC3339),
		"updatedAt": doc.UpdatedAt.Format(time.RFC3339),
	}

	// Merge data fields
	for k, v := range doc.Data {
		result[k] = v
	}

	// Add status if present
	if doc.Status != "" {
		result["_status"] = doc.Status
	}

	return result
}
