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

// devFlags holds flags for the dev command.
type devFlags struct {
	cmd string
}

// devEvent represents a lifecycle event for JSON output.
type devEvent struct {
	Event     string `json:"event"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message,omitempty"`
	ExitCode  int    `json:"exit_code,omitempty"`
}

//nolint:cyclop // CLI command with sequential logic
func runDev(args []string, gf *globalFlags) int {
	out := newOutput(gf.json, gf.quiet, gf.noColor, gf.verbose)
	df := &devFlags{}

	// Parse flags
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	fs.StringVar(&df.cmd, "cmd", "", "Explicit main package path")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			usageDev()
			return exitOK
		}
		return exitUsage
	}

	// Find main package
	mainPkg, err := findMainPackage(df.cmd)
	if err != nil {
		emitDevEvent(out, "error", err.Error(), 0)
		if !out.json {
			out.errorf("error: %v\n", err)
		}
		return exitNoProject
	}

	out.verbosef(1, "discovered main package: %s\n", mainPkg)
	emitDevEvent(out, "starting", fmt.Sprintf("running %s", mainPkg), 0)

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

	// Create command
	cmd := exec.CommandContext(ctx, "go", goArgs...) //nolint:gosec // args are constructed safely
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()

	// Start process
	if err := cmd.Start(); err != nil {
		emitDevEvent(out, "error", err.Error(), 0)
		if !out.json {
			out.errorf("error: failed to start: %v\n", err)
		}
		return exitError
	}

	emitDevEvent(out, "started", fmt.Sprintf("pid %d", cmd.Process.Pid), 0)

	// Wait for signal or process exit
	doneCh := make(chan error, 1)
	go func() {
		doneCh <- cmd.Wait()
	}()

	select {
	case sig := <-sigCh:
		emitDevEvent(out, "signal", sig.String(), 0)
		if !out.json {
			out.print("\nReceived %s, shutting down...\n", sig)
		}

		// Send signal to process
		_ = cmd.Process.Signal(sig)

		// Wait for graceful shutdown with timeout
		select {
		case err := <-doneCh:
			return handleDevProcessExit(out, err)
		case <-time.After(15 * time.Second):
			emitDevEvent(out, "timeout", "forcing shutdown", 0)
			_ = cmd.Process.Kill()
			return exitError
		}

	case err := <-doneCh:
		return handleDevProcessExit(out, err)
	}
}

func handleDevProcessExit(out *output, err error) int {
	if err == nil {
		emitDevEvent(out, "stopped", "clean exit", 0)
		return exitOK
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		code := exitErr.ExitCode()
		emitDevEvent(out, "stopped", fmt.Sprintf("exit code %d", code), code)
		return code
	}

	emitDevEvent(out, "error", err.Error(), 1)
	return exitError
}

func emitDevEvent(out *output, event, message string, exitCode int) {
	if !out.json {
		return
	}

	ev := devEvent{
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

func usageDev() {
	fmt.Println("Usage:")
	fmt.Println("  mizu dev [flags] [-- <args>]")
	fmt.Println()
	fmt.Println("Run the current project in development mode.")
	fmt.Println()
	fmt.Println("Behavior:")
	fmt.Println("  - auto-detect main package (cmd/* or main package)")
	fmt.Println("  - build if needed")
	fmt.Println("  - run until interrupted (SIGINT/SIGTERM)")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("      --cmd <path>         Explicit main package path")
	fmt.Println("      --json               Emit lifecycle events as JSON")
	fmt.Println("  -h, --help               Show help")
}
