package live

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	stdsync "sync"

	mizusync "github.com/go-mizu/mizu/sync"
)

const (
	defaultQueueSize = 256
)

// Options configures the Server.
type Options struct {
	// Codec for message encoding. Default: JSONCodec.
	Codec Codec

	// QueueSize is the per-session send queue size. Default: 256.
	// When the queue fills up, the session is closed.
	QueueSize int

	// OnAuth is called to authenticate new connections.
	// Return Meta with user info on success, or an error to reject.
	// If nil, all connections are accepted without authentication.
	OnAuth func(ctx context.Context, r *http.Request) (Meta, error)

	// OnMessage is called when a message is received from a client.
	// This is where you implement subscribe/unsubscribe/publish logic.
	OnMessage func(ctx context.Context, s *Session, msg Message)

	// OnClose is called when a session is closed.
	// The error may be nil for clean closes.
	OnClose func(s *Session, err error)

	// Origins is a list of allowed origins for WebSocket connections.
	// If empty, all origins are allowed.
	Origins []string

	// IDGenerator generates unique session IDs.
	// If nil, random hex IDs are generated.
	IDGenerator func() string
}

// Server owns sessions, pubsub state, and the WebSocket handler.
type Server struct {
	opts     Options
	pubsub   *memPubSub
	sessions stdsync.Map // map[string]*Session
}

// New creates a new live server with the given options.
func New(opts Options) *Server {
	if opts.Codec == nil {
		opts.Codec = JSONCodec{}
	}
	if opts.QueueSize <= 0 {
		opts.QueueSize = defaultQueueSize
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
	srv.pubsub.Publish(topic, msg)
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

// Session returns the session with the given ID, or nil if not found.
func (srv *Server) Session(id string) *Session {
	if v, ok := srv.sessions.Load(id); ok {
		return v.(*Session)
	}
	return nil
}

// Sessions returns a snapshot of all connected sessions.
func (srv *Server) Sessions() []*Session {
	var sessions []*Session
	srv.sessions.Range(func(_, value any) bool {
		if s, ok := value.(*Session); ok {
			sessions = append(sessions, s)
		}
		return true
	})
	return sessions
}

// SessionCount returns the number of connected sessions.
func (srv *Server) SessionCount() int {
	count := 0
	srv.sessions.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// PubSub returns the server's PubSub instance.
func (srv *Server) PubSub() PubSub {
	return srv.pubsub
}

// Options returns the server's configuration.
func (srv *Server) Options() Options {
	return srv.opts
}

// addSession registers a session with the server.
func (srv *Server) addSession(s *Session) {
	srv.sessions.Store(s.id, s)
}

// removeSession unregisters a session from the server.
func (srv *Server) removeSession(s *Session) {
	srv.sessions.Delete(s.id)
	srv.pubsub.UnsubscribeAll(s)
}

// generateID generates a random hex session ID.
func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// SyncNotifier returns a sync.Notifier that publishes to live topics.
// When a sync scope advances, it publishes to "{prefix}{scope}" topic.
func SyncNotifier(srv *Server, prefix string) mizusync.Notifier {
	return mizusync.NotifierFunc(func(scope string, cursor uint64) {
		topic := prefix + scope
		srv.Publish(topic, Message{
			Type:  "sync",
			Topic: topic,
			Body:  []byte(`{"cursor":` + uintToString(cursor) + `}`),
		})
	})
}

// uintToString converts uint64 to string without importing strconv.
func uintToString(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
