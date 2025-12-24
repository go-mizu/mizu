package tags

import (
	"context"
	"errors"
	"regexp"
	"strings"
)

// Errors
var (
	ErrNotFound    = errors.New("tag not found")
	ErrInvalidName = errors.New("invalid tag name")
	ErrNameTaken   = errors.New("tag name already taken")
)

// Validation
var (
	TagNameRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

const (
	NameMinLen = 2
	NameMaxLen = 25
)

// Tag represents a story tag/category.
type Tag struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
	StoryCount  int64  `json:"story_count"`
}

// CreateIn contains input for creating a tag.
type CreateIn struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}

// Validate validates the create input.
func (in *CreateIn) Validate() error {
	in.Name = strings.ToLower(strings.TrimSpace(in.Name))

	if len(in.Name) < NameMinLen || len(in.Name) > NameMaxLen {
		return ErrInvalidName
	}
	if !TagNameRegex.MatchString(in.Name) {
		return ErrInvalidName
	}

	return nil
}

// API defines the tags service interface.
type API interface {
	GetByID(ctx context.Context, id string) (*Tag, error)
	GetByName(ctx context.Context, name string) (*Tag, error)
	GetByNames(ctx context.Context, names []string) ([]*Tag, error)

	// Lists
	List(ctx context.Context, limit int) ([]*Tag, error)
	ListPopular(ctx context.Context, limit int) ([]*Tag, error)
}

// Store defines the data storage interface for tags.
type Store interface {
	GetByID(ctx context.Context, id string) (*Tag, error)
	GetByName(ctx context.Context, name string) (*Tag, error)
	GetByNames(ctx context.Context, names []string) ([]*Tag, error)

	// Lists
	List(ctx context.Context, limit int) ([]*Tag, error)
	ListPopular(ctx context.Context, limit int) ([]*Tag, error)
}
