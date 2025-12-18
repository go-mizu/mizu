package todo

import (
	"context"
	"encoding/json"
	"fmt"

	contract "github.com/go-mizu/mizu/contract/v2"
)

// Invoker implements contract.Invoker for the todo service.
type Invoker struct {
	svc      *Service
	contract *contract.Service
}

// NewInvoker creates an invoker for the todo service.
func NewInvoker(svc *Service, c *contract.Service) *Invoker {
	return &Invoker{svc: svc, contract: c}
}

// Descriptor returns the service descriptor.
func (i *Invoker) Descriptor() *contract.Service {
	return i.contract
}

// Call dispatches a method call to the service.
func (i *Invoker) Call(ctx context.Context, resource, method string, in any) (any, error) {
	key := resource + "." + method
	switch key {
	case "todos.list":
		return i.svc.List(ctx)

	case "todos.create":
		req, err := decodeInput[CreateIn](in)
		if err != nil {
			return nil, err
		}
		return i.svc.Create(ctx, req)

	case "todos.get":
		req, err := decodeInput[GetIn](in)
		if err != nil {
			return nil, err
		}
		return i.svc.Get(ctx, req)

	case "todos.update":
		req, err := decodeInput[UpdateIn](in)
		if err != nil {
			return nil, err
		}
		return i.svc.Update(ctx, req)

	case "todos.delete":
		req, err := decodeInput[DeleteIn](in)
		if err != nil {
			return nil, err
		}
		if err := i.svc.Delete(ctx, req); err != nil {
			return nil, err
		}
		return nil, nil

	case "health.check":
		if err := i.svc.Health(ctx); err != nil {
			return &HealthStatus{Status: "unhealthy"}, nil
		}
		return &HealthStatus{Status: "ok"}, nil

	default:
		return nil, fmt.Errorf("unknown method: %s", key)
	}
}

// NewInput creates a new input instance for a method.
func (i *Invoker) NewInput(resource, method string) (any, error) {
	key := resource + "." + method
	switch key {
	case "todos.create":
		return &CreateIn{}, nil
	case "todos.get":
		return &GetIn{}, nil
	case "todos.update":
		return &UpdateIn{}, nil
	case "todos.delete":
		return &DeleteIn{}, nil
	default:
		return nil, nil
	}
}

// Stream is not supported for this service.
func (i *Invoker) Stream(ctx context.Context, resource, method string, in any) (contract.Stream, error) {
	return nil, contract.ErrUnsupported
}

func decodeInput[T any](in any) (*T, error) {
	if in == nil {
		return new(T), nil
	}
	if v, ok := in.(*T); ok {
		return v, nil
	}
	// Handle map[string]any from JSON decode
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
