package friendcode

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// CodeExpiration is the default expiration duration for friend codes.
	CodeExpiration = 24 * time.Hour
	// CodeLength is the length of the generated code.
	CodeLength = 12
)

// Service implements the friend code API.
type Service struct {
	store        Store
	userStore    UserStore
	contactStore ContactStore
	baseURL      string
}

// NewService creates a new friend code service.
func NewService(store Store, userStore UserStore, contactStore ContactStore, baseURL string) *Service {
	return &Service{
		store:        store,
		userStore:    userStore,
		contactStore: contactStore,
		baseURL:      baseURL,
	}
}

// Generate creates or returns existing valid friend code for user.
func (s *Service) Generate(ctx context.Context, userID string) (*FriendCodeResponse, error) {
	// Check for existing valid code
	existing, err := s.store.GetByUserID(ctx, userID)
	if err == nil && existing != nil && existing.ExpiresAt.After(time.Now()) {
		return &FriendCodeResponse{
			ID:        existing.ID,
			Code:      existing.Code,
			ExpiresAt: existing.ExpiresAt,
			QRData:    s.generateQRData(existing.Code),
		}, nil
	}

	// Delete any expired codes
	if existing != nil {
		s.store.Delete(ctx, existing.ID)
	}

	// Generate new code
	code, err := generateCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	fc := &FriendCode{
		ID:        uuid.NewString(),
		UserID:    userID,
		Code:      code,
		ExpiresAt: time.Now().Add(CodeExpiration),
		CreatedAt: time.Now(),
	}

	if err := s.store.Insert(ctx, fc); err != nil {
		return nil, fmt.Errorf("failed to save friend code: %w", err)
	}

	return &FriendCodeResponse{
		ID:        fc.ID,
		Code:      fc.Code,
		ExpiresAt: fc.ExpiresAt,
		QRData:    s.generateQRData(fc.Code),
	}, nil
}

// Resolve validates a code and returns the associated user info.
func (s *Service) Resolve(ctx context.Context, code string) (*ResolvedUser, error) {
	code = strings.ToUpper(strings.TrimSpace(code))

	fc, err := s.store.GetByCode(ctx, code)
	if err != nil {
		return nil, ErrNotFound
	}

	if fc.ExpiresAt.Before(time.Now()) {
		return nil, ErrExpired
	}

	user, err := s.userStore.GetByID(ctx, fc.UserID)
	if err != nil {
		return nil, ErrNotFound
	}

	return user, nil
}

// AddFriend adds a contact using a friend code.
func (s *Service) AddFriend(ctx context.Context, userID, code string) (*Contact, error) {
	code = strings.ToUpper(strings.TrimSpace(code))

	fc, err := s.store.GetByCode(ctx, code)
	if err != nil {
		return nil, ErrNotFound
	}

	if fc.ExpiresAt.Before(time.Now()) {
		return nil, ErrExpired
	}

	// Cannot add yourself
	if fc.UserID == userID {
		return nil, ErrSelfAdd
	}

	// Check if already a contact
	exists, err := s.contactStore.Exists(ctx, userID, fc.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check contact: %w", err)
	}
	if exists {
		return nil, ErrAlreadyAdded
	}

	// Get user info for display name
	user, err := s.userStore.GetByID(ctx, fc.UserID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Add as contact
	displayName := user.DisplayName
	if displayName == "" {
		displayName = user.Username
	}

	if err := s.contactStore.Insert(ctx, userID, fc.UserID, displayName); err != nil {
		return nil, fmt.Errorf("failed to add contact: %w", err)
	}

	return &Contact{
		UserID:        userID,
		ContactUserID: fc.UserID,
		DisplayName:   displayName,
		CreatedAt:     time.Now(),
	}, nil
}

// Revoke invalidates a user's current friend code.
func (s *Service) Revoke(ctx context.Context, userID string) error {
	return s.store.DeleteByUserID(ctx, userID)
}

// generateCode creates a random alphanumeric code.
func generateCode() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Use base32 encoding for readable codes (no confusing 0/O, 1/I)
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)
	// Format as MIZU-XXXXXXXX
	if len(code) > 8 {
		code = code[:8]
	}
	return "MIZU-" + code, nil
}

// generateQRData creates the QR code data URL.
func (s *Service) generateQRData(code string) string {
	if s.baseURL != "" {
		return fmt.Sprintf("%s/add-friend/%s", s.baseURL, code)
	}
	return fmt.Sprintf("/add-friend/%s", code)
}
