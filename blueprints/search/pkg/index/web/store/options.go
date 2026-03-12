package store

import "time"

// Option configures Store behavior at open time.
type Option func(*options)

type options struct {
	busyTimeoutMS int
}

// WithBusyTimeout sets the DuckDB busy timeout.
func WithBusyTimeout(d time.Duration) Option {
	return func(o *options) { o.busyTimeoutMS = int(d.Milliseconds()) }
}
