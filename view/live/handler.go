package live

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/view"
)

// sessionHandler manages a single live session.
type sessionHandler struct {
	live    *Live
	page    pageHandler
	session *sessionBase
	view    View
	ctx     *Ctx
	mizuCtx *mizu.Ctx

	conn      *wsConn
	connMu    sync.Mutex
	isConn    atomic.Bool
	serverCh  chan any
	closeCh   chan struct{}
	closeOnce sync.Once

	timers   map[string]*time.Timer
	timersMu sync.Mutex
	timerID  uint64
}

// newSessionHandler creates a new session handler.
func newSessionHandler(l *Live, page pageHandler, mc *mizu.Ctx) *sessionHandler {
	sessionID := generateSessionID()

	h := &sessionHandler{
		live:     l,
		page:     page,
		session:  page.newSession(sessionID),
		mizuCtx:  mc,
		serverCh: make(chan any, 64),
		closeCh:  make(chan struct{}),
		timers:   make(map[string]*time.Timer),
	}

	h.ctx = newCtx(mc, sessionID, l, h)

	// Set session creation time.
	now := time.Now()
	h.session.setCreated(now)
	h.session.setLastSeen(now)

	return h
}

// generateSessionID creates a cryptographically random session ID.
func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to time-based (not ideal but better than panic).
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// run is the main session event loop.
func (h *sessionHandler) run(ctx context.Context, conn *wsConn, join *JoinPayload) error {
	h.conn = conn
	h.isConn.Store(true)
	defer h.cleanup()

	// Register with pubsub.
	if h.live.pubsub != nil {
		h.live.pubsub.register(h.session.getID(), h.serverCh)
		defer h.live.pubsub.unregister(h.session.getID())
		defer h.live.pubsub.UnsubscribeAll(h.session.getID())

		// Subscribe to sync scopes from join payload.
		if len(join.Scopes) > 0 {
			h.live.pubsub.Subscribe(h.session.getID(), join.Scopes...)
		}
	}

	// Store session.
	if h.live.store != nil {
		h.live.store.Set(h.session.getID(), h.session)
		defer h.live.store.Delete(h.session.getID())
	}

	// Call Mount.
	h.session.lock()
	if err := h.page.mount(h.ctx, h.session); err != nil {
		h.session.unlock()
		return fmt.Errorf("mount: %w", err)
	}

	// Get initial view.
	v, err := h.page.render(h.ctx, h.session)
	if err != nil {
		h.session.unlock()
		return fmt.Errorf("render: %w", err)
	}
	h.view = v
	h.session.unlock()

	// Send initial render.
	if err := h.sendInitialRender(); err != nil {
		return err
	}

	// Notify session start.
	if h.live.opts.OnSessionStart != nil {
		h.live.opts.OnSessionStart(h.ctx, h.session.getID())
	}

	// Run event loop.
	return h.eventLoop(ctx)
}

// eventLoop processes events until connection closes.
func (h *sessionHandler) eventLoop(ctx context.Context) error {
	clientCh := make(chan *Message, 16)
	errCh := make(chan error, 1)

	// Start read goroutine.
	go func() {
		for {
			msg, err := h.readMessage()
			if err != nil {
				if err != ErrConnectionClose {
					errCh <- err
				}
				close(clientCh)
				return
			}
			clientCh <- msg
		}
	}()

	// Heartbeat ticker.
	heartbeat := time.NewTicker(h.live.opts.HeartbeatInterval)
	defer heartbeat.Stop()

	// Session timeout timer.
	timeout := time.NewTimer(h.live.opts.SessionTimeout)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-h.closeCh:
			return nil

		case err := <-errCh:
			return err

		case msg, ok := <-clientCh:
			if !ok {
				return nil // Connection closed.
			}

			// Reset timeout on activity.
			if !timeout.Stop() {
				select {
				case <-timeout.C:
				default:
				}
			}
			timeout.Reset(h.live.opts.SessionTimeout)

			if err := h.handleClientMessage(msg); err != nil {
				h.sendError(err)
			}

		case msg := <-h.serverCh:
			if err := h.handleServerMessage(msg); err != nil {
				h.sendError(err)
			}

		case <-heartbeat.C:
			h.sendHeartbeat()

		case <-timeout.C:
			h.sendClose("session_expired", "Session timed out")
			return nil
		}
	}
}

// handleClientMessage processes a message from the client.
func (h *sessionHandler) handleClientMessage(msg *Message) error {
	h.session.setLastSeen(time.Now())

	switch msg.Type {
	case MsgTypeEvent:
		return h.handleEvent(msg)

	case MsgTypeHeartbeat:
		var payload HeartbeatPayload
		if err := msg.parsePayload(&payload); err != nil {
			return err
		}
		// Send pong.
		h.send(MsgTypeHeartbeat, msg.Ref, HeartbeatPayload{Pong: payload.Ping})
		return nil

	case MsgTypeLeave:
		h.closeOnce.Do(func() { close(h.closeCh) })
		return nil

	case MsgTypeSubscribe:
		var payload SubscribePayload
		if err := msg.parsePayload(&payload); err != nil {
			return err
		}
		if h.live.pubsub != nil && len(payload.Scopes) > 0 {
			h.live.pubsub.Subscribe(h.session.getID(), payload.Scopes...)
		}
		h.send(MsgTypeReply, msg.Ref, ReplyPayload{Status: "ok"})
		return nil

	case MsgTypeUnsubscribe:
		var payload UnsubscribePayload
		if err := msg.parsePayload(&payload); err != nil {
			return err
		}
		if h.live.pubsub != nil && len(payload.Scopes) > 0 {
			h.live.pubsub.Unsubscribe(h.session.getID(), payload.Scopes...)
		}
		h.send(MsgTypeReply, msg.Ref, ReplyPayload{Status: "ok"})
		return nil

	default:
		return fmt.Errorf("unknown message type: %d", msg.Type)
	}
}

// handleEvent processes a client event.
func (h *sessionHandler) handleEvent(msg *Message) error {
	var payload eventPayload
	if err := msg.parsePayload(&payload); err != nil {
		return err
	}

	event := payload.toEvent()

	h.session.lock()
	defer h.session.unlock()

	// Call Handle.
	if err := h.page.handle(h.ctx, h.session, event); err != nil {
		// Send error reply.
		h.send(MsgTypeReply, msg.Ref, ReplyPayload{
			Status:  "error",
			Reason:  "handler_error",
			Message: err.Error(),
		})

		if h.live.opts.OnError != nil {
			h.live.opts.OnError(h.ctx, err)
		}
		return nil
	}

	// Send success reply.
	h.send(MsgTypeReply, msg.Ref, ReplyPayload{Status: "ok"})

	// Send patches if dirty.
	return h.sendPatches()
}

// handleServerMessage processes a server-originated message.
func (h *sessionHandler) handleServerMessage(msg any) error {
	// Check if this is a sync poke - these are forwarded directly to the client.
	if poke, ok := msg.(Poke); ok {
		h.send(MsgTypePoke, 0, PokePayload{
			Scope:  poke.Scope,
			Cursor: poke.Cursor,
		})
		return nil
	}

	h.session.lock()
	defer h.session.unlock()

	// Call Info.
	if err := h.page.info(h.ctx, h.session, msg); err != nil {
		if h.live.opts.OnError != nil {
			h.live.opts.OnError(h.ctx, err)
		}
		return nil
	}

	// Send patches if dirty.
	return h.sendPatches()
}

// sendInitialRender sends the initial rendered regions.
func (h *sessionHandler) sendInitialRender() error {
	h.session.lock()
	defer h.session.unlock()

	rendered := make(map[string]string)

	// Render all regions.
	dirty := h.session.getDirty()
	if dirty.IsAll() || dirty.IsEmpty() {
		// Render all defined regions.
		for id, tmpl := range h.view.Regions {
			html, err := h.renderRegion(id, tmpl)
			if err != nil {
				return err
			}
			rendered[id] = html
			h.session.setRegion(id, html)
		}
	} else {
		// Render only dirty regions.
		for _, id := range dirty.List() {
			if tmpl, ok := h.view.Regions[id]; ok {
				html, err := h.renderRegion(id, tmpl)
				if err != nil {
					return err
				}
				rendered[id] = html
				h.session.setRegion(id, html)
			}
		}
	}

	dirty.Clear()

	// Send reply with rendered content.
	h.send(MsgTypeReply, 1, ReplyPayload{
		Status:    "ok",
		SessionID: h.session.getID(),
		Rendered:  rendered,
	})

	// Send any queued commands.
	h.sendCommands()

	return nil
}

// sendPatches renders dirty regions and sends patches.
func (h *sessionHandler) sendPatches() error {
	dirty := h.session.getDirty()
	if dirty.IsEmpty() {
		// Still send commands if any.
		h.sendCommands()
		return nil
	}

	var patches []RegionPatch

	if dirty.IsAll() {
		// Render all regions.
		for id, tmpl := range h.view.Regions {
			html, err := h.renderRegion(id, tmpl)
			if err != nil {
				return err
			}

			oldHTML := h.session.getRegions()[id]
			if html != oldHTML {
				patches = append(patches, RegionPatch{
					ID:     id,
					HTML:   html,
					Action: "morph",
				})
				h.session.setRegion(id, html)
			}
		}
	} else {
		// Render only dirty regions.
		for _, id := range dirty.List() {
			tmpl, ok := h.view.Regions[id]
			if !ok {
				// Not a region, skip.
				continue
			}

			html, err := h.renderRegion(id, tmpl)
			if err != nil {
				return err
			}

			oldHTML := h.session.getRegions()[id]
			if html != oldHTML {
				patches = append(patches, RegionPatch{
					ID:     id,
					HTML:   html,
					Action: "morph",
				})
				h.session.setRegion(id, html)
			}
		}
	}

	dirty.Clear()

	if len(patches) > 0 {
		h.send(MsgTypePatch, 0, PatchPayload{Regions: patches})
	}

	// Send any queued commands.
	h.sendCommands()

	return nil
}

// renderRegion renders a partial template for a region.
func (h *sessionHandler) renderRegion(id, tmpl string) (string, error) {
	// Build template data.
	data := h.buildTemplateData()

	var buf bytes.Buffer
	if err := h.live.view.RenderPartial(&buf, tmpl, data); err != nil {
		return "", fmt.Errorf("render region %s: %w", id, err)
	}

	return buf.String(), nil
}

// buildTemplateData builds the data passed to templates.
func (h *sessionHandler) buildTemplateData() view.Data {
	return view.Data{
		"State": h.getState(),
		"Flash": h.session.getFlash(),
	}
}

// getState returns the session state via accessor.
func (h *sessionHandler) getState() any {
	switch s := h.session.typed().(type) {
	case interface{ getState() any }:
		return s.getState()
	default:
		// Use reflection-free approach for Session[T].
		return getSessionState(h.session.session)
	}
}

// getSessionState extracts State from Session[T].
type stateAccessor interface {
	getStateField() any
}

func (s *Session[T]) getStateField() any { return s.State }

func getSessionState(sess any) any {
	if a, ok := sess.(stateAccessor); ok {
		return a.getStateField()
	}
	return nil
}

// sendCommands sends queued client commands.
func (h *sessionHandler) sendCommands() {
	cmds := h.session.getCommands()
	if len(cmds) == 0 {
		return
	}

	var envelopes []commandEnvelope
	for _, cmd := range cmds {
		envelopes = append(envelopes, wrapCommand(cmd))
	}

	h.send(MsgTypeCommand, 0, CommandPayload{Commands: envelopes})
	h.session.clearCommands()
}

// sendHeartbeat sends a heartbeat ping.
func (h *sessionHandler) sendHeartbeat() {
	h.send(MsgTypeHeartbeat, 0, HeartbeatPayload{Ping: time.Now().UnixMilli()})
}

// sendError sends an error message.
func (h *sessionHandler) sendError(err error) {
	h.send(MsgTypeError, 0, ErrorPayload{
		Code:        "error",
		Message:     err.Error(),
		Recoverable: true,
	})
}

// sendClose sends a close message.
func (h *sessionHandler) sendClose(reason, message string) {
	h.send(MsgTypeClose, 0, ClosePayload{
		Reason:  reason,
		Message: message,
	})
}

// send sends a message over the WebSocket.
func (h *sessionHandler) send(msgType byte, ref uint32, payload any) {
	data, err := encodeMessage(msgType, ref, payload)
	if err != nil {
		return
	}

	h.connMu.Lock()
	defer h.connMu.Unlock()

	if h.conn != nil {
		h.conn.WriteMessage(string(data))
	}
}

// readMessage reads a message from the WebSocket.
func (h *sessionHandler) readMessage() (*Message, error) {
	data, err := h.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	return decodeMessage([]byte(data))
}

// sendServerMsg sends a message to the server channel.
func (h *sessionHandler) sendServerMsg(msg any) {
	select {
	case h.serverCh <- msg:
	default:
		// Channel full, drop message.
	}
}

// sendAfter schedules a message after delay.
func (h *sessionHandler) sendAfter(msg any, delay time.Duration) *Timer {
	h.timersMu.Lock()
	defer h.timersMu.Unlock()

	id := fmt.Sprintf("t%d", atomic.AddUint64(&h.timerID, 1))

	timer := time.AfterFunc(delay, func() {
		h.sendServerMsg(msg)

		h.timersMu.Lock()
		delete(h.timers, id)
		h.timersMu.Unlock()
	})

	h.timers[id] = timer

	return &Timer{
		id: id,
		cancel: func() {
			h.timersMu.Lock()
			defer h.timersMu.Unlock()
			if t, ok := h.timers[id]; ok {
				t.Stop()
				delete(h.timers, id)
			}
		},
	}
}

// connected returns true if the WebSocket is connected.
func (h *sessionHandler) connected() bool {
	return h.isConn.Load()
}

// cleanup releases resources when the session ends.
func (h *sessionHandler) cleanup() {
	h.isConn.Store(false)

	// Cancel all timers.
	h.timersMu.Lock()
	for _, t := range h.timers {
		t.Stop()
	}
	h.timers = nil
	h.timersMu.Unlock()

	// Notify session end.
	if h.live.opts.OnSessionEnd != nil {
		h.live.opts.OnSessionEnd(h.session.getID())
	}
}
