package async

const (
	defaultMaxBodySize = 1 << 20 // 1MB
	defaultBufferSize  = 16
)

// ErrorMapper converts Go errors to async error responses.
type ErrorMapper func(error) (code string, message string)

// Option configures async transport behavior.
type Option func(*options)

type options struct {
	maxBodySize  int64
	bufferSize   int
	errorMapper  ErrorMapper
	onConnect    func(clientID string)
	onDisconnect func(clientID string)
}

func defaultOptions() *options {
	return &options{
		maxBodySize: defaultMaxBodySize,
		bufferSize:  defaultBufferSize,
		errorMapper: defaultErrorMapper,
	}
}

func applyOptions(opts []Option) *options {
	o := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	return o
}

func defaultErrorMapper(err error) (string, string) {
	return "error", err.Error()
}

// WithErrorMapper sets custom error mapping.
func WithErrorMapper(m ErrorMapper) Option {
	return func(o *options) {
		if m != nil {
			o.errorMapper = m
		}
	}
}

// WithMaxBodySize limits request body size.
func WithMaxBodySize(n int64) Option {
	return func(o *options) {
		if n > 0 {
			o.maxBodySize = n
		}
	}
}

// WithBufferSize sets the per-client event buffer size.
func WithBufferSize(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.bufferSize = n
		}
	}
}

// WithOnConnect sets a callback when a client connects to SSE stream.
func WithOnConnect(fn func(clientID string)) Option {
	return func(o *options) {
		o.onConnect = fn
	}
}

// WithOnDisconnect sets a callback when a client disconnects.
func WithOnDisconnect(fn func(clientID string)) Option {
	return func(o *options) {
		o.onDisconnect = fn
	}
}
