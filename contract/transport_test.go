package contract

import (
	"context"
	"testing"
)

func TestTrimJSONSpace(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"no whitespace", []byte(`{"key":"value"}`), `{"key":"value"}`},
		{"leading spaces", []byte(`   {"key":"value"}`), `{"key":"value"}`},
		{"trailing spaces", []byte(`{"key":"value"}   `), `{"key":"value"}`},
		{"both sides", []byte(`  {"key":"value"}  `), `{"key":"value"}`},
		{"tabs and newlines", []byte("\t\n{}\n\t"), `{}`},
		{"empty", []byte(""), ""},
		{"only whitespace", []byte("   \t\n  "), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrimJSONSpace(tt.input)
			if string(got) != tt.want {
				t.Errorf("TrimJSONSpace(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsJSONNull(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  bool
	}{
		{"null lowercase", []byte("null"), true},
		{"null uppercase", []byte("NULL"), true},
		{"null mixed case", []byte("NuLl"), true},
		{"null with spaces", []byte("  null  "), true},
		{"empty", []byte(""), false},
		{"empty object", []byte("{}"), false},
		{"string", []byte(`"null"`), false},
		{"number", []byte("0"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsJSONNull(tt.input)
			if got != tt.want {
				t.Errorf("IsJSONNull(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSafeErrorString(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil error", nil, ""},
		{"simple error", ErrNotFound("not found"), "not found"},
		{"internal error", ErrInternal("oops"), "oops"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeErrorString(tt.err)
			if got != tt.want {
				t.Errorf("SafeErrorString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestServiceResolver(t *testing.T) {
	svc, err := Register("test", &TodoService{})
	if err != nil {
		t.Fatalf("register error: %v", err)
	}

	resolver := &ServiceResolver{Service: svc}

	tests := []struct {
		name    string
		input   string
		wantNil bool
	}{
		{"method only", "Create", false},
		{"service.method", "test.Create", false},
		{"wrong service", "other.Create", true},
		{"unknown method", "Unknown", true},
		{"empty", "", true},
		{"spaces only", "   ", true},
		{"invalid format", "a.b.c", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := resolver.Resolve(tt.input)
			if tt.wantNil {
				if m != nil {
					t.Errorf("Resolve(%q) should be nil", tt.input)
				}
			} else {
				if m == nil {
					t.Errorf("Resolve(%q) should not be nil", tt.input)
				}
			}
		})
	}
}

func TestDefaultInvoker(t *testing.T) {
	svc, err := Register("test", &TodoService{})
	if err != nil {
		t.Fatalf("register error: %v", err)
	}

	invoker := &DefaultInvoker{}
	ctx := context.Background()

	tests := []struct {
		name    string
		method  string
		input   []byte
		wantErr bool
	}{
		{
			name:   "valid input",
			method: "Create",
			input:  []byte(`{"title":"Test"}`),
		},
		{
			name:   "empty input for method with params",
			method: "Create",
			input:  []byte(``),
		},
		{
			name:   "null input",
			method: "Create",
			input:  []byte(`null`),
		},
		{
			name:    "invalid JSON",
			method:  "Create",
			input:   []byte(`{invalid}`),
			wantErr: true,
		},
		{
			name:    "array input",
			method:  "Create",
			input:   []byte(`["not","object"]`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := svc.Method(tt.method)
			if m == nil {
				t.Fatalf("method %q not found", tt.method)
			}

			_, err := invoker.Invoke(ctx, m, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestTransportError(t *testing.T) {
	err := &TransportError{
		Code:    -32600,
		Message: "Invalid Request",
		Data:    map[string]string{"reason": "test"},
	}

	if err.Error() != "Invalid Request" {
		t.Errorf("Error() = %q, want %q", err.Error(), "Invalid Request")
	}

	// Test with no message
	err2 := &TransportError{Code: -32600}
	if !contains(err2.Error(), "-32600") {
		t.Errorf("Error() should contain code, got %q", err2.Error())
	}

	// Test Unwrap
	cause := ErrInternal("cause")
	err3 := &TransportError{
		Code:  -32603,
		Cause: cause,
	}
	if err3.Unwrap() != cause {
		t.Error("Unwrap() should return cause")
	}
}

func TestApplyTransportOptions(t *testing.T) {
	svc, _ := Register("test", &TodoService{})

	// Default options
	opts := ApplyTransportOptions(svc)
	if opts.Resolver == nil {
		t.Error("expected default resolver")
	}
	if opts.Invoker == nil {
		t.Error("expected default invoker")
	}

	// Custom resolver
	customResolver := &ServiceResolver{Service: svc}
	opts = ApplyTransportOptions(svc, WithResolver(customResolver))
	if opts.Resolver != customResolver {
		t.Error("expected custom resolver")
	}

	// Custom invoker
	customInvoker := &DefaultInvoker{}
	opts = ApplyTransportOptions(svc, WithTransportInvoker(customInvoker))
	if opts.Invoker != customInvoker {
		t.Error("expected custom invoker")
	}

	// Middleware
	mw := func(next MethodInvoker) MethodInvoker { return next }
	opts = ApplyTransportOptions(svc, WithTransportMiddleware(mw))
	if len(opts.Middleware) != 1 {
		t.Error("expected middleware")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
