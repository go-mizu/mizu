// Package live provides low-latency realtime message delivery over WebSocket
// with topic-based publish and subscribe.
//
// It is designed as a transport and fanout layer, not a correctness layer.
// Messages are best-effort. If a client disconnects or misses messages,
// recovery must happen through another mechanism such as sync, polling, or reload.
//
// # Design principles
//
//   - Transport-only: live moves messages, it does not interpret or validate state
//   - Best-effort delivery: no durability or replay guarantees
//   - Topic-based fanout: scalable, simple routing model
//   - Opaque payloads: higher layers define schemas
//   - Minimal surface area: few types, predictable behavior
//   - Independent: no dependency on sync, view, or application logic
//
// # Basic usage
//
//	server := live.New(live.Options{
//	    OnAuth: func(ctx context.Context, r *http.Request) (any, error) {
//	        token := r.Header.Get("Authorization")
//	        if !validateToken(token) {
//	            return nil, errors.New("invalid token")
//	        }
//	        return UserInfo{ID: getUserID(token)}, nil
//	    },
//	    OnMessage: func(ctx context.Context, s *live.Session, topic string, data []byte) {
//	        var cmd Command
//	        json.Unmarshal(data, &cmd)
//	        switch cmd.Type {
//	        case "subscribe":
//	            server.Subscribe(s, topic)
//	        case "unsubscribe":
//	            server.Unsubscribe(s, topic)
//	        }
//	    },
//	    OnClose: func(s *live.Session, err error) {
//	        log.Printf("session %s closed: %v", s.ID(), err)
//	    },
//	})
//
//	app := mizu.New()
//	app.Get("/ws", mizu.Compat(server.Handler()))
//	app.Listen(":8080")
//
// # Connection lifecycle
//
//  1. HTTP request arrives at handler
//  2. OnAuth called (optional) to authenticate
//  3. WebSocket upgrade performed
//  4. Session created and registered
//  5. Read loop receives messages and calls OnMessage
//  6. Write loop sends queued messages to client
//  7. On disconnect: cleanup subscriptions, call OnClose with close reason
//
// # Backpressure
//
// Each session has a bounded send queue (default 256 messages).
// If the queue fills up, the session is closed to protect server health.
// This is intentional: slow clients should not affect other clients.
package live

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/go-mizu/mizu/live/internal/ws"
)

// -----------------------------------------------------------------------------
// Errors
// -----------------------------------------------------------------------------

var (
	// ErrSessionClosed is returned when sending to a closed session.
	ErrSessionClosed = errors.New("live: session closed")

	// ErrQueueFull is returned when the send queue is full.
	// When this happens, the session is closed to protect server health.
	ErrQueueFull = errors.New("live: send queue full")
)

// -----------------------------------------------------------------------------
// Types
// -----------------------------------------------------------------------------

// Message is the transport envelope for pub/sub.
// It carries topic and opaque data between clients and server.
type Message struct {
	// Topic is the routing key for pub/sub operations.
	Topic string `json:"topic,omitempty"`

	// Data contains the message payload as opaque JSON.
	// Higher layers define the schema.
	Data json.RawMessage `json:"data,omitempty"`
}

// -----------------------------------------------------------------------------
// Options
// -----------------------------------------------------------------------------

const (
	defaultQueueSize = 256
	defaultReadLimit = 4 * 1024 * 1024 // 4MB
)

// Options configures the Server.
type Options struct {
	// QueueSize is the per-session send queue size. Default: 256.
	// When the queue fills up, the session is closed.
	QueueSize int

	// ReadLimit is the maximum message size in bytes. Default: 4MB.
	// Messages exceeding this limit cause the connection to be closed.
	ReadLimit int

	// OnAuth is called to authenticate new connections.
	// Return any value on success (accessible via Session.Value()),
	// or an error to reject the connection.
	// If nil, all connections are accepted without authentication.
	OnAuth func(ctx context.Context, r *http.Request) (any, error)

	// OnMessage is called when a message is received from a client.
	// The topic and data are extracted from the incoming JSON message.
	// This is where you implement subscribe/unsubscribe/publish logic.
	OnMessage func(ctx context.Context, s *Session, topic string, data []byte)

	// OnClose is called when a session is closed.
	// The error contains the actual close reason (may be nil for clean closes).
	OnClose func(s *Session, err error)

	// Origins is a list of allowed origins for WebSocket connections.
	// If empty and CheckOrigin is nil, all origins are allowed.
	// For more complex origin validation, use CheckOrigin instead.
	Origins []string

	// CheckOrigin validates the Origin header. Return true to allow.
	// If set, this takes precedence over Origins.
	// If nil and Origins is empty, all origins are allowed.
	CheckOrigin func(r *http.Request) bool

	// IDGenerator generates unique session IDs.
	// If nil, random hex IDs are generated.
	IDGenerator func() string
}

// -----------------------------------------------------------------------------
// Server
// -----------------------------------------------------------------------------

// Server owns sessions, pubsub state, and the WebSocket handler.
type Server struct {
	opts     Options
	pubsub   *memPubSub
	sessions sync.Map // map[string]*Session
}

// New creates a new live server with the given options.
func New(opts Options) *Server {
	if opts.QueueSize <= 0 {
		opts.QueueSize = defaultQueueSize
	}
	if opts.ReadLimit <= 0 {
		opts.ReadLimit = defaultReadLimit
	}
	if opts.IDGenerator == nil {
		opts.IDGenerator = generateID
	}

	return &Server{
		opts:   opts,
		pubsub: newMemPubSub(),
	}
}

// Handler returns an http.Handler that upgrades connections to WebSocket.
func (srv *Server) Handler() http.Handler {
	return http.HandlerFunc(srv.handleConn)
}

// Publish sends data to all subscribers of a topic.
func (srv *Server) Publish(topic string, data []byte) {
	srv.pubsub.publish(topic, data)
}

// Subscribe adds a session to a topic.
func (srv *Server) Subscribe(s *Session, topic string) {
	srv.pubsub.subscribe(s, topic)
}

// Unsubscribe removes a session from a topic.
func (srv *Server) Unsubscribe(s *Session, topic string) {
	srv.pubsub.unsubscribe(s, topic)
}

// addSession registers a session with the server.
func (srv *Server) addSession(s *Session) {
	srv.sessions.Store(s.id, s)
}

// removeSession unregisters a session from the server.
func (srv *Server) removeSession(s *Session) {
	srv.sessions.Delete(s.id)
	srv.pubsub.unsubscribeAll(s)
}

// generateID generates a random hex session ID.
func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return encodeHex(b)
}

func encodeHex(b []byte) string {
	const hextable = "0123456789abcdef"
	dst := make([]byte, len(b)*2)
	for i, v := range b {
		dst[i*2] = hextable[v>>4]
		dst[i*2+1] = hextable[v&0x0f]
	}
	return string(dst)
}

// -----------------------------------------------------------------------------
// Session
// -----------------------------------------------------------------------------

// Session represents a single connected WebSocket client.
// It is safe for concurrent use.
type Session struct {
	id       string
	value    any
	sendCh   chan Message
	server   *Server
	topics   map[string]struct{}
	mu       sync.RWMutex
	closed   atomic.Bool
	closeErr atomic.Value // stores error
	doneCh   chan struct{}
	wsConn   *ws.Conn // underlying WebSocket connection
}

// newSession creates a new session with the given ID and value.
func newSession(id string, value any, queueSize int, server *Server) *Session {
	if queueSize <= 0 {
		queueSize = defaultQueueSize
	}
	return &Session{
		id:     id,
		value:  value,
		sendCh: make(chan Message, queueSize),
		server: server,
		topics: make(map[string]struct{}),
		doneCh: make(chan struct{}),
	}
}

// ID returns the session's unique identifier.
func (s *Session) ID() string {
	return s.id
}

// Value returns the opaque value set during authentication.
// Cast to your application type as needed.
func (s *Session) Value() any {
	return s.value
}

// Send queues a message for delivery to the client.
// It is non-blocking and returns an error if the session is closed
// or if the send queue is full.
//
// When the queue is full, the session is automatically closed to
// enforce backpressure and protect server health.
func (s *Session) Send(msg Message) error {
	if s.closed.Load() {
		return ErrSessionClosed
	}

	select {
	case s.sendCh <- msg:
		return nil
	default:
		// Queue is full - close session to enforce backpressure
		_ = s.closeWithError(ErrQueueFull)
		return ErrQueueFull
	}
}

// Close gracefully closes the session.
// It is safe to call multiple times.
func (s *Session) Close() error {
	return s.closeWithError(nil)
}

// closeWithError closes the session with the given error.
// The error is stored and passed to OnClose.
func (s *Session) closeWithError(err error) error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}
	if err != nil {
		s.closeErr.Store(err)
	}
	close(s.doneCh)
	// Close the underlying connection to unblock readLoop
	if s.wsConn != nil {
		_ = s.wsConn.Close()
	}
	return nil
}

// CloseError returns the error that caused the session to close, if any.
func (s *Session) CloseError() error {
	v := s.closeErr.Load()
	if v == nil {
		return nil
	}
	return v.(error)
}

// IsClosed returns true if the session has been closed.
func (s *Session) IsClosed() bool {
	return s.closed.Load()
}

// addTopic adds a topic to the session's subscription set.
func (s *Session) addTopic(topic string) {
	s.mu.Lock()
	s.topics[topic] = struct{}{}
	s.mu.Unlock()
}

// removeTopic removes a topic from the session's subscription set.
func (s *Session) removeTopic(topic string) {
	s.mu.Lock()
	delete(s.topics, topic)
	s.mu.Unlock()
}

// clearTopics removes all topics from the session's subscription set.
func (s *Session) clearTopics() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	topics := make([]string, 0, len(s.topics))
	for t := range s.topics {
		topics = append(topics, t)
	}
	s.topics = make(map[string]struct{})
	return topics
}

// -----------------------------------------------------------------------------
// PubSub (internal)
// -----------------------------------------------------------------------------

// memPubSub is the in-memory PubSub implementation.
type memPubSub struct {
	mu     sync.RWMutex
	topics map[string]map[*Session]struct{}
}

// newMemPubSub creates a new in-memory PubSub.
func newMemPubSub() *memPubSub {
	return &memPubSub{
		topics: make(map[string]map[*Session]struct{}),
	}
}

// subscribe adds a session to a topic.
func (p *memPubSub) subscribe(s *Session, topic string) {
	if s == nil || topic == "" {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	subs, ok := p.topics[topic]
	if !ok {
		subs = make(map[*Session]struct{})
		p.topics[topic] = subs
	}
	subs[s] = struct{}{}
	s.addTopic(topic)
}

// unsubscribe removes a session from a topic.
func (p *memPubSub) unsubscribe(s *Session, topic string) {
	if s == nil || topic == "" {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	subs, ok := p.topics[topic]
	if !ok {
		return
	}
	delete(subs, s)
	s.removeTopic(topic)

	// Clean up empty topics
	if len(subs) == 0 {
		delete(p.topics, topic)
	}
}

// unsubscribeAll removes a session from all topics.
func (p *memPubSub) unsubscribeAll(s *Session) {
	if s == nil {
		return
	}

	topics := s.clearTopics()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, topic := range topics {
		subs, ok := p.topics[topic]
		if !ok {
			continue
		}
		delete(subs, s)
		if len(subs) == 0 {
			delete(p.topics, topic)
		}
	}
}

// publish sends data to all subscribers of a topic.
// Messages are sent asynchronously; slow receivers don't block the publisher.
func (p *memPubSub) publish(topic string, data []byte) {
	if topic == "" {
		return
	}

	p.mu.RLock()
	subs, ok := p.topics[topic]
	if !ok || len(subs) == 0 {
		p.mu.RUnlock()
		return
	}

	// Take a snapshot to avoid holding lock during sends
	sessions := make([]*Session, 0, len(subs))
	for s := range subs {
		sessions = append(sessions, s)
	}
	p.mu.RUnlock()

	// Create message with topic and data
	msg := Message{
		Topic: topic,
		Data:  data,
	}

	// Send to all subscribers
	for _, s := range sessions {
		// Non-blocking send; if it fails (queue full), session will be closed
		_ = s.Send(msg)
	}
}

// -----------------------------------------------------------------------------
// Codec (internal)
// -----------------------------------------------------------------------------

// encodeMessage serializes a message to JSON.
func encodeMessage(m Message) ([]byte, error) {
	return json.Marshal(m)
}

// decodeMessage deserializes JSON to extract topic and data.
func decodeMessage(data []byte) (topic string, payload []byte, err error) {
	var m Message
	if err := json.Unmarshal(data, &m); err != nil {
		return "", nil, err
	}
	return m.Topic, m.Data, nil
}

// -----------------------------------------------------------------------------
// Connection Handler
// -----------------------------------------------------------------------------

// handleConn handles a new WebSocket connection.
//
//nolint:cyclop // Connection handling requires multiple steps
func (srv *Server) handleConn(w http.ResponseWriter, r *http.Request) {
	// Check if it's a WebSocket upgrade request
	if !ws.IsUpgradeRequest(r) {
		http.Error(w, "websocket upgrade required", http.StatusBadRequest)
		return
	}

	// Validate WebSocket version (RFC 6455 requires version 13)
	version := r.Header.Get("Sec-WebSocket-Version")
	if version != "13" {
		w.Header().Set("Sec-WebSocket-Version", "13")
		http.Error(w, "unsupported WebSocket version", http.StatusUpgradeRequired)
		return
	}

	// Check origin using CheckOrigin callback or Origins list
	if srv.opts.CheckOrigin != nil {
		if !srv.opts.CheckOrigin(r) {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
	} else if len(srv.opts.Origins) > 0 {
		origin := r.Header.Get("Origin")
		allowed := false
		for _, o := range srv.opts.Origins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}
		if !allowed {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
	}

	// Authenticate if OnAuth is set
	var value any
	if srv.opts.OnAuth != nil {
		var err error
		value, err = srv.opts.OnAuth(r.Context(), r)
		if err != nil {
			http.Error(w, "authentication failed", http.StatusUnauthorized)
			return
		}
	}

	// Validate WebSocket key
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" || !ws.ValidateKey(key) {
		http.Error(w, "invalid Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}

	// Upgrade to WebSocket
	wsConn, err := ws.Upgrade(w, r, srv.opts.ReadLimit)
	if err != nil {
		return // Error already sent to client
	}

	// Create session
	session := newSession(srv.opts.IDGenerator(), value, srv.opts.QueueSize, srv)
	session.wsConn = wsConn
	srv.addSession(session)

	// Start write loop
	go srv.writeLoop(session)

	// Run read loop (blocking)
	readErr := srv.readLoop(r, session)

	// Close session with the read error as reason
	_ = session.closeWithError(readErr)
	srv.removeSession(session)

	// Call OnClose callback with actual close reason
	if srv.opts.OnClose != nil {
		srv.opts.OnClose(session, session.CloseError())
	}
}

// readLoop reads messages from the WebSocket and dispatches to OnMessage.
func (srv *Server) readLoop(r *http.Request, session *Session) error {
	ctx := r.Context()

	for {
		opcode, data, err := session.wsConn.ReadMessage()
		if err != nil {
			return err
		}

		// Handle control frames
		switch opcode {
		case ws.OpClose:
			return nil
		case ws.OpPing:
			_ = session.wsConn.WriteMessage(ws.OpPong, data)
			continue
		case ws.OpPong:
			continue
		}

		// Only process text/binary messages
		if opcode != ws.OpText && opcode != ws.OpBinary {
			continue
		}

		// Decode message to get topic and payload
		topic, payload, err := decodeMessage(data)
		if err != nil {
			continue // Skip invalid messages
		}

		// Dispatch to handler
		if srv.opts.OnMessage != nil {
			srv.opts.OnMessage(ctx, session, topic, payload)
		}
	}
}

// writeLoop sends messages from the session queue to the WebSocket.
func (srv *Server) writeLoop(session *Session) {
	for {
		select {
		case msg := <-session.sendCh:
			data, err := encodeMessage(msg)
			if err != nil {
				continue
			}
			if err := session.wsConn.WriteMessage(ws.OpText, data); err != nil {
				_ = session.closeWithError(err)
				return
			}
		case <-session.doneCh:
			// Send close frame
			_ = session.wsConn.WriteClose(ws.CloseNormal)
			return
		}
	}
}
