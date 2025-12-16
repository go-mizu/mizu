package contract

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode represents a portable error code that maps consistently across transports.
// Codes are aligned with gRPC status codes for interoperability.
type ErrorCode string

const (
	ErrCodeOK                 ErrorCode = "OK"
	ErrCodeCanceled           ErrorCode = "CANCELED"
	ErrCodeUnknown            ErrorCode = "UNKNOWN"
	ErrCodeInvalidArgument    ErrorCode = "INVALID_ARGUMENT"
	ErrCodeDeadlineExceeded   ErrorCode = "DEADLINE_EXCEEDED"
	ErrCodeNotFound           ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists      ErrorCode = "ALREADY_EXISTS"
	ErrCodePermissionDenied   ErrorCode = "PERMISSION_DENIED"
	ErrCodeResourceExhausted  ErrorCode = "RESOURCE_EXHAUSTED"
	ErrCodeFailedPrecondition ErrorCode = "FAILED_PRECONDITION"
	ErrCodeAborted            ErrorCode = "ABORTED"
	ErrCodeOutOfRange         ErrorCode = "OUT_OF_RANGE"
	ErrCodeUnimplemented      ErrorCode = "UNIMPLEMENTED"
	ErrCodeInternal           ErrorCode = "INTERNAL"
	ErrCodeUnavailable        ErrorCode = "UNAVAILABLE"
	ErrCodeDataLoss           ErrorCode = "DATA_LOSS"
	ErrCodeUnauthenticated    ErrorCode = "UNAUTHENTICATED"
)

// Error is the portable error type used across all transports.
// It implements the error interface and provides consistent mapping
// to HTTP status codes, JSON-RPC error codes, and gRPC status codes.
type Error struct {
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	cause   error
}

// NewError creates a new Error with the given code and message.
func NewError(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Errorf creates a new Error with formatted message.
func Errorf(code ErrorCode, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// WithDetails adds details to the error.
func (e *Error) WithDetails(details map[string]any) *Error {
	e.Details = details
	return e
}

// WithDetail adds a single detail key-value pair.
func (e *Error) WithDetail(key string, value any) *Error {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// WithCause wraps an underlying error.
func (e *Error) WithCause(cause error) *Error {
	e.cause = cause
	return e
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return string(e.Code)
}

// Unwrap returns the underlying cause.
func (e *Error) Unwrap() error {
	return e.cause
}

// Is reports whether target matches this error.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// HTTPStatus returns the appropriate HTTP status code.
func (e *Error) HTTPStatus() int {
	return ErrorCodeToHTTPStatus(e.Code)
}

// JSONRPCCode returns the JSON-RPC 2.0 error code.
func (e *Error) JSONRPCCode() int {
	return ErrorCodeToJSONRPC(e.Code)
}

// GRPCCode returns the gRPC status code.
func (e *Error) GRPCCode() int {
	return ErrorCodeToGRPC(e.Code)
}

// ErrorCodeToHTTPStatus maps an ErrorCode to an HTTP status code.
func ErrorCodeToHTTPStatus(code ErrorCode) int {
	switch code {
	case ErrCodeOK:
		return http.StatusOK
	case ErrCodeCanceled:
		return 499 // Client Closed Request
	case ErrCodeUnknown:
		return http.StatusInternalServerError
	case ErrCodeInvalidArgument:
		return http.StatusBadRequest
	case ErrCodeDeadlineExceeded:
		return http.StatusGatewayTimeout
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeAlreadyExists:
		return http.StatusConflict
	case ErrCodePermissionDenied:
		return http.StatusForbidden
	case ErrCodeResourceExhausted:
		return http.StatusTooManyRequests
	case ErrCodeFailedPrecondition:
		return http.StatusPreconditionFailed
	case ErrCodeAborted:
		return http.StatusConflict
	case ErrCodeOutOfRange:
		return http.StatusBadRequest
	case ErrCodeUnimplemented:
		return http.StatusNotImplemented
	case ErrCodeInternal:
		return http.StatusInternalServerError
	case ErrCodeUnavailable:
		return http.StatusServiceUnavailable
	case ErrCodeDataLoss:
		return http.StatusInternalServerError
	case ErrCodeUnauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// ErrorCodeToJSONRPC maps an ErrorCode to a JSON-RPC 2.0 error code.
// Standard JSON-RPC codes: -32700 to -32600
// Server errors: -32099 to -32000
// Application errors: -32099 to -32000 (we use -32000 - grpc_code)
func ErrorCodeToJSONRPC(code ErrorCode) int {
	switch code {
	case ErrCodeOK:
		return 0
	case ErrCodeCanceled:
		return -32001
	case ErrCodeUnknown:
		return -32002
	case ErrCodeInvalidArgument:
		return -32602 // Invalid params (standard)
	case ErrCodeDeadlineExceeded:
		return -32004
	case ErrCodeNotFound:
		return -32601 // Method not found (closest match)
	case ErrCodeAlreadyExists:
		return -32006
	case ErrCodePermissionDenied:
		return -32007
	case ErrCodeResourceExhausted:
		return -32008
	case ErrCodeFailedPrecondition:
		return -32009
	case ErrCodeAborted:
		return -32010
	case ErrCodeOutOfRange:
		return -32011
	case ErrCodeUnimplemented:
		return -32601 // Method not found
	case ErrCodeInternal:
		return -32603 // Internal error (standard)
	case ErrCodeUnavailable:
		return -32014
	case ErrCodeDataLoss:
		return -32015
	case ErrCodeUnauthenticated:
		return -32016
	default:
		return -32603
	}
}

// ErrorCodeToGRPC maps an ErrorCode to a gRPC status code.
func ErrorCodeToGRPC(code ErrorCode) int {
	switch code {
	case ErrCodeOK:
		return 0
	case ErrCodeCanceled:
		return 1
	case ErrCodeUnknown:
		return 2
	case ErrCodeInvalidArgument:
		return 3
	case ErrCodeDeadlineExceeded:
		return 4
	case ErrCodeNotFound:
		return 5
	case ErrCodeAlreadyExists:
		return 6
	case ErrCodePermissionDenied:
		return 7
	case ErrCodeResourceExhausted:
		return 8
	case ErrCodeFailedPrecondition:
		return 9
	case ErrCodeAborted:
		return 10
	case ErrCodeOutOfRange:
		return 11
	case ErrCodeUnimplemented:
		return 12
	case ErrCodeInternal:
		return 13
	case ErrCodeUnavailable:
		return 14
	case ErrCodeDataLoss:
		return 15
	case ErrCodeUnauthenticated:
		return 16
	default:
		return 2 // UNKNOWN
	}
}

// HTTPStatusToErrorCode maps an HTTP status code to an ErrorCode.
func HTTPStatusToErrorCode(status int) ErrorCode {
	switch status {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		return ErrCodeOK
	case http.StatusBadRequest:
		return ErrCodeInvalidArgument
	case http.StatusUnauthorized:
		return ErrCodeUnauthenticated
	case http.StatusForbidden:
		return ErrCodePermissionDenied
	case http.StatusNotFound:
		return ErrCodeNotFound
	case http.StatusConflict:
		return ErrCodeAlreadyExists
	case http.StatusPreconditionFailed:
		return ErrCodeFailedPrecondition
	case http.StatusTooManyRequests:
		return ErrCodeResourceExhausted
	case http.StatusInternalServerError:
		return ErrCodeInternal
	case http.StatusNotImplemented:
		return ErrCodeUnimplemented
	case http.StatusServiceUnavailable:
		return ErrCodeUnavailable
	case http.StatusGatewayTimeout:
		return ErrCodeDeadlineExceeded
	default:
		if status >= 400 && status < 500 {
			return ErrCodeInvalidArgument
		}
		return ErrCodeInternal
	}
}

// AsError extracts an *Error from err, or wraps it if not already an *Error.
func AsError(err error) *Error {
	if err == nil {
		return nil
	}

	var e *Error
	if errors.As(err, &e) {
		return e
	}

	// Wrap unknown errors as INTERNAL
	return &Error{
		Code:    ErrCodeInternal,
		Message: err.Error(),
		cause:   err,
	}
}

// Common error constructors for convenience

// ErrInvalidArgument creates an INVALID_ARGUMENT error.
func ErrInvalidArgument(message string) *Error {
	return NewError(ErrCodeInvalidArgument, message)
}

// ErrNotFound creates a NOT_FOUND error.
func ErrNotFound(message string) *Error {
	return NewError(ErrCodeNotFound, message)
}

// ErrAlreadyExists creates an ALREADY_EXISTS error.
func ErrAlreadyExists(message string) *Error {
	return NewError(ErrCodeAlreadyExists, message)
}

// ErrPermissionDenied creates a PERMISSION_DENIED error.
func ErrPermissionDenied(message string) *Error {
	return NewError(ErrCodePermissionDenied, message)
}

// ErrUnauthenticated creates an UNAUTHENTICATED error.
func ErrUnauthenticated(message string) *Error {
	return NewError(ErrCodeUnauthenticated, message)
}

// ErrInternal creates an INTERNAL error.
func ErrInternal(message string) *Error {
	return NewError(ErrCodeInternal, message)
}

// ErrUnimplemented creates an UNIMPLEMENTED error.
func ErrUnimplemented(message string) *Error {
	return NewError(ErrCodeUnimplemented, message)
}

// ErrUnavailable creates an UNAVAILABLE error.
func ErrUnavailable(message string) *Error {
	return NewError(ErrCodeUnavailable, message)
}

// ErrResourceExhausted creates a RESOURCE_EXHAUSTED error.
func ErrResourceExhausted(message string) *Error {
	return NewError(ErrCodeResourceExhausted, message)
}

// ErrFailedPrecondition creates a FAILED_PRECONDITION error.
func ErrFailedPrecondition(message string) *Error {
	return NewError(ErrCodeFailedPrecondition, message)
}

// ErrAborted creates an ABORTED error.
func ErrAborted(message string) *Error {
	return NewError(ErrCodeAborted, message)
}

// ErrDeadlineExceeded creates a DEADLINE_EXCEEDED error.
func ErrDeadlineExceeded(message string) *Error {
	return NewError(ErrCodeDeadlineExceeded, message)
}
