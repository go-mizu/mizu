package cli

import (
	"fmt"
	"runtime"
)

type versionInfo struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	Commit    string `json:"commit"`
	BuiltAt   string `json:"built_at"`
}

func runVersion(args []string, gf *globalFlags) int {
	out := newOutput(gf.json, gf.quiet, gf.noColor, gf.verbose)

	// Check for help flag
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			usageVersion()
			return exitOK
		}
	}

	info := versionInfo{
		Version:   Version,
		GoVersion: runtime.Version(),
		Commit:    Commit,
		BuiltAt:   BuildTime,
	}

	if out.json {
		out.writeJSON(info)
		return exitOK
	}

	out.print("mizu version %s\n", info.Version)
	out.print("go version: %s\n", info.GoVersion)
	out.print("commit: %s\n", info.Commit)
	out.print("built: %s\n", info.BuiltAt)

	return exitOK
}

func usageVersion() {
	fmt.Println("Usage:")
	fmt.Println("  mizu version [flags]")
	fmt.Println()
	fmt.Println("Print version information.")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("      --json    Output as JSON")
	fmt.Println("  -h, --help    Show help")
}
