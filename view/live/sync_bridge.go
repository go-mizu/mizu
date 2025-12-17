package live

// Poke represents a sync poke message for internal routing.
type Poke struct {
	Scope  string
	Cursor uint64
}

// SyncPokeBroker implements sync.PokeBroker using live's PubSub.
// This bridges the sync package to the live package, allowing sync
// to notify live connections when data changes.
type SyncPokeBroker struct {
	pubsub PubSub
}

// NewSyncPokeBroker creates a poke broker backed by live pubsub.
func NewSyncPokeBroker(pubsub PubSub) *SyncPokeBroker {
	return &SyncPokeBroker{pubsub: pubsub}
}

// Poke sends a poke message to all watchers of a scope.
// This satisfies the sync.PokeBroker interface.
func (b *SyncPokeBroker) Poke(scope string, cursor uint64) {
	// The scope becomes the pubsub topic
	b.pubsub.Publish(scope, Poke{
		Scope:  scope,
		Cursor: cursor,
	})
}
