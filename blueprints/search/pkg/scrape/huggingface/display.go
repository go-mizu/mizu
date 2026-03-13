package huggingface

import "fmt"

func PrintStats(db *DB, state *State) error {
	stats, err := db.GetStats()
	if err != nil {
		return err
	}
	fmt.Println("── Hugging Face Database Stats ──")
	fmt.Printf("  Models:       %d\n", stats.Models)
	fmt.Printf("  Datasets:     %d\n", stats.Datasets)
	fmt.Printf("  Spaces:       %d\n", stats.Spaces)
	fmt.Printf("  Collections:  %d\n", stats.Collections)
	fmt.Printf("  Papers:       %d\n", stats.Papers)
	fmt.Printf("  Repo files:   %d\n", stats.RepoFiles)
	fmt.Printf("  Repo links:   %d\n", stats.RepoLinks)
	fmt.Printf("  DB size:      %.1f MB\n", float64(stats.DBSize)/(1024*1024))
	fmt.Printf("  DB path:      %s\n", db.Path())
	if state != nil {
		pending, inProgress, done, failed := state.QueueStats()
		fmt.Println("── Queue ──")
		fmt.Printf("  Pending:      %d\n", pending)
		fmt.Printf("  In progress:  %d\n", inProgress)
		fmt.Printf("  Done:         %d\n", done)
		fmt.Printf("  Failed:       %d\n", failed)
	}
	models, err := db.RecentModels(5)
	if err == nil && len(models) > 0 {
		fmt.Println("── Recent Models ──")
		for _, model := range models {
			fmt.Printf("  %s", model.RepoID)
			if model.Author != "" {
				fmt.Printf(" by %s", model.Author)
			}
			if model.PipelineTag != "" {
				fmt.Printf(" [%s]", model.PipelineTag)
			}
			fmt.Println()
		}
	}
	return nil
}

func PrintCrawlProgress(state *CrawlState) {
	fmt.Printf("\r  done=%-6d pending=%-6d failed=%-4d in-flight=%-3d rps=%.1f    ",
		state.Done, state.Pending, state.Failed, len(state.InFlight), state.RPS)
}
