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

// NewSeedSE creates the seed se command.
func NewSeedSE() *cobra.Command {
	var sourceDir string
	var batchSize int

	cmd := &cobra.Command{
		Use:   "se",
		Short: "Import StackExchange dump",
		Long:  "Imports StackExchange XML dumps into the QA database.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSeedSE(cmd.Context(), sourceDir, batchSize)
		},
	}

	homeDir, _ := os.UserHomeDir()
	defaultSource := filepath.Join(homeDir, "Downloads", "data", "ai.stackexchange.com")

	cmd.Flags().StringVar(&sourceDir, "source", defaultSource, "Directory containing StackExchange XML files")
	cmd.Flags().IntVar(&batchSize, "batch", 5000, "Insert batch size")
	return cmd
}

func runSeedSE(ctx context.Context, sourceDir string, batchSize int) error {
	ui := NewUI()
	ui.Header(iconDatabase, "Importing StackExchange Data")
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

	summary, err := importer.ImportDir(ctx, sourceDir)
	if err != nil {
		ui.Error(fmt.Sprintf("Import failed: %v", err))
		return err
	}

	ui.Success("StackExchange import complete")
	ui.Summary([][2]string{
		{"Users", fmt.Sprintf("%d", summary.Users)},
		{"Tags", fmt.Sprintf("%d", summary.Tags)},
		{"Questions", fmt.Sprintf("%d", summary.Questions)},
		{"Answers", fmt.Sprintf("%d", summary.Answers)},
		{"Comments", fmt.Sprintf("%d", summary.Comments)},
		{"Votes", fmt.Sprintf("%d", summary.Votes)},
		{"Bookmarks", fmt.Sprintf("%d", summary.Bookmarks)},
		{"Badges", fmt.Sprintf("%d", summary.Badges)},
		{"Badge awards", fmt.Sprintf("%d", summary.BadgeAwards)},
		{"Question tags", fmt.Sprintf("%d", summary.QuestionTags)},
	})
	ui.Blank()
	ui.Hint("Next: run 'qa serve' to start the server")
	return nil
}

type seedSEProgress struct {
	ui        *UI
	current   string
	lastRows  int
	lastBytes int64
}

func (p *seedSEProgress) StartFile(name string, size int64) {
	p.current = name
	p.lastRows = 0
	p.lastBytes = 0
	p.ui.StartSpinner(fmt.Sprintf("Importing %s...", name))
}

func (p *seedSEProgress) Advance(name string, rows int, bytesRead int64) {
	if name == "" {
		name = p.current
	}
	if rows == p.lastRows && bytesRead == p.lastBytes {
		return
	}
	p.lastRows = rows
	p.lastBytes = bytesRead
	p.ui.UpdateSpinner(fmt.Sprintf("Importing %s: %d rows", name, rows))
}

func (p *seedSEProgress) EndFile(name string, rows int, duration time.Duration) {
	p.ui.StopSpinner(fmt.Sprintf("Imported %s (%d rows)", name, rows), duration)
}

func (p *seedSEProgress) Logf(format string, args ...any) {
	p.ui.Step(fmt.Sprintf(format, args...))
}
