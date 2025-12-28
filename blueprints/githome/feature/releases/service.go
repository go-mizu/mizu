package releases

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Service implements the releases API
type Service struct {
	store       Store
	repoStore   repos.Store
	userStore   users.Store
	baseURL     string
	storagePath string
}

// NewService creates a new releases service
func NewService(store Store, repoStore repos.Store, userStore users.Store, baseURL, storagePath string) *Service {
	return &Service{
		store:       store,
		repoStore:   repoStore,
		userStore:   userStore,
		baseURL:     baseURL,
		storagePath: storagePath,
	}
}

// List returns releases for a repository
func (s *Service) List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Release, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	releases, err := s.store.List(ctx, r.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, rel := range releases {
		s.populateURLs(rel, owner, repo)
	}
	return releases, nil
}

// Get retrieves a release by ID
func (s *Service) Get(ctx context.Context, owner, repo string, releaseID int64) (*Release, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	rel, err := s.store.GetByID(ctx, releaseID)
	if err != nil {
		return nil, err
	}
	if rel == nil || rel.RepoID != r.ID {
		return nil, ErrNotFound
	}

	// Load assets
	assets, err := s.store.ListAssets(ctx, releaseID, &ListOpts{PerPage: 100})
	if err != nil {
		return nil, err
	}
	rel.Assets = assets
	for _, a := range rel.Assets {
		s.populateAssetURLs(a, owner, repo, releaseID)
	}

	s.populateURLs(rel, owner, repo)
	return rel, nil
}

// GetLatest retrieves the latest release
func (s *Service) GetLatest(ctx context.Context, owner, repo string) (*Release, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	rel, err := s.store.GetLatest(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	if rel == nil {
		return nil, ErrNotFound
	}

	s.populateURLs(rel, owner, repo)
	return rel, nil
}

// GetByTag retrieves a release by tag
func (s *Service) GetByTag(ctx context.Context, owner, repo, tag string) (*Release, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	rel, err := s.store.GetByTag(ctx, r.ID, tag)
	if err != nil {
		return nil, err
	}
	if rel == nil {
		return nil, ErrNotFound
	}

	s.populateURLs(rel, owner, repo)
	return rel, nil
}

// Create creates a new release
func (s *Service) Create(ctx context.Context, owner, repo string, authorID int64, in *CreateIn) (*Release, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Check if release with tag exists
	existing, err := s.store.GetByTag(ctx, r.ID, in.TagName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrReleaseExists
	}

	author, err := s.userStore.GetByID(ctx, authorID)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, users.ErrNotFound
	}

	targetCommitish := in.TargetCommitish
	if targetCommitish == "" {
		targetCommitish = r.DefaultBranch
	}

	now := time.Now()
	var publishedAt *time.Time
	if !in.Draft {
		publishedAt = &now
	}

	rel := &Release{
		TagName:         in.TagName,
		TargetCommitish: targetCommitish,
		Name:            in.Name,
		Body:            in.Body,
		Draft:           in.Draft,
		Prerelease:      in.Prerelease,
		CreatedAt:       now,
		PublishedAt:     publishedAt,
		Author:          author.ToSimple(),
		Assets:          []*Asset{},
		RepoID:          r.ID,
		AuthorID:        authorID,
	}

	if err := s.store.Create(ctx, rel); err != nil {
		return nil, err
	}

	s.populateURLs(rel, owner, repo)
	return rel, nil
}

// Update updates a release
func (s *Service) Update(ctx context.Context, owner, repo string, releaseID int64, in *UpdateIn) (*Release, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	rel, err := s.store.GetByID(ctx, releaseID)
	if err != nil {
		return nil, err
	}
	if rel == nil || rel.RepoID != r.ID {
		return nil, ErrNotFound
	}

	if err := s.store.Update(ctx, releaseID, in); err != nil {
		return nil, err
	}

	return s.Get(ctx, owner, repo, releaseID)
}

// Delete removes a release
func (s *Service) Delete(ctx context.Context, owner, repo string, releaseID int64) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	rel, err := s.store.GetByID(ctx, releaseID)
	if err != nil {
		return err
	}
	if rel == nil || rel.RepoID != r.ID {
		return ErrNotFound
	}

	return s.store.Delete(ctx, releaseID)
}

// GenerateNotes generates release notes
func (s *Service) GenerateNotes(ctx context.Context, owner, repo string, in *GenerateNotesIn) (*ReleaseNotes, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Would generate notes from commit history between previous and new tag
	return &ReleaseNotes{
		Name: in.TagName,
		Body: fmt.Sprintf("## What's Changed\n\nFull Changelog: %s/compare/%s...%s", s.baseURL, in.PreviousTagName, in.TagName),
	}, nil
}

// ListAssets returns assets for a release
func (s *Service) ListAssets(ctx context.Context, owner, repo string, releaseID int64, opts *ListOpts) ([]*Asset, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	rel, err := s.store.GetByID(ctx, releaseID)
	if err != nil {
		return nil, err
	}
	if rel == nil || rel.RepoID != r.ID {
		return nil, ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	assets, err := s.store.ListAssets(ctx, releaseID, opts)
	if err != nil {
		return nil, err
	}

	for _, a := range assets {
		s.populateAssetURLs(a, owner, repo, releaseID)
	}
	return assets, nil
}

// GetAsset retrieves an asset by ID
func (s *Service) GetAsset(ctx context.Context, owner, repo string, assetID int64) (*Asset, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	a, err := s.store.GetAssetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAssetNotFound
	}

	s.populateAssetURLs(a, owner, repo, a.ReleaseID)
	return a, nil
}

// UploadAsset uploads a new asset
func (s *Service) UploadAsset(ctx context.Context, owner, repo string, releaseID int64, uploaderID int64, name, contentType string, file io.Reader) (*Asset, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	rel, err := s.store.GetByID(ctx, releaseID)
	if err != nil {
		return nil, err
	}
	if rel == nil || rel.RepoID != r.ID {
		return nil, ErrNotFound
	}

	uploader, err := s.userStore.GetByID(ctx, uploaderID)
	if err != nil {
		return nil, err
	}
	if uploader == nil {
		return nil, users.ErrNotFound
	}

	// Save file to storage
	storagePath := filepath.Join(s.storagePath, owner, repo, "releases", fmt.Sprintf("%d", releaseID), name)
	if err := os.MkdirAll(filepath.Dir(storagePath), 0755); err != nil {
		return nil, err
	}

	f, err := os.Create(storagePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	size, err := io.Copy(f, file)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	a := &Asset{
		Name:        name,
		ContentType: contentType,
		Size:        int(size),
		State:       "uploaded",
		CreatedAt:   now,
		UpdatedAt:   now,
		Uploader:    uploader.ToSimple(),
		ReleaseID:   releaseID,
		UploaderID:  uploaderID,
		StoragePath: storagePath,
	}

	if err := s.store.CreateAsset(ctx, a); err != nil {
		return nil, err
	}

	s.populateAssetURLs(a, owner, repo, releaseID)
	return a, nil
}

// UpdateAsset updates an asset
func (s *Service) UpdateAsset(ctx context.Context, owner, repo string, assetID int64, in *UpdateAssetIn) (*Asset, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	a, err := s.store.GetAssetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAssetNotFound
	}

	if err := s.store.UpdateAsset(ctx, assetID, in); err != nil {
		return nil, err
	}

	return s.GetAsset(ctx, owner, repo, assetID)
}

// DeleteAsset removes an asset
func (s *Service) DeleteAsset(ctx context.Context, owner, repo string, assetID int64) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	a, err := s.store.GetAssetByID(ctx, assetID)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrAssetNotFound
	}

	// Delete file from storage
	if a.StoragePath != "" {
		_ = os.Remove(a.StoragePath)
	}

	return s.store.DeleteAsset(ctx, assetID)
}

// DownloadAsset returns the asset file content
func (s *Service) DownloadAsset(ctx context.Context, owner, repo string, assetID int64) (io.ReadCloser, string, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, "", err
	}
	if r == nil {
		return nil, "", repos.ErrNotFound
	}

	a, err := s.store.GetAssetByID(ctx, assetID)
	if err != nil {
		return nil, "", err
	}
	if a == nil {
		return nil, "", ErrAssetNotFound
	}

	f, err := os.Open(a.StoragePath)
	if err != nil {
		return nil, "", err
	}

	// Increment download count
	_ = s.store.IncrementDownloadCount(ctx, assetID)

	return f, a.ContentType, nil
}

// populateURLs fills in the URL fields for a release
func (s *Service) populateURLs(rel *Release, owner, repo string) {
	rel.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Release:%d", rel.ID)))
	rel.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/releases/%d", s.baseURL, owner, repo, rel.ID)
	rel.HTMLURL = fmt.Sprintf("%s/%s/%s/releases/tag/%s", s.baseURL, owner, repo, rel.TagName)
	rel.AssetsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/releases/%d/assets", s.baseURL, owner, repo, rel.ID)
	rel.UploadURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/releases/%d/assets{?name,label}", s.baseURL, owner, repo, rel.ID)
	rel.TarballURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/tarball/%s", s.baseURL, owner, repo, rel.TagName)
	rel.ZipballURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/zipball/%s", s.baseURL, owner, repo, rel.TagName)
}

// populateAssetURLs fills in the URL fields for an asset
func (s *Service) populateAssetURLs(a *Asset, owner, repo string, releaseID int64) {
	a.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("ReleaseAsset:%d", a.ID)))
	a.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/releases/assets/%d", s.baseURL, owner, repo, a.ID)
	a.BrowserDownloadURL = fmt.Sprintf("%s/%s/%s/releases/download/%d/%s", s.baseURL, owner, repo, releaseID, a.Name)
}
