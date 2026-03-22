package chat

import "context"

// ChatStore stores chats.
type ChatStore interface {
	// Create creates a chat.
	Create(ctx context.Context, chat Chat) error

	// Get returns a chat.
	Get(ctx context.Context, id string) (Chat, error)

	// List returns chats.
	List(ctx context.Context, in ListInput) (Chats, error)
}

// MemberStore stores chat members.
type MemberStore interface {
	// Join adds an actor to a chat.
	Join(ctx context.Context, chat string, actor string) error

	// Leave removes an actor from a chat.
	Leave(ctx context.Context, chat string, actor string) error

	// Has reports whether an actor is in a chat.
	Has(ctx context.Context, chat string, actor string) (bool, error)

	// List returns members in a chat.
	List(ctx context.Context, chat string, limit int) ([]string, error)
}

// MessageStore stores messages.
type MessageStore interface {
	// Create creates a message.
	Create(ctx context.Context, msg Message) error

	// Get returns a message.
	Get(ctx context.Context, id string) (Message, error)

	// List returns messages.
	List(ctx context.Context, in MessagesInput) (Messages, error)
}
