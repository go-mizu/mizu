package storage

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var globalFlags struct {
	json     bool
	quiet    bool
	noColor  bool
	token    string
	endpoint string
}

// New returns the root storage command.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "CLI for Liteio Storage API",
		Long: `storage — CLI for Liteio Storage API
https://storage.liteio.dev

Upload, download, and share files from your terminal.
Zero dependencies, pipe-friendly, works everywhere.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Example: `  storage login
  storage put report.pdf docs/
  storage ls docs/
  storage get docs/report.pdf
  storage share docs/report.pdf --expires 7d
  echo "hello" | storage put - notes/hello.txt
  storage cat docs/data.json | jq '.items'
  storage find quarterly --json`,
	}

	// Global flags
	pf := cmd.PersistentFlags()
	pf.BoolVarP(&globalFlags.json, "json", "j", false, "JSON output")
	pf.BoolVarP(&globalFlags.quiet, "quiet", "q", false, "suppress non-essential output")
	pf.BoolVar(&globalFlags.noColor, "no-color", false, "disable colors")
	pf.StringVarP(&globalFlags.token, "token", "t", "", "bearer token or API key")
	pf.StringVarP(&globalFlags.endpoint, "endpoint", "e", "", "API base URL")

	// Register commands
	cmd.AddCommand(
		newLoginCmd(),
		newLogoutCmd(),
		newTokenCmd(),
		newLsCmd(),
		newPutCmd(),
		newGetCmd(),
		newCatCmd(),
		newRmCmd(),
		newMvCmd(),
		newShareCmd(),
		newFindCmd(),
		newStatCmd(),
		newKeyCmd(),
	)

	return cmd
}

// deps creates shared dependencies from global flags.
func deps() *Deps {
	cfg := LoadConfig(globalFlags.token, globalFlags.endpoint)
	return &Deps{
		Config: cfg,
		Client: NewClient(cfg),
		Out:    NewOutput(globalFlags.noColor, globalFlags.quiet),
	}
}

// handleError prints a CLI error and exits with the appropriate code.
func handleError(cmd *cobra.Command, err error) {
	out := NewOutput(globalFlags.noColor, globalFlags.quiet)

	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		out.PrintError(cliErr.Msg, cliErr.Hint, "")
		os.Exit(cliErr.Code)
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401:
			out.PrintError("authentication failed", apiErr.Message, "Run 'storage login' to re-authenticate")
		case 403:
			out.PrintError("permission denied", apiErr.Message, "Check your API key prefix restrictions")
		case 404:
			out.PrintError("not found", apiErr.Message, "")
		case 409:
			out.PrintError("conflict", apiErr.Message, "")
		default:
			out.PrintError(fmt.Sprintf("request failed (%d)", apiErr.StatusCode), apiErr.Message, "")
		}
		os.Exit(apiErr.ExitCode())
	}

	out.PrintError(err.Error(), "", "")
	os.Exit(ExitError)
}

// wrapRun wraps a command function with error handling.
func wrapRun(fn func(*cobra.Command, []string) error) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		if err := fn(cmd, args); err != nil {
			handleError(cmd, err)
		}
	}
}
