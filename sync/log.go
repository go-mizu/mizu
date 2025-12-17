package sync

import "context"

// Log records ordered changes and serves them by cursor.
type Log interface {
	// Append adds changes to the log and returns the final cursor.
	// Changes are assigned sequential cursor values starting after
	// the current cursor. The returned cursor is the last assigned.
	Append(ctx context.Context, scope string, changes []Change) (uint64, error)

	// Since returns changes after the given cursor for a scope.
	// Returns up to limit changes.
	Since(ctx context.Context, scope string, cursor uint64, limit int) ([]Change, error)

	// Cursor returns the current latest cursor for a scope.
	Cursor(ctx context.Context, scope string) (uint64, error)

	// Trim removes changes before the given cursor (for compaction).
	Trim(ctx context.Context, scope string, before uint64) error
}
