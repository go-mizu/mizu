package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/qa/pkg/seed/se"
	"github.com/go-mizu/mizu/blueprints/qa/store/duckdb"
)

// NewSeedFixTags creates the fix-tags command.
func NewSeedFixTags() *cobra.Command {
	var sourceDir string
	var batchSize int

	cmd := &cobra.Command{
		Use:   "fix-tags",
		Short: "Populate question_tags from StackExchange dump",
		Long:  "Re-imports question-tag associations from Posts.xml into the question_tags table.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSeedFixTags(cmd.Context(), sourceDir, batchSize)
		},
	}

	homeDir, _ := os.UserHomeDir()
	defaultSource := filepath.Join(homeDir, "Downloads", "data", "ai.stackexchange.com")

	cmd.Flags().StringVar(&sourceDir, "source", defaultSource, "Directory containing StackExchange XML files")
	cmd.Flags().IntVar(&batchSize, "batch", 5000, "Insert batch size")
	return cmd
}

func runSeedFixTags(ctx context.Context, sourceDir string, batchSize int) error {
	ui := NewUI()
	ui.Header(iconDatabase, "Fixing Question Tags")
	ui.Blank()

	start := time.Now()
	ui.StartSpinner("Opening database...")

	store, err := duckdb.Open(dataDir)
	if err != nil {
		ui.StopSpinnerError("Failed to open database")
		return err
	}
	defer store.Close()

	ui.StopSpinner("Database opened", time.Since(start))

	progress := &seedSEProgress{ui: ui}
	importer := se.NewImporter(store.DB()).WithBatchSize(batchSize).WithProgress(progress)

	postsPath := filepath.Join(sourceDir, "Posts.xml")
	count, err := importer.ImportQuestionTags(ctx, postsPath)
	if err != nil {
		ui.Error(fmt.Sprintf("Import failed: %v", err))
		return err
	}

	ui.Success("Question tags import complete")
	ui.Summary([][2]string{
		{"Question tags", fmt.Sprintf("%d", count)},
	})
	ui.Blank()
	ui.Hint("Next: run 'qa serve' to start the server")
	return nil
}
