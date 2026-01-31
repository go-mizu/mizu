package tools

import "context"

// Tool defines a tool that the LLM can invoke.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any // JSON Schema for the input parameters
	Execute     func(ctx context.Context, input map[string]any) (string, error)
}

// Registry holds available tools keyed by name.
type Registry struct {
	tools map[string]*Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]*Tool)}
}

// Register adds a tool to the registry. Overwrites if name already exists.
func (r *Registry) Register(t *Tool) {
	r.tools[t.Name] = t
}

// Get returns a tool by name, or nil if not found.
func (r *Registry) Get(name string) *Tool {
	return r.tools[name]
}

// All returns all registered tools.
func (r *Registry) All() []*Tool {
	result := make([]*Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// Definitions returns tool definitions in the Anthropic API format.
// Each definition has "name", "description", and "input_schema" fields.
func (r *Registry) Definitions() []map[string]any {
	defs := make([]map[string]any, 0, len(r.tools))
	for _, t := range r.tools {
		def := map[string]any{
			"name":         t.Name,
			"description":  t.Description,
			"input_schema": t.InputSchema,
		}
		defs = append(defs, def)
	}
	return defs
}
