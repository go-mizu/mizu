package live

import (
	"sync"
	"time"
)

// Session holds the state for a live connection.
type Session[T any] struct {
	// ID is the unique session identifier.
	ID string

	// State is the user-defined typed state.
	State T

	// Flash provides flash message support.
	Flash Flash

	// UserID is set from auth middleware (optional).
	UserID string

	// dirty tracks which regions need re-rendering.
	dirty *DirtySet

	// regions stores last rendered HTML for each region.
	regions map[string]string

	// commands queued for client.
	commands []Command

	// created is when the session was created.
	created time.Time

	// lastSeen is the last activity time.
	lastSeen time.Time
}

// Mark marks regions as dirty, triggering re-render.
func (s *Session[T]) Mark(ids ...string) {
	for _, id := range ids {
		s.dirty.Add(id)
	}
}

// MarkAll marks all regions as dirty.
func (s *Session[T]) MarkAll() {
	s.dirty.AddAll()
}

// ReplaceState replaces the entire state and marks all regions dirty.
func (s *Session[T]) ReplaceState(next T) {
	s.State = next
	s.MarkAll()
}

// IsDirty checks if any regions are marked dirty.
func (s *Session[T]) IsDirty() bool {
	return !s.dirty.IsEmpty()
}

// Push queues a client-side command.
func (s *Session[T]) Push(cmd Command) {
	s.commands = append(s.commands, cmd)
}

// typed returns the session as an any for the wrapper.
func (s *Session[T]) typed() any {
	return s
}

// sessionBase wraps any Session[T] for internal use.
type sessionBase struct {
	session any
	mu      sync.Mutex
}

func (b *sessionBase) typed() any {
	return b.session
}

func (b *sessionBase) lock()   { b.mu.Lock() }
func (b *sessionBase) unlock() { b.mu.Unlock() }

func (b *sessionBase) getID() string {
	switch s := b.session.(type) {
	case interface{ getID() string }:
		return s.getID()
	default:
		// Use reflection-free approach via generic accessor
		return getSessionID(b.session)
	}
}

func (b *sessionBase) getDirty() *DirtySet {
	return getSessionDirty(b.session)
}

func (b *sessionBase) getRegions() map[string]string {
	return getSessionRegions(b.session)
}

func (b *sessionBase) setRegion(id, html string) {
	getSessionRegions(b.session)[id] = html
}

func (b *sessionBase) getCommands() []Command {
	return getSessionCommands(b.session)
}

func (b *sessionBase) clearCommands() {
	clearSessionCommands(b.session)
}

func (b *sessionBase) getFlash() *Flash {
	return getSessionFlash(b.session)
}

func (b *sessionBase) setCreated(t time.Time) {
	setSessionCreated(b.session, t)
}

func (b *sessionBase) setLastSeen(t time.Time) {
	setSessionLastSeen(b.session, t)
}

func (b *sessionBase) getLastSeen() time.Time {
	return getSessionLastSeen(b.session)
}

// Helper functions to access Session[T] fields without reflection.
// These use type assertions on a common interface.

type sessionAccessor interface {
	getIDField() string
	getDirtyField() *DirtySet
	getRegionsField() map[string]string
	getCommandsField() []Command
	clearCommandsField()
	getFlashField() *Flash
	setCreatedField(time.Time)
	setLastSeenField(time.Time)
	getLastSeenField() time.Time
}

func (s *Session[T]) getIDField() string              { return s.ID }
func (s *Session[T]) getDirtyField() *DirtySet        { return s.dirty }
func (s *Session[T]) getRegionsField() map[string]string { return s.regions }
func (s *Session[T]) getCommandsField() []Command     { return s.commands }
func (s *Session[T]) clearCommandsField()             { s.commands = nil }
func (s *Session[T]) getFlashField() *Flash           { return &s.Flash }
func (s *Session[T]) setCreatedField(t time.Time)     { s.created = t }
func (s *Session[T]) setLastSeenField(t time.Time)    { s.lastSeen = t }
func (s *Session[T]) getLastSeenField() time.Time     { return s.lastSeen }

func getSessionID(s any) string {
	if a, ok := s.(sessionAccessor); ok {
		return a.getIDField()
	}
	return ""
}

func getSessionDirty(s any) *DirtySet {
	if a, ok := s.(sessionAccessor); ok {
		return a.getDirtyField()
	}
	return nil
}

func getSessionRegions(s any) map[string]string {
	if a, ok := s.(sessionAccessor); ok {
		return a.getRegionsField()
	}
	return nil
}

func getSessionCommands(s any) []Command {
	if a, ok := s.(sessionAccessor); ok {
		return a.getCommandsField()
	}
	return nil
}

func clearSessionCommands(s any) {
	if a, ok := s.(sessionAccessor); ok {
		a.clearCommandsField()
	}
}

func getSessionFlash(s any) *Flash {
	if a, ok := s.(sessionAccessor); ok {
		return a.getFlashField()
	}
	return nil
}

func setSessionCreated(s any, t time.Time) {
	if a, ok := s.(sessionAccessor); ok {
		a.setCreatedField(t)
	}
}

func setSessionLastSeen(s any, t time.Time) {
	if a, ok := s.(sessionAccessor); ok {
		a.setLastSeenField(t)
	}
}

func getSessionLastSeen(s any) time.Time {
	if a, ok := s.(sessionAccessor); ok {
		return a.getLastSeenField()
	}
	return time.Time{}
}

// DirtySet tracks which regions need re-rendering.
type DirtySet struct {
	mu      sync.RWMutex
	all     bool
	regions map[string]struct{}
}

func newDirtySet() *DirtySet {
	return &DirtySet{
		regions: make(map[string]struct{}),
	}
}

// Add marks a region as dirty.
func (d *DirtySet) Add(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.regions[id] = struct{}{}
}

// AddAll marks all regions as dirty.
func (d *DirtySet) AddAll() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.all = true
}

// Has checks if a region is dirty.
func (d *DirtySet) Has(id string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.all {
		return true
	}
	_, ok := d.regions[id]
	return ok
}

// IsAll returns true if all regions are marked dirty.
func (d *DirtySet) IsAll() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.all
}

// IsEmpty returns true if no regions are dirty.
func (d *DirtySet) IsEmpty() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return !d.all && len(d.regions) == 0
}

// List returns all dirty region IDs.
func (d *DirtySet) List() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make([]string, 0, len(d.regions))
	for id := range d.regions {
		result = append(result, id)
	}
	return result
}

// Clear resets the dirty set.
func (d *DirtySet) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.all = false
	d.regions = make(map[string]struct{})
}

// Flash provides flash message support.
type Flash struct {
	Success []string
	Error   []string
	Warning []string
	Info    []string
}

// AddSuccess adds a success flash message.
func (f *Flash) AddSuccess(msg string) {
	f.Success = append(f.Success, msg)
}

// AddError adds an error flash message.
func (f *Flash) AddError(msg string) {
	f.Error = append(f.Error, msg)
}

// AddWarning adds a warning flash message.
func (f *Flash) AddWarning(msg string) {
	f.Warning = append(f.Warning, msg)
}

// AddInfo adds an info flash message.
func (f *Flash) AddInfo(msg string) {
	f.Info = append(f.Info, msg)
}

// Clear removes all flash messages.
func (f *Flash) Clear() {
	f.Success = nil
	f.Error = nil
	f.Warning = nil
	f.Info = nil
}

// IsEmpty returns true if there are no flash messages.
func (f *Flash) IsEmpty() bool {
	return len(f.Success) == 0 && len(f.Error) == 0 &&
		len(f.Warning) == 0 && len(f.Info) == 0
}
