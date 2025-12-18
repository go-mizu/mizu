// contract/transport/async/server.go
package async

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	contract "github.com/go-mizu/mizu/contract/v2"
)

// Broker is the minimal pubsub surface required by this transport.
//
// Implementations can be backed by NATS, Redis PubSub, Kafka, MQTT, WebSocket fanout, etc.
type Broker interface {
	Publish(ctx context.Context, topic string, payload []byte) error

	// Subscribe registers a handler for messages on topic.
	// The returned function must cancel the subscription.
	Subscribe(ctx context.Context, topic string, handler func(payload []byte)) (unsubscribe func(), err error)
}

// Server exposes a contract.Invoker over an async message broker.
//
// For each method with an Async binding, Server subscribes to the request topic.
// Requests can be notifications (no reply_to) or calls (reply_to set).
type Server struct {
	inv contract.Invoker
	svc *contract.Service
	br  Broker

	unsubs []func()
}

// NewServer creates a server bound to a broker.
func NewServer(inv contract.Invoker, br Broker) (*Server, error) {
	if inv == nil {
		return nil, errors.New("async: nil invoker")
	}
	if br == nil {
		return nil, errors.New("async: nil broker")
	}
	svc := inv.Descriptor()
	if svc == nil {
		return nil, errors.New("async: nil descriptor")
	}
	return &Server{inv: inv, svc: svc, br: br}, nil
}

// Start subscribes to all request topics derived from the service descriptor.
// Call Stop to unsubscribe.
func (s *Server) Start(ctx context.Context) error {
	if s.svc == nil {
		return errors.New("async: nil service")
	}

	for _, res := range s.svc.Resources {
		if res == nil {
			continue
		}
		for _, m := range res.Methods {
			if m == nil {
				continue
			}

			reqTopic := RequestTopic(s.svc.Name, res.Name, m.Name)
			resource := res.Name
			method := m.Name

			unsub, err := s.br.Subscribe(ctx, reqTopic, func(payload []byte) {
				s.handleOne(ctx, resource, method, payload)
			})
			if err != nil {
				s.Stop()
				return fmt.Errorf("async: subscribe %s: %w", reqTopic, err)
			}
			s.unsubs = append(s.unsubs, unsub)
		}
	}
	if len(s.unsubs) == 0 {
		return errors.New("async: no resources/methods to subscribe")
	}
	return nil
}

// Stop unsubscribes from all topics registered by Start.
func (s *Server) Stop() {
	for i := len(s.unsubs) - 1; i >= 0; i-- {
		if s.unsubs[i] != nil {
			s.unsubs[i]()
		}
	}
	s.unsubs = nil
}

func (s *Server) handleOne(ctx context.Context, resource string, method string, payload []byte) {
	var req Envelope
	if err := json.Unmarshal(payload, &req); err != nil {
		// Cannot reply reliably without reply_to. Drop.
		return
	}

	// Allocate and decode input if required.
	desc := s.svc.Method(resource, method)
	if desc == nil {
		s.replyIfNeeded(ctx, req, Envelope{
			ID:    req.ID,
			Error: &Error{Code: "method_not_found", Message: "unknown method"},
		})
		return
	}

	var in any
	if desc.Input != "" {
		var err error
		in, err = s.inv.NewInput(resource, method)
		if err != nil || in == nil {
			s.replyIfNeeded(ctx, req, Envelope{
				ID:    req.ID,
				Error: &Error{Code: "new_input_failed", Message: "failed to allocate input"},
			})
			return
		}
		if len(req.Params) != 0 && string(req.Params) != "null" {
			if err := json.Unmarshal(req.Params, in); err != nil {
				s.replyIfNeeded(ctx, req, Envelope{
					ID:    req.ID,
					Error: &Error{Code: "invalid_params", Message: err.Error()},
				})
				return
			}
		}
	}

	out, err := s.inv.Call(ctx, resource, method, in)
	if err != nil {
		s.replyIfNeeded(ctx, req, Envelope{
			ID:    req.ID,
			Error: &Error{Code: "call_failed", Message: err.Error()},
		})
		return
	}

	// Notification: no reply
	if strings.TrimSpace(req.ReplyTo) == "" {
		return
	}

	// No output: return null result
	if desc.Output == "" {
		s.replyIfNeeded(ctx, req, Envelope{
			ID:     req.ID,
			Result: json.RawMessage("null"),
		})
		return
	}

	b, mErr := json.Marshal(out)
	if mErr != nil {
		s.replyIfNeeded(ctx, req, Envelope{
			ID:    req.ID,
			Error: &Error{Code: "marshal_failed", Message: mErr.Error()},
		})
		return
	}
	s.replyIfNeeded(ctx, req, Envelope{
		ID:     req.ID,
		Result: json.RawMessage(b),
	})
}

func (s *Server) replyIfNeeded(ctx context.Context, req Envelope, resp Envelope) {
	if strings.TrimSpace(req.ReplyTo) == "" {
		return
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_ = s.br.Publish(ctx, req.ReplyTo, b)
}

// RequestTopic returns the canonical request topic name.
func RequestTopic(service, resource, method string) string {
	return sanitizeTopic(service) + "." + sanitizeTopic(resource) + "." + sanitizeTopic(method) + ".request"
}

// ResponseTopic returns the canonical response topic name.
func ResponseTopic(service, resource, method string) string {
	return sanitizeTopic(service) + "." + sanitizeTopic(resource) + "." + sanitizeTopic(method) + ".response"
}

func sanitizeTopic(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}

// Envelope is the wire message for async request-reply.
//
// - Params carries request input encoded as JSON object
// - Result carries method output encoded as JSON
// - ReplyTo indicates where the server should publish the response
type Envelope struct {
	ID      string          `json:"id,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	ReplyTo string          `json:"reply_to,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error is a structured error payload for async responses.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
