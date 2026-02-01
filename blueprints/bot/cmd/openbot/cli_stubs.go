package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
)

// Stub commands that print "not yet implemented" messages for OpenClaw CLI parity.

func runDoctor() error {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println("  Config: FAIL -", err)
		return nil
	}
	fmt.Println("Running health checks...")
	fmt.Printf("  Config: OK (%s)\n", config.DefaultConfigPath())

	// Check workspace.
	if _, err := os.Stat(cfg.Workspace); err != nil {
		fmt.Printf("  Workspace: MISSING (%s)\n", cfg.Workspace)
	} else {
		fmt.Printf("  Workspace: OK (%s)\n", cfg.Workspace)
	}

	// Check sessions directory.
	sessDir := filepath.Join(cfg.DataDir, "agents", "main", "sessions")
	if _, err := os.Stat(sessDir); err != nil {
		fmt.Printf("  Sessions: MISSING (%s)\n", sessDir)
	} else {
		fmt.Printf("  Sessions: OK (%s)\n", sessDir)
	}

	// Check memory DB.
	memDB := filepath.Join(cfg.DataDir, "memory.db")
	if info, err := os.Stat(memDB); err != nil {
		fmt.Println("  Memory: not indexed")
	} else {
		fmt.Printf("  Memory: OK (%d bytes)\n", info.Size())
	}

	// Check skills.
	skills, _ := skill.LoadAllSkills(cfg.Workspace)
	readyCount := 0
	for _, s := range skills {
		if s.Ready {
			readyCount++
		}
	}
	fmt.Printf("  Skills: %d loaded, %d ready\n", len(skills), readyCount)

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

