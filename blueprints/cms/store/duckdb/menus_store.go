package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/feature/menus"
)

// MenusStore handles menu data access.
type MenusStore struct {
	db *sql.DB
}

// NewMenusStore creates a new menus store.
func NewMenusStore(db *sql.DB) *MenusStore {
	return &MenusStore{db: db}
}

func (s *MenusStore) CreateMenu(ctx context.Context, m *menus.Menu) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO menus (id, name, slug, location, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, m.ID, m.Name, m.Slug, nullString(m.Location), m.CreatedAt, m.UpdatedAt)
	return err
}

func (s *MenusStore) GetMenu(ctx context.Context, id string) (*menus.Menu, error) {
	return s.scanMenu(s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, location, created_at, updated_at
		FROM menus WHERE id = $1
	`, id))
}

func (s *MenusStore) GetMenuBySlug(ctx context.Context, slug string) (*menus.Menu, error) {
	return s.scanMenu(s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, location, created_at, updated_at
		FROM menus WHERE slug = $1
	`, slug))
}

func (s *MenusStore) GetMenuByLocation(ctx context.Context, location string) (*menus.Menu, error) {
	return s.scanMenu(s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, location, created_at, updated_at
		FROM menus WHERE location = $1
	`, location))
}

func (s *MenusStore) ListMenus(ctx context.Context) ([]*menus.Menu, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, slug, location, created_at, updated_at
		FROM menus
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*menus.Menu
	for rows.Next() {
		m, err := s.scanMenuRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (s *MenusStore) UpdateMenu(ctx context.Context, id string, in *menus.UpdateMenuIn) error {
	var sets []string
	var args []any
	argNum := 1

	if in.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *in.Name)
		argNum++
	}
	if in.Slug != nil {
		sets = append(sets, fmt.Sprintf("slug = $%d", argNum))
		args = append(args, *in.Slug)
		argNum++
	}
	if in.Location != nil {
		sets = append(sets, fmt.Sprintf("location = $%d", argNum))
		args = append(args, nullString(*in.Location))
		argNum++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++

	args = append(args, id)
	query := fmt.Sprintf("UPDATE menus SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *MenusStore) DeleteMenu(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM menus WHERE id = $1`, id)
	return err
}

// Menu items

func (s *MenusStore) CreateItem(ctx context.Context, item *menus.MenuItem) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO menu_items (id, menu_id, parent_id, title, url, target, link_type, link_id, css_class, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, item.ID, item.MenuID, nullString(item.ParentID), item.Title, nullString(item.URL), item.Target, nullString(item.LinkType), nullString(item.LinkID), nullString(item.CSSClass), item.SortOrder, item.CreatedAt)
	return err
}

func (s *MenusStore) GetItem(ctx context.Context, id string) (*menus.MenuItem, error) {
	return s.scanItem(s.db.QueryRowContext(ctx, `
		SELECT id, menu_id, parent_id, title, url, target, link_type, link_id, css_class, sort_order, created_at
		FROM menu_items WHERE id = $1
	`, id))
}

func (s *MenusStore) GetItemsByMenu(ctx context.Context, menuID string) ([]*menus.MenuItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, menu_id, parent_id, title, url, target, link_type, link_id, css_class, sort_order, created_at
		FROM menu_items WHERE menu_id = $1
		ORDER BY sort_order, title
	`, menuID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*menus.MenuItem
	for rows.Next() {
		item, err := s.scanItemRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (s *MenusStore) UpdateItem(ctx context.Context, id string, in *menus.UpdateItemIn) error {
	var sets []string
	var args []any
	argNum := 1

	if in.ParentID != nil {
		sets = append(sets, fmt.Sprintf("parent_id = $%d", argNum))
		args = append(args, nullString(*in.ParentID))
		argNum++
	}
	if in.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", argNum))
		args = append(args, *in.Title)
		argNum++
	}
	if in.URL != nil {
		sets = append(sets, fmt.Sprintf("url = $%d", argNum))
		args = append(args, nullString(*in.URL))
		argNum++
	}
	if in.Target != nil {
		sets = append(sets, fmt.Sprintf("target = $%d", argNum))
		args = append(args, *in.Target)
		argNum++
	}
	if in.LinkType != nil {
		sets = append(sets, fmt.Sprintf("link_type = $%d", argNum))
		args = append(args, nullString(*in.LinkType))
		argNum++
	}
	if in.LinkID != nil {
		sets = append(sets, fmt.Sprintf("link_id = $%d", argNum))
		args = append(args, nullString(*in.LinkID))
		argNum++
	}
	if in.CSSClass != nil {
		sets = append(sets, fmt.Sprintf("css_class = $%d", argNum))
		args = append(args, nullString(*in.CSSClass))
		argNum++
	}
	if in.SortOrder != nil {
		sets = append(sets, fmt.Sprintf("sort_order = $%d", argNum))
		args = append(args, *in.SortOrder)
		argNum++
	}

	if len(sets) == 0 {
		return nil
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE menu_items SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *MenusStore) DeleteItem(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM menu_items WHERE id = $1`, id)
	return err
}

func (s *MenusStore) DeleteItemsByMenu(ctx context.Context, menuID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM menu_items WHERE menu_id = $1`, menuID)
	return err
}

func (s *MenusStore) scanMenu(row *sql.Row) (*menus.Menu, error) {
	m := &menus.Menu{}
	var location sql.NullString
	err := row.Scan(&m.ID, &m.Name, &m.Slug, &location, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.Location = location.String
	return m, nil
}

func (s *MenusStore) scanMenuRow(rows *sql.Rows) (*menus.Menu, error) {
	m := &menus.Menu{}
	var location sql.NullString
	err := rows.Scan(&m.ID, &m.Name, &m.Slug, &location, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	m.Location = location.String
	return m, nil
}

func (s *MenusStore) scanItem(row *sql.Row) (*menus.MenuItem, error) {
	item := &menus.MenuItem{}
	var parentID, url, linkType, linkID, cssClass sql.NullString
	err := row.Scan(&item.ID, &item.MenuID, &parentID, &item.Title, &url, &item.Target, &linkType, &linkID, &cssClass, &item.SortOrder, &item.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item.ParentID = parentID.String
	item.URL = url.String
	item.LinkType = linkType.String
	item.LinkID = linkID.String
	item.CSSClass = cssClass.String
	return item, nil
}

func (s *MenusStore) scanItemRow(rows *sql.Rows) (*menus.MenuItem, error) {
	item := &menus.MenuItem{}
	var parentID, url, linkType, linkID, cssClass sql.NullString
	err := rows.Scan(&item.ID, &item.MenuID, &parentID, &item.Title, &url, &item.Target, &linkType, &linkID, &cssClass, &item.SortOrder, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	item.ParentID = parentID.String
	item.URL = url.String
	item.LinkType = linkType.String
	item.LinkID = linkID.String
	item.CSSClass = cssClass.String
	return item, nil
}
