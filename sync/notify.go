package sync

// Notifier is an optional integration hook.
// It is called after changes are committed to notify
// external systems (e.g., live/websocket connections).
type Notifier interface {
	Notify(scope string, cursor uint64)
}

// NotifierFunc wraps a function as a Notifier.
type NotifierFunc func(scope string, cursor uint64)

// Notify implements Notifier.
func (f NotifierFunc) Notify(scope string, cursor uint64) {
	f(scope, cursor)
}

// MultiNotifier fans out notifications to multiple notifiers.
type MultiNotifier []Notifier

// Notify calls all notifiers.
func (m MultiNotifier) Notify(scope string, cursor uint64) {
	for _, n := range m {
		n.Notify(scope, cursor)
	}
}
