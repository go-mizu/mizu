package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var devFlags struct {
	cmd string
}

var devCmd = &cobra.Command{
	Use:   "dev [flags] [-- args...]",
	Short: "Run the current project in development mode",
	Long: `Run the current project in development mode.

Automatically discovers the main package in cmd/* or the current directory,
builds it, and runs until interrupted.`,
	Example: `  # Auto-discover and run main package
  mizu dev

  # Specify explicit main package
  mizu dev --cmd ./cmd/server

  # Pass arguments to the application
  mizu dev -- --port 3000`,
	RunE: wrapRunE(runDevCmd),
}

func init() {
	devCmd.Flags().StringVar(&devFlags.cmd, "cmd", "", "Explicit main package path")
}

// devEvent represents a lifecycle event for JSON output.
type devEvent struct {
	Event     string `json:"event"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message,omitempty"`
	ExitCode  int    `json:"exit_code,omitempty"`
}

func runDevCmd(cmd *cobra.Command, args []string) error {
	out := NewOutput()

	// Find main package
	mainPkg, err := findMainPackage(devFlags.cmd)
	if err != nil {
		emitDevEventNew(out, "error", err.Error(), 0)
		if !Flags.JSON {
			out.PrintError("%v", err)
		}
		return err
	}

	out.Verbosef(1, "discovered main package: %s\n", mainPkg)
	emitDevEventNew(out, "starting", fmt.Sprintf("running %s", mainPkg), 0)

	if !Flags.JSON {
		out.Print("Starting %s...\n", out.Cyan(mainPkg))
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Build arguments for go run
	goArgs := []string{"run", mainPkg}

	// Create command
	execCmd := exec.CommandContext(ctx, "go", goArgs...) //nolint:gosec
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin
	execCmd.Env = os.Environ()

	// Start process
	if err := execCmd.Start(); err != nil {
		emitDevEventNew(out, "error", err.Error(), 0)
		if !Flags.JSON {
			out.PrintError("failed to start: %v", err)
		}
		return err
	}

	emitDevEventNew(out, "started", fmt.Sprintf("pid %d", execCmd.Process.Pid), 0)

	// Wait for signal or process exit
	doneCh := make(chan error, 1)
	go func() {
		doneCh <- execCmd.Wait()
	}()

	select {
	case sig := <-sigCh:
		emitDevEventNew(out, "signal", sig.String(), 0)
		if !Flags.JSON {
			out.Print("\nReceived %s, shutting down...\n", sig)
		}

		// Send signal to process
		_ = execCmd.Process.Signal(sig)

		// Wait for graceful shutdown with timeout
		select {
		case err := <-doneCh:
			return handleDevProcessExitNew(out, err)
		case <-time.After(15 * time.Second):
			emitDevEventNew(out, "timeout", "forcing shutdown", 0)
			_ = execCmd.Process.Kill()
			return fmt.Errorf("timeout waiting for shutdown")
		}

	case err := <-doneCh:
		return handleDevProcessExitNew(out, err)
	}
}

func handleDevProcessExitNew(out *Output, err error) error {
	if err == nil {
		emitDevEventNew(out, "stopped", "clean exit", 0)
		return nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		code := exitErr.ExitCode()
		emitDevEventNew(out, "stopped", fmt.Sprintf("exit code %d", code), code)
		return fmt.Errorf("exit code %d", code)
	}

	emitDevEventNew(out, "error", err.Error(), 1)
	return err
}

func emitDevEventNew(out *Output, event, message string, exitCode int) {
	if !Flags.JSON {
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

	out.WriteJSON(ev)
}

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
			content, err := readFileString(filepath.Join(dir, e.Name()))
			if err == nil && strings.Contains(content, "package main") {
				return true
			}
		}
	}

	return false
}
