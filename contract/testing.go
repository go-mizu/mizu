package contract

import (
	"context"
	"encoding/json"
)

// Client is a test client for calling service methods.
type Client struct {
	service *Service
}

// NewClient creates a test client for a service.
func NewClient(svc *Service) *Client {
	return &Client{service: svc}
}

// Call invokes a method by name with the given input.
func (c *Client) Call(ctx context.Context, method string, in any) (any, error) {
	m := c.service.Method(method)
	if m == nil {
		return nil, NewError(NotFound, "method not found: "+method)
	}
	return m.Call(ctx, in)
}

// CallJSON invokes a method with JSON input and returns JSON output.
func (c *Client) CallJSON(ctx context.Context, method string, input []byte) ([]byte, error) {
	m := c.service.Method(method)
	if m == nil {
		return nil, NewError(NotFound, "method not found: "+method)
	}

	var in any
	if m.HasInput() {
		in = m.NewInput()
		if len(input) > 0 {
			if err := json.Unmarshal(input, in); err != nil {
				return nil, NewError(InvalidArgument, "invalid JSON: "+err.Error())
			}
		}
	}

	out, err := m.Call(ctx, in)
	if err != nil {
		return nil, err
	}

	if out == nil {
		return nil, nil
	}

	return json.Marshal(out)
}

// Service returns the underlying service.
func (c *Client) Service() *Service {
	return c.service
}

// Methods returns all method names.
func (c *Client) Methods() []string {
	return c.service.MethodNames()
}
