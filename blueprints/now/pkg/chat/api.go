package chat

import (
	"context"
	"time"

	"now/pkg/auth"
)

const (
	// KindDirect is a one to one chat.
	KindDirect = "direct"

	// KindRoom is a group chat.
	KindRoom = "room"
)

// Chat represents a conversation.
type Chat struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	Title       string    `json:"title,omitempty"`
	Creator     string    `json:"creator,omitempty"`
	Fingerprint string    `json:"fingerprint,omitempty"`
	Time        time.Time `json:"created_at"`
}

// Message represents a message in a chat.
type Message struct {
	ID          string    `json:"id"`
	Chat        string    `json:"chat"`
	Actor       string    `json:"actor"`
	Fingerprint string    `json:"fingerprint"`
	Text        string    `json:"text"`
	Signature   string    `json:"signature"`
	Time        time.Time `json:"created_at"`
}

// CreateInput is used to create a chat.
type CreateInput struct {
	Kind       string `json:"kind"`
	Title      string `json:"title,omitempty"`
	Visibility string `json:"visibility,omitempty"`
}

// JoinInput is used to join a chat.
type JoinInput struct {
	Chat  string `json:"chat"`
	Token string `json:"token,omitempty"`
}

// GetInput identifies a chat by id.
type GetInput struct {
	ID string `json:"id"`
}

// ListInput filters chats.
type ListInput struct {
	Kind  string `json:"kind,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

// SendInput is used to send a message.
type SendInput struct {
	Chat      string `json:"chat"`
	Text      string `json:"text"`
	Signature string `json:"signature,omitempty"`
}

// MessagesInput lists messages in a chat.
type MessagesInput struct {
	Chat   string `json:"chat"`
	Before string `json:"before,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// Chats is a list of chats.
type Chats struct {
	Items []Chat `json:"items"`
}

// Messages is a list of messages.
type Messages struct {
	Items []Message `json:"items"`
}

// API defines chat operations. All methods require a VerifiedActor
// produced by the auth layer.
type API interface {
	Create(ctx context.Context, in CreateInput, actor auth.VerifiedActor) (Chat, error)
	Join(ctx context.Context, in JoinInput, actor auth.VerifiedActor) error
	Get(ctx context.Context, in GetInput, actor auth.VerifiedActor) (Chat, error)
	List(ctx context.Context, in ListInput, actor auth.VerifiedActor) (Chats, error)
	Send(ctx context.Context, in SendInput, actor auth.VerifiedActor) (Message, error)
	Messages(ctx context.Context, in MessagesInput, actor auth.VerifiedActor) (Messages, error)
}
