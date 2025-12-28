package releases

import (
	"context"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/pkg/ulid"
)

// Service implements the releases API
type Service struct {
	store Store
}

// NewService creates a new releases service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new release
func (s *Service) Create(ctx context.Context, repoID, authorID string, in *CreateIn) (*Release, error) {
	if in.TagName == "" {
		return nil, ErrMissingTag
	}

	// Check if tag already exists
	existing, _ := s.store.GetByTag(ctx, repoID, in.TagName)
	if existing != nil {
		return nil, ErrExists
	}

	targetCommitish := in.TargetCommitish
	if targetCommitish == "" {
		targetCommitish = "main"
	}

	now := time.Now()
	release := &Release{
		ID:              ulid.New(),
		RepoID:          repoID,
		TagName:         strings.TrimSpace(in.TagName),
		TargetCommitish: targetCommitish,
		Name:            in.Name,
		Body:            in.Body,
		IsDraft:         in.IsDraft,
		IsPrerelease:    in.IsPrerelease,
		AuthorID:        authorID,
		CreatedAt:       now,
	}

	// If not a draft, set published time
	if !in.IsDraft {
		release.PublishedAt = &now
	}

	if err := s.store.Create(ctx, release); err != nil {
		return nil, err
	}

	return release, nil
}

// GetByID retrieves a release by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Release, error) {
	release, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, ErrNotFound
	}

	// Load assets
	release.Assets, _ = s.store.ListAssets(ctx, id)

	return release, nil
}

// GetByTag retrieves a release by repository ID and tag name
func (s *Service) GetByTag(ctx context.Context, repoID, tagName string) (*Release, error) {
	release, err := s.store.GetByTag(ctx, repoID, tagName)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, ErrNotFound
	}

	// Load assets
	release.Assets, _ = s.store.ListAssets(ctx, release.ID)

	return release, nil
}

// GetLatest retrieves the latest published release
func (s *Service) GetLatest(ctx context.Context, repoID string) (*Release, error) {
	release, err := s.store.GetLatest(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, ErrNotFound
	}

	// Load assets
	release.Assets, _ = s.store.ListAssets(ctx, release.ID)

	return release, nil
}

// Update updates a release
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Release, error) {
	release, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, ErrNotFound
	}

	if in.TagName != nil {
		newTag := strings.TrimSpace(*in.TagName)
		// Check if new tag conflicts
		if newTag != release.TagName {
			existing, _ := s.store.GetByTag(ctx, release.RepoID, newTag)
			if existing != nil {
				return nil, ErrExists
			}
		}
		release.TagName = newTag
	}
	if in.TargetCommitish != nil {
		release.TargetCommitish = *in.TargetCommitish
	}
	if in.Name != nil {
		release.Name = *in.Name
	}
	if in.Body != nil {
		release.Body = *in.Body
	}
	if in.IsDraft != nil {
		// If going from draft to published
		if release.IsDraft && !*in.IsDraft && release.PublishedAt == nil {
			now := time.Now()
			release.PublishedAt = &now
		}
		release.IsDraft = *in.IsDraft
	}
	if in.IsPrerelease != nil {
		release.IsPrerelease = *in.IsPrerelease
	}

	if err := s.store.Update(ctx, release); err != nil {
		return nil, err
	}

	return release, nil
}

// Delete deletes a release
func (s *Service) Delete(ctx context.Context, id string) error {
	release, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if release == nil {
		return ErrNotFound
	}
	return s.store.Delete(ctx, id)
}

// List lists releases for a repository
func (s *Service) List(ctx context.Context, repoID string, opts *ListOpts) ([]*Release, error) {
	limit := 30
	offset := 0
	if opts != nil {
		if opts.Limit > 0 && opts.Limit <= 100 {
			limit = opts.Limit
		}
		if opts.Offset >= 0 {
			offset = opts.Offset
		}
	}

	releases, err := s.store.List(ctx, repoID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Load assets for each release
	for _, r := range releases {
		r.Assets, _ = s.store.ListAssets(ctx, r.ID)
	}

	return releases, nil
}

// Publish publishes a draft release
func (s *Service) Publish(ctx context.Context, id string) (*Release, error) {
	release, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, ErrNotFound
	}

	if !release.IsDraft {
		return nil, ErrAlreadyPublished
	}

	now := time.Now()
	release.IsDraft = false
	release.PublishedAt = &now

	if err := s.store.Update(ctx, release); err != nil {
		return nil, err
	}

	return release, nil
}

// UploadAsset uploads an asset to a release
func (s *Service) UploadAsset(ctx context.Context, releaseID, uploaderID string, in *UploadAssetIn) (*Asset, error) {
	if in.Name == "" {
		return nil, ErrInvalidInput
	}

	now := time.Now()
	asset := &Asset{
		ID:          ulid.New(),
		ReleaseID:   releaseID,
		Name:        in.Name,
		Label:       in.Label,
		ContentType: in.ContentType,
		SizeBytes:   in.SizeBytes,
		UploaderID:  uploaderID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.CreateAsset(ctx, asset); err != nil {
		return nil, err
	}

	return asset, nil
}

// GetAsset retrieves an asset by ID
func (s *Service) GetAsset(ctx context.Context, id string) (*Asset, error) {
	asset, err := s.store.GetAsset(ctx, id)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, ErrAssetNotFound
	}
	return asset, nil
}

// UpdateAsset updates an asset's name and label
func (s *Service) UpdateAsset(ctx context.Context, id string, name, label string) (*Asset, error) {
	asset, err := s.store.GetAsset(ctx, id)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, ErrAssetNotFound
	}

	if name != "" {
		asset.Name = name
	}
	asset.Label = label
	asset.UpdatedAt = time.Now()

	if err := s.store.UpdateAsset(ctx, asset); err != nil {
		return nil, err
	}

	return asset, nil
}

// DeleteAsset deletes an asset
func (s *Service) DeleteAsset(ctx context.Context, id string) error {
	asset, err := s.store.GetAsset(ctx, id)
	if err != nil {
		return err
	}
	if asset == nil {
		return ErrAssetNotFound
	}
	return s.store.DeleteAsset(ctx, id)
}

// ListAssets lists assets for a release
func (s *Service) ListAssets(ctx context.Context, releaseID string) ([]*Asset, error) {
	return s.store.ListAssets(ctx, releaseID)
}

// IncrementDownload increments the download count for an asset
func (s *Service) IncrementDownload(ctx context.Context, assetID string) error {
	return s.store.IncrementDownload(ctx, assetID)
}
