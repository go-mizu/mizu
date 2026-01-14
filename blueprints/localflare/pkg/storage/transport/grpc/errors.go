// File: lib/storage/transport/grpc/errors.go

package grpc

import (
	"errors"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mapStorageError converts storage package errors to gRPC status errors.
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
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// mapGRPCError converts gRPC status errors back to storage package errors.
func mapGRPCError(err error) error {
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
	case codes.PermissionDenied:
		return storage.ErrPermission
	case codes.Unimplemented:
		return storage.ErrUnsupported
	default:
		return errors.New(st.Message())
	}
}
