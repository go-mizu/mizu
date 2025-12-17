package todo

import (
	"context"
	"encoding/json"
	gosync "sync"
	"time"

	"github.com/go-mizu/mizu/sync"
)

// Todo represents a todo item.
type Todo struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store holds todo data and provides sync.ApplyFunc and sync.SnapshotFunc.
type Store struct {
	mu    gosync.RWMutex
	todos map[string]map[string]*Todo // scope -> id -> todo
}

// NewStore creates a new todo store.
func NewStore() *Store {
	return &Store{
		todos: make(map[string]map[string]*Todo),
	}
}

// Apply implements sync.ApplyFunc.
func (s *Store) Apply(ctx context.Context, m sync.Mutation) ([]sync.Change, error) {
	scope := m.Scope
	if scope == "" {
		scope = sync.DefaultScope
	}

	var args map[string]any
	if len(m.Args) > 0 {
		if err := json.Unmarshal(m.Args, &args); err != nil {
			return nil, sync.ErrInvalidMutation
		}
	}

	switch m.Name {
	case "todo.create":
		return s.createTodo(scope, args)
	case "todo.update":
		return s.updateTodo(scope, args)
	case "todo.delete":
		return s.deleteTodo(scope, args)
	case "todo.toggle":
		return s.toggleTodo(scope, args)
	default:
		return nil, sync.ErrInvalidMutation
	}
}

// Snapshot implements sync.SnapshotFunc.
func (s *Store) Snapshot(ctx context.Context, scope string) (json.RawMessage, uint64, error) {
	if scope == "" {
		scope = sync.DefaultScope
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	todos := s.todos[scope]
	if todos == nil {
		return json.RawMessage(`{"todos":[]}`), 0, nil
	}

	todoList := make([]*Todo, 0, len(todos))
	for _, t := range todos {
		todoList = append(todoList, t)
	}

	data, err := json.Marshal(map[string]any{"todos": todoList})
	if err != nil {
		return nil, 0, err
	}

	return data, 0, nil
}

// GetAll returns all todos for a scope.
func (s *Store) GetAll(scope string) []*Todo {
	if scope == "" {
		scope = sync.DefaultScope
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	todos := s.todos[scope]
	if todos == nil {
		return nil
	}

	result := make([]*Todo, 0, len(todos))
	for _, t := range todos {
		result = append(result, t)
	}
	return result
}

func (s *Store) createTodo(scope string, args map[string]any) ([]sync.Change, error) {
	id, _ := args["id"].(string)
	title, _ := args["title"].(string)

	if id == "" || title == "" {
		return nil, sync.ErrInvalidMutation
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.todos[scope] == nil {
		s.todos[scope] = make(map[string]*Todo)
	}

	// Check if already exists
	if _, exists := s.todos[scope][id]; exists {
		return nil, sync.ErrConflict
	}

	todo := &Todo{
		ID:        id,
		Title:     title,
		Done:      false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.todos[scope][id] = todo

	data, err := json.Marshal(map[string]any{
		"op":   "create",
		"todo": todo,
	})
	if err != nil {
		return nil, err
	}

	return makeChanges(scope, data), nil
}

func (s *Store) updateTodo(scope string, args map[string]any) ([]sync.Change, error) {
	id, _ := args["id"].(string)
	if id == "" {
		return nil, sync.ErrInvalidMutation
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	todos := s.todos[scope]
	if todos == nil {
		return nil, sync.ErrNotFound
	}

	todo, exists := todos[id]
	if !exists {
		return nil, sync.ErrNotFound
	}

	// Apply updates
	if title, ok := args["title"].(string); ok {
		todo.Title = title
	}
	if done, ok := args["done"].(bool); ok {
		todo.Done = done
	}
	todo.UpdatedAt = time.Now()

	data, err := json.Marshal(map[string]any{
		"op":   "update",
		"todo": todo,
	})
	if err != nil {
		return nil, err
	}

	return makeChanges(scope, data), nil
}

func (s *Store) deleteTodo(scope string, args map[string]any) ([]sync.Change, error) {
	id, _ := args["id"].(string)
	if id == "" {
		return nil, sync.ErrInvalidMutation
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	todos := s.todos[scope]
	if todos == nil {
		return nil, sync.ErrNotFound
	}

	if _, exists := todos[id]; !exists {
		return nil, sync.ErrNotFound
	}

	delete(todos, id)

	data, err := json.Marshal(map[string]any{
		"op": "delete",
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	return makeChanges(scope, data), nil
}

func (s *Store) toggleTodo(scope string, args map[string]any) ([]sync.Change, error) {
	id, _ := args["id"].(string)
	if id == "" {
		return nil, sync.ErrInvalidMutation
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	todos := s.todos[scope]
	if todos == nil {
		return nil, sync.ErrNotFound
	}

	todo, exists := todos[id]
	if !exists {
		return nil, sync.ErrNotFound
	}

	// Toggle done status
	todo.Done = !todo.Done
	todo.UpdatedAt = time.Now()

	data, err := json.Marshal(map[string]any{
		"op":   "update",
		"todo": todo,
	})
	if err != nil {
		return nil, err
	}

	return makeChanges(scope, data), nil
}

// makeChanges creates a single-element change slice.
func makeChanges(scope string, data json.RawMessage) []sync.Change {
	return []sync.Change{
		{Scope: scope, Data: data},
	}
}
