// Package calls provides voice/video call signaling.
package calls

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("call not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrCallEnded    = errors.New("call already ended")
	ErrCallBusy     = errors.New("user is busy")
)

// CallType represents the type of call.
type CallType string

const (
	TypeVoice CallType = "voice"
	TypeVideo CallType = "video"
)

// CallStatus represents call status.
type CallStatus string

const (
	StatusInitiated CallStatus = "initiated"
	StatusRinging   CallStatus = "ringing"
	StatusOngoing   CallStatus = "ongoing"
	StatusEnded     CallStatus = "ended"
	StatusMissed    CallStatus = "missed"
	StatusDeclined  CallStatus = "declined"
)

// Call represents a voice/video call.
type Call struct {
	ID        string     `json:"id"`
	ChatID    string     `json:"chat_id,omitempty"`
	CallerID  string     `json:"caller_id"`
	Type      CallType   `json:"type"`
	Status    CallStatus `json:"status"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Duration  int        `json:"duration,omitempty"` // seconds
	CreatedAt time.Time  `json:"created_at"`

	// Populated from joins
	Caller       any            `json:"caller,omitempty"`
	Participants []*Participant `json:"participants,omitempty"`
}

// Participant represents a call participant.
type Participant struct {
	CallID     string     `json:"call_id"`
	UserID     string     `json:"user_id"`
	Status     string     `json:"status"` // pending, joined, left
	JoinedAt   *time.Time `json:"joined_at,omitempty"`
	LeftAt     *time.Time `json:"left_at,omitempty"`
	IsMuted    bool       `json:"is_muted"`
	IsVideoOff bool       `json:"is_video_off"`

	// Joined user info
	User any `json:"user,omitempty"`
}

// SignalData represents WebRTC signaling data.
type SignalData struct {
	Type      string `json:"type"` // offer, answer, ice-candidate
	SDP       string `json:"sdp,omitempty"`
	Candidate string `json:"candidate,omitempty"`
}

// InitiateIn contains input for initiating a call.
type InitiateIn struct {
	RecipientIDs []string `json:"recipient_ids"`
	Type         CallType `json:"type"`
	ChatID       string   `json:"chat_id,omitempty"`
}

// API defines the calls service contract.
type API interface {
	Initiate(ctx context.Context, callerID string, in *InitiateIn) (*Call, error)
	GetByID(ctx context.Context, id string) (*Call, error)
	Accept(ctx context.Context, callID, userID string) (*Call, error)
	Decline(ctx context.Context, callID, userID string) error
	End(ctx context.Context, callID, userID string) error
	Join(ctx context.Context, callID, userID string) error
	Leave(ctx context.Context, callID, userID string) error
	Mute(ctx context.Context, callID, userID string, muted bool) error
	SetVideoEnabled(ctx context.Context, callID, userID string, enabled bool) error
	GetHistory(ctx context.Context, userID string, limit int) ([]*Call, error)
	GetActive(ctx context.Context, userID string) (*Call, error)
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, c *Call) error
	GetByID(ctx context.Context, id string) (*Call, error)
	Update(ctx context.Context, id string, updates map[string]any) error
	UpdateStatus(ctx context.Context, id string, status CallStatus) error
	GetHistory(ctx context.Context, userID string, limit int) ([]*Call, error)
	GetActiveForUser(ctx context.Context, userID string) (*Call, error)

	// Participants
	InsertParticipant(ctx context.Context, p *Participant) error
	UpdateParticipant(ctx context.Context, callID, userID string, updates map[string]any) error
	GetParticipants(ctx context.Context, callID string) ([]*Participant, error)
	GetParticipant(ctx context.Context, callID, userID string) (*Participant, error)
}
