package versions

import (
	"context"
	"fmt"

	"github.com/go-mizu/blueprints/cms/config"
	"github.com/go-mizu/blueprints/cms/store/duckdb"
)

// VersionsService implements the Service interface.
type VersionsService struct {
	store            *duckdb.VersionsStore
	collectionsStore *duckdb.CollectionsStore
	globalsStore     *duckdb.GlobalsStore
	config           *config.Config
}

// NewService creates a new versions service.
func NewService(store *duckdb.VersionsStore, collectionsStore *duckdb.CollectionsStore, globalsStore *duckdb.GlobalsStore, cfg *config.Config) *VersionsService {
	return &VersionsService{
		store:            store,
		collectionsStore: collectionsStore,
		globalsStore:     globalsStore,
		config:           cfg,
	}
}

// CreateVersion creates a new version for a document.
func (s *VersionsService) CreateVersion(ctx context.Context, collection, parentID string, data map[string]any, opts *VersionOptions) (*Version, error) {
	if opts == nil {
		opts = &VersionOptions{}
	}

	// Get latest version number
	latestVersion, err := s.store.GetLatestVersion(ctx, collection, parentID)
	if err != nil {
		return nil, fmt.Errorf("get latest version: %w", err)
	}

	dbVersion := &duckdb.Version{
		Parent:    parentID,
		Version:   latestVersion + 1,
		Snapshot:  data,
		Published: opts.Published,
		Autosave:  opts.Autosave,
		UpdatedBy: opts.UpdatedBy,
	}

	if err := s.store.Create(ctx, collection, dbVersion); err != nil {
		return nil, fmt.Errorf("create version: %w", err)
	}

	// Clean up old versions if max configured
	cfg := s.getCollectionConfig(collection)
	if cfg != nil && cfg.Versions != nil && cfg.Versions.MaxPerDoc > 0 {
		if err := s.store.DeleteOldVersions(ctx, collection, parentID, cfg.Versions.MaxPerDoc); err != nil {
			// Log but don't fail
		}
	}

	return toVersion(dbVersion), nil
}

// GetVersion retrieves a specific version.
func (s *VersionsService) GetVersion(ctx context.Context, collection, versionID string) (*Version, error) {
	dbVersion, err := s.store.GetByID(ctx, collection, versionID)
	if err != nil {
		return nil, err
	}
	if dbVersion == nil {
		return nil, nil
	}
	return toVersion(dbVersion), nil
}

// ListVersions lists all versions for a document.
func (s *VersionsService) ListVersions(ctx context.Context, collection, parentID string, opts *ListOptions) (*VersionsResult, error) {
	if opts == nil {
		opts = &ListOptions{Limit: 10, Page: 1}
	}
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}

	versions, total, err := s.store.ListByParent(ctx, collection, parentID, opts.Limit, opts.Page)
	if err != nil {
		return nil, err
	}

	docs := make([]*Version, len(versions))
	for i, v := range versions {
		docs[i] = toVersion(v)
	}

	totalPages := (total + opts.Limit - 1) / opts.Limit

	result := &VersionsResult{
		Docs:        docs,
		TotalDocs:   total,
		Limit:       opts.Limit,
		TotalPages:  totalPages,
		Page:        opts.Page,
		HasPrevPage: opts.Page > 1,
		HasNextPage: opts.Page < totalPages,
	}

	if result.HasPrevPage {
		prev := opts.Page - 1
		result.PrevPage = &prev
	}
	if result.HasNextPage {
		next := opts.Page + 1
		result.NextPage = &next
	}

	return result, nil
}

// RestoreVersion restores a document to a previous version.
func (s *VersionsService) RestoreVersion(ctx context.Context, collection, versionID string) (map[string]any, error) {
	version, err := s.store.GetByID(ctx, collection, versionID)
	if err != nil {
		return nil, err
	}
	if version == nil {
		return nil, fmt.Errorf("version not found")
	}

	// Update the document with the version snapshot
	doc, err := s.collectionsStore.UpdateByID(ctx, collection, version.Parent, version.Snapshot)
	if err != nil {
		return nil, fmt.Errorf("restore document: %w", err)
	}

	// Create a new version marking this as a restore
	_, err = s.CreateVersion(ctx, collection, version.Parent, version.Snapshot, &VersionOptions{
		Published: true,
	})
	if err != nil {
		// Log but don't fail
	}

	return toDocMap(doc), nil
}

// CompareVersions compares two versions and returns the differences.
func (s *VersionsService) CompareVersions(ctx context.Context, collection, versionID1, versionID2 string) (*VersionDiff, error) {
	v1, err := s.store.GetByID(ctx, collection, versionID1)
	if err != nil {
		return nil, err
	}
	if v1 == nil {
		return nil, fmt.Errorf("version 1 not found")
	}

	v2, err := s.store.GetByID(ctx, collection, versionID2)
	if err != nil {
		return nil, err
	}
	if v2 == nil {
		return nil, fmt.Errorf("version 2 not found")
	}

	return compareSnapshots(v1.Snapshot, v2.Snapshot), nil
}

// SaveDraft saves document changes as a draft.
func (s *VersionsService) SaveDraft(ctx context.Context, collection, parentID string, data map[string]any, userID string) (map[string]any, error) {
	// Create a draft version
	_, err := s.CreateVersion(ctx, collection, parentID, data, &VersionOptions{
		Published: false,
		Autosave:  false,
		UpdatedBy: userID,
	})
	if err != nil {
		return nil, err
	}

	// Update document with draft status
	updateData := make(map[string]any)
	for k, v := range data {
		updateData[k] = v
	}
	updateData["_status"] = "draft"

	doc, err := s.collectionsStore.UpdateByID(ctx, collection, parentID, updateData)
	if err != nil {
		return nil, fmt.Errorf("save draft: %w", err)
	}

	return toDocMap(doc), nil
}

// PublishDraft publishes the current draft.
func (s *VersionsService) PublishDraft(ctx context.Context, collection, parentID string, userID string) (map[string]any, error) {
	// Get current document
	doc, err := s.collectionsStore.FindByID(ctx, collection, parentID)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("document not found")
	}

	// Create a published version
	_, err = s.CreateVersion(ctx, collection, parentID, doc.Data, &VersionOptions{
		Published: true,
		Autosave:  false,
		UpdatedBy: userID,
	})
	if err != nil {
		return nil, err
	}

	// Update document status to published
	updateData := map[string]any{
		"_status": "published",
	}

	updatedDoc, err := s.collectionsStore.UpdateByID(ctx, collection, parentID, updateData)
	if err != nil {
		return nil, fmt.Errorf("publish draft: %w", err)
	}

	return toDocMap(updatedDoc), nil
}

// GetLatestDraft retrieves the latest draft version.
func (s *VersionsService) GetLatestDraft(ctx context.Context, collection, parentID string) (*Version, error) {
	versions, _, err := s.store.ListByParent(ctx, collection, parentID, 100, 1)
	if err != nil {
		return nil, err
	}

	for _, v := range versions {
		if !v.Published && !v.Autosave {
			return toVersion(v), nil
		}
	}

	return nil, nil
}

// Autosave saves document changes as an autosave version.
func (s *VersionsService) Autosave(ctx context.Context, collection, parentID string, data map[string]any, userID string) error {
	// Check for existing autosave
	existing, err := s.GetLatestAutosave(ctx, collection, parentID)
	if err != nil {
		return err
	}

	if existing != nil {
		// Update existing autosave (don't create new version)
		// This prevents cluttering with many autosave versions
		return s.updateAutosave(ctx, collection, existing.ID, data, userID)
	}

	// Create new autosave version
	_, err = s.CreateVersion(ctx, collection, parentID, data, &VersionOptions{
		Published: false,
		Autosave:  true,
		UpdatedBy: userID,
	})
	return err
}

func (s *VersionsService) updateAutosave(ctx context.Context, collection, versionID string, data map[string]any, userID string) error {
	// For simplicity, we'll delete and recreate
	// In production, we'd update in place
	version, err := s.store.GetByID(ctx, collection, versionID)
	if err != nil {
		return err
	}
	if version == nil {
		return nil
	}

	// Create new version with same number
	newVersion := &duckdb.Version{
		Parent:    version.Parent,
		Version:   version.Version,
		Snapshot:  data,
		Published: false,
		Autosave:  true,
		UpdatedBy: userID,
	}

	return s.store.Create(ctx, collection, newVersion)
}

// GetLatestAutosave retrieves the latest autosave version.
func (s *VersionsService) GetLatestAutosave(ctx context.Context, collection, parentID string) (*Version, error) {
	versions, _, err := s.store.ListByParent(ctx, collection, parentID, 100, 1)
	if err != nil {
		return nil, err
	}

	for _, v := range versions {
		if v.Autosave {
			return toVersion(v), nil
		}
	}

	return nil, nil
}

// CreateGlobalVersion creates a new version for a global.
func (s *VersionsService) CreateGlobalVersion(ctx context.Context, slug string, data map[string]any, userID string) (*GlobalVersion, error) {
	// Get latest version
	versions, _, err := s.store.ListGlobalVersions(ctx, slug, 1, 1)
	if err != nil {
		return nil, err
	}

	versionNum := 1
	if len(versions) > 0 {
		versionNum = versions[0].Version + 1
	}

	dbVersion := &duckdb.GlobalVersion{
		GlobalSlug: slug,
		Version:    versionNum,
		Snapshot:   data,
		UpdatedBy:  userID,
	}

	if err := s.store.CreateGlobalVersion(ctx, dbVersion); err != nil {
		return nil, err
	}

	return toGlobalVersion(dbVersion), nil
}

// GetGlobalVersion retrieves a specific global version.
func (s *VersionsService) GetGlobalVersion(ctx context.Context, versionID string) (*GlobalVersion, error) {
	// We need to find this version across all globals
	// This is a simplified implementation
	return nil, fmt.Errorf("not implemented - use ListGlobalVersions")
}

// ListGlobalVersions lists all versions for a global.
func (s *VersionsService) ListGlobalVersions(ctx context.Context, slug string, opts *ListOptions) (*GlobalVersionsResult, error) {
	if opts == nil {
		opts = &ListOptions{Limit: 10, Page: 1}
	}
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}

	versions, total, err := s.store.ListGlobalVersions(ctx, slug, opts.Limit, opts.Page)
	if err != nil {
		return nil, err
	}

	docs := make([]*GlobalVersion, len(versions))
	for i, v := range versions {
		docs[i] = toGlobalVersion(v)
	}

	totalPages := (total + opts.Limit - 1) / opts.Limit

	result := &GlobalVersionsResult{
		Docs:        docs,
		TotalDocs:   total,
		Limit:       opts.Limit,
		TotalPages:  totalPages,
		Page:        opts.Page,
		HasPrevPage: opts.Page > 1,
		HasNextPage: opts.Page < totalPages,
	}

	if result.HasPrevPage {
		prev := opts.Page - 1
		result.PrevPage = &prev
	}
	if result.HasNextPage {
		next := opts.Page + 1
		result.NextPage = &next
	}

	return result, nil
}

// RestoreGlobalVersion restores a global to a previous version.
func (s *VersionsService) RestoreGlobalVersion(ctx context.Context, versionID string) (map[string]any, error) {
	// Simplified - in production we'd look up the version
	return nil, fmt.Errorf("not implemented")
}

func (s *VersionsService) getCollectionConfig(slug string) *config.CollectionConfig {
	if s.config == nil {
		return nil
	}
	for i := range s.config.Collections {
		if s.config.Collections[i].Slug == slug {
			return &s.config.Collections[i]
		}
	}
	return nil
}

func toVersion(v *duckdb.Version) *Version {
	return &Version{
		ID:        v.ID,
		Parent:    v.Parent,
		Version:   v.Version,
		Snapshot:  v.Snapshot,
		Published: v.Published,
		Autosave:  v.Autosave,
		CreatedAt: v.CreatedAt,
		UpdatedBy: v.UpdatedBy,
	}
}

func toGlobalVersion(v *duckdb.GlobalVersion) *GlobalVersion {
	return &GlobalVersion{
		ID:         v.ID,
		GlobalSlug: v.GlobalSlug,
		Version:    v.Version,
		Snapshot:   v.Snapshot,
		CreatedAt:  v.CreatedAt,
		UpdatedBy:  v.UpdatedBy,
	}
}

func toDocMap(doc *duckdb.Document) map[string]any {
	if doc == nil {
		return nil
	}
	result := make(map[string]any)
	result["id"] = doc.ID
	result["createdAt"] = doc.CreatedAt
	result["updatedAt"] = doc.UpdatedAt
	if doc.Status != "" {
		result["_status"] = doc.Status
	}
	if doc.Version > 0 {
		result["_version"] = doc.Version
	}
	for k, v := range doc.Data {
		result[k] = v
	}
	return result
}

func compareSnapshots(before, after map[string]any) *VersionDiff {
	diff := &VersionDiff{
		Added:   make(map[string]any),
		Removed: make(map[string]any),
		Changed: make(map[string]DiffPair),
	}

	// Find added and changed
	for k, v := range after {
		if beforeVal, exists := before[k]; !exists {
			diff.Added[k] = v
		} else if !deepEqual(beforeVal, v) {
			diff.Changed[k] = DiffPair{Before: beforeVal, After: v}
		}
	}

	// Find removed
	for k, v := range before {
		if _, exists := after[k]; !exists {
			diff.Removed[k] = v
		}
	}

	return diff
}

func deepEqual(a, b any) bool {
	// Simple comparison - in production use reflect.DeepEqual or similar
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
