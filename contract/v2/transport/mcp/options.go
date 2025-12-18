package mcp

// Option configures MCP transport behavior.
type Option func(*options)

// ErrorMapper converts Go errors to MCP tool errors.
// Returns: isError flag, error message
type ErrorMapper func(error) (isError bool, message string)

type options struct {
	maxBodySize   int64
	errorMapper   ErrorMapper
	serverName    string
	serverVersion string
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

// WithServerInfo sets server name and version for capability negotiation.
func WithServerInfo(name, version string) Option {
	return func(o *options) {
		o.serverName = name
		o.serverVersion = version
	}
}

// WithErrorMapper sets custom error-to-MCP-error mapping.
// Default maps all errors to isError=true with err.Error() message.
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

// defaultErrorMapper returns isError=true for all errors.
func defaultErrorMapper(err error) (bool, string) {
	return true, err.Error()
}
