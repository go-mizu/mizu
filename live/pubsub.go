package live

import (
	"sync"
)

// PubSub routes messages by topic.
type PubSub interface {
	// Subscribe adds a session to a topic.
	Subscribe(s *Session, topic string)

	// Unsubscribe removes a session from a topic.
	Unsubscribe(s *Session, topic string)

	// Publish sends a message to all subscribers of a topic.
	Publish(topic string, msg Message)

	// Who returns all sessions subscribed to a topic.
	Who(topic string) []*Session

	// Topics returns all active topics with at least one subscriber.
	Topics() []string

	// Count returns the number of subscribers for a topic.
	Count(topic string) int
}

// memPubSub is the default in-memory PubSub implementation.
type memPubSub struct {
	mu     sync.RWMutex
	topics map[string]map[*Session]struct{}
}

// newMemPubSub creates a new in-memory PubSub.
func newMemPubSub() *memPubSub {
	return &memPubSub{
		topics: make(map[string]map[*Session]struct{}),
	}
}

// Subscribe adds a session to a topic.
func (p *memPubSub) Subscribe(s *Session, topic string) {
	if s == nil || topic == "" {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	subs, ok := p.topics[topic]
	if !ok {
		subs = make(map[*Session]struct{})
		p.topics[topic] = subs
	}
	subs[s] = struct{}{}
	s.addTopic(topic)
}

// Unsubscribe removes a session from a topic.
func (p *memPubSub) Unsubscribe(s *Session, topic string) {
	if s == nil || topic == "" {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	subs, ok := p.topics[topic]
	if !ok {
		return
	}
	delete(subs, s)
	s.removeTopic(topic)

	// Clean up empty topics
	if len(subs) == 0 {
		delete(p.topics, topic)
	}
}

// UnsubscribeAll removes a session from all topics.
func (p *memPubSub) UnsubscribeAll(s *Session) {
	if s == nil {
		return
	}

	topics := s.clearTopics()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, topic := range topics {
		subs, ok := p.topics[topic]
		if !ok {
			continue
		}
		delete(subs, s)
		if len(subs) == 0 {
			delete(p.topics, topic)
		}
	}
}

// Publish sends a message to all subscribers of a topic.
// Messages are sent asynchronously; slow receivers don't block the publisher.
func (p *memPubSub) Publish(topic string, msg Message) {
	if topic == "" {
		return
	}

	// Set the topic on the message if not already set
	if msg.Topic == "" {
		msg.Topic = topic
	}

	p.mu.RLock()
	subs, ok := p.topics[topic]
	if !ok || len(subs) == 0 {
		p.mu.RUnlock()
		return
	}

	// Take a snapshot to avoid holding lock during sends
	sessions := make([]*Session, 0, len(subs))
	for s := range subs {
		sessions = append(sessions, s)
	}
	p.mu.RUnlock()

	// Send to all subscribers
	for _, s := range sessions {
		// Non-blocking send; if it fails (queue full), session will be closed
		_ = s.Send(msg)
	}
}

// Who returns all sessions subscribed to a topic.
func (p *memPubSub) Who(topic string) []*Session {
	p.mu.RLock()
	defer p.mu.RUnlock()

	subs, ok := p.topics[topic]
	if !ok {
		return nil
	}

	sessions := make([]*Session, 0, len(subs))
	for s := range subs {
		sessions = append(sessions, s)
	}
	return sessions
}

// Topics returns all active topics with at least one subscriber.
func (p *memPubSub) Topics() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	topics := make([]string, 0, len(p.topics))
	for t := range p.topics {
		topics = append(topics, t)
	}
	return topics
}

// Count returns the number of subscribers for a topic.
func (p *memPubSub) Count(topic string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	subs, ok := p.topics[topic]
	if !ok {
		return 0
	}
	return len(subs)
}

// Ensure memPubSub implements PubSub.
var _ PubSub = (*memPubSub)(nil)
