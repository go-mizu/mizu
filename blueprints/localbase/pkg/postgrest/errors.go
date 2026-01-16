// Package postgrest provides a PostgREST-compatible REST API implementation.
package postgrest

import (
	"fmt"
	"net/http"
	"strings"
)

// Error represents a PostgREST-compatible error response.
type Error struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Details *string `json:"details"`
	Hint    *string `json:"hint"`
	Status  int     `json:"-"`
}

func (e *Error) Error() string {
	return e.Message
}

// NewError creates a new PostgREST error.
func NewError(code string, status int, message string) *Error {
	return &Error{
		Code:    code,
		Status:  status,
		Message: message,
	}
}

// WithDetails adds details to the error.
func (e *Error) WithDetails(details string) *Error {
	e.Details = &details
	return e
}

// WithHint adds a hint to the error.
func (e *Error) WithHint(hint string) *Error {
	e.Hint = &hint
	return e
}

// PGRST Error Codes - Group 0: Connection
var (
	ErrPGRST000 = func(msg string) *Error {
		return NewError("PGRST000", http.StatusServiceUnavailable, "Database connection failed: "+msg)
	}
	ErrPGRST001 = func(msg string) *Error {
		return NewError("PGRST001", http.StatusServiceUnavailable, "Internal connection error: "+msg)
	}
	ErrPGRST002 = func() *Error {
		return NewError("PGRST002", http.StatusServiceUnavailable, "Cannot build schema cache, PostgreSQL may be unavailable")
	}
	ErrPGRST003 = func() *Error {
		return NewError("PGRST003", http.StatusGatewayTimeout, "Timeout acquiring connection from pool")
	}
)

// PGRST Error Codes - Group 1: API Request
var (
	ErrPGRST100 = func(detail string) *Error {
		return NewError("PGRST100", http.StatusBadRequest, "Failed to parse query string").WithDetails(detail)
	}
	ErrPGRST101 = func() *Error {
		return NewError("PGRST101", http.StatusMethodNotAllowed, "Only GET and POST methods are allowed for RPC")
	}
	ErrPGRST102 = func(detail string) *Error {
		return NewError("PGRST102", http.StatusBadRequest, "Invalid request body: "+detail)
	}
	ErrPGRST103 = func() *Error {
		return NewError("PGRST103", http.StatusRequestedRangeNotSatisfiable, "Requested range not satisfiable")
	}
	ErrPGRST105 = func() *Error {
		return NewError("PGRST105", http.StatusMethodNotAllowed, "PUT is not allowed without a primary key")
	}
	ErrPGRST106 = func(schema string) *Error {
		return NewError("PGRST106", http.StatusNotAcceptable, fmt.Sprintf("Schema '%s' not found in db-schemas", schema))
	}
	ErrPGRST107 = func(contentType string) *Error {
		return NewError("PGRST107", http.StatusUnsupportedMediaType, fmt.Sprintf("Invalid Content-Type: %s", contentType))
	}
	ErrPGRST108 = func(resource string) *Error {
		return NewError("PGRST108", http.StatusBadRequest, fmt.Sprintf("Filter on '%s' requires it to be part of the select", resource))
	}
	ErrPGRST116 = func(count int) *Error {
		msg := fmt.Sprintf("JSON object requested, multiple (or no) rows returned. Got %d rows", count)
		return NewError("PGRST116", http.StatusNotAcceptable, msg)
	}
	ErrPGRST117 = func() *Error {
		return NewError("PGRST117", http.StatusMethodNotAllowed, "Unsupported HTTP verb")
	}
	ErrPGRST120 = func() *Error {
		return NewError("PGRST120", http.StatusBadRequest, "Embedded resource filter only supports is.null and not.is.null operators")
	}
	ErrPGRST121 = func(msg string) *Error {
		return NewError("PGRST121", http.StatusInternalServerError, "Cannot parse JSON in RAISE: "+msg)
	}
	ErrPGRST122 = func(pref string) *Error {
		return NewError("PGRST122", http.StatusBadRequest, fmt.Sprintf("Invalid preference: %s", pref))
	}
	ErrPGRST124 = func(affected, max int) *Error {
		return NewError("PGRST124", http.StatusBadRequest,
			fmt.Sprintf("max-affected preference violated: %d rows affected, max allowed is %d", affected, max))
	}
)

// PGRST Error Codes - Group 2: Schema Cache
var (
	ErrPGRST200 = func(detail string) *Error {
		return NewError("PGRST200", http.StatusBadRequest, "Schema cache is stale").WithDetails(detail)
	}
	ErrPGRST201 = func(options []string) *Error {
		return NewError("PGRST201", 300, // Multiple Choices
			"Ambiguous embedding request, could be: "+strings.Join(options, ", "))
	}
	ErrPGRST202 = func(fn string) *Error {
		return NewError("PGRST202", http.StatusNotFound, fmt.Sprintf("Function '%s' not found", fn))
	}
	ErrPGRST203 = func(fn string) *Error {
		return NewError("PGRST203", 300, // Multiple Choices
			fmt.Sprintf("Multiple functions named '%s' with matching arguments", fn))
	}
	ErrPGRST204 = func(col, table string) *Error {
		return NewError("PGRST204", http.StatusBadRequest,
			fmt.Sprintf("Column '%s' not found in table '%s'", col, table)).
			WithDetails(fmt.Sprintf("Could not find column '%s' in relation '%s'", col, table))
	}
	ErrPGRST205 = func(table string) *Error {
		return NewError("PGRST205", http.StatusNotFound,
			fmt.Sprintf("Relation '%s' does not exist", table))
	}
)

// PGRST Error Codes - Group 3: JWT
var (
	ErrPGRST300 = func() *Error {
		return NewError("PGRST300", http.StatusInternalServerError, "JWT secret is not configured")
	}
	ErrPGRST301 = func(detail string) *Error {
		return NewError("PGRST301", http.StatusUnauthorized, "JWT invalid: "+detail)
	}
	ErrPGRST302 = func() *Error {
		return NewError("PGRST302", http.StatusUnauthorized, "Anonymous access disabled, authentication required")
	}
	ErrPGRST303 = func(detail string) *Error {
		return NewError("PGRST303", http.StatusUnauthorized, "JWT claims validation failed: "+detail)
	}
)

// PostgreSQL Error Code Mapping
var pgErrorStatusMap = map[string]int{
	"23505": http.StatusConflict,    // unique_violation
	"23503": http.StatusConflict,    // foreign_key_violation
	"23502": http.StatusBadRequest,  // not_null_violation
	"23514": http.StatusBadRequest,  // check_violation
	"23P01": http.StatusBadRequest,  // exclusion_violation
	"42P01": http.StatusNotFound,    // undefined_table
	"42703": http.StatusBadRequest,  // undefined_column
	"42883": http.StatusNotFound,    // undefined_function
	"42501": http.StatusForbidden,   // insufficient_privilege
	"22P02": http.StatusBadRequest,  // invalid_text_representation
	"22003": http.StatusBadRequest,  // numeric_value_out_of_range
	"22001": http.StatusBadRequest,  // string_data_right_truncation
	"22007": http.StatusBadRequest,  // invalid_datetime_format
	"21000": http.StatusBadRequest,  // cardinality_violation (mass UPDATE/DELETE)
	"P0001": http.StatusBadRequest,  // raise_exception
}

// ParsePGError converts a PostgreSQL error to a PostgREST error.
func ParsePGError(err error) *Error {
	if err == nil {
		return nil
	}

	msg := err.Error()

	// Check for common PostgreSQL error patterns
	// PostgreSQL errors typically contain "ERROR:" followed by an error code

	// Check for specific error codes in the message
	for code, status := range pgErrorStatusMap {
		if strings.Contains(msg, code) || strings.Contains(msg, "(SQLSTATE "+code+")") {
			return NewError(code, status, msg)
		}
	}

	// Check for common error patterns
	if strings.Contains(msg, "does not exist") {
		if strings.Contains(msg, "relation") {
			// Extract table name
			return ErrPGRST205(extractRelationName(msg))
		}
		if strings.Contains(msg, "column") {
			return NewError("42703", http.StatusBadRequest, msg)
		}
		if strings.Contains(msg, "function") {
			return NewError("42883", http.StatusNotFound, msg)
		}
	}

	if strings.Contains(msg, "permission denied") {
		return NewError("42501", http.StatusForbidden, msg)
	}

	if strings.Contains(msg, "unique constraint") || strings.Contains(msg, "duplicate key") {
		return NewError("23505", http.StatusConflict, msg)
	}

	if strings.Contains(msg, "foreign key constraint") {
		return NewError("23503", http.StatusConflict, msg)
	}

	if strings.Contains(msg, "not-null constraint") || strings.Contains(msg, "null value in column") {
		return NewError("23502", http.StatusBadRequest, msg)
	}

	if strings.Contains(msg, "violates check constraint") {
		return NewError("23514", http.StatusBadRequest, msg)
	}

	// Default to 400 Bad Request
	return NewError("PGRST100", http.StatusBadRequest, msg)
}

func extractRelationName(msg string) string {
	// Try to extract table name from error message like:
	// 'relation "tablename" does not exist'
	if idx := strings.Index(msg, `relation "`); idx >= 0 {
		start := idx + len(`relation "`)
		if end := strings.Index(msg[start:], `"`); end >= 0 {
			return msg[start : start+end]
		}
	}
	return "unknown"
}

// MassOperationError is returned when a mass UPDATE/DELETE is attempted without filters.
func MassOperationError() *Error {
	return NewError("21000", http.StatusBadRequest,
		"DELETE/UPDATE without filters is not allowed. Use filters or set db-aggregates-enabled").
		WithHint("Add a filter condition to limit the affected rows")
}
