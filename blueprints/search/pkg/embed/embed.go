// Package embed defines minimal interfaces for embedding generation.
//
// The package provides a small abstraction over embedding models.
// Implementations may call remote APIs, run local models, or use
// embedded inference libraries.
package embed

import "context"

// Model represents an embedding model.
//
// A Model converts text or binary content into a fixed dimension
// vector representation suitable for vector search or clustering.
//
// Implementations may wrap:
//   - Remote APIs (OpenAI, Voyage, Cohere, etc.).
//   - Local inference runtimes.
//   - Embedded models.
type Model interface {
	// Name returns the model identifier.
	//
	// This value is implementation defined and may correspond
	// to a remote model name or local model ID.
	Name() string

	// Dimension returns the embedding vector dimension.
	//
	// Drivers should return a constant dimension for the model.
	Dimension() int

	// Embed encodes a batch of inputs into embedding vectors.
	//
	// Contract:
	//   - The returned slice must have the same length as inputs.
	//   - Each vector must have Dimension() elements.
	//   - Implementations should batch efficiently where possible.
	//   - ctx should be respected for cancellation and timeouts.
	Embed(ctx context.Context, inputs []Input) ([]Vector, error)
}

// Input represents a single embedding input.
//
// Contract:
//   - Exactly one of Text or Bytes should be provided.
//   - Text is the common case for natural language embeddings.
//   - Bytes allows embedding arbitrary binary content when supported.
type Input struct {
	Text  string `json:"text,omitempty"`
	Bytes []byte `json:"bytes,omitempty"`
}

// Vector is a single embedding output.
//
// Contract:
//   - Values must have consistent length equal to Model.Dimension().
//   - ID optionally links the embedding to an external document.
type Vector struct {
	ID     string    `json:"id,omitempty"`
	Values []float32 `json:"values"`
}