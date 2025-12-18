package rest

import "github.com/go-mizu/mizu"

// Route represents a single HTTP endpoint from a contract.
type Route struct {
	Method   string       // HTTP method (GET, POST, etc.)
	Path     string       // URL path with {param} placeholders
	Resource string       // Contract resource name
	Name     string       // Contract method name
	Handler  mizu.Handler // Handler function
}

// mizuRoute is the internal route representation with additional metadata.
// Named to avoid conflict with legacy route type in server.go.
type mizuRoute struct {
	httpMethod string
	path       string
	resource   string
	method     string
	pathParams []string
	hasInput   bool
	hasOutput  bool
}
