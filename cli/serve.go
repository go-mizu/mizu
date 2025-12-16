package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// serveFlags holds flags for the serve command.
type serveFlags struct {
	cmd  string
	envs envList
	args string
}

// envList is a flag.Value for repeatable --env flags.
type envList []string

func (e *envList) String() string { return strings.Join(*e, ",") }
func (e *envList) Set(s string) error {
	*e = append(*e, s)
	return nil
}

// serveEvent represents a lifecycle event for JSON output.
type serveEvent struct {
	Event     string `json:"event"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message,omitempty"`
	ExitCode  int    `json:"exit_code,omitempty"`
}

//nolint:cyclop // CLI command with sequential logic
func runServe(args []string, gf *globalFlags) int {
	out := newOutput(gf.json, gf.quiet, gf.noColor, gf.verbose)
	sf := &serveFlags{}

	// Parse flags
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.StringVar(&sf.cmd, "cmd", "", "Explicit main package path")
	fs.Var(&sf.envs, "env", "Set env var (k=v, repeatable)")
	fs.StringVar(&sf.args, "args", "", "Args passed to the program")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			usageServe()
			return exitOK
		}
		return exitUsage
	}

	// Find main package
	mainPkg, err := findMainPackage(sf.cmd)
	if err != nil {
		emitEvent(out, "error", err.Error(), 0)
		if !out.json {
			out.errorf("error: %v\n", err)
		}
		return exitNoProject
	}

	out.verbosef(1, "discovered main package: %s\n", mainPkg)
	emitEvent(out, "starting", fmt.Sprintf("running %s", mainPkg), 0)

	if !out.json {
		out.print("Starting %s...\n", out.cyan(mainPkg))
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Build arguments for go run
	goArgs := []string{"run", mainPkg}
	if sf.args != "" {
		goArgs = append(goArgs, strings.Fields(sf.args)...)
	}

	// Create command
	cmd := exec.CommandContext(ctx, "go", goArgs...) //nolint:gosec // args are constructed safely
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Set environment
	cmd.Env = os.Environ()
	for _, e := range sf.envs {
		cmd.Env = append(cmd.Env, e)
	}

	// Start process
	if err := cmd.Start(); err != nil {
		emitEvent(out, "error", err.Error(), 0)
		if !out.json {
			out.errorf("error: failed to start: %v\n", err)
		}
		return exitError
	}

	emitEvent(out, "started", fmt.Sprintf("pid %d", cmd.Process.Pid), 0)

	// Wait for signal or process exit
	doneCh := make(chan error, 1)
	go func() {
		doneCh <- cmd.Wait()
	}()

	select {
	case sig := <-sigCh:
		emitEvent(out, "signal", sig.String(), 0)
		if !out.json {
			out.print("\nReceived %s, shutting down...\n", sig)
		}

		// Send signal to process
		_ = cmd.Process.Signal(sig)

		// Wait for graceful shutdown with timeout
		select {
		case err := <-doneCh:
			return handleProcessExit(out, err)
		case <-time.After(15 * time.Second):
			emitEvent(out, "timeout", "forcing shutdown", 0)
			_ = cmd.Process.Kill()
			return exitError
		}

	case err := <-doneCh:
		return handleProcessExit(out, err)
	}
}

func handleProcessExit(out *output, err error) int {
	if err == nil {
		emitEvent(out, "stopped", "clean exit", 0)
		return exitOK
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		code := exitErr.ExitCode()
		emitEvent(out, "stopped", fmt.Sprintf("exit code %d", code), code)
		return code
	}

	emitEvent(out, "error", err.Error(), 1)
	return exitError
}

func emitEvent(out *output, event, message string, exitCode int) {
	if !out.json {
		return
	}

	ev := serveEvent{
		Event:     event,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Message:   message,
	}
	if exitCode != 0 {
		ev.ExitCode = exitCode
	}

	enc := newJSONEncoder(out.stdout)
	_ = enc.encode(ev)
}

//nolint:cyclop // discovery logic with multiple fallback paths
func findMainPackage(explicit string) (string, error) {
	if explicit != "" {
		// Validate explicit path exists
		if !dirExists(explicit) && !fileExists(explicit) {
			return "", fmt.Errorf("path does not exist: %s", explicit)
		}
		return explicit, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Look for cmd/ directory
	cmdDir := filepath.Join(cwd, "cmd")
	if dirExists(cmdDir) {
		entries, err := os.ReadDir(cmdDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					// Use "./" prefix to ensure Go treats it as a local path
					candidate := "./cmd/" + e.Name()
					if hasMainGo(filepath.Join(cwd, "cmd", e.Name())) {
						return candidate, nil
					}
				}
			}
		}
	}

	// Check for main.go in current directory
	if hasMainGo(cwd) {
		return ".", nil
	}

	return "", errNoProject("no runnable main package found (try --cmd)")
}

func hasMainGo(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			// Check if file contains main package
			content, err := readFileString(filepath.Join(dir, e.Name()))
			if err == nil && strings.Contains(content, "package main") {
				return true
			}
		}
	}

	return false
}

func usageServe() {
	fmt.Println("Usage:")
	fmt.Println("  mizu serve [flags]")
	fmt.Println()
	fmt.Println("Run the project with graceful shutdown.")
	fmt.Println()
	fmt.Println("Behavior:")
	fmt.Println("  - Detects a runnable main package (cmd/* preferred, then module root)")
	fmt.Println("  - Runs it with predictable env and graceful shutdown")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("      --cmd <path>     Explicit main package path (ex: ./cmd/api)")
	fmt.Println("      --env k=v        Set env var (repeatable)")
	fmt.Println("      --args \"<args>\"  Args passed to the program")
	fmt.Println("      --json           Emit lifecycle events as JSON")
	fmt.Println("  -h, --help           Show help")
}
