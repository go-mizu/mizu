package pagination

import (
	"strconv"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 30
	MaxPerPage     = 100
)

// Params represents pagination parameters
type Params struct {
	Page    int
	PerPage int
}

// NewParams creates pagination params with defaults
func NewParams(page, perPage int) Params {
	if page < 1 {
		page = DefaultPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	return Params{Page: page, PerPage: perPage}
}

// FromStrings parses pagination from string values
func FromStrings(pageStr, perPageStr string) Params {
	page, _ := strconv.Atoi(pageStr)
	perPage, _ := strconv.Atoi(perPageStr)
	return NewParams(page, perPage)
}

// Offset returns the SQL offset for the current page
func (p Params) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// Limit returns the SQL limit
func (p Params) Limit() int {
	return p.PerPage
}

// Result represents a paginated result
type Result struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// NewResult creates a pagination result
func NewResult(params Params, total int) Result {
	totalPages := total / params.PerPage
	if total%params.PerPage > 0 {
		totalPages++
	}
	return Result{
		Page:       params.Page,
		PerPage:    params.PerPage,
		Total:      total,
		TotalPages: totalPages,
	}
}

// HasNext returns true if there are more pages
func (r Result) HasNext() bool {
	return r.Page < r.TotalPages
}

// HasPrev returns true if there are previous pages
func (r Result) HasPrev() bool {
	return r.Page > 1
}
