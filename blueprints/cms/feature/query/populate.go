package query

import (
	"context"
	"fmt"

	"github.com/go-mizu/blueprints/cms/config"
	"github.com/go-mizu/blueprints/cms/store/duckdb"
)

// PopulatorService implements relationship population.
type PopulatorService struct {
	store *duckdb.CollectionsStore
	cfg   *config.Config
}

// NewPopulator creates a new populator service.
func NewPopulator(store *duckdb.CollectionsStore, cfg *config.Config) *PopulatorService {
	return &PopulatorService{
		store: store,
		cfg:   cfg,
	}
}

// Populate populates relationships in a document.
func (p *PopulatorService) Populate(ctx context.Context, doc map[string]any, fields []config.Field, opts *PopulateOptions) (map[string]any, error) {
	if opts == nil || opts.Depth <= 0 {
		return doc, nil
	}

	result := make(map[string]any)
	for k, v := range doc {
		result[k] = v
	}

	for i := range fields {
		field := &fields[i]
		value, exists := doc[field.Name]
		if !exists || value == nil {
			continue
		}

		// Check if we should populate this field
		if opts.Populate != nil && !opts.Populate[field.Name] {
			continue
		}

		if field.IsRelational() {
			populated, err := p.PopulateField(ctx, value, field, opts)
			if err != nil {
				return nil, fmt.Errorf("populate %s: %w", field.Name, err)
			}
			result[field.Name] = populated
		} else if field.HasNestedFields() {
			// Recursively populate nested fields
			nested, err := p.populateNested(ctx, value, field, opts)
			if err != nil {
				return nil, err
			}
			result[field.Name] = nested
		}
	}

	return result, nil
}

// PopulateField populates a specific relationship field.
func (p *PopulatorService) PopulateField(ctx context.Context, value any, field *config.Field, opts *PopulateOptions) (any, error) {
	if opts == nil || opts.Depth <= 0 {
		return value, nil
	}

	switch field.Type {
	case config.FieldTypeRelationship:
		return p.populateRelationship(ctx, value, field, opts)
	case config.FieldTypeUpload:
		return p.populateUpload(ctx, value, field, opts)
	default:
		return value, nil
	}
}

func (p *PopulatorService) populateRelationship(ctx context.Context, value any, field *config.Field, opts *PopulateOptions) (any, error) {
	if len(field.RelationTo) == 0 {
		return value, nil
	}

	// Handle hasMany
	if field.HasMany {
		return p.populateMany(ctx, value, field, opts)
	}

	// Single relationship
	return p.populateSingle(ctx, value, field, opts)
}

func (p *PopulatorService) populateSingle(ctx context.Context, value any, field *config.Field, opts *PopulateOptions) (any, error) {
	// Check for polymorphic relationship
	if len(field.RelationTo) > 1 {
		// Polymorphic: value should be {relationTo: "collection", value: "id"}
		if obj, ok := value.(map[string]any); ok {
			collection, _ := obj["relationTo"].(string)
			id, _ := obj["value"].(string)
			if collection != "" && id != "" {
				populated, err := p.fetchDocument(ctx, collection, id, opts)
				if err != nil {
					return value, nil // Return original on error
				}
				return map[string]any{
					"relationTo": collection,
					"value":      populated,
				}, nil
			}
		}
		return value, nil
	}

	// Single collection relationship
	collection := field.RelationTo[0]
	id, ok := value.(string)
	if !ok {
		return value, nil
	}

	return p.fetchDocument(ctx, collection, id, opts)
}

func (p *PopulatorService) populateMany(ctx context.Context, value any, field *config.Field, opts *PopulateOptions) (any, error) {
	arr, ok := value.([]any)
	if !ok {
		// Try string array
		if strArr, ok := value.([]string); ok {
			arr = make([]any, len(strArr))
			for i, s := range strArr {
				arr[i] = s
			}
		} else {
			return value, nil
		}
	}

	result := make([]any, 0, len(arr))
	for _, item := range arr {
		populated, err := p.populateSingle(ctx, item, field, &PopulateOptions{
			Depth:          opts.Depth - 1,
			Locale:         opts.Locale,
			FallbackLocale: opts.FallbackLocale,
		})
		if err != nil {
			result = append(result, item) // Keep original on error
			continue
		}
		result = append(result, populated)
	}

	return result, nil
}

func (p *PopulatorService) populateUpload(ctx context.Context, value any, field *config.Field, opts *PopulateOptions) (any, error) {
	// Uploads are always in the "media" collection
	if field.HasMany {
		return p.populateManyMedia(ctx, value, opts)
	}

	id, ok := value.(string)
	if !ok {
		return value, nil
	}

	return p.fetchDocument(ctx, "media", id, opts)
}

func (p *PopulatorService) populateManyMedia(ctx context.Context, value any, opts *PopulateOptions) (any, error) {
	arr, ok := value.([]any)
	if !ok {
		if strArr, ok := value.([]string); ok {
			arr = make([]any, len(strArr))
			for i, s := range strArr {
				arr[i] = s
			}
		} else {
			return value, nil
		}
	}

	result := make([]any, 0, len(arr))
	for _, item := range arr {
		id, ok := item.(string)
		if !ok {
			result = append(result, item)
			continue
		}
		populated, err := p.fetchDocument(ctx, "media", id, &PopulateOptions{Depth: 0})
		if err != nil {
			result = append(result, item)
			continue
		}
		result = append(result, populated)
	}

	return result, nil
}

func (p *PopulatorService) fetchDocument(ctx context.Context, collection, id string, opts *PopulateOptions) (any, error) {
	doc, err := p.store.FindByID(ctx, collection, id)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, nil
	}

	result := make(map[string]any)
	result["id"] = doc.ID
	result["createdAt"] = doc.CreatedAt
	result["updatedAt"] = doc.UpdatedAt
	for k, v := range doc.Data {
		result[k] = v
	}

	// Recursively populate if depth allows
	if opts.Depth > 1 {
		collConfig := p.getCollectionConfig(collection)
		if collConfig != nil {
			return p.Populate(ctx, result, collConfig.Fields, &PopulateOptions{
				Depth:          opts.Depth - 1,
				Locale:         opts.Locale,
				FallbackLocale: opts.FallbackLocale,
			})
		}
	}

	return result, nil
}

func (p *PopulatorService) populateNested(ctx context.Context, value any, field *config.Field, opts *PopulateOptions) (any, error) {
	switch field.Type {
	case config.FieldTypeArray:
		return p.populateArray(ctx, value, field, opts)
	case config.FieldTypeGroup:
		if data, ok := value.(map[string]any); ok {
			return p.Populate(ctx, data, field.Fields, opts)
		}
	case config.FieldTypeBlocks:
		return p.populateBlocks(ctx, value, field, opts)
	case config.FieldTypeTabs:
		return p.populateTabs(ctx, value, field, opts)
	}
	return value, nil
}

func (p *PopulatorService) populateArray(ctx context.Context, value any, field *config.Field, opts *PopulateOptions) (any, error) {
	arr, ok := value.([]any)
	if !ok {
		return value, nil
	}

	result := make([]any, len(arr))
	for i, item := range arr {
		if itemMap, ok := item.(map[string]any); ok {
			populated, err := p.Populate(ctx, itemMap, field.Fields, opts)
			if err != nil {
				return nil, err
			}
			result[i] = populated
		} else {
			result[i] = item
		}
	}
	return result, nil
}

func (p *PopulatorService) populateBlocks(ctx context.Context, value any, field *config.Field, opts *PopulateOptions) (any, error) {
	arr, ok := value.([]any)
	if !ok {
		return value, nil
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
			populated, err := p.Populate(ctx, itemMap, blockFields, opts)
			if err != nil {
				return nil, err
			}
			result[i] = populated
		} else {
			result[i] = item
		}
	}
	return result, nil
}

func (p *PopulatorService) populateTabs(ctx context.Context, value any, field *config.Field, opts *PopulateOptions) (any, error) {
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
			// Unnamed tab - populate at root level
			populated, err := p.Populate(ctx, data, tab.Fields, opts)
			if err != nil {
				return nil, err
			}
			for k, v := range populated {
				result[k] = v
			}
		} else if tabData, ok := data[tab.Name].(map[string]any); ok {
			populated, err := p.Populate(ctx, tabData, tab.Fields, opts)
			if err != nil {
				return nil, err
			}
			result[tab.Name] = populated
		}
	}

	return result, nil
}

// PopulateDocs populates relationships in multiple documents.
func (p *PopulatorService) PopulateDocs(ctx context.Context, docs []map[string]any, fields []config.Field, opts *PopulateOptions) ([]map[string]any, error) {
	result := make([]map[string]any, len(docs))
	for i, doc := range docs {
		populated, err := p.Populate(ctx, doc, fields, opts)
		if err != nil {
			return nil, err
		}
		result[i] = populated
	}
	return result, nil
}

func (p *PopulatorService) getCollectionConfig(slug string) *config.CollectionConfig {
	if p.cfg == nil {
		return nil
	}
	for i := range p.cfg.Collections {
		if p.cfg.Collections[i].Slug == slug {
			return &p.cfg.Collections[i]
		}
	}
	return nil
}
