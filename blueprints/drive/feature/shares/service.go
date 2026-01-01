package shares

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/drive/pkg/password"
	"github.com/go-mizu/blueprints/drive/pkg/ulid"
	"github.com/go-mizu/blueprints/drive/store/duckdb"
)

var (
	ErrNotFound     = errors.New("share not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrExpired      = errors.New("share expired")
	ErrDownloadLimit = errors.New("download limit reached")
)

// Service implements the shares API.
type Service struct {
	store *duckdb.Store
}

// NewService creates a new shares service.
func NewService(store *duckdb.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, ownerID, resourceID, resourceType, sharedWithID, permission string) (*Share, error) {
	now := time.Now()
	dbShare := &duckdb.Share{
		ID:           ulid.New(),
		ResourceType: resourceType,
		ResourceID:   resourceID,
		OwnerID:      ownerID,
		SharedWithID: sql.NullString{String: sharedWithID, Valid: sharedWithID != ""},
		Permission:   permission,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.CreateShare(ctx, dbShare); err != nil {
		return nil, err
	}

	return dbShareToShare(dbShare), nil
}

func (s *Service) CreateLink(ctx context.Context, ownerID, resourceID, resourceType string, in *CreateLinkIn) (*Share, error) {
	token := generateToken()
	now := time.Now()

	dbShare := &duckdb.Share{
		ID:           ulid.New(),
		ResourceType: resourceType,
		ResourceID:   resourceID,
		OwnerID:      ownerID,
		Permission:   in.Permission,
		LinkToken:    sql.NullString{String: token, Valid: true},
		PreventDownload: in.PreventDownload,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if in.Password != "" {
		hash, err := password.Hash(in.Password)
		if err != nil {
			return nil, err
		}
		dbShare.LinkPasswordHash = sql.NullString{String: hash, Valid: true}
	}

	if !in.ExpiresAt.IsZero() {
		dbShare.ExpiresAt = sql.NullTime{Time: in.ExpiresAt, Valid: true}
	}

	if in.DownloadLimit > 0 {
		dbShare.DownloadLimit = sql.NullInt64{Int64: in.DownloadLimit, Valid: true}
	}

	if err := s.store.CreateShare(ctx, dbShare); err != nil {
		return nil, err
	}

	return dbShareToShare(dbShare), nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Share, error) {
	dbShare, err := s.store.GetShareByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dbShare == nil {
		return nil, ErrNotFound
	}
	return dbShareToShare(dbShare), nil
}

func (s *Service) GetByToken(ctx context.Context, token string) (*Share, error) {
	dbShare, err := s.store.GetShareByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if dbShare == nil {
		return nil, ErrNotFound
	}

	// Check if expired
	if dbShare.ExpiresAt.Valid && time.Now().After(dbShare.ExpiresAt.Time) {
		return nil, ErrExpired
	}

	// Check download limit
	if dbShare.DownloadLimit.Valid && int64(dbShare.DownloadCount) >= dbShare.DownloadLimit.Int64 {
		return nil, ErrDownloadLimit
	}

	return dbShareToShare(dbShare), nil
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Share, error) {
	dbShare, err := s.store.GetShareByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dbShare == nil {
		return nil, ErrNotFound
	}

	if in.Permission != nil {
		dbShare.Permission = *in.Permission
	}
	if in.ExpiresAt != nil {
		dbShare.ExpiresAt = sql.NullTime{Time: *in.ExpiresAt, Valid: !in.ExpiresAt.IsZero()}
	}
	if in.DownloadLimit != nil {
		dbShare.DownloadLimit = sql.NullInt64{Int64: *in.DownloadLimit, Valid: *in.DownloadLimit > 0}
	}
	if in.PreventDownload != nil {
		dbShare.PreventDownload = *in.PreventDownload
	}
	dbShare.UpdatedAt = time.Now()

	if err := s.store.UpdateShare(ctx, dbShare); err != nil {
		return nil, err
	}

	return dbShareToShare(dbShare), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.DeleteShare(ctx, id)
}

func (s *Service) ListByOwner(ctx context.Context, ownerID string) ([]*Share, error) {
	dbShares, err := s.store.ListSharesByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	return dbSharesToShares(dbShares), nil
}

func (s *Service) ListSharedWithMe(ctx context.Context, userID string) ([]*Share, error) {
	dbShares, err := s.store.ListSharesWithUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dbSharesToShares(dbShares), nil
}

func (s *Service) ListForResource(ctx context.Context, resourceType, resourceID string) ([]*Share, error) {
	dbShares, err := s.store.ListSharesForResource(ctx, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	return dbSharesToShares(dbShares), nil
}

func (s *Service) CheckAccess(ctx context.Context, userID, resourceType, resourceID string) (*Share, error) {
	dbShare, err := s.store.GetShareForUserAndResource(ctx, userID, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	if dbShare == nil {
		return nil, nil
	}
	return dbShareToShare(dbShare), nil
}

func (s *Service) IncrementDownload(ctx context.Context, id string) error {
	return s.store.IncrementDownloadCount(ctx, id)
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func dbShareToShare(sh *duckdb.Share) *Share {
	share := &Share{
		ID:              sh.ID,
		ResourceType:   sh.ResourceType,
		ResourceID:     sh.ResourceID,
		OwnerID:        sh.OwnerID,
		Permission:     sh.Permission,
		DownloadCount:  sh.DownloadCount,
		PreventDownload: sh.PreventDownload,
		CreatedAt:      sh.CreatedAt,
		UpdatedAt:      sh.UpdatedAt,
	}
	if sh.SharedWithID.Valid {
		share.SharedWithID = sh.SharedWithID.String
	}
	if sh.LinkToken.Valid {
		share.LinkToken = sh.LinkToken.String
	}
	if sh.LinkPasswordHash.Valid {
		share.LinkPasswordHash = sh.LinkPasswordHash.String
	}
	if sh.ExpiresAt.Valid {
		share.ExpiresAt = sh.ExpiresAt.Time
	}
	if sh.DownloadLimit.Valid {
		share.DownloadLimit = sh.DownloadLimit.Int64
	}
	return share
}

func dbSharesToShares(dbShares []*duckdb.Share) []*Share {
	shares := make([]*Share, len(dbShares))
	for i, sh := range dbShares {
		shares[i] = dbShareToShare(sh)
	}
	return shares
}
