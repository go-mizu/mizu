package contract

import (
	"errors"
	"fmt"
	"net/http"
)

// Code represents a portable error code aligned with gRPC status codes.
type Code string

const (
	OK                 Code = "OK"
	Canceled           Code = "CANCELED"
	Unknown            Code = "UNKNOWN"
	InvalidArgument    Code = "INVALID_ARGUMENT"
	DeadlineExceeded   Code = "DEADLINE_EXCEEDED"
	NotFound           Code = "NOT_FOUND"
	AlreadyExists      Code = "ALREADY_EXISTS"
	PermissionDenied   Code = "PERMISSION_DENIED"
	ResourceExhausted  Code = "RESOURCE_EXHAUSTED"
	FailedPrecondition Code = "FAILED_PRECONDITION"
	Aborted            Code = "ABORTED"
	OutOfRange         Code = "OUT_OF_RANGE"
	Unimplemented      Code = "UNIMPLEMENTED"
	Internal           Code = "INTERNAL"
	Unavailable        Code = "UNAVAILABLE"
	DataLoss           Code = "DATA_LOSS"
	Unauthenticated    Code = "UNAUTHENTICATED"
)

// Error is the portable error type used across all transports.
type Error struct {
	Code    Code           `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	cause   error
}

// NewError creates a new Error with the given code and message.
func NewError(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Errorf creates a new Error with formatted message.
func Errorf(code Code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
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

// WithDetails adds details to the error.
func (e *Error) WithDetails(details map[string]any) *Error {
	e.Details = details
	return e
}

// WithCause wraps an underlying error.
func (e *Error) WithCause(cause error) *Error {
	e.cause = cause
	return e
}

// HTTPStatus returns the appropriate HTTP status code.
func (e *Error) HTTPStatus() int {
	return CodeToHTTP(e.Code)
}

// CodeToHTTP maps a Code to an HTTP status code.
func CodeToHTTP(code Code) int {
	switch code {
	case OK:
		return http.StatusOK
	case Canceled:
		return 499
	case Unknown:
		return http.StatusInternalServerError
	case InvalidArgument:
		return http.StatusBadRequest
	case DeadlineExceeded:
		return http.StatusGatewayTimeout
	case NotFound:
		return http.StatusNotFound
	case AlreadyExists:
		return http.StatusConflict
	case PermissionDenied:
		return http.StatusForbidden
	case ResourceExhausted:
		return http.StatusTooManyRequests
	case FailedPrecondition:
		return http.StatusPreconditionFailed
	case Aborted:
		return http.StatusConflict
	case OutOfRange:
		return http.StatusBadRequest
	case Unimplemented:
		return http.StatusNotImplemented
	case Internal:
		return http.StatusInternalServerError
	case Unavailable:
		return http.StatusServiceUnavailable
	case DataLoss:
		return http.StatusInternalServerError
	case Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// CodeToJSONRPC maps a Code to a JSON-RPC 2.0 error code.
func CodeToJSONRPC(code Code) int {
	switch code {
	case OK:
		return 0
	case InvalidArgument:
		return -32602
	case NotFound, Unimplemented:
		return -32601
	case Internal:
		return -32603
	default:
		return -32603
	}
}

// HTTPToCode maps an HTTP status code to a Code.
func HTTPToCode(status int) Code {
	switch status {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		return OK
	case http.StatusBadRequest:
		return InvalidArgument
	case http.StatusUnauthorized:
		return Unauthenticated
	case http.StatusForbidden:
		return PermissionDenied
	case http.StatusNotFound:
		return NotFound
	case http.StatusConflict:
		return AlreadyExists
	case http.StatusPreconditionFailed:
		return FailedPrecondition
	case http.StatusTooManyRequests:
		return ResourceExhausted
	case http.StatusInternalServerError:
		return Internal
	case http.StatusNotImplemented:
		return Unimplemented
	case http.StatusServiceUnavailable:
		return Unavailable
	case http.StatusGatewayTimeout:
		return DeadlineExceeded
	default:
		if status >= 400 && status < 500 {
			return InvalidArgument
		}
		return Internal
	}
}

// AsError extracts an *Error from err, or wraps it as Internal.
func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return &Error{Code: Internal, Message: err.Error(), cause: err}
}
