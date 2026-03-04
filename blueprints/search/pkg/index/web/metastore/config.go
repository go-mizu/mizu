package metastore

import "time"

// Options controls backend-specific connection behavior.
type Options struct {
	BusyTimeout time.Duration
	JournalMode string
}
