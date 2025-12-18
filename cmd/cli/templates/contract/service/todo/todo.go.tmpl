// Package todo implements a simple todo service.
//
// This is a plain Go service with no framework dependencies.
// It can be easily tested and reused across different transports.
package todo

import (
	"context"
	"errors"
	"sync"
)

// Priority levels for todos.
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// Service is the todo business logic.
// It has no HTTP or transport dependencies.
type Service struct {
	mu    sync.RWMutex
	todos map[string]*Todo
	seq   int
}

// Todo represents a todo item.
type Todo struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Completed bool     `json:"completed"`
	Priority  Priority `json:"priority,omitempty"`
}

// CreateIn is the input for creating a todo.
type CreateIn struct {
	Title    string   `json:"title"`
	Priority Priority `json:"priority,omitempty"`
}

// GetIn is the input for getting a todo by ID.
type GetIn struct {
	ID string `json:"id"`
}

// UpdateIn is the input for updating a todo.
type UpdateIn struct {
	ID        string   `json:"id"`
	Title     string   `json:"title,omitempty"`
	Completed bool     `json:"completed,omitempty"`
	Priority  Priority `json:"priority,omitempty"`
}

// DeleteIn is the input for deleting a todo.
type DeleteIn struct {
	ID string `json:"id"`
}

// TodoList is a list of todos.
type TodoList struct {
	Items []*Todo `json:"items"`
	Total int     `json:"total"`
}

// HealthStatus represents the service health.
type HealthStatus struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

// Errors returned by the service.
var (
	ErrNotFound   = errors.New("todo not found")
	ErrTitleEmpty = errors.New("title cannot be empty")
)

// Create creates a new todo.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Todo, error) {
	if in.Title == "" {
		return nil, ErrTitleEmpty
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.todos == nil {
		s.todos = make(map[string]*Todo)
	}

	s.seq++
	todo := &Todo{
		ID:        formatID(s.seq),
		Title:     in.Title,
		Completed: false,
		Priority:  in.Priority,
	}
	if todo.Priority == "" {
		todo.Priority = PriorityMedium
	}
	s.todos[todo.ID] = todo

	return todo, nil
}

// Get retrieves a todo by ID.
func (s *Service) Get(ctx context.Context, in *GetIn) (*Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todo, ok := s.todos[in.ID]
	if !ok {
		return nil, ErrNotFound
	}

	return todo, nil
}

// List returns all todos.
func (s *Service) List(ctx context.Context) (*TodoList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*Todo, 0, len(s.todos))
	for _, t := range s.todos {
		items = append(items, t)
	}

	return &TodoList{Items: items, Total: len(items)}, nil
}

// Update updates an existing todo.
func (s *Service) Update(ctx context.Context, in *UpdateIn) (*Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo, ok := s.todos[in.ID]
	if !ok {
		return nil, ErrNotFound
	}

	if in.Title != "" {
		todo.Title = in.Title
	}
	todo.Completed = in.Completed
	if in.Priority != "" {
		todo.Priority = in.Priority
	}

	return todo, nil
}

// Delete removes a todo.
func (s *Service) Delete(ctx context.Context, in *DeleteIn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.todos[in.ID]; !ok {
		return ErrNotFound
	}

	delete(s.todos, in.ID)
	return nil
}

// Health returns nil if the service is healthy.
func (s *Service) Health(ctx context.Context) error {
	return nil
}

func formatID(n int) string {
	return "todo_" + itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf) - 1
	for n > 0 {
		buf[i] = byte('0' + n%10)
		n /= 10
		i--
	}
	return string(buf[i+1:])
}
