package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Error represents a GitHub API error response
type Error struct {
	Message          string        `json:"message"`
	DocumentationURL string        `json:"documentation_url,omitempty"`
	Errors           []ErrorDetail `json:"errors,omitempty"`
}

// ErrorDetail represents a detailed error
type ErrorDetail struct {
	Resource string `json:"resource,omitempty"`
	Field    string `json:"field,omitempty"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		json.NewEncoder(w).Encode(v)
	}
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, &Error{
		Message: message,
	})
}

// WriteValidationError writes a validation error response
func WriteValidationError(w http.ResponseWriter, message string, errors []ErrorDetail) {
	WriteJSON(w, http.StatusUnprocessableEntity, &Error{
		Message: message,
		Errors:  errors,
	})
}

// WriteNotFound writes a 404 response
func WriteNotFound(w http.ResponseWriter, resource string) {
	WriteJSON(w, http.StatusNotFound, &Error{
		Message: fmt.Sprintf("%s not found", resource),
	})
}

// WriteForbidden writes a 403 response
func WriteForbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Must have admin rights to Repository"
	}
	WriteJSON(w, http.StatusForbidden, &Error{
		Message: message,
	})
}

// WriteUnauthorized writes a 401 response
func WriteUnauthorized(w http.ResponseWriter) {
	WriteJSON(w, http.StatusUnauthorized, &Error{
		Message: "Requires authentication",
	})
}

// WriteBadRequest writes a 400 response
func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteJSON(w, http.StatusBadRequest, &Error{
		Message: message,
	})
}

// WriteConflict writes a 409 response
func WriteConflict(w http.ResponseWriter, message string) {
	WriteJSON(w, http.StatusConflict, &Error{
		Message: message,
	})
}

// WriteNoContent writes a 204 response
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteCreated writes a 201 response with JSON body
func WriteCreated(w http.ResponseWriter, v interface{}) {
	WriteJSON(w, http.StatusCreated, v)
}

// WriteAccepted writes a 202 response with JSON body
func WriteAccepted(w http.ResponseWriter, v interface{}) {
	WriteJSON(w, http.StatusAccepted, v)
}

// Pagination helpers

// PaginationParams extracts pagination parameters from request
type PaginationParams struct {
	Page    int
	PerPage int
}

// GetPaginationParams extracts pagination params from request
func GetPaginationParams(r *http.Request) PaginationParams {
	p := PaginationParams{
		Page:    1,
		PerPage: 30,
	}

	if page := r.URL.Query().Get("page"); page != "" {
		if n, err := strconv.Atoi(page); err == nil && n > 0 {
			p.Page = n
		}
	}

	if perPage := r.URL.Query().Get("per_page"); perPage != "" {
		if n, err := strconv.Atoi(perPage); err == nil && n > 0 && n <= 100 {
			p.PerPage = n
		}
	}

	return p
}

// SetLinkHeader sets the Link header for pagination
func SetLinkHeader(w http.ResponseWriter, r *http.Request, page, perPage, totalCount int) {
	baseURL := r.URL.Path
	query := r.URL.Query()

	totalPages := (totalCount + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}

	var links []string

	// First page
	if page > 1 {
		query.Set("page", "1")
		links = append(links, fmt.Sprintf(`<%s?%s>; rel="first"`, baseURL, query.Encode()))
	}

	// Prev page
	if page > 1 {
		query.Set("page", strconv.Itoa(page-1))
		links = append(links, fmt.Sprintf(`<%s?%s>; rel="prev"`, baseURL, query.Encode()))
	}

	// Next page
	if page < totalPages {
		query.Set("page", strconv.Itoa(page+1))
		links = append(links, fmt.Sprintf(`<%s?%s>; rel="next"`, baseURL, query.Encode()))
	}

	// Last page
	if page < totalPages {
		query.Set("page", strconv.Itoa(totalPages))
		links = append(links, fmt.Sprintf(`<%s?%s>; rel="last"`, baseURL, query.Encode()))
	}

	if len(links) > 0 {
		w.Header().Set("Link", strings.Join(links, ", "))
	}
}

// DecodeJSON decodes JSON request body
func DecodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// PathParam extracts a path parameter from the request
func PathParam(r *http.Request, name string) string {
	return r.PathValue(name)
}

// PathParamInt extracts an integer path parameter
func PathParamInt(r *http.Request, name string) (int, error) {
	s := r.PathValue(name)
	return strconv.Atoi(s)
}

// PathParamInt64 extracts an int64 path parameter
func PathParamInt64(r *http.Request, name string) (int64, error) {
	s := r.PathValue(name)
	return strconv.ParseInt(s, 10, 64)
}

// QueryParam extracts a query parameter
func QueryParam(r *http.Request, name string) string {
	return r.URL.Query().Get(name)
}

// QueryParamInt extracts an integer query parameter with default
func QueryParamInt(r *http.Request, name string, defaultVal int) int {
	s := r.URL.Query().Get(name)
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return n
}

// QueryParamBool extracts a boolean query parameter
func QueryParamBool(r *http.Request, name string) bool {
	s := r.URL.Query().Get(name)
	return s == "true" || s == "1"
}
