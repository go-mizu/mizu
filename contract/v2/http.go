package contract

import (
	"reflect"
	"strings"
)

// inferHTTPBinding infers HTTP method and path from method name conventions.
func inferHTTPBinding(methodName, resourceName string, inputType reflect.Type) *MethodHTTP {
	lowerName := strings.ToLower(methodName)

	// Determine HTTP method based on naming conventions
	var httpMethod string
	var pathSuffix string
	var needsID bool

	switch {
	case strings.HasPrefix(lowerName, "create"),
		strings.HasPrefix(lowerName, "add"),
		strings.HasPrefix(lowerName, "new"):
		httpMethod = "POST"
		pathSuffix = ""

	case strings.HasPrefix(lowerName, "list"),
		strings.HasPrefix(lowerName, "all"),
		strings.HasPrefix(lowerName, "search"),
		strings.HasPrefix(lowerName, "find") && strings.HasSuffix(lowerName, "s"):
		httpMethod = "GET"
		pathSuffix = ""

	case strings.HasPrefix(lowerName, "get"),
		strings.HasPrefix(lowerName, "find"),
		strings.HasPrefix(lowerName, "fetch"),
		strings.HasPrefix(lowerName, "read"):
		httpMethod = "GET"
		needsID = true

	case strings.HasPrefix(lowerName, "update"),
		strings.HasPrefix(lowerName, "edit"),
		strings.HasPrefix(lowerName, "modify"),
		strings.HasPrefix(lowerName, "set"):
		httpMethod = "PUT"
		needsID = true

	case strings.HasPrefix(lowerName, "delete"),
		strings.HasPrefix(lowerName, "remove"):
		httpMethod = "DELETE"
		needsID = true

	case strings.HasPrefix(lowerName, "patch"):
		httpMethod = "PATCH"
		needsID = true

	default:
		// Default to POST with method name as action
		httpMethod = "POST"
		pathSuffix = "/" + toLowerSnake(methodName)
	}

	// Build path
	path := "/" + resourceName
	if needsID {
		// Check input type for ID field to determine path parameter name
		idParam := extractIDParam(inputType)
		if idParam != "" {
			path += "/{" + idParam + "}"
		}
	}
	path += pathSuffix

	return &MethodHTTP{
		Method: httpMethod,
		Path:   path,
	}
}

// extractIDParam finds the ID parameter from an input type.
func extractIDParam(t reflect.Type) string {
	if t == nil {
		return "id"
	}

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return "id"
	}

	// Look for fields that are likely IDs
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		// Check for explicit path tag
		if pathTag := f.Tag.Get("path"); pathTag != "" {
			return pathTag
		}

		// Check field name patterns
		name := strings.ToLower(f.Name)
		if name == "id" {
			return getJSONName(f)
		}
		if strings.HasSuffix(name, "id") {
			return getJSONName(f)
		}
	}

	// Default to "id"
	return "id"
}

// extractPathParamsFromType extracts path parameters from an input type.
// Returns a map of param name -> field name.
func extractPathParamsFromType(t reflect.Type) map[string]string {
	result := make(map[string]string)

	if t == nil {
		return result
	}

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return result
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		// Check for explicit path tag
		if pathTag := f.Tag.Get("path"); pathTag != "" {
			result[pathTag] = f.Name
		}
	}

	return result
}

// extractQueryParams extracts query parameters from an input type.
// Returns field names that should be query parameters (not in path).
func extractQueryParams(t reflect.Type, pathParams map[string]string) []string {
	var result []string

	if t == nil {
		return result
	}

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return result
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		// Skip if it's a path parameter
		isPath := false
		for _, fieldName := range pathParams {
			if fieldName == f.Name {
				isPath = true
				break
			}
		}

		if !isPath {
			// Check for explicit query tag
			if queryTag := f.Tag.Get("query"); queryTag != "" {
				result = append(result, queryTag)
			} else {
				result = append(result, getJSONName(f))
			}
		}
	}

	return result
}
