// Package flight provides an Apache Arrow Flight transport layer for storage.Storage backends.
//
// This package implements a Flight server that exposes any storage.Storage
// implementation over Arrow Flight RPC, enabling high-performance columnar
// data transfer with zero-copy serialization.
//
// Example:
//
//	store, _ := storage.Open(ctx, "local:///data")
//
//	cfg := &flight.Config{
//	    Addr: ":8080",
//	    Auth: &flight.AuthConfig{
//	        TokenValidator: validateToken,
//	    },
//	}
//
//	server := flight.New(store, cfg)
//	server.Serve()
package flight

import (
	"context"
	"errors"

	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mapStorageError converts a storage error to a gRPC status error.
func mapStorageError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, storage.ErrNotExist):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, storage.ErrExist):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, storage.ErrPermission):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, storage.ErrUnsupported):
		return status.Error(codes.Unimplemented, err.Error())
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// mapStatusError converts a gRPC status error to a storage error.
func mapStatusError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	switch st.Code() {
	case codes.NotFound:
		return storage.ErrNotExist
	case codes.AlreadyExists:
		return storage.ErrExist
	case codes.PermissionDenied, codes.Unauthenticated:
		return storage.ErrPermission
	case codes.Unimplemented:
		return storage.ErrUnsupported
	case codes.Canceled:
		return context.Canceled
	case codes.DeadlineExceeded:
		return context.DeadlineExceeded
	case codes.InvalidArgument:
		return errors.New(st.Message())
	case codes.OK:
		return nil
	default:
		return errors.New(st.Message())
	}
}

// isNotFoundError checks if an error is a not found error.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, storage.ErrNotExist) {
		return true
	}
	st, ok := status.FromError(err)
	return ok && st.Code() == codes.NotFound
}

// isPermissionError checks if an error is a permission error.
func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, storage.ErrPermission) {
		return true
	}
	st, ok := status.FromError(err)
	return ok && (st.Code() == codes.PermissionDenied || st.Code() == codes.Unauthenticated)
}

// isUnsupportedError checks if an error is an unsupported error.
func isUnsupportedError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, storage.ErrUnsupported) {
		return true
	}
	st, ok := status.FromError(err)
	return ok && st.Code() == codes.Unimplemented
}

// invalidArgumentError creates an invalid argument error.
func invalidArgumentError(msg string) error {
	return status.Error(codes.InvalidArgument, msg)
}

// unauthenticatedError creates an unauthenticated error.
func unauthenticatedError(msg string) error {
	return status.Error(codes.Unauthenticated, msg)
}
