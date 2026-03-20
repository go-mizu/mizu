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
func New(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "CLI for storage.now",
		Long: `storage — CLI for storage.now
https://storage.liteio.dev

Upload, download, and share files from your terminal.
One binary, zero configuration, pipe-friendly.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Intercept errors after execution to print them nicely.
		},
	}

	// Global flags
	pf := cmd.PersistentFlags()
	pf.BoolVarP(&globalFlags.json, "json", "j", false, "JSON output")
	pf.BoolVarP(&globalFlags.quiet, "quiet", "q", false, "suppress non-essential output")
	pf.BoolVar(&globalFlags.noColor, "no-color", false, "disable colors")
	pf.StringVarP(&globalFlags.token, "token", "t", "", "bearer token or API key")
	pf.StringVarP(&globalFlags.endpoint, "endpoint", "e", "", "API base URL")

	// Override version template
	cmd.SetVersionTemplate("storage {{.Version}}\n")

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
		newCpCmd(),
		newShareCmd(),
		newInfoCmd(),
		newSearchCmd(),
		newStatsCmd(),
		newBucketCmd(),
		newKeyCmd(),
	)

	// Custom error handling
	originalRun := cmd.PersistentPostRunE
	cmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		if originalRun != nil {
			return originalRun(cmd, args)
		}
		return nil
	}

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
			out.PrintError("permission denied", apiErr.Message, "Check your API key scopes")
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
