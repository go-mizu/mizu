package ai

import (
	"context"
	"sync"

	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
)

// Capability represents what a model can do.
type Capability string

const (
	CapabilityText       Capability = "text"
	CapabilityVision     Capability = "vision"
	CapabilityEmbeddings Capability = "embeddings"
	CapabilityVoice      Capability = "voice"
)

// ModelInfo describes an available model.
type ModelInfo struct {
	ID           string       `json:"id"`
	Provider     string       `json:"provider"`
	Name         string       `json:"name"`
	Description  string       `json:"description,omitempty"`
	Capabilities []Capability `json:"capabilities"`
	ContextSize  int          `json:"context_size"`
	Speed        string       `json:"speed"` // fast, balanced, thorough
	IsDefault    bool         `json:"is_default,omitempty"`
	Available    bool         `json:"available"`
}

// ModelRegistry manages available models.
type ModelRegistry struct {
	mu        sync.RWMutex
	models    map[string]*ModelInfo
	providers map[string]llm.Provider
	defaults  map[Capability]string // Default model per capability
}

// NewModelRegistry creates a new model registry.
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		models:    make(map[string]*ModelInfo),
		providers: make(map[string]llm.Provider),
		defaults:  make(map[Capability]string),
	}
}

// RegisterModel adds a model to the registry.
func (r *ModelRegistry) RegisterModel(info ModelInfo, provider llm.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.models[info.ID] = &info
	r.providers[info.ID] = provider

	// Set as default for capabilities if no default exists
	for _, cap := range info.Capabilities {
		if _, exists := r.defaults[cap]; !exists {
			r.defaults[cap] = info.ID
		}
	}
}

// SetDefault sets the default model for a capability.
func (r *ModelRegistry) SetDefault(cap Capability, modelID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.models[modelID]; exists {
		r.defaults[cap] = modelID
	}
}

// GetModel returns a model by ID.
func (r *ModelRegistry) GetModel(id string) (*ModelInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, ok := r.models[id]
	return info, ok
}

// GetProvider returns the provider for a model.
func (r *ModelRegistry) GetProvider(modelID string) (llm.Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[modelID]
	return provider, ok
}

// GetDefaultModel returns the default model for a capability.
func (r *ModelRegistry) GetDefaultModel(cap Capability) (*ModelInfo, llm.Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modelID, exists := r.defaults[cap]
	if !exists {
		return nil, nil, false
	}

	info := r.models[modelID]
	provider := r.providers[modelID]
	return info, provider, true
}

// ListModels returns all registered models.
func (r *ModelRegistry) ListModels() []ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models := make([]ModelInfo, 0, len(r.models))
	for _, info := range r.models {
		m := *info
		// Mark defaults
		for cap, defaultID := range r.defaults {
			if defaultID == m.ID {
				for _, c := range m.Capabilities {
					if c == cap {
						m.IsDefault = true
						break
					}
				}
			}
		}
		models = append(models, m)
	}
	return models
}

// ListModelsByCapability returns models with a specific capability.
func (r *ModelRegistry) ListModelsByCapability(cap Capability) []ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var models []ModelInfo
	for _, info := range r.models {
		for _, c := range info.Capabilities {
			if c == cap {
				m := *info
				if r.defaults[cap] == m.ID {
					m.IsDefault = true
				}
				models = append(models, m)
				break
			}
		}
	}
	return models
}

// CheckHealth checks if a model is available.
func (r *ModelRegistry) CheckHealth(ctx context.Context, modelID string) bool {
	provider, ok := r.GetProvider(modelID)
	if !ok {
		return false
	}
	return provider.Ping(ctx) == nil
}

// UpdateAvailability updates the availability status of all models.
func (r *ModelRegistry) UpdateAvailability(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, info := range r.models {
		provider := r.providers[id]
		if provider != nil {
			info.Available = provider.Ping(ctx) == nil
		} else {
			info.Available = false
		}
	}
}

// GetModelForQuery returns the appropriate model for a query.
// It considers the requested model, mode, and falls back to defaults.
func (r *ModelRegistry) GetModelForQuery(modelID string, mode Mode, hasImages bool) (llm.Provider, *ModelInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// If specific model requested, use it
	if modelID != "" {
		if provider, ok := r.providers[modelID]; ok {
			return provider, r.models[modelID], nil
		}
	}

	// If query has images, use vision model
	if hasImages {
		if defaultID, ok := r.defaults[CapabilityVision]; ok {
			return r.providers[defaultID], r.models[defaultID], nil
		}
	}

	// Use default text model
	if defaultID, ok := r.defaults[CapabilityText]; ok {
		return r.providers[defaultID], r.models[defaultID], nil
	}

	// Fallback to any available model
	for id, provider := range r.providers {
		return provider, r.models[id], nil
	}

	return nil, nil, ErrNoModelAvailable
}

// Errors
var ErrNoModelAvailable = &ModelError{Message: "no model available"}

type ModelError struct {
	Message string
}

func (e *ModelError) Error() string {
	return e.Message
}
