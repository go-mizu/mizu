// Package sdk defines minimal shared primitives for SDK generation.
//
// This package is intentionally tiny and stable.
// It provides language-agnostic structures that generators emit,
// without embedding language-specific semantics.
package sdk

// File represents a generated source file.
type File struct {
	// Path is the relative path of the file within the SDK output.
	// Examples:
	//   "client.go"
	//   "responses/client.go"
	//   "types/response.go"
	Path string

	// Content is the full textual content of the file.
	Content string
}
