package channel

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// Driver is the interface that all channel drivers must implement.
// Each messaging platform (Telegram, Discord, etc.) provides its own Driver.
type Driver interface {
	// Type returns the channel type identifier.
	Type() types.ChannelType

	// Connect initializes the connection to the messaging platform.
	Connect(ctx context.Context) error

	// Disconnect closes the connection.
	Disconnect(ctx context.Context) error

	// Send sends a message to a peer on the platform.
	Send(ctx context.Context, msg *types.OutboundMessage) error

	// Status returns the current connection status.
	Status() string
}

// MessageHandler is called when a channel driver receives an inbound message.
type MessageHandler func(ctx context.Context, msg *types.InboundMessage) error

// registry holds registered driver constructors.
var registry = map[types.ChannelType]func(config string, handler MessageHandler) (Driver, error){}

// Register adds a driver constructor to the registry.
func Register(ct types.ChannelType, fn func(config string, handler MessageHandler) (Driver, error)) {
	registry[ct] = fn
}

// New creates a new driver for the given channel type.
func New(ct types.ChannelType, config string, handler MessageHandler) (Driver, error) {
	fn, ok := registry[ct]
	if !ok {
		return nil, &UnsupportedError{ChannelType: ct}
	}
	return fn(config, handler)
}

// UnsupportedError is returned when a channel type has no registered driver.
type UnsupportedError struct {
	ChannelType types.ChannelType
}

func (e *UnsupportedError) Error() string {
	return "unsupported channel type: " + string(e.ChannelType)
}
