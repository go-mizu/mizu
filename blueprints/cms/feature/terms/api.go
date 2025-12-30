// Package terms provides taxonomy term management (categories, tags, custom taxonomies).
package terms

import (
	"context"
	"errors"
)

// Errors
var (
	ErrNotFound   = errors.New("term not found")
	ErrSlugTaken  = errors.New("slug already taken")
	ErrForbidden  = errors.New("forbidden")
)

// Taxonomies
const (
	TaxonomyCategory = "category"
	TaxonomyPostTag  = "post_tag"
	TaxonomyNavMenu  = "nav_menu"
	TaxonomyLinkCat  = "link_category"
	TaxonomyPostFormat = "post_format"
)

// Term represents a WordPress-compatible term.
type Term struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Taxonomy    string                 `json:"taxonomy"`
	Description string                 `json:"description"`
	Parent      string                 `json:"parent,omitempty"`
	Count       int64                  `json:"count"`
	Link        string                 `json:"link,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
}

// CreateIn contains input for creating a term.
type CreateIn struct {
	Name        string `json:"name"`
	Slug        string `json:"slug,omitempty"`
	Taxonomy    string `json:"taxonomy"`
	Description string `json:"description,omitempty"`
	Parent      string `json:"parent,omitempty"`
}

// UpdateIn contains input for updating a term.
type UpdateIn struct {
	Name        *string `json:"name,omitempty"`
	Slug        *string `json:"slug,omitempty"`
	Description *string `json:"description,omitempty"`
	Parent      *string `json:"parent,omitempty"`
}

// ListOpts contains options for listing terms.
type ListOpts struct {
	Page        int      `json:"page"`
	PerPage     int      `json:"per_page"`
	Search      string   `json:"search"`
	Include     []string `json:"include"`
	Exclude     []string `json:"exclude"`
	Parent      *string  `json:"parent"`
	HideEmpty   bool     `json:"hide_empty"`
	OrderBy     string   `json:"orderby"`
	Order       string   `json:"order"`
	Slug        []string `json:"slug"`
	Taxonomy    string   `json:"-"`
}

// TaxonomyInfo describes a taxonomy.
type TaxonomyInfo struct {
	Name         string   `json:"name"`
	Slug         string   `json:"slug"`
	Description  string   `json:"description"`
	Hierarchical bool     `json:"hierarchical"`
	ShowCloud    bool     `json:"show_cloud"`
	Types        []string `json:"types"`
	RestBase     string   `json:"rest_base"`
}

// API defines the terms service interface.
type API interface {
	// Term management
	Create(ctx context.Context, in CreateIn) (*Term, error)
	GetByID(ctx context.Context, id string) (*Term, error)
	GetBySlug(ctx context.Context, slug, taxonomy string) (*Term, error)
	Update(ctx context.Context, id string, in UpdateIn) (*Term, error)
	Delete(ctx context.Context, id string, force bool) error

	// Lists
	List(ctx context.Context, opts ListOpts) ([]*Term, int, error)

	// Meta
	GetMeta(ctx context.Context, termID, key string) (string, error)
	SetMeta(ctx context.Context, termID, key, value string) error
	DeleteMeta(ctx context.Context, termID, key string) error

	// Taxonomy info
	GetTaxonomy(ctx context.Context, taxonomy string) (*TaxonomyInfo, error)
	GetTaxonomies(ctx context.Context) ([]*TaxonomyInfo, error)

	// Hierarchical utilities
	GetChildren(ctx context.Context, parentID, taxonomy string) ([]*Term, error)
	GetAncestors(ctx context.Context, termID string) ([]*Term, error)
}
