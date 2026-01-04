package views

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrNotFound = errors.New("view not found")
)

// Service implements the views API.
type Service struct {
	store Store
	pages pages.API
}

// NewService creates a new views service.
func NewService(store Store, pages pages.API) *Service {
	return &Service{store: store, pages: pages}
}

// Create creates a new view.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*View, error) {
	// Get position
	views, _ := s.store.ListByDatabase(ctx, in.DatabaseID)
	position := len(views)

	view := &View{
		ID:         ulid.New(),
		DatabaseID: in.DatabaseID,
		Name:       in.Name,
		Type:       in.Type,
		Filter:     in.Filter,
		Sorts:      in.Sorts,
		GroupBy:    in.GroupBy,
		SubGroupBy: in.SubGroupBy,
		CalendarBy: in.CalendarBy,
		Config:     in.Config,
		Position:   position,
		CreatedBy:  in.CreatedBy,
		CreatedAt:  time.Now(),
	}

	if view.Name == "" {
		view.Name = string(view.Type)
	}

	if view.Type == "" {
		view.Type = ViewTable
	}

	if err := s.store.Create(ctx, view); err != nil {
		return nil, err
	}

	return view, nil
}

// GetByID retrieves a view by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*View, error) {
	view, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return view, nil
}

// Update updates a view.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*View, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// Delete deletes a view.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ListByDatabase lists views for a database.
func (s *Service) ListByDatabase(ctx context.Context, databaseID string) ([]*View, error) {
	return s.store.ListByDatabase(ctx, databaseID)
}

// Reorder reorders views.
func (s *Service) Reorder(ctx context.Context, databaseID string, viewIDs []string) error {
	return s.store.Reorder(ctx, databaseID, viewIDs)
}

// Duplicate creates a copy of a view.
func (s *Service) Duplicate(ctx context.Context, id string, userID string) (*View, error) {
	original, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	return s.Create(ctx, &CreateIn{
		DatabaseID: original.DatabaseID,
		Name:       original.Name + " (copy)",
		Type:       original.Type,
		Filter:     original.Filter,
		Sorts:      original.Sorts,
		GroupBy:    original.GroupBy,
		CalendarBy: original.CalendarBy,
		CreatedBy:  userID,
	})
}

// Query queries a view and returns matching pages.
func (s *Service) Query(ctx context.Context, viewID string, cursor string, limit int) (*QueryResult, error) {
	view, err := s.store.GetByID(ctx, viewID)
	if err != nil {
		return nil, ErrNotFound
	}

	if limit <= 0 {
		limit = 50
	}

	// Get all pages for the database (as database items)
	items, err := s.pages.ListByParent(ctx, view.DatabaseID, pages.ParentDatabase)
	if err != nil {
		return nil, err
	}

	// Apply filter if present
	if view.Filter != nil {
		items = s.applyFilter(items, view.Filter)
	}

	// Apply sorting if present
	if len(view.Sorts) > 0 {
		items = s.applySort(items, view.Sorts)
	}

	// Apply pagination
	start := 0
	if cursor != "" {
		for i, item := range items {
			if item.ID == cursor {
				start = i + 1
				break
			}
		}
	}

	end := start + limit
	if end > len(items) {
		end = len(items)
	}

	result := &QueryResult{
		Items:   items[start:end],
		HasMore: end < len(items),
	}

	if result.HasMore && len(result.Items) > 0 {
		result.NextCursor = result.Items[len(result.Items)-1].ID
	}

	return result, nil
}

// applyFilter filters pages based on the filter configuration.
func (s *Service) applyFilter(items []*pages.Page, filter *Filter) []*pages.Page {
	if filter == nil {
		return items
	}

	result := make([]*pages.Page, 0, len(items))
	for _, item := range items {
		if s.evaluateFilter(item, filter) {
			result = append(result, item)
		}
	}
	return result
}

// evaluateFilter evaluates a filter condition against a page.
func (s *Service) evaluateFilter(item *pages.Page, filter *Filter) bool {
	// Handle AND conditions
	if len(filter.And) > 0 {
		for _, f := range filter.And {
			if !s.evaluateFilter(item, &f) {
				return false
			}
		}
		return true
	}

	// Handle OR conditions
	if len(filter.Or) > 0 {
		for _, f := range filter.Or {
			if s.evaluateFilter(item, &f) {
				return true
			}
		}
		return false
	}

	// Handle property condition
	if filter.PropertyID == "" {
		return true
	}

	prop, ok := item.Properties[filter.PropertyID]
	if !ok {
		return s.evaluateEmpty(filter.Operator)
	}

	return s.evaluateCondition(prop.Value, filter.Operator, filter.Value)
}

// evaluateEmpty returns the result for empty property values.
func (s *Service) evaluateEmpty(operator string) bool {
	switch operator {
	case "is_empty":
		return true
	case "is_not_empty":
		return false
	default:
		return false
	}
}

// evaluateCondition evaluates a condition against a value.
func (s *Service) evaluateCondition(value interface{}, operator string, target interface{}) bool {
	valueStr := toString(value)
	targetStr := toString(target)

	switch operator {
	case "equals", "is":
		return valueStr == targetStr
	case "does_not_equal", "is_not":
		return valueStr != targetStr
	case "contains":
		return containsIgnoreCase(valueStr, targetStr)
	case "does_not_contain":
		return !containsIgnoreCase(valueStr, targetStr)
	case "starts_with":
		return startsWithIgnoreCase(valueStr, targetStr)
	case "ends_with":
		return endsWithIgnoreCase(valueStr, targetStr)
	case "is_empty":
		return value == nil || valueStr == ""
	case "is_not_empty":
		return value != nil && valueStr != ""
	case "greater_than":
		return toFloat(value) > toFloat(target)
	case "less_than":
		return toFloat(value) < toFloat(target)
	case "greater_than_or_equal_to":
		return toFloat(value) >= toFloat(target)
	case "less_than_or_equal_to":
		return toFloat(value) <= toFloat(target)
	case "before":
		return toTime(value).Before(toTime(target))
	case "after":
		return toTime(value).After(toTime(target))
	case "on_or_before":
		tv, tt := toTime(value), toTime(target)
		return tv.Before(tt) || tv.Equal(tt)
	case "on_or_after":
		tv, tt := toTime(value), toTime(target)
		return tv.After(tt) || tv.Equal(tt)
	default:
		return true
	}
}

// applySort sorts pages based on the sort configuration.
func (s *Service) applySort(items []*pages.Page, sorts []Sort) []*pages.Page {
	if len(sorts) == 0 {
		return items
	}

	result := make([]*pages.Page, len(items))
	copy(result, items)

	// Use stable sort to preserve relative order for equal elements
	for i := len(sorts) - 1; i >= 0; i-- {
		st := sorts[i]
		stableSort(result, func(a, b *pages.Page) bool {
			cmp := s.compareProperty(a, b, st.PropertyID)
			if st.Direction == "desc" {
				return cmp > 0
			}
			return cmp < 0
		})
	}

	return result
}

// compareProperty compares a property value between two pages.
func (s *Service) compareProperty(a, b *pages.Page, propertyID string) int {
	va := a.Properties[propertyID]
	vb := b.Properties[propertyID]

	// Handle nil cases
	if va.Value == nil && vb.Value == nil {
		return 0
	}
	if va.Value == nil {
		return -1
	}
	if vb.Value == nil {
		return 1
	}

	// Compare based on value type
	switch v := va.Value.(type) {
	case string:
		vbStr, _ := vb.Value.(string)
		return compareStrings(v, vbStr)
	case float64:
		vbFloat := toFloat(vb.Value)
		if v < vbFloat {
			return -1
		} else if v > vbFloat {
			return 1
		}
		return 0
	case bool:
		vbBool, _ := vb.Value.(bool)
		if v == vbBool {
			return 0
		}
		if v {
			return 1
		}
		return -1
	default:
		return compareStrings(toString(v), toString(vb.Value))
	}
}

// Helper functions

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func toFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

func toTime(v interface{}) time.Time {
	if v == nil {
		return time.Time{}
	}
	switch val := v.(type) {
	case time.Time:
		return val
	case string:
		// Try multiple date formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02",
			"01/02/2006",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, val); err == nil {
				return t
			}
		}
		return time.Time{}
	default:
		return time.Time{}
	}
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			(len(s) > 0 && containsLower(toLower(s), toLower(substr))))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func startsWithIgnoreCase(s, prefix string) bool {
	return len(s) >= len(prefix) && toLower(s[:len(prefix)]) == toLower(prefix)
}

func endsWithIgnoreCase(s, suffix string) bool {
	return len(s) >= len(suffix) && toLower(s[len(s)-len(suffix):]) == toLower(suffix)
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func compareStrings(a, b string) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

// stableSort performs a stable sort using the provided less function.
func stableSort(items []*pages.Page, less func(a, b *pages.Page) bool) {
	n := len(items)
	for i := 1; i < n; i++ {
		for j := i; j > 0 && less(items[j], items[j-1]); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}
