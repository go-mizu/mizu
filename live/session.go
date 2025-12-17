package live

import (
	"sync"
	"sync/atomic"
)

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
	closeErr error
	closeMu  sync.Mutex
	doneCh   chan struct{}
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
func (s *Session) closeWithError(err error) error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	s.closeMu.Lock()
	s.closeErr = err
	s.closeMu.Unlock()

	close(s.doneCh)
	return nil
}

// CloseError returns the error that caused the session to close, if any.
func (s *Session) CloseError() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	return s.closeErr
}

// Done returns a channel that is closed when the session is closed.
func (s *Session) Done() <-chan struct{} {
	return s.doneCh
}

// IsClosed returns true if the session has been closed.
func (s *Session) IsClosed() bool {
	return s.closed.Load()
}

// Topics returns a snapshot of the topics this session is subscribed to.
func (s *Session) Topics() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	topics := make([]string, 0, len(s.topics))
	for t := range s.topics {
		topics = append(topics, t)
	}
	return topics
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

// hasTopic returns true if the session is subscribed to the given topic.
func (s *Session) hasTopic(topic string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.topics[topic]
	return ok
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
