package cli

import (
	"fmt"
	"os"
	"strings"
)

// Version information set at build time.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// globalFlags holds flags shared across all commands.
type globalFlags struct {
	json    bool
	noColor bool
	quiet   bool
	verbose int
	help    bool
}

// command represents a CLI subcommand.
type command struct {
	name  string
	short string
	run   func(args []string, gf *globalFlags) int
	usage func()
}

var commands = []*command{
	{name: "new", short: "Create a new project from a template", run: runNew, usage: usageNew},
	{name: "dev", short: "Run the current project in development mode", run: runDev, usage: usageDev},
	{name: "contract", short: "Work with service contracts", run: runContract, usage: usageContract},
	{name: "middleware", short: "Explore available middlewares", run: runMiddleware, usage: usageMiddleware},
	{name: "version", short: "Print version information", run: runVersion, usage: usageVersion},
}

// hiddenCommands are aliases that print deprecation notices
var hiddenCommands = map[string]func(args []string, gf *globalFlags) int{
	"serve": runServeDeprecated,
}

// Run is the main entry point for the CLI.
func Run() int {
	gf := &globalFlags{}

	// Manual flag parsing for global flags before subcommand
	args := os.Args[1:]
	args, gf = parseGlobalFlags(args, gf)

	// No subcommand or help flag
	if len(args) == 0 || gf.help {
		printUsage()
		if gf.help {
			return exitOK
		}
		return exitUsage
	}

	// Find and run subcommand
	cmdName := args[0]
	cmdArgs := args[1:]

	for _, cmd := range commands {
		if cmd.name == cmdName {
			return cmd.run(cmdArgs, gf)
		}
	}

	// Check hidden commands (deprecated aliases)
	if hidden, ok := hiddenCommands[cmdName]; ok {
		return hidden(cmdArgs, gf)
	}

	// Unknown command
	fmt.Fprintf(os.Stderr, "error: unknown command %q\n", cmdName)
	fmt.Fprintf(os.Stderr, "Run 'mizu --help' for usage.\n")
	return exitUsage
}

func parseGlobalFlags(args []string, gf *globalFlags) ([]string, *globalFlags) {
	var remaining []string
	i := 0

	// First pass: parse global flags before subcommand
	for i < len(args) {
		arg := args[i]

		// Stop at first non-flag argument (the subcommand)
		if !strings.HasPrefix(arg, "-") {
			remaining = append(remaining, arg)
			i++
			break
		}

		if !parseGlobalFlag(arg, args, &i, gf) {
			// Unknown flag before subcommand, keep it
			remaining = append(remaining, arg)
			i++
		} else {
			i++
		}
	}

	// Second pass: extract global flags from remaining args (after subcommand)
	for i < len(args) {
		arg := args[i]
		if parseGlobalFlag(arg, args, &i, gf) {
			i++
			continue
		}
		remaining = append(remaining, arg)
		i++
	}

	return remaining, gf
}

func parseGlobalFlag(arg string, _ []string, _ *int, gf *globalFlags) bool {
	switch arg {
	case "-h", "--help":
		gf.help = true
		return true
	case "-q", "--quiet":
		gf.quiet = true
		return true
	case "-v", "--verbose":
		gf.verbose++
		return true
	case "--json":
		gf.json = true
		return true
	case "--no-color":
		gf.noColor = true
		return true
	default:
		return false
	}
}

func printUsage() {
	fmt.Println("mizu is the project toolkit for the go-mizu framework.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  mizu <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	for _, cmd := range commands {
		fmt.Printf("  %-10s %s\n", cmd.name, cmd.short)
	}
	fmt.Println()
	fmt.Println("Global Flags:")
	fmt.Println("      --json               Emit machine readable output")
	fmt.Println("      --no-color           Disable color output")
	fmt.Println("  -q, --quiet              Reduce output (errors only)")
	fmt.Println("  -v, --verbose            Increase verbosity (repeatable)")
	fmt.Println("  -h, --help               Show help")
}

// runServeDeprecated prints deprecation notice and runs dev command
func runServeDeprecated(args []string, gf *globalFlags) int {
	fmt.Fprintln(os.Stderr, "mizu serve is deprecated. Use: mizu dev")
	return runDev(args, gf)
}
