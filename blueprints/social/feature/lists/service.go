package lists

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/social/feature/accounts"
	"github.com/go-mizu/blueprints/social/pkg/ulid"
)

// Service implements the lists API.
type Service struct {
	store    Store
	accounts accounts.API
}

// NewService creates a new lists service.
func NewService(store Store, accountsSvc accounts.API) *Service {
	return &Service{
		store:    store,
		accounts: accountsSvc,
	}
}

// Create creates a new list.
func (s *Service) Create(ctx context.Context, accountID string, in *CreateIn) (*List, error) {
	now := time.Now()
	list := &List{
		ID:        ulid.New(),
		AccountID: accountID,
		Title:     in.Title,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Insert(ctx, list); err != nil {
		return nil, err
	}

	return list, nil
}

// GetByID retrieves a list by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*List, error) {
	list, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	count, err := s.store.GetMemberCount(ctx, id)
	if err == nil {
		list.MemberCount = count
	}

	return list, nil
}

// GetByAccount retrieves lists owned by an account.
func (s *Service) GetByAccount(ctx context.Context, accountID string) ([]*List, error) {
	lists, err := s.store.GetByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	for _, list := range lists {
		count, err := s.store.GetMemberCount(ctx, list.ID)
		if err == nil {
			list.MemberCount = count
		}
	}

	return lists, nil
}

// Update updates a list.
func (s *Service) Update(ctx context.Context, accountID, listID string, in *UpdateIn) (*List, error) {
	list, err := s.store.GetByID(ctx, listID)
	if err != nil {
		return nil, ErrNotFound
	}

	if list.AccountID != accountID {
		return nil, ErrUnauthorized
	}

	if err := s.store.Update(ctx, listID, in); err != nil {
		return nil, err
	}

	return s.GetByID(ctx, listID)
}

// Delete deletes a list.
func (s *Service) Delete(ctx context.Context, accountID, listID string) error {
	list, err := s.store.GetByID(ctx, listID)
	if err != nil {
		return ErrNotFound
	}

	if list.AccountID != accountID {
		return ErrUnauthorized
	}

	return s.store.Delete(ctx, listID)
}

// GetMembers returns members of a list.
func (s *Service) GetMembers(ctx context.Context, listID string, limit, offset int) ([]*ListMember, error) {
	members, err := s.store.GetMembers(ctx, listID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Populate accounts
	if s.accounts != nil {
		for _, m := range members {
			acc, err := s.accounts.GetByID(ctx, m.AccountID)
			if err == nil {
				m.Account = acc
			}
		}
	}

	return members, nil
}

// AddMember adds a member to a list.
func (s *Service) AddMember(ctx context.Context, accountID, listID, memberID string) error {
	list, err := s.store.GetByID(ctx, listID)
	if err != nil {
		return ErrNotFound
	}

	if list.AccountID != accountID {
		return ErrUnauthorized
	}

	exists, err := s.store.ExistsMember(ctx, listID, memberID)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyMember
	}

	member := &ListMember{
		ListID:    listID,
		AccountID: memberID,
		CreatedAt: time.Now(),
	}

	return s.store.InsertMember(ctx, member)
}

// RemoveMember removes a member from a list.
func (s *Service) RemoveMember(ctx context.Context, accountID, listID, memberID string) error {
	list, err := s.store.GetByID(ctx, listID)
	if err != nil {
		return ErrNotFound
	}

	if list.AccountID != accountID {
		return ErrUnauthorized
	}

	exists, err := s.store.ExistsMember(ctx, listID, memberID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotMember
	}

	return s.store.DeleteMember(ctx, listID, memberID)
}

// GetListsContaining returns lists that contain a specific account.
func (s *Service) GetListsContaining(ctx context.Context, accountID, targetID string) ([]*List, error) {
	lists, err := s.store.GetListsContaining(ctx, targetID)
	if err != nil {
		return nil, err
	}

	// Filter to only lists owned by accountID
	owned := make([]*List, 0)
	for _, list := range lists {
		if list.AccountID == accountID {
			owned = append(owned, list)
		}
	}

	return owned, nil
}
