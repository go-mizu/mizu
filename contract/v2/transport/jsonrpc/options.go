package jsonrpc

// Option configures JSON-RPC transport behavior.
type Option func(*options)

// ErrorMapper converts Go errors to JSON-RPC error responses.
// Returns: code (JSON-RPC error code), message, data (optional)
type ErrorMapper func(error) (code int, message string, data any)

type options struct {
	maxBodySize int64
	errorMapper ErrorMapper
}

func defaultOptions() *options {
	return &options{
		maxBodySize: 1 << 20, // 1MB
		errorMapper: defaultErrorMapper,
	}
}

func applyOptions(opts []Option) *options {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithErrorMapper sets custom error-to-JSON-RPC-error mapping.
// Default maps all errors to code -32000 (server error).
func WithErrorMapper(m ErrorMapper) Option {
	return func(o *options) {
		if m != nil {
			o.errorMapper = m
		}
	}
}

// WithMaxBodySize limits request body size (default: 1MB).
func WithMaxBodySize(n int64) Option {
	return func(o *options) {
		if n > 0 {
			o.maxBodySize = n
		}
	}
}

// defaultErrorMapper returns -32000 (server error) for all errors.
func defaultErrorMapper(err error) (int, string, any) {
	return errServer, "server error", err.Error()
}
