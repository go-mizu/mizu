package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// Version information set at build time.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// GlobalFlags holds flags shared across all commands.
type GlobalFlags struct {
	JSON    bool
	NoColor bool
	Quiet   bool
	Verbose int
	MD      bool // print markdown help
}

// Flags is the global flags instance.
var Flags = &GlobalFlags{}

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "mizu",
	Short: "Project toolkit for the go-mizu framework",
	Long: `mizu is the project toolkit for the go-mizu framework.

It provides commands for creating new projects, running development servers,
working with service contracts, and exploring available middlewares.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if Flags.MD {
			return printMarkdownHelp(cmd)
		}
		return cmd.Help()
	},
}

// Note: man page generation and shell completions are now handled by Fang automatically

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&Flags.JSON, "json", false, "Emit machine readable output")
	rootCmd.PersistentFlags().BoolVar(&Flags.NoColor, "no-color", false, "Disable color output")
	rootCmd.PersistentFlags().BoolVarP(&Flags.Quiet, "quiet", "q", false, "Reduce output (errors only)")
	rootCmd.PersistentFlags().CountVarP(&Flags.Verbose, "verbose", "v", "Increase verbosity (repeatable)")
	rootCmd.PersistentFlags().BoolVar(&Flags.MD, "md", false, "Print help as rendered markdown")

	// Register subcommands
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(contractCmd)
	rootCmd.AddCommand(middlewareCmd)
	rootCmd.AddCommand(versionCmd)
	// Note: man and completion commands are now provided by Fang automatically
}

// Run is the main entry point for the CLI.
func Run() int {
	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		Flags.NoColor = true
	}

	// Execute with Fang for enhanced DX
	// Fang provides styled help, automatic version handling, man pages, and completions
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithVersion(Version),
		fang.WithCommit(Commit),
	); err != nil {
		return 1
	}
	return 0
}

// Styles for CLI output
var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	cyanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	boldStyle    = lipgloss.NewStyle().Bold(true)
)

// printMarkdownHelp renders and prints markdown help for a command.
func printMarkdownHelp(cmd *cobra.Command) error {
	name := cmd.Name()
	if cmd.Parent() != nil && cmd.Parent().Name() != "mizu" && cmd.Parent().Name() != "" {
		name = cmd.Parent().Name() + "-" + name
	}

	// Try to find matching markdown file
	content, err := DocsFS.ReadFile("docs/commands/" + name + ".md")
	if err != nil {
		// Fall back to main manual
		content, err = DocsFS.ReadFile("docs/mizu.md")
		if err != nil {
			// Generate help from command
			return cmd.Help()
		}
	}

	return renderMarkdown(os.Stdout, string(content))
}

// renderMarkdown renders markdown content with glamour.
func renderMarkdown(w *os.File, content string) error {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		// Fallback: print raw markdown
		fmt.Fprint(w, content)
		return nil
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		fmt.Fprint(w, content)
		return nil
	}

	fmt.Fprint(w, rendered)
	return nil
}

// wrapRunE wraps a command's RunE to handle --md flag.
func wrapRunE(original func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if Flags.MD {
			return printMarkdownHelp(cmd)
		}
		return original(cmd, args)
	}
}
