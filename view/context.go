package view

import "context"

// getEngine retrieves the engine from the context.
func getEngine(ctx context.Context) *Engine {
	if e, ok := ctx.Value(engineKey{}).(*Engine); ok {
		return e
	}
	return nil
}
