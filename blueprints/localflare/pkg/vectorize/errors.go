package vectorize

import "errors"

// Common errors returned by vector database operations.
var (
	// ErrIndexNotFound is returned when an index doesn't exist.
	ErrIndexNotFound = errors.New("vectorize: index not found")

	// ErrIndexExists is returned when trying to create an index that already exists.
	ErrIndexExists = errors.New("vectorize: index already exists")

	// ErrVectorNotFound is returned when a vector doesn't exist.
	ErrVectorNotFound = errors.New("vectorize: vector not found")

	// ErrVectorExists is returned when inserting a vector that already exists.
	ErrVectorExists = errors.New("vectorize: vector already exists")

	// ErrDimensionMismatch is returned when vector dimensions don't match the index.
	ErrDimensionMismatch = errors.New("vectorize: vector dimension mismatch")

	// ErrInvalidDSN is returned when the connection string is malformed.
	ErrInvalidDSN = errors.New("vectorize: invalid DSN")

	// ErrConnectionFailed is returned when unable to connect to the database.
	ErrConnectionFailed = errors.New("vectorize: connection failed")

	// ErrClosed is returned when operating on a closed connection.
	ErrClosed = errors.New("vectorize: connection closed")

	// ErrInvalidMetric is returned when an unsupported distance metric is specified.
	ErrInvalidMetric = errors.New("vectorize: invalid distance metric")

	// ErrEmptyVector is returned when an empty vector is provided.
	ErrEmptyVector = errors.New("vectorize: empty vector")

	// ErrInvalidFilter is returned when a filter expression is invalid.
	ErrInvalidFilter = errors.New("vectorize: invalid filter")

	// ErrNotSupported is returned when an operation is not supported by the driver.
	ErrNotSupported = errors.New("vectorize: operation not supported")
)
