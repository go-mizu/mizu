package contract

import (
	"net/http"
	"strings"
)

// pluralize naively pluralizes a word.
func pluralize(s string) string {
	if strings.HasSuffix(s, "s") {
		return s
	}
	return s + "s"
}

// restVerb infers an HTTP method from a method name.
func restVerb(name string) string {
	switch {
	case strings.HasPrefix(name, "Create"):
		return http.MethodPost
	case strings.HasPrefix(name, "Get"):
		return http.MethodGet
	case strings.HasPrefix(name, "List"):
		return http.MethodGet
	case strings.HasPrefix(name, "Update"):
		return http.MethodPut
	case strings.HasPrefix(name, "Delete"):
		return http.MethodDelete
	case strings.HasPrefix(name, "Patch"):
		return http.MethodPatch
	default:
		return http.MethodPost
	}
}

// needsID checks if a method needs an ID in the path.
func needsID(m *Method) bool {
	if m.Input == nil {
		return false
	}
	// convention: field named ID implies /{id}
	return strings.Contains(strings.ToLower(m.Input.Name), "id")
}
