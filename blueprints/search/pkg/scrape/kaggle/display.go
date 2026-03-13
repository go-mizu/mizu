package kaggle

import "fmt"

func PrintStats(db *DB, stateDB *State) error {
	stats, err := db.GetStats()
	if err != nil {
		return err
	}
	fmt.Println("── Kaggle Database Stats ──")
	fmt.Printf("  Datasets:     %d\n", stats.Datasets)
	fmt.Printf("  Models:       %d\n", stats.Models)
	fmt.Printf("  Competitions: %d\n", stats.Competitions)
	fmt.Printf("  Notebooks:    %d\n", stats.Notebooks)
	fmt.Printf("  Profiles:     %d\n", stats.Profiles)
	fmt.Printf("  DB size:      %.1f MB\n", float64(stats.DBSize)/(1024*1024))
	fmt.Printf("  DB path:      %s\n", db.Path())
	if stateDB != nil {
		pending, inProgress, done, failed := stateDB.QueueStats()
		fmt.Println("── Queue ──")
		fmt.Printf("  Pending:     %d\n", pending)
		fmt.Printf("  In progress: %d\n", inProgress)
		fmt.Printf("  Done:        %d\n", done)
		fmt.Printf("  Failed:      %d\n", failed)
	}
	items, err := db.RecentDatasets(5)
	if err == nil && len(items) > 0 {
		fmt.Println("── Recent Datasets ──")
		for _, item := range items {
			fmt.Printf("  [%s] %s\n", item.OwnerRef, item.Title)
		}
	}
	return nil
}

func PrintCrawlProgress(state *CrawlState) {
	fmt.Printf("\r  done=%-6d  pending=%-6d  failed=%-4d  in-flight=%-3d  rps=%.1f    ",
		state.Done, state.Pending, state.Failed, len(state.InFlight), state.RPS)
}
