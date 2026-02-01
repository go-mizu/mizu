package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
)

// runModels dispatches models subcommands: list, status, set
func runModels() error {
	sub := ""
	if len(os.Args) > 2 {
		sub = os.Args[2]
	}

	switch sub {
	case "list", "":
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "MODEL\tPROVIDER")
		fmt.Fprintln(w, "claude-sonnet-4-20250514\tanthropic")
		fmt.Fprintln(w, "claude-opus-4-5-20251101\tanthropic")
		fmt.Fprintln(w, "claude-haiku-35-20241022\tanthropic")
		w.Flush()
		return nil

	case "status":
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		modelsPath := filepath.Join(cfg.DataDir, "agents", "main", "agent", "models.json")
		data, err := os.ReadFile(modelsPath)
		if err != nil {
			fmt.Println("Default model: claude-sonnet-4-20250514")
			return nil
		}
		var models map[string]string
		if err := json.Unmarshal(data, &models); err != nil {
			fmt.Println("Default model: claude-sonnet-4-20250514")
			return nil
		}
		def := models["default"]
		if def == "" {
			def = "claude-sonnet-4-20250514"
		}
		fmt.Printf("Default model: %s\n", def)
		return nil

	case "set":
		if len(os.Args) < 4 {
			return fmt.Errorf("usage: openbot models set <model-name>")
		}
		modelName := os.Args[3]

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		modelsDir := filepath.Join(cfg.DataDir, "agents", "main", "agent")
		if err := os.MkdirAll(modelsDir, 0o755); err != nil {
			return fmt.Errorf("create models dir: %w", err)
		}

		modelsPath := filepath.Join(modelsDir, "models.json")
		models := map[string]string{"default": modelName}
		data, err := json.MarshalIndent(models, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal models: %w", err)
		}
		if err := os.WriteFile(modelsPath, append(data, '\n'), 0o644); err != nil {
			return fmt.Errorf("write models: %w", err)
		}
		fmt.Printf("Default model set to: %s\n", modelName)
		return nil

	default:
		fmt.Println("Usage: openbot models [list|status|set <model>]")
		return nil
	}
}

// Ensure config import is used.
var _ = config.DefaultConfigPath
