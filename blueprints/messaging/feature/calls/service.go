package calls

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/messaging/pkg/ulid"
)

// Service implements the calls API.
type Service struct {
	store Store
}

// NewService creates a new calls service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Initiate initiates a new call.
func (s *Service) Initiate(ctx context.Context, callerID string, in *InitiateIn) (*Call, error) {
	// Check if caller is already in a call
	active, _ := s.store.GetActiveForUser(ctx, callerID)
	if active != nil {
		return nil, ErrCallBusy
	}

	now := time.Now()
	call := &Call{
		ID:        ulid.New(),
		ChatID:    in.ChatID,
		CallerID:  callerID,
		Type:      in.Type,
		Status:    StatusInitiated,
		CreatedAt: now,
	}

	if err := s.store.Insert(ctx, call); err != nil {
		return nil, err
	}

	// Add caller as participant
	callerP := &Participant{
		CallID:   call.ID,
		UserID:   callerID,
		Status:   "joined",
		JoinedAt: &now,
	}
	if err := s.store.InsertParticipant(ctx, callerP); err != nil {
		return nil, err
	}

	// Add recipients as pending participants
	for _, recipientID := range in.RecipientIDs {
		p := &Participant{
			CallID: call.ID,
			UserID: recipientID,
			Status: "pending",
		}
		if err := s.store.InsertParticipant(ctx, p); err != nil {
			return nil, err
		}
	}

	// Update status to ringing
	if err := s.store.UpdateStatus(ctx, call.ID, StatusRinging); err != nil {
		return nil, err
	}
	call.Status = StatusRinging

	return call, nil
}

// GetByID retrieves a call by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Call, error) {
	return s.store.GetByID(ctx, id)
}

// Accept accepts an incoming call.
func (s *Service) Accept(ctx context.Context, callID, userID string) (*Call, error) {
	call, err := s.store.GetByID(ctx, callID)
	if err != nil {
		return nil, err
	}

	if call.Status == StatusEnded {
		return nil, ErrCallEnded
	}

	now := time.Now()
	updates := map[string]any{
		"status":    "joined",
		"joined_at": now,
	}
	if err := s.store.UpdateParticipant(ctx, callID, userID, updates); err != nil {
		return nil, err
	}

	// Update call status to ongoing if first accept
	if call.Status == StatusRinging {
		callUpdates := map[string]any{
			"status":     StatusOngoing,
			"started_at": now,
		}
		if err := s.store.Update(ctx, callID, callUpdates); err != nil {
			return nil, err
		}
	}

	return s.store.GetByID(ctx, callID)
}

// Decline declines an incoming call.
func (s *Service) Decline(ctx context.Context, callID, userID string) error {
	call, err := s.store.GetByID(ctx, callID)
	if err != nil {
		return err
	}

	if call.Status == StatusEnded {
		return ErrCallEnded
	}

	updates := map[string]any{
		"status": "declined",
	}
	if err := s.store.UpdateParticipant(ctx, callID, userID, updates); err != nil {
		return err
	}

	// Check if all recipients declined
	participants, err := s.store.GetParticipants(ctx, callID)
	if err != nil {
		return err
	}

	allDeclined := true
	for _, p := range participants {
		if p.UserID != call.CallerID && p.Status != "declined" {
			allDeclined = false
			break
		}
	}

	if allDeclined {
		return s.store.UpdateStatus(ctx, callID, StatusDeclined)
	}

	return nil
}

// End ends a call.
func (s *Service) End(ctx context.Context, callID, userID string) error {
	call, err := s.store.GetByID(ctx, callID)
	if err != nil {
		return err
	}

	if call.Status == StatusEnded {
		return nil
	}

	now := time.Now()
	var duration int
	if call.StartedAt != nil {
		duration = int(now.Sub(*call.StartedAt).Seconds())
	}

	updates := map[string]any{
		"status":    StatusEnded,
		"ended_at":  now,
		"duration":  duration,
	}

	return s.store.Update(ctx, callID, updates)
}

// Join joins an ongoing call.
func (s *Service) Join(ctx context.Context, callID, userID string) error {
	call, err := s.store.GetByID(ctx, callID)
	if err != nil {
		return err
	}

	if call.Status == StatusEnded {
		return ErrCallEnded
	}

	now := time.Now()
	updates := map[string]any{
		"status":    "joined",
		"joined_at": now,
	}

	return s.store.UpdateParticipant(ctx, callID, userID, updates)
}

// Leave leaves a call.
func (s *Service) Leave(ctx context.Context, callID, userID string) error {
	now := time.Now()
	updates := map[string]any{
		"status":  "left",
		"left_at": now,
	}

	return s.store.UpdateParticipant(ctx, callID, userID, updates)
}

// Mute mutes/unmutes in a call.
func (s *Service) Mute(ctx context.Context, callID, userID string, muted bool) error {
	updates := map[string]any{
		"is_muted": muted,
	}
	return s.store.UpdateParticipant(ctx, callID, userID, updates)
}

// SetVideoEnabled enables/disables video in a call.
func (s *Service) SetVideoEnabled(ctx context.Context, callID, userID string, enabled bool) error {
	updates := map[string]any{
		"is_video_off": !enabled,
	}
	return s.store.UpdateParticipant(ctx, callID, userID, updates)
}

// GetHistory gets call history for a user.
func (s *Service) GetHistory(ctx context.Context, userID string, limit int) ([]*Call, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.store.GetHistory(ctx, userID, limit)
}

// GetActive gets the active call for a user.
func (s *Service) GetActive(ctx context.Context, userID string) (*Call, error) {
	return s.store.GetActiveForUser(ctx, userID)
}
