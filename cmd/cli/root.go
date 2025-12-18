package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/mango-cobra"
	"github.com/muesli/roff"
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

// manCmd generates man pages (hidden).
var manCmd = &cobra.Command{
	Use:    "man",
	Short:  "Generate man pages",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		manPage, err := mcobra.NewManPage(1, rootCmd)
		if err != nil {
			return err
		}

		manPage = manPage.WithSection("Authors", "The go-mizu contributors <https://github.com/go-mizu/mizu>")
		manPage = manPage.WithSection("Bugs", "Report bugs at https://github.com/go-mizu/mizu/issues")

		fmt.Println(manPage.Build(roff.NewDocument()))
		return nil
	},
}

// completionCmd generates shell completions.
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for mizu.

To load completions:

Bash:
  $ source <(mizu completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ mizu completion bash > /etc/bash_completion.d/mizu
  # macOS:
  $ mizu completion bash > $(brew --prefix)/etc/bash_completion.d/mizu

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  # To load completions for each session, execute once:
  $ mizu completion zsh > "${fpath[1]}/_mizu"
  # You will need to start a new shell for this setup to take effect.

Fish:
  $ mizu completion fish | source
  # To load completions for each session, execute once:
  $ mizu completion fish > ~/.config/fish/completions/mizu.fish

PowerShell:
  PS> mizu completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> mizu completion powershell > mizu.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unknown shell: %s", args[0])
		}
	},
}

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
	rootCmd.AddCommand(manCmd)
	rootCmd.AddCommand(completionCmd)
}

// Run is the main entry point for the CLI.
func Run() int {
	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		Flags.NoColor = true
	}

	if err := rootCmd.Execute(); err != nil {
		// Print styled error
		out := NewOutput()
		out.Errorf("Error: %v\n", err)
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
