// Package menus provides navigation menu management.
package menus

import (
	"context"
	"time"
)

// Menu represents a navigation menu.
type Menu struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Slug      string      `json:"slug"`
	Location  string      `json:"location,omitempty"`
	Items     []*MenuItem `json:"items,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// MenuItem represents an item in a menu.
type MenuItem struct {
	ID        string      `json:"id"`
	MenuID    string      `json:"menu_id"`
	ParentID  string      `json:"parent_id,omitempty"`
	Title     string      `json:"title"`
	URL       string      `json:"url,omitempty"`
	Target    string      `json:"target"`
	LinkType  string      `json:"link_type,omitempty"`
	LinkID    string      `json:"link_id,omitempty"`
	CSSClass  string      `json:"css_class,omitempty"`
	SortOrder int         `json:"sort_order"`
	Children  []*MenuItem `json:"children,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

// CreateMenuIn contains input for creating a menu.
type CreateMenuIn struct {
	Name     string `json:"name"`
	Slug     string `json:"slug,omitempty"`
	Location string `json:"location,omitempty"`
}

// UpdateMenuIn contains input for updating a menu.
type UpdateMenuIn struct {
	Name     *string `json:"name,omitempty"`
	Slug     *string `json:"slug,omitempty"`
	Location *string `json:"location,omitempty"`
}

// CreateItemIn contains input for creating a menu item.
type CreateItemIn struct {
	ParentID  string `json:"parent_id,omitempty"`
	Title     string `json:"title"`
	URL       string `json:"url,omitempty"`
	Target    string `json:"target,omitempty"`
	LinkType  string `json:"link_type,omitempty"`
	LinkID    string `json:"link_id,omitempty"`
	CSSClass  string `json:"css_class,omitempty"`
	SortOrder int    `json:"sort_order,omitempty"`
}

// UpdateItemIn contains input for updating a menu item.
type UpdateItemIn struct {
	ParentID  *string `json:"parent_id,omitempty"`
	Title     *string `json:"title,omitempty"`
	URL       *string `json:"url,omitempty"`
	Target    *string `json:"target,omitempty"`
	LinkType  *string `json:"link_type,omitempty"`
	LinkID    *string `json:"link_id,omitempty"`
	CSSClass  *string `json:"css_class,omitempty"`
	SortOrder *int    `json:"sort_order,omitempty"`
}

// API defines the menus service contract.
type API interface {
	CreateMenu(ctx context.Context, in *CreateMenuIn) (*Menu, error)
	GetMenu(ctx context.Context, id string) (*Menu, error)
	GetMenuBySlug(ctx context.Context, slug string) (*Menu, error)
	GetMenuByLocation(ctx context.Context, location string) (*Menu, error)
	ListMenus(ctx context.Context) ([]*Menu, error)
	UpdateMenu(ctx context.Context, id string, in *UpdateMenuIn) (*Menu, error)
	DeleteMenu(ctx context.Context, id string) error
	CreateItem(ctx context.Context, menuID string, in *CreateItemIn) (*MenuItem, error)
	UpdateItem(ctx context.Context, id string, in *UpdateItemIn) (*MenuItem, error)
	DeleteItem(ctx context.Context, id string) error
	ReorderItems(ctx context.Context, menuID string, itemIDs []string) error
}

// Store defines the data access contract for menus.
type Store interface {
	CreateMenu(ctx context.Context, m *Menu) error
	GetMenu(ctx context.Context, id string) (*Menu, error)
	GetMenuBySlug(ctx context.Context, slug string) (*Menu, error)
	GetMenuByLocation(ctx context.Context, location string) (*Menu, error)
	ListMenus(ctx context.Context) ([]*Menu, error)
	UpdateMenu(ctx context.Context, id string, in *UpdateMenuIn) error
	DeleteMenu(ctx context.Context, id string) error
	CreateItem(ctx context.Context, item *MenuItem) error
	GetItem(ctx context.Context, id string) (*MenuItem, error)
	GetItemsByMenu(ctx context.Context, menuID string) ([]*MenuItem, error)
	UpdateItem(ctx context.Context, id string, in *UpdateItemIn) error
	DeleteItem(ctx context.Context, id string) error
	DeleteItemsByMenu(ctx context.Context, menuID string) error
}
