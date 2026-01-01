package activity

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/drive/pkg/ulid"
	"github.com/go-mizu/blueprints/drive/store/duckdb"
)

// Service implements the activity API.
type Service struct {
	store *duckdb.Store
}

// NewService creates a new activity service.
func NewService(store *duckdb.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Log(ctx context.Context, userID string, in *LogIn) (*Activity, error) {
	now := time.Now()
	dbActivity := &duckdb.Activity{
		ID:           ulid.New(),
		UserID:       userID,
		Action:       in.Action,
		ResourceType: in.ResourceType,
		ResourceID:   in.ResourceID,
		ResourceName: sql.NullString{String: in.ResourceName, Valid: in.ResourceName != ""},
		Details:      sql.NullString{String: in.Details, Valid: in.Details != ""},
		IPAddress:    sql.NullString{String: in.IPAddress, Valid: in.IPAddress != ""},
		UserAgent:    sql.NullString{String: in.UserAgent, Valid: in.UserAgent != ""},
		CreatedAt:    now,
	}

	if err := s.store.CreateActivity(ctx, dbActivity); err != nil {
		return nil, err
	}

	return dbActivityToActivity(dbActivity), nil
}

func (s *Service) ListByUser(ctx context.Context, userID string, limit int) ([]*Activity, error) {
	if limit <= 0 {
		limit = 50
	}
	dbActivities, err := s.store.ListActivitiesByUser(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	return dbActivitiesToActivities(dbActivities), nil
}

func (s *Service) ListForResource(ctx context.Context, resourceType, resourceID string, limit int) ([]*Activity, error) {
	if limit <= 0 {
		limit = 50
	}
	dbActivities, err := s.store.ListActivitiesForResource(ctx, resourceType, resourceID, limit)
	if err != nil {
		return nil, err
	}
	return dbActivitiesToActivities(dbActivities), nil
}

func (s *Service) ListRecent(ctx context.Context, limit int) ([]*Activity, error) {
	if limit <= 0 {
		limit = 50
	}
	dbActivities, err := s.store.ListRecentActivities(ctx, limit)
	if err != nil {
		return nil, err
	}
	return dbActivitiesToActivities(dbActivities), nil
}

func dbActivityToActivity(a *duckdb.Activity) *Activity {
	return &Activity{
		ID:           a.ID,
		UserID:       a.UserID,
		Action:       a.Action,
		ResourceType: a.ResourceType,
		ResourceID:   a.ResourceID,
		ResourceName: a.ResourceName.String,
		Details:      a.Details.String,
		IPAddress:    a.IPAddress.String,
		UserAgent:    a.UserAgent.String,
		CreatedAt:    a.CreatedAt,
	}
}

func dbActivitiesToActivities(dbActivities []*duckdb.Activity) []*Activity {
	activities := make([]*Activity, len(dbActivities))
	for i, a := range dbActivities {
		activities[i] = dbActivityToActivity(a)
	}
	return activities
}
