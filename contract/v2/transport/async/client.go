// contract/transport/async/client.go
package async

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/contract"
)

// Client calls an async transport surface over a Broker.
// It supports request-reply and notifications.
type Client struct {
	Svc *contract.Service
	Br  Broker

	// ReplyPrefix is used to build per-call reply topics.
	// Example: "reply.myapp" gives "reply.myapp.<random>".
	ReplyPrefix string

	// Timeout is used when waiting for reply in Call.
	Timeout time.Duration
}

// NewClient constructs a client.
func NewClient(svc *contract.Service, br Broker) (*Client, error) {
	if svc == nil {
		return nil, errors.New("async: nil service")
	}
	if br == nil {
		return nil, errors.New("async: nil broker")
	}
	return &Client{
		Svc:         svc,
		Br:          br,
		ReplyPrefix: "reply",
		Timeout:     30 * time.Second,
	}, nil
}

// Notify publishes a notification (no reply expected).
func (c *Client) Notify(ctx context.Context, resource, method string, in any) error {
	if c.Svc == nil || c.Br == nil {
		return errors.New("async: not initialized")
	}
	desc := c.Svc.Method(resource, method)
	if desc == nil {
		return fmt.Errorf("async: unknown method %s.%s", resource, method)
	}

	reqTopic := RequestTopic(c.Svc.Name, resource, method)
	var params json.RawMessage
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		params = b
	}

	env := Envelope{
		ID:     newID(),
		Params: params,
		// ReplyTo empty => notification
	}
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return c.Br.Publish(ctx, reqTopic, b)
}

// Call publishes a request and waits for a response (request-reply).
// out must be a pointer to the output struct, unless the method has no output.
func (c *Client) Call(ctx context.Context, resource, method string, in any, out any) error {
	if c.Svc == nil || c.Br == nil {
		return errors.New("async: not initialized")
	}
	desc := c.Svc.Method(resource, method)
	if desc == nil {
		return fmt.Errorf("async: unknown method %s.%s", resource, method)
	}
	if desc.Output == "" && out != nil {
		return errors.New("async: out must be nil for methods with no output")
	}

	reqTopic := RequestTopic(c.Svc.Name, resource, method)
	replyTopic := c.replyTopic(resource, method)

	var (
		mu    sync.Mutex
		done  = make(chan Envelope, 1)
		unsub func()
	)

	unsub, err := c.Br.Subscribe(ctx, replyTopic, func(payload []byte) {
		var env Envelope
		if uErr := json.Unmarshal(payload, &env); uErr != nil {
			return
		}
		mu.Lock()
		select {
		case done <- env:
		default:
		}
		mu.Unlock()
	})
	if err != nil {
		return err
	}
	defer unsub()

	var params json.RawMessage
	if in != nil {
		b, mErr := json.Marshal(in)
		if mErr != nil {
			return mErr
		}
		params = b
	}

	id := newID()
	req := Envelope{
		ID:      id,
		Params:  params,
		ReplyTo: replyTopic,
	}
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if err := c.Br.Publish(ctx, reqTopic, b); err != nil {
		return err
	}

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	select {
	case resp := <-done:
		if resp.Error != nil {
			return fmt.Errorf("async: %s.%s: %s: %s", resource, method, resp.Error.Code, resp.Error.Message)
		}
		if desc.Output == "" || out == nil {
			return nil
		}
		if len(resp.Result) == 0 || string(resp.Result) == "null" {
			return nil
		}
		return json.Unmarshal(resp.Result, out)

	case <-time.After(timeout):
		return fmt.Errorf("async: timeout waiting for %s.%s", resource, method)

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) replyTopic(resource, method string) string {
	prefix := strings.TrimSpace(c.ReplyPrefix)
	if prefix == "" {
		prefix = "reply"
	}
	return prefix + "." + sanitizeTopic(c.Svc.Name) + "." + sanitizeTopic(resource) + "." + sanitizeTopic(method) + "." + newID()
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
