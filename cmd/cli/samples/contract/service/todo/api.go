// Package todo implements a simple todo service.
//
// This package defines both the API interface and its implementation.
// The interface is used for code-first contract registration.
package todo

import "context"

// API defines the Todo service contract.
// This interface is the source of truth for the API definition.
type API interface {
	// Create creates a new todo item.
	Create(ctx context.Context, in *CreateIn) (*Todo, error)

	// List returns all todo items.
	List(ctx context.Context) (*TodoList, error)

	// Get retrieves a todo by ID.
	Get(ctx context.Context, in *GetIn) (*Todo, error)

	// Update modifies an existing todo.
	Update(ctx context.Context, in *UpdateIn) (*Todo, error)

	// Delete removes a todo.
	Delete(ctx context.Context, in *DeleteIn) error
}

// Ensure Service implements API
var _ API = (*Service)(nil)
