package live

import (
	"time"

	"github.com/go-mizu/mizu"
)

// Ctx is the live page context.
type Ctx struct {
	// Mizu context for request access.
	*mizu.Ctx

	// SessionID is the live session identifier.
	SessionID string

	// live engine reference for pubsub access.
	live *Live

	// handler for internal operations.
	handler *sessionHandler
}

// Subscribe subscribes the session to pubsub topics.
func (c *Ctx) Subscribe(topics ...string) error {
	if c.live == nil || c.live.pubsub == nil {
		return nil
	}
	return c.live.pubsub.Subscribe(c.SessionID, topics...)
}

// Unsubscribe removes subscriptions from topics.
func (c *Ctx) Unsubscribe(topics ...string) error {
	if c.live == nil || c.live.pubsub == nil {
		return nil
	}
	return c.live.pubsub.Unsubscribe(c.SessionID, topics...)
}

// SendInfo sends a message to this session's Info handler.
func (c *Ctx) SendInfo(msg any) error {
	if c.handler != nil {
		c.handler.sendServerMsg(msg)
	}
	return nil
}

// SendAfter sends a message to the Info handler after a delay.
func (c *Ctx) SendAfter(msg any, delay time.Duration) *Timer {
	if c.handler == nil {
		return &Timer{}
	}
	return c.handler.sendAfter(msg, delay)
}

// Connected returns true if the WebSocket is connected.
func (c *Ctx) Connected() bool {
	return c.handler != nil && c.handler.connected()
}

// Timer represents a scheduled message.
type Timer struct {
	id     string
	cancel func()
}

// Cancel cancels the scheduled message.
func (t *Timer) Cancel() {
	if t.cancel != nil {
		t.cancel()
	}
}

// newCtx creates a new live context.
func newCtx(mc *mizu.Ctx, sessionID string, l *Live, h *sessionHandler) *Ctx {
	return &Ctx{
		Ctx:       mc,
		SessionID: sessionID,
		live:      l,
		handler:   h,
	}
}
