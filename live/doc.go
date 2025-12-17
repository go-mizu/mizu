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
//	            server.PubSub().Subscribe(s, msg.Topic)
//	            s.Send(live.Message{Type: "ack", Topic: msg.Topic, Ref: msg.Ref})
//	        case "unsubscribe":
//	            server.PubSub().Unsubscribe(s, msg.Topic)
//	        case "publish":
//	            server.PubSub().Publish(msg.Topic, msg)
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
//
// # Integration with sync
//
// The live package can accelerate sync by notifying clients when data changes:
//
//	notifier := live.SyncNotifier(liveServer, "sync:")
//	engine := sync.New(sync.Options{
//	    // ...
//	    Notify: notifier,
//	})
//
// Clients subscribed to "sync:{scope}" topics receive notifications
// when that scope's cursor advances, prompting immediate pull.
package live
