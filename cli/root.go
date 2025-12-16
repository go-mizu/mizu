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
	chdir   string
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
	{name: "serve", short: "Run the project (detects main package or cmd/*)", run: runServe, usage: usageServe},
	{name: "doctor", short: "Diagnose environment, module, and project layout", run: runDoctor, usage: usageDoctor},
	{name: "version", short: "Print version information", run: runVersion, usage: usageVersion},
}

// Run is the main entry point for the CLI.
func Run() int {
	gf := &globalFlags{}

	// Manual flag parsing for global flags before subcommand
	args := os.Args[1:]
	args, gf = parseGlobalFlags(args, gf)

	// Handle --chdir
	if gf.chdir != "" {
		if err := os.Chdir(gf.chdir); err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot change to directory %q: %v\n", gf.chdir, err)
			return exitError
		}
	}

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

//nolint:cyclop // flag parsing switch statement
func parseGlobalFlag(arg string, args []string, i *int, gf *globalFlags) bool {
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
	case "-C", "--chdir":
		*i++
		if *i < len(args) {
			gf.chdir = args[*i]
		}
		return true
	default:
		// Check for -C=value form
		if strings.HasPrefix(arg, "-C=") {
			gf.chdir = strings.TrimPrefix(arg, "-C=")
			return true
		}
		if strings.HasPrefix(arg, "--chdir=") {
			gf.chdir = strings.TrimPrefix(arg, "--chdir=")
			return true
		}
		return false
	}
}

func printUsage() {
	fmt.Println("mizu is the project toolkit for go-mizu.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  mizu <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	for _, cmd := range commands {
		fmt.Printf("  %-10s %s\n", cmd.name, cmd.short)
	}
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -C, --chdir <dir>        Run as if started in <dir>")
	fmt.Println("      --json               Emit machine-readable output where supported")
	fmt.Println("      --no-color           Disable color output")
	fmt.Println("  -q, --quiet              Reduce output (still prints errors)")
	fmt.Println("  -v, --verbose            More diagnostics (repeatable)")
	fmt.Println("  -h, --help               Show help")
}
