package cli

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// finding represents a diagnostic result.
type finding struct {
	Level   string `json:"level"` // "ok", "info", "warn", "error"
	Message string `json:"message"`
}

// doctorResult holds all diagnostic findings.
type doctorResult struct {
	Environment []finding `json:"environment"`
	Project     []finding `json:"project"`
}

func runDoctor(args []string, gf *globalFlags) int {
	out := newOutput(gf.json, gf.quiet, gf.noColor, gf.verbose)

	// Parse flags
	var fix bool
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.BoolVar(&fix, "fix", false, "Apply safe fixes")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			usageDoctor()
			return exitOK
		}
		return exitUsage
	}

	result := &doctorResult{}

	// Check environment
	if !out.json && !out.quiet {
		out.print("Checking environment...\n")
	}
	result.Environment = checkEnvironment(out)

	// Check project
	if !out.json && !out.quiet {
		out.print("\nChecking project...\n")
	}
	result.Project = checkProject(out, fix)

	// JSON output
	if out.json {
		out.writeJSON(result)
		return exitOK
	}

	// Summary
	errors := countLevel(result, "error")
	warns := countLevel(result, "warn")

	if errors > 0 {
		out.print("\n%s found, %s\n",
			out.red(fmt.Sprintf("%d %s", errors, pluralize(errors, "error", "errors"))),
			out.yellow(fmt.Sprintf("%d %s", warns, pluralize(warns, "warning", "warnings"))))
		return exitError
	}

	if warns > 0 {
		out.print("\n%s\n", out.yellow(fmt.Sprintf("%d %s found", warns, pluralize(warns, "warning", "warnings"))))
	} else {
		out.print("\n%s\n", out.green("All checks passed"))
	}

	return exitOK
}

func checkEnvironment(out *output) []finding {
	var findings []finding

	// Go version
	goVersion := runtime.Version()
	findings = append(findings, finding{Level: "ok", Message: fmt.Sprintf("go version: %s", goVersion)})
	printFinding(out, findings[len(findings)-1])

	// Go path
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		findings = append(findings, finding{Level: "ok", Message: fmt.Sprintf("GOPATH: %s", gopath)})
	} else {
		findings = append(findings, finding{Level: "info", Message: "GOPATH not set (using default)"})
	}
	printFinding(out, findings[len(findings)-1])

	// GO111MODULE
	gomod := os.Getenv("GO111MODULE")
	if gomod == "" || gomod == "on" || gomod == "auto" {
		findings = append(findings, finding{Level: "ok", Message: "GO111MODULE: " + valueOr(gomod, "auto (default)")})
	} else {
		findings = append(findings, finding{Level: "warn", Message: fmt.Sprintf("GO111MODULE=%s (expected on/auto)", gomod)})
	}
	printFinding(out, findings[len(findings)-1])

	// Check go command works
	if _, err := exec.LookPath("go"); err != nil {
		findings = append(findings, finding{Level: "error", Message: "go command not found in PATH"})
		printFinding(out, findings[len(findings)-1])
	}

	return findings
}

//nolint:cyclop // diagnostic checks with multiple steps
func checkProject(out *output, fix bool) []finding {
	var findings []finding
	cwd, _ := os.Getwd()

	// Check go.mod
	gomodPath, err := findGoMod(cwd)
	if err != nil || gomodPath == "" {
		findings = append(findings, finding{Level: "error", Message: "go.mod not found"})
		printFinding(out, findings[len(findings)-1])
		return findings
	}

	findings = append(findings, finding{Level: "ok", Message: "go.mod found"})
	printFinding(out, findings[len(findings)-1])

	// Parse go.mod for module name
	gomodContent, err := readFileString(gomodPath)
	if err != nil {
		findings = append(findings, finding{Level: "error", Message: fmt.Sprintf("cannot read go.mod: %v", err)})
		printFinding(out, findings[len(findings)-1])
		return findings
	}

	moduleName := parseModuleName(gomodContent)
	if moduleName != "" {
		findings = append(findings, finding{Level: "ok", Message: fmt.Sprintf("module: %s", moduleName)})
		printFinding(out, findings[len(findings)-1])
	}

	// Check for replace directives
	if strings.Contains(gomodContent, "replace ") {
		findings = append(findings, finding{Level: "info", Message: "go.mod contains replace directives"})
		printFinding(out, findings[len(findings)-1])
	}

	// Check project structure
	projectRoot := filepath.Dir(gomodPath)

	// Check for cmd directory
	cmdDir := filepath.Join(projectRoot, "cmd")
	if dirExists(cmdDir) {
		entries, _ := os.ReadDir(cmdDir)
		cmdCount := 0
		for _, e := range entries {
			if e.IsDir() {
				cmdCount++
			}
		}
		if cmdCount > 0 {
			findings = append(findings, finding{Level: "ok", Message: fmt.Sprintf("cmd/ contains %d %s", cmdCount, pluralize(cmdCount, "command", "commands"))})
		} else {
			findings = append(findings, finding{Level: "warn", Message: "cmd/ directory is empty"})
		}
	} else {
		// Check for main.go in root
		if fileExists(filepath.Join(projectRoot, "main.go")) {
			findings = append(findings, finding{Level: "ok", Message: "main.go found in project root"})
		} else {
			findings = append(findings, finding{Level: "warn", Message: "no cmd/ directory or main.go found"})
		}
	}
	printFinding(out, findings[len(findings)-1])

	// Apply fixes if requested
	if fix {
		fixFindings := applyFixes(out, projectRoot)
		findings = append(findings, fixFindings...)
	}

	return findings
}

func applyFixes(out *output, projectRoot string) []finding {
	var findings []finding

	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectRoot
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		findings = append(findings, finding{Level: "error", Message: fmt.Sprintf("go mod tidy failed: %s", stderr.String())})
	} else {
		findings = append(findings, finding{Level: "ok", Message: "ran go mod tidy"})
	}
	printFinding(out, findings[len(findings)-1])

	// Run go fmt
	cmd = exec.Command("go", "fmt", "./...")
	cmd.Dir = projectRoot
	stderr.Reset()
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		findings = append(findings, finding{Level: "warn", Message: fmt.Sprintf("go fmt failed: %s", stderr.String())})
	} else {
		findings = append(findings, finding{Level: "ok", Message: "ran go fmt"})
	}
	printFinding(out, findings[len(findings)-1])

	return findings
}

func printFinding(out *output, f finding) {
	if out.json || out.quiet {
		return
	}

	switch f.Level {
	case "ok":
		out.ok(f.Message)
	case "info":
		out.info(f.Message)
	case "warn":
		out.warn(f.Message)
	case "error":
		out.fail(f.Message)
	}
}

func countLevel(result *doctorResult, level string) int {
	count := 0
	for _, f := range result.Environment {
		if f.Level == level {
			count++
		}
	}
	for _, f := range result.Project {
		if f.Level == level {
			count++
		}
	}
	return count
}

func parseModuleName(content string) string {
	re := regexp.MustCompile(`^module\s+(\S+)`)
	for _, line := range strings.Split(content, "\n") {
		if matches := re.FindStringSubmatch(strings.TrimSpace(line)); len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

func valueOr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func usageDoctor() {
	fmt.Println("Usage:")
	fmt.Println("  mizu doctor [flags]")
	fmt.Println()
	fmt.Println("Diagnose environment, module, and project layout.")
	fmt.Println()
	fmt.Println("Checks:")
	fmt.Println("  - go version, env, module mode")
	fmt.Println("  - go.mod validity, replace directives")
	fmt.Println("  - project layout sanity (cmd, main discovery)")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("      --fix     Apply safe fixes (go mod tidy, go fmt)")
	fmt.Println("      --json    Emit findings as JSON")
	fmt.Println("  -h, --help    Show help")
}
