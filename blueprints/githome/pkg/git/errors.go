package git

import "errors"

var (
	// ErrNotFound indicates a git object was not found
	ErrNotFound = errors.New("object not found")

	// ErrInvalidSHA indicates an invalid SHA format
	ErrInvalidSHA = errors.New("invalid SHA")

	// ErrRefNotFound indicates a reference was not found
	ErrRefNotFound = errors.New("reference not found")

	// ErrRefExists indicates a reference already exists
	ErrRefExists = errors.New("reference already exists")

	// ErrNotARepository indicates the path is not a git repository
	ErrNotARepository = errors.New("not a git repository")

	// ErrEmptyRepository indicates the repository has no commits
	ErrEmptyRepository = errors.New("repository is empty")

	// ErrInvalidRef indicates an invalid reference name
	ErrInvalidRef = errors.New("invalid reference name")

	// ErrNonFastForward indicates a non-fast-forward update was rejected
	ErrNonFastForward = errors.New("non-fast-forward update rejected")
)
