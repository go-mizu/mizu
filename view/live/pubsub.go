package live

import (
	"sync"
)

// PubSub enables server-to-client messaging.
type PubSub interface {
	// Subscribe adds a session to topics.
	Subscribe(sessionID string, topics ...string) error

	// Unsubscribe removes a session from topics.
	Unsubscribe(sessionID string, topics ...string) error

	// UnsubscribeAll removes a session from all topics.
	UnsubscribeAll(sessionID string) error

	// Publish sends a message to all subscribers of a topic.
	Publish(topic string, msg any) error

	// Broadcast sends a message to all sessions.
	Broadcast(msg any) error

	// register associates a session with a message channel.
	register(sessionID string, ch chan<- any)

	// unregister removes a session's message channel.
	unregister(sessionID string)
}

// InmemPubSub is an in-memory pubsub implementation.
type InmemPubSub struct {
	mu sync.RWMutex

	// topics maps topic -> set of session IDs.
	topics map[string]map[string]struct{}

	// sessions maps session ID -> message channel.
	sessions map[string]chan<- any

	// subscriptions maps session ID -> set of topics.
	subscriptions map[string]map[string]struct{}
}

// NewInmemPubSub creates a new in-memory pubsub.
func NewInmemPubSub() *InmemPubSub {
	return &InmemPubSub{
		topics:        make(map[string]map[string]struct{}),
		sessions:      make(map[string]chan<- any),
		subscriptions: make(map[string]map[string]struct{}),
	}
}

// Subscribe adds a session to topics.
func (p *InmemPubSub) Subscribe(sessionID string, topics ...string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, topic := range topics {
		if p.topics[topic] == nil {
			p.topics[topic] = make(map[string]struct{})
		}
		p.topics[topic][sessionID] = struct{}{}

		if p.subscriptions[sessionID] == nil {
			p.subscriptions[sessionID] = make(map[string]struct{})
		}
		p.subscriptions[sessionID][topic] = struct{}{}
	}
	return nil
}

// Unsubscribe removes a session from topics.
func (p *InmemPubSub) Unsubscribe(sessionID string, topics ...string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, topic := range topics {
		if subs, ok := p.topics[topic]; ok {
			delete(subs, sessionID)
			if len(subs) == 0 {
				delete(p.topics, topic)
			}
		}

		if subs, ok := p.subscriptions[sessionID]; ok {
			delete(subs, topic)
		}
	}
	return nil
}

// UnsubscribeAll removes a session from all topics.
func (p *InmemPubSub) UnsubscribeAll(sessionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if topics, ok := p.subscriptions[sessionID]; ok {
		for topic := range topics {
			if subs, ok := p.topics[topic]; ok {
				delete(subs, sessionID)
				if len(subs) == 0 {
					delete(p.topics, topic)
				}
			}
		}
		delete(p.subscriptions, sessionID)
	}
	return nil
}

// Publish sends a message to all subscribers of a topic.
func (p *InmemPubSub) Publish(topic string, msg any) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	subs, ok := p.topics[topic]
	if !ok {
		return nil
	}

	for sessionID := range subs {
		if ch, ok := p.sessions[sessionID]; ok {
			select {
			case ch <- msg:
			default:
				// Channel full, skip this message.
			}
		}
	}
	return nil
}

// Broadcast sends a message to all sessions.
func (p *InmemPubSub) Broadcast(msg any) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, ch := range p.sessions {
		select {
		case ch <- msg:
		default:
			// Channel full, skip this message.
		}
	}
	return nil
}

// register associates a session with a message channel.
func (p *InmemPubSub) register(sessionID string, ch chan<- any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sessions[sessionID] = ch
}

// unregister removes a session's message channel.
func (p *InmemPubSub) unregister(sessionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.sessions, sessionID)
}

// TopicSubscribers returns the number of subscribers for a topic.
func (p *InmemPubSub) TopicSubscribers(topic string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.topics[topic])
}

// SessionCount returns the number of active sessions.
func (p *InmemPubSub) SessionCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.sessions)
}
