package agents

import (
	"fmt"
	"sync"
)

// Registry manages agent registration and discovery.
type Registry struct {
	agents map[AgentType]Agent
	mu     sync.RWMutex
}

// NewRegistry creates a new agent registry.
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[AgentType]Agent),
	}
}

// Register adds an agent to the registry.
func (r *Registry) Register(agent Agent) error {
	if agent == nil {
		return fmt.Errorf("agent cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[agent.Type()]; exists {
		return fmt.Errorf("agent %s already registered", agent.Type())
	}

	r.agents[agent.Type()] = agent
	return nil
}

// Unregister removes an agent from the registry.
func (r *Registry) Unregister(agentType AgentType) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[agentType]; !exists {
		return fmt.Errorf("agent %s not found", agentType)
	}

	delete(r.agents, agentType)
	return nil
}

// Get retrieves an agent by type.
func (r *Registry) Get(agentType AgentType) (Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, ok := r.agents[agentType]
	return agent, ok
}

// List returns all registered agents.
func (r *Registry) List() []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]AgentInfo, 0, len(r.agents))
	for _, agent := range r.agents {
		infos = append(infos, AgentInfo{
			Type:         agent.Type(),
			Name:         agent.Name(),
			Description:  agent.Description(),
			Capabilities: agent.Capabilities(),
			Status:       "active",
		})
	}
	return infos
}

// FindByCapability finds agents that can handle an action.
func (r *Registry) FindByCapability(action string) []Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Agent
	for _, agent := range r.agents {
		if agent.CanHandle(action) {
			result = append(result, agent)
		}
	}
	return result
}

// Count returns the number of registered agents.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}
