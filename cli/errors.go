package cli

import "fmt"

// Exit codes for CLI operations.
const (
	exitOK        = 0
	exitError     = 1
	exitUsage     = 2
	exitNoProject = 3
)

// cliError represents a CLI error with exit code and optional hint.
type cliError struct {
	code    int
	message string
	hint    string
}

func (e *cliError) Error() string {
	if e.hint != "" {
		return fmt.Sprintf("%s\n  hint: %s", e.message, e.hint)
	}
	return e.message
}

func errNoProject(msg string) *cliError {
	return &cliError{code: exitNoProject, message: msg}
}
