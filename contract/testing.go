package contract

import (
	"context"
	"encoding/json"
)

// TestClient provides a transport-free way to call service methods.
// Useful for unit testing services without HTTP overhead.
type TestClient struct {
	service *Service
}

// NewTestClient creates a test client for a service.
func NewTestClient(svc *Service) *TestClient {
	return &TestClient{service: svc}
}

// Call invokes a method by name with the given input.
// The input can be nil for methods that don't require input.
func (c *TestClient) Call(ctx context.Context, method string, in any) (any, error) {
	m := c.service.Method(method)
	if m == nil {
		return nil, ErrNotFound("method not found: " + method)
	}

	return m.Invoker.Call(ctx, in)
}

// MustCall invokes a method and panics on error.
// Useful for test setup where errors indicate test bugs.
func (c *TestClient) MustCall(ctx context.Context, method string, in any) any {
	out, err := c.Call(ctx, method, in)
	if err != nil {
		panic("TestClient.MustCall: " + err.Error())
	}
	return out
}

// CallJSON invokes a method with JSON input and returns JSON output.
// Useful for testing JSON serialization round-trips.
func (c *TestClient) CallJSON(ctx context.Context, method string, inputJSON []byte) ([]byte, error) {
	m := c.service.Method(method)
	if m == nil {
		return nil, ErrNotFound("method not found: " + method)
	}

	var in any
	if m.HasInput() {
		in = m.NewInput()
		if len(inputJSON) > 0 {
			if err := json.Unmarshal(inputJSON, in); err != nil {
				return nil, ErrInvalidArgument("invalid JSON input: " + err.Error())
			}
		}
	}

	out, err := m.Invoker.Call(ctx, in)
	if err != nil {
		return nil, err
	}

	if out == nil {
		return nil, nil
	}

	return json.Marshal(out)
}

// Service returns the underlying service.
func (c *TestClient) Service() *Service {
	return c.service
}

// Methods returns all method names.
func (c *TestClient) Methods() []string {
	return c.service.MethodNames()
}

// HasMethod returns true if the service has the named method.
func (c *TestClient) HasMethod(name string) bool {
	return c.service.Method(name) != nil
}

// MockService provides a way to mock service methods for testing consumers.
type MockService struct {
	Name     string
	handlers map[string]MockHandler
}

// MockHandler is a function that handles a mock method call.
type MockHandler func(ctx context.Context, in any) (any, error)

// NewMockService creates a new mock service with the given name.
func NewMockService(name string) *MockService {
	return &MockService{
		Name:     name,
		handlers: make(map[string]MockHandler),
	}
}

// On registers a handler for a method.
func (m *MockService) On(method string, handler MockHandler) *MockService {
	m.handlers[method] = handler
	return m
}

// OnReturn registers a handler that returns the given values.
func (m *MockService) OnReturn(method string, out any, err error) *MockService {
	m.handlers[method] = func(ctx context.Context, in any) (any, error) {
		return out, err
	}
	return m
}

// OnError registers a handler that returns the given error.
func (m *MockService) OnError(method string, err error) *MockService {
	return m.OnReturn(method, nil, err)
}

// Call invokes a mock method.
func (m *MockService) Call(ctx context.Context, method string, in any) (any, error) {
	handler, ok := m.handlers[method]
	if !ok {
		return nil, ErrUnimplemented("mock not configured for: " + method)
	}
	return handler(ctx, in)
}

// AssertInput is a test helper that asserts the input matches expected.
type AssertInput struct {
	Method string
	Input  any
	Calls  []any
}

// RecordingMock records all calls for later assertion.
type RecordingMock struct {
	*MockService
	calls []RecordedCall
}

// RecordedCall represents a recorded method call.
type RecordedCall struct {
	Method string
	Input  any
	Output any
	Error  error
}

// NewRecordingMock creates a mock that records calls.
func NewRecordingMock(name string) *RecordingMock {
	return &RecordingMock{
		MockService: NewMockService(name),
		calls:       make([]RecordedCall, 0),
	}
}

// OnRecord registers a handler and records the call.
func (r *RecordingMock) OnRecord(method string, handler MockHandler) *RecordingMock {
	r.handlers[method] = func(ctx context.Context, in any) (any, error) {
		out, err := handler(ctx, in)
		r.calls = append(r.calls, RecordedCall{
			Method: method,
			Input:  in,
			Output: out,
			Error:  err,
		})
		return out, err
	}
	return r
}

// Calls returns all recorded calls.
func (r *RecordingMock) Calls() []RecordedCall {
	return r.calls
}

// CallsFor returns calls for a specific method.
func (r *RecordingMock) CallsFor(method string) []RecordedCall {
	var result []RecordedCall
	for _, c := range r.calls {
		if c.Method == method {
			result = append(result, c)
		}
	}
	return result
}

// CallCount returns the number of calls to a method.
func (r *RecordingMock) CallCount(method string) int {
	return len(r.CallsFor(method))
}

// Reset clears all recorded calls.
func (r *RecordingMock) Reset() {
	r.calls = r.calls[:0]
}

// TestService is a helper for testing services.
type TestService struct {
	*Service
	client *TestClient
}

// NewTestService registers a service and returns a test wrapper.
func NewTestService(name string, svc any) (*TestService, error) {
	s, err := Register(name, svc)
	if err != nil {
		return nil, err
	}
	return &TestService{
		Service: s,
		client:  NewTestClient(s),
	}, nil
}

// MustNewTestService registers a service and panics on error.
func MustNewTestService(name string, svc any) *TestService {
	ts, err := NewTestService(name, svc)
	if err != nil {
		panic("MustNewTestService: " + err.Error())
	}
	return ts
}

// Call invokes a method by name.
func (t *TestService) Call(ctx context.Context, method string, in any) (any, error) {
	return t.client.Call(ctx, method, in)
}

// MustCall invokes a method and panics on error.
func (t *TestService) MustCall(ctx context.Context, method string, in any) any {
	return t.client.MustCall(ctx, method, in)
}

// Client returns the test client.
func (t *TestService) Client() *TestClient {
	return t.client
}
