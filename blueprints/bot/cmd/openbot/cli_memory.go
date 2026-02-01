package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/memory"
)

// runMemory dispatches memory subcommands: status, index, search
func runMemory() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: openbot memory <status|index|search>")
	}

	switch os.Args[2] {
	case "status":
		return runMemoryStatus()
	case "index":
		return runMemoryIndex()
	case "search":
		return runMemorySearch()
	default:
		return fmt.Errorf("unknown memory subcommand: %s", os.Args[2])
	}
}

func runMemoryStatus() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(cfg.DataDir, "memory.db")
	info, err := os.Stat(dbPath)
	if err != nil {
		fmt.Printf("Memory DB: %s\n", dbPath)
		fmt.Println("Status:    missing")
		return nil
	}

	fmt.Printf("Memory DB: %s\n", dbPath)
	fmt.Printf("Size:      %d bytes\n", info.Size())
	fmt.Println("Status:    active")
	return nil
}

func runMemoryIndex() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(cfg.DataDir, "memory.db")
	memCfg := memory.DefaultMemoryConfig()
	memCfg.WorkspaceDir = cfg.Workspace

	mgr, err := memory.NewMemoryManager(dbPath, cfg.Workspace, memCfg)
	if err != nil {
		return fmt.Errorf("create memory manager: %w", err)
	}
	defer mgr.Close()

	fmt.Printf("Indexing workspace: %s\n", cfg.Workspace)
	if err := mgr.IndexAll(); err != nil {
		return fmt.Errorf("index all: %w", err)
	}

	fmt.Println("Indexing complete.")
	return nil
}

func runMemorySearch() error {
	fs := flag.NewFlagSet("memory-search", flag.ExitOnError)
	query := fs.String("q", "", "Search query (required)")
	fs.Parse(os.Args[3:])

	if *query == "" {
		return fmt.Errorf("usage: openbot memory search -q <query>")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(cfg.DataDir, "memory.db")
	memCfg := memory.DefaultMemoryConfig()
	memCfg.WorkspaceDir = cfg.Workspace

	mgr, err := memory.NewMemoryManager(dbPath, cfg.Workspace, memCfg)
	if err != nil {
		return fmt.Errorf("create memory manager: %w", err)
	}
	defer mgr.Close()

	ctx := context.Background()
	results, err := mgr.Search(ctx, *query, 6, 0)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	for i, r := range results {
		fmt.Printf("[%d] %s (lines %d-%d, score %.2f)\n", i+1, r.Path, r.StartLine, r.EndLine, r.Score)
		snippet := r.Snippet
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		fmt.Printf("    %s\n\n", snippet)
	}
	return nil
}
