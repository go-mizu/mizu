package sync

// PokeBroker notifies live connections when data changes.
type PokeBroker interface {
	// Poke notifies watchers of a scope that data has changed.
	Poke(scope string, cursor uint64)
}

// Poke is the message sent to live connections.
type Poke struct {
	Scope  string `json:"scope"`
	Cursor uint64 `json:"cursor"`
}

// NopBroker is a no-op implementation of PokeBroker.
type NopBroker struct{}

// Poke does nothing.
func (NopBroker) Poke(scope string, cursor uint64) {}

// FuncBroker wraps a function as a PokeBroker.
type FuncBroker func(scope string, cursor uint64)

// Poke calls the underlying function.
func (f FuncBroker) Poke(scope string, cursor uint64) {
	f(scope, cursor)
}

// MultiBroker fans out pokes to multiple brokers.
type MultiBroker struct {
	brokers []PokeBroker
}

// NewMultiBroker creates a MultiBroker.
func NewMultiBroker(brokers ...PokeBroker) *MultiBroker {
	return &MultiBroker{brokers: brokers}
}

// Poke notifies all brokers.
func (m *MultiBroker) Poke(scope string, cursor uint64) {
	for _, b := range m.brokers {
		b.Poke(scope, cursor)
	}
}

// Add adds a broker.
func (m *MultiBroker) Add(broker PokeBroker) {
	m.brokers = append(m.brokers, broker)
}
