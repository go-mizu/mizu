package globals

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/cms/config"
	"github.com/go-mizu/blueprints/cms/store/duckdb"
)

var ErrNotFound = errors.New("global not found")

// Service implements the globals API.
type Service struct {
	store    *duckdb.GlobalsStore
	versions *duckdb.VersionsStore
	globals  map[string]*config.GlobalConfig
}

// NewService creates a new globals service.
func NewService(store *duckdb.GlobalsStore, versions *duckdb.VersionsStore, globals []config.GlobalConfig) *Service {
	globalMap := make(map[string]*config.GlobalConfig)
	for i := range globals {
		globalMap[globals[i].Slug] = &globals[i]
	}
	return &Service{
		store:    store,
		versions: versions,
		globals:  globalMap,
	}
}

// Get retrieves a global by slug.
func (s *Service) Get(ctx context.Context, slug string) (map[string]any, error) {
	cfg := s.globals[slug]
	if cfg == nil {
		return nil, ErrNotFound
	}

	g, err := s.store.Get(ctx, slug)
	if err != nil {
		return nil, err
	}

	if g == nil {
		// Return empty structure with defaults
		return s.getDefaultData(cfg), nil
	}

	// Add metadata
	result := g.Data
	if result == nil {
		result = make(map[string]any)
	}
	result["id"] = g.ID
	result["createdAt"] = g.CreatedAt.Format(time.RFC3339)
	result["updatedAt"] = g.UpdatedAt.Format(time.RFC3339)

	return result, nil
}

// Update updates a global.
func (s *Service) Update(ctx context.Context, slug string, data map[string]any) (map[string]any, error) {
	cfg := s.globals[slug]
	if cfg == nil {
		return nil, ErrNotFound
	}

	// Remove metadata from data
	delete(data, "id")
	delete(data, "createdAt")
	delete(data, "updatedAt")

	g, err := s.store.Update(ctx, slug, data)
	if err != nil {
		return nil, err
	}

	// Create version if versioning enabled
	if cfg.Versions != nil {
		latestVersion := 0
		versions, _, _ := s.versions.ListGlobalVersions(ctx, slug, 1, 1)
		if len(versions) > 0 {
			latestVersion = versions[0].Version
		}

		version := &duckdb.GlobalVersion{
			GlobalSlug: slug,
			Version:    latestVersion + 1,
			Snapshot:   data,
		}
		s.versions.CreateGlobalVersion(ctx, version)
	}

	// Add metadata
	result := g.Data
	if result == nil {
		result = make(map[string]any)
	}
	result["id"] = g.ID
	result["createdAt"] = g.CreatedAt.Format(time.RFC3339)
	result["updatedAt"] = g.UpdatedAt.Format(time.RFC3339)

	return result, nil
}

// getDefaultData returns default data for a global based on its field definitions.
func (s *Service) getDefaultData(cfg *config.GlobalConfig) map[string]any {
	result := make(map[string]any)
	for _, field := range cfg.Fields {
		if field.DefaultValue != nil {
			result[field.Name] = field.DefaultValue
		}
	}
	return result
}
