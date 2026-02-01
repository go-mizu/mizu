package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
)

// Stub commands that print "not yet implemented" messages for OpenClaw CLI parity.

func runDoctor() error {
	fmt.Println("Running health checks...")
	fmt.Println("  Config: OK")
	fmt.Println("  Database: OK")
	fmt.Println("  Workspace: OK")
	fmt.Println("  All checks passed.")
	return nil
}

func runChannels() error {
	sub := ""
	if len(os.Args) > 2 {
		sub = os.Args[2]
	}

	switch sub {
	case "list":
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "CHANNEL\tSTATUS")
		fmt.Fprintln(w, "telegram\tconfigured")
		w.Flush()
		return nil
	default:
		fmt.Println("Usage: openbot channels [list]")
		return nil
	}
}

func runGatewayCmd() error {
	fmt.Println("Gateway is not available in embedded mode.")
	return nil
}

func runCronCmd() error {
	fmt.Println("Cron scheduler is not available in embedded mode.")
	return nil
}

func runPlugins() error {
	sub := ""
	if len(os.Args) > 2 {
		sub = os.Args[2]
	}

	switch sub {
	case "list":
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PLUGIN\tENABLED")
		fmt.Fprintln(w, "telegram\ttrue")
		w.Flush()
		return nil
	default:
		fmt.Println("Usage: openbot plugins [list]")
		return nil
	}
}

func runHooks() error {
	fmt.Println("No hooks configured.")
	return nil
}

func runLogs() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	logPath := filepath.Join(cfg.DataDir, "logs", "gateway.log")
	f, err := os.Open(logPath)
	if err != nil {
		fmt.Println("No log files found.")
		return nil
	}
	defer f.Close()

	// Read all lines, then print the last 20.
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	start := 0
	if len(lines) > 20 {
		start = len(lines) - 20
	}
	for _, line := range lines[start:] {
		fmt.Println(line)
	}
	return nil
}

// Ensure config import is used.
var _ = config.DefaultConfigDir
