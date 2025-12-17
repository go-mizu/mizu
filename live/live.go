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
//	    OnAuth: func(ctx context.Context, r *http.Request) (live.Meta, error) {
//	        token := r.Header.Get("Authorization")
//	        if !validateToken(token) {
//	            return nil, live.ErrAuthFailed
//	        }
//	        return live.Meta{"user_id": getUserID(token)}, nil
//	    },
//	    OnMessage: func(ctx context.Context, s *live.Session, msg live.Message) {
//	        switch msg.Type {
//	        case "subscribe":
//	            server.Subscribe(s, msg.Topic)
//	            s.Send(live.Message{Type: "ack", Topic: msg.Topic, Ref: msg.Ref})
//	        case "unsubscribe":
//	            server.Unsubscribe(s, msg.Topic)
//	        case "publish":
//	            server.Publish(msg.Topic, msg)
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
//  5. Read loop decodes messages and calls OnMessage
//  6. Write loop sends queued messages to client
//  7. On disconnect: cleanup subscriptions, call OnClose
//
// # Backpressure
//
// Each session has a bounded send queue (default 256 messages).
// If the queue fills up, the session is closed to protect server health.
// This is intentional: slow clients should not affect other clients.
package live

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // G505: SHA1 is required by WebSocket protocol (RFC 6455)
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
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

	// ErrAuthFailed is returned when authentication fails.
	ErrAuthFailed = errors.New("live: authentication failed")

	// ErrInvalidVersion is returned when the WebSocket version is not supported.
	ErrInvalidVersion = errors.New("live: unsupported WebSocket version")

	// ErrMessageTooLarge is returned when a message exceeds the read limit.
	ErrMessageTooLarge = errors.New("live: message too large")

	// ErrProtocolError is returned for WebSocket protocol violations.
	ErrProtocolError = errors.New("live: protocol error")
)

// -----------------------------------------------------------------------------
// Types
// -----------------------------------------------------------------------------

// Message is the transport envelope for all live communications.
// It carries typed messages between clients and server.
type Message struct {
	// Type identifies the message purpose (e.g., "subscribe", "message", "ack").
	// The live package does not interpret this field; higher layers define semantics.
	Type string `json:"type"`

	// Topic is the routing key for pub/sub operations.
	// Empty for messages that don't target a specific topic.
	Topic string `json:"topic,omitempty"`

	// Ref is a client-generated reference for correlating request/response pairs.
	// Servers should echo Ref in acknowledgments.
	Ref string `json:"ref,omitempty"`

	// Body contains the message payload as opaque bytes.
	// Higher layers define the schema. When using JSON codec, this is base64-encoded.
	Body []byte `json:"body,omitempty"`
}

// Meta holds authenticated connection metadata.
// Populated by OnAuth callback and accessible via Session.Meta().
// The live package treats this as read-only after creation.
type Meta map[string]any

// Get returns the value for key, or nil if not present.
func (m Meta) Get(key string) any {
	if m == nil {
		return nil
	}
	return m[key]
}

// GetString returns the string value for key, or empty string if not present or not a string.
func (m Meta) GetString(key string) string {
	v, _ := m.Get(key).(string)
	return v
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
	// Messages exceeding this limit cause the connection to be closed
	// with a 1009 (message too big) close code.
	ReadLimit int

	// OnAuth is called to authenticate new connections.
	// Return Meta with user info on success, or an error to reject.
	// If nil, all connections are accepted without authentication.
	OnAuth func(ctx context.Context, r *http.Request) (Meta, error)

	// OnMessage is called when a message is received from a client.
	// This is where you implement subscribe/unsubscribe/publish logic.
	// The live package does not interpret message types; higher layers
	// define the protocol (e.g., "subscribe", "publish", "unsubscribe").
	OnMessage func(ctx context.Context, s *Session, msg Message)

	// OnClose is called when a session is closed.
	// The error may be nil for clean closes.
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
	opts         Options
	pubsub       *memPubSub
	sessions     sync.Map // map[string]*Session
	sessionCount atomic.Int64
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

// Publish sends a message to all subscribers of a topic.
func (srv *Server) Publish(topic string, msg Message) {
	srv.pubsub.publish(topic, msg)
}

// Broadcast sends a message to all connected sessions.
func (srv *Server) Broadcast(msg Message) {
	srv.sessions.Range(func(_, value any) bool {
		if s, ok := value.(*Session); ok {
			_ = s.Send(msg)
		}
		return true
	})
}

// Subscribe adds a session to a topic.
func (srv *Server) Subscribe(s *Session, topic string) {
	srv.pubsub.subscribe(s, topic)
}

// Unsubscribe removes a session from a topic.
func (srv *Server) Unsubscribe(s *Session, topic string) {
	srv.pubsub.unsubscribe(s, topic)
}

// SessionCount returns the number of connected sessions.
// This is an O(1) operation using an atomic counter.
func (srv *Server) SessionCount() int {
	return int(srv.sessionCount.Load())
}

// addSession registers a session with the server.
func (srv *Server) addSession(s *Session) {
	srv.sessions.Store(s.id, s)
	srv.sessionCount.Add(1)
}

// removeSession unregisters a session from the server.
func (srv *Server) removeSession(s *Session) {
	srv.sessions.Delete(s.id)
	srv.sessionCount.Add(-1)
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
	meta     Meta
	sendCh   chan Message
	server   *Server
	topics   map[string]struct{}
	mu       sync.RWMutex
	closed   atomic.Bool
	doneCh   chan struct{}
	conn     net.Conn // underlying connection for cleanup
}

// newSession creates a new session with the given ID and metadata.
func newSession(id string, meta Meta, queueSize int, server *Server) *Session {
	if queueSize <= 0 {
		queueSize = defaultQueueSize
	}
	return &Session{
		id:     id,
		meta:   meta,
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

// Meta returns the session's metadata set during authentication.
func (s *Session) Meta() Meta {
	return s.meta
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
		s.closeWithError(ErrQueueFull)
		return ErrQueueFull
	}
}

// Close gracefully closes the session.
// It is safe to call multiple times.
func (s *Session) Close() error {
	return s.closeWithError(nil)
}

// closeWithError closes the session with the given error.
// It also closes the underlying connection to unblock any blocked reads.
func (s *Session) closeWithError(err error) error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}
	_ = err // error stored for internal use only
	close(s.doneCh)
	// Close the underlying connection to unblock readLoop
	if s.conn != nil {
		_ = s.conn.Close()
	}
	return nil
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

// publish sends a message to all subscribers of a topic.
// Messages are sent asynchronously; slow receivers don't block the publisher.
func (p *memPubSub) publish(topic string, msg Message) {
	if topic == "" {
		return
	}

	// Set the topic on the message if not already set
	if msg.Topic == "" {
		msg.Topic = topic
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

	// Send to all subscribers
	for _, s := range sessions {
		// Non-blocking send; if it fails (queue full), session will be closed
		_ = s.Send(msg)
	}
}

// count returns the number of subscribers for a topic.
func (p *memPubSub) count(topic string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	subs, ok := p.topics[topic]
	if !ok {
		return 0
	}
	return len(subs)
}

// -----------------------------------------------------------------------------
// Codec (internal)
// -----------------------------------------------------------------------------

// encodeMessage serializes a message to JSON.
func encodeMessage(m Message) ([]byte, error) {
	return json.Marshal(m)
}

// decodeMessage deserializes JSON to a message.
func decodeMessage(data []byte) (Message, error) {
	var m Message
	if err := json.Unmarshal(data, &m); err != nil {
		return Message{}, err
	}
	return m, nil
}

// -----------------------------------------------------------------------------
// WebSocket
// -----------------------------------------------------------------------------

const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// WebSocket message types (opcodes)
const (
	wsContinuation  = 0
	wsTextMessage   = 1
	wsBinaryMessage = 2
	wsCloseMessage  = 8
	wsPingMessage   = 9
	wsPongMessage   = 10
)

// WebSocket close codes per RFC 6455
const (
	wsCloseNormal        = 1000 // Normal closure
	wsCloseGoingAway     = 1001 // Endpoint going away
	wsCloseProtocolError = 1002 // Protocol error
	wsCloseUnsupported   = 1003 // Unsupported data type
	wsCloseTooLarge      = 1009 // Message too big
)

// wsConn represents a WebSocket connection.
type wsConn struct {
	conn      net.Conn
	reader    *bufio.Reader
	writer    *bufio.Writer
	mu        sync.Mutex
	readLimit int // maximum message size
}

// handleConn handles a new WebSocket connection.
//
//nolint:cyclop // Connection handling requires multiple steps
func (srv *Server) handleConn(w http.ResponseWriter, r *http.Request) {
	// Check if it's a WebSocket upgrade request
	if !isWebSocketUpgrade(r) {
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
	var meta Meta
	if srv.opts.OnAuth != nil {
		var err error
		meta, err = srv.opts.OnAuth(r.Context(), r)
		if err != nil {
			http.Error(w, "authentication failed", http.StatusUnauthorized)
			return
		}
	}

	// Get and validate WebSocket key
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}
	if !validateWebSocketKey(key) {
		http.Error(w, "invalid Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}

	// Calculate accept key
	acceptKey := computeAcceptKey(key)

	// Hijack connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket: hijack not supported", http.StatusInternalServerError)
		return
	}

	conn, bufrw, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send upgrade response
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"

	_, _ = bufrw.WriteString(response)
	_ = bufrw.Flush()

	// Create WebSocket connection wrapper
	ws := &wsConn{
		conn:      conn,
		reader:    bufrw.Reader,
		writer:    bufrw.Writer,
		readLimit: srv.opts.ReadLimit,
	}

	// Create session and store connection for cleanup
	session := newSession(srv.opts.IDGenerator(), meta, srv.opts.QueueSize, srv)
	session.conn = conn
	srv.addSession(session)

	// Start write loop
	go srv.writeLoop(session, ws)

	// Run read loop (blocking)
	readErr := srv.readLoop(r, session, ws)

	// Cleanup (closeWithError already closes the underlying conn)
	session.closeWithError(readErr)
	srv.removeSession(session)

	// Call OnClose callback
	if srv.opts.OnClose != nil {
		srv.opts.OnClose(session, readErr)
	}
}

// validateWebSocketKey validates that the Sec-WebSocket-Key is valid base64
// and decodes to exactly 16 bytes (per RFC 6455).
func validateWebSocketKey(key string) bool {
	decoded, err := base64.StdEncoding.DecodeString(key)
	return err == nil && len(decoded) == 16
}

// readLoop reads messages from the WebSocket and dispatches to OnMessage.
func (srv *Server) readLoop(r *http.Request, session *Session, ws *wsConn) error {
	ctx := r.Context()

	for {
		msgType, data, err := ws.readMessage()
		if err != nil {
			return err
		}

		// Handle control frames
		switch msgType {
		case wsCloseMessage:
			return nil
		case wsPingMessage:
			_ = ws.writeMessage(wsPongMessage, data)
			continue
		case wsPongMessage:
			continue
		}

		// Only process text/binary messages
		if msgType != wsTextMessage && msgType != wsBinaryMessage {
			continue
		}

		// Decode message
		msg, err := decodeMessage(data)
		if err != nil {
			continue // Skip invalid messages
		}

		// Dispatch to handler
		if srv.opts.OnMessage != nil {
			srv.opts.OnMessage(ctx, session, msg)
		}
	}
}

// writeLoop sends messages from the session queue to the WebSocket.
func (srv *Server) writeLoop(session *Session, ws *wsConn) {
	for {
		select {
		case msg := <-session.sendCh:
			data, err := encodeMessage(msg)
			if err != nil {
				continue
			}
			if err := ws.writeMessage(wsTextMessage, data); err != nil {
				session.closeWithError(err)
				return
			}
		case <-session.doneCh:
			// Send close frame
			_ = ws.writeMessage(wsCloseMessage, []byte{0x03, 0xe8}) // 1000 normal closure
			return
		}
	}
}

// isWebSocketUpgrade checks if request is a WebSocket upgrade.
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// computeAcceptKey computes the Sec-WebSocket-Accept header value.
func computeAcceptKey(key string) string {
	h := sha1.New() //nolint:gosec // G401: SHA1 required by WebSocket protocol (RFC 6455)
	h.Write([]byte(key + websocketGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// readMessage reads a WebSocket frame.
// It enforces RFC 6455 requirements:
// - Client frames must be masked
// - Control frames must have payload ≤ 125 bytes and FIN=1
// - Fragmentation is rejected (FIN=0 or opcode=0 continuation)
// - Message size is limited by readLimit
//
//nolint:cyclop // WebSocket frame parsing requires multiple format checks
func (ws *wsConn) readMessage() (messageType int, data []byte, err error) {
	// Read first byte (FIN + RSV + opcode)
	b, err := ws.reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}

	fin := b&0x80 != 0
	opcode := int(b & 0x0F)

	// Reject continuation frames (we don't support fragmentation)
	if opcode == wsContinuation {
		return 0, nil, ErrProtocolError
	}

	// Reject fragmented data frames (FIN=0)
	// Control frames (8-10) must always have FIN=1
	isControlFrame := opcode >= wsCloseMessage
	if !fin {
		return 0, nil, ErrProtocolError
	}

	// Read second byte (MASK + payload length)
	b, err = ws.reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}

	masked := b&0x80 != 0
	length := int(b & 0x7F)

	// RFC 6455: Client frames MUST be masked
	if !masked {
		return 0, nil, ErrProtocolError
	}

	// Control frames must have payload length ≤ 125
	if isControlFrame && length > 125 {
		return 0, nil, ErrProtocolError
	}

	// Extended payload length
	switch length {
	case 126:
		lenBytes := make([]byte, 2)
		if _, err := io.ReadFull(ws.reader, lenBytes); err != nil {
			return 0, nil, err
		}
		length = int(binary.BigEndian.Uint16(lenBytes))
	case 127:
		lenBytes := make([]byte, 8)
		if _, err := io.ReadFull(ws.reader, lenBytes); err != nil {
			return 0, nil, err
		}
		length64 := binary.BigEndian.Uint64(lenBytes)
		// Check for overflow and read limit
		if length64 > uint64(ws.readLimit) {
			return 0, nil, ErrMessageTooLarge
		}
		length = int(length64)
	}

	// Check read limit for all payload lengths
	if ws.readLimit > 0 && length > ws.readLimit {
		return 0, nil, ErrMessageTooLarge
	}

	// Read masking key (always present since we require masked frames)
	mask := make([]byte, 4)
	if _, err := io.ReadFull(ws.reader, mask); err != nil {
		return 0, nil, err
	}

	// Read payload
	data = make([]byte, length)
	if _, err := io.ReadFull(ws.reader, data); err != nil {
		return 0, nil, err
	}

	// Unmask data
	for i := range data {
		data[i] ^= mask[i%4]
	}

	return opcode, data, nil
}

// writeMessage writes a WebSocket frame.
func (ws *wsConn) writeMessage(messageType int, data []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	// First byte: FIN + opcode
	_ = ws.writer.WriteByte(0x80 | byte(messageType))

	// Second byte: payload length (server never masks)
	length := len(data)
	if length <= 125 {
		_ = ws.writer.WriteByte(byte(length))
	} else if length <= 65535 {
		_ = ws.writer.WriteByte(126)
		_ = ws.writer.WriteByte(byte(length >> 8))
		_ = ws.writer.WriteByte(byte(length))
	} else {
		_ = ws.writer.WriteByte(127)
		for i := 7; i >= 0; i-- {
			_ = ws.writer.WriteByte(byte(length >> (8 * i)))
		}
	}

	// Write payload
	_, _ = ws.writer.Write(data)

	return ws.writer.Flush()
}
