package rest

import "net/http"

// ErrorMapper maps an error to HTTP status code, error code, and message.
type ErrorMapper func(error) (status int, code string, message string)

// DefaultErrorMapper returns 400 Bad Request for all errors.
func DefaultErrorMapper(err error) (int, string, string) {
	return http.StatusBadRequest, "request_error", err.Error()
}

// options holds configuration for REST transport.
type options struct {
	errorMapper   ErrorMapper
	maxBodySize   int64
	strictRouting bool
}

func defaultOptions() *options {
	return &options{
		errorMapper: DefaultErrorMapper,
		maxBodySize: 1 << 20, // 1MB
	}
}

// Option configures REST transport behavior.
type Option func(*options)

// WithErrorMapper sets a custom error mapper.
func WithErrorMapper(m ErrorMapper) Option {
	return func(o *options) {
		if m != nil {
			o.errorMapper = m
		}
	}
}

// WithMaxBodySize sets the maximum request body size in bytes.
// Default is 1MB.
func WithMaxBodySize(n int64) Option {
	return func(o *options) {
		if n > 0 {
			o.maxBodySize = n
		}
	}
}

// WithStrictRouting disables query parameter fallback for non-GET requests.
// By default, query params can supplement JSON body for any method.
func WithStrictRouting() Option {
	return func(o *options) {
		o.strictRouting = true
	}
}

func applyOptions(opts []Option) *options {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return o
}
