package view

import (
	"html/template"
	"sync"
)

// renderContext holds the state for a single render operation.
type renderContext struct {
	engine *Engine

	mu       sync.Mutex
	slots    map[string]string
	stacks   map[string][]string
	childBuf string
}

// newRenderContext creates a new render context.
func newRenderContext(e *Engine) *renderContext {
	return &renderContext{
		engine: e,
		slots:  make(map[string]string),
		stacks: make(map[string][]string),
	}
}

// setSlot sets the content for a named slot.
func (ctx *renderContext) setSlot(name, content string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.slots[name] = content
}

// slot returns the content for a named slot, or the default if not set.
func (ctx *renderContext) slot(name string, defaults ...any) template.HTML {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if content, ok := ctx.slots[name]; ok {
		return template.HTML(content)
	}

	// Return default if provided
	if len(defaults) > 0 {
		switch v := defaults[0].(type) {
		case string:
			return template.HTML(v)
		case template.HTML:
			return v
		default:
			return ""
		}
	}

	return ""
}

// push adds content to a stack.
func (ctx *renderContext) push(name, content string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.stacks[name] = append(ctx.stacks[name], content)
}

// stack returns all content for a stack, optionally deduplicating.
func (ctx *renderContext) stack(name string, dedupe bool) template.HTML {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	items, ok := ctx.stacks[name]
	if !ok || len(items) == 0 {
		return ""
	}

	if !dedupe {
		var result string
		for _, item := range items {
			result += item
		}
		return template.HTML(result)
	}

	// Deduplicate
	seen := make(map[string]bool)
	var result string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result += item
		}
	}
	return template.HTML(result)
}

// setChildren sets the children content for component rendering.
func (ctx *renderContext) setChildren(content string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.childBuf = content
}

// children returns the children content.
func (ctx *renderContext) children() template.HTML {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return template.HTML(ctx.childBuf)
}
