package amazon

import "fmt"

// PrintStats prints a summary of the Amazon database stats.
func PrintStats(db *DB, stateDB *State) error {
	stats, err := db.GetStats()
	if err != nil {
		return err
	}

	fmt.Println("── Amazon Database Stats ──")
	fmt.Printf("  Products:    %d\n", stats.Products)
	fmt.Printf("  Brands:      %d\n", stats.Brands)
	fmt.Printf("  Authors:     %d\n", stats.Authors)
	fmt.Printf("  Categories:  %d\n", stats.Categories)
	fmt.Printf("  Bestsellers: %d\n", stats.Bestsellers)
	fmt.Printf("  Reviews:     %d\n", stats.Reviews)
	fmt.Printf("  QAs:         %d\n", stats.QAs)
	fmt.Printf("  Sellers:     %d\n", stats.Sellers)
	fmt.Printf("  Searches:    %d\n", stats.Searches)
	fmt.Printf("  DB size:  %.1f MB\n", float64(stats.DBSize)/(1024*1024))
	fmt.Printf("  DB path:  %s\n", db.Path())

	if stateDB != nil {
		pending, inProgress, done, failed := stateDB.QueueStats()
		fmt.Println("── Queue ──")
		fmt.Printf("  Pending:     %d\n", pending)
		fmt.Printf("  In progress: %d\n", inProgress)
		fmt.Printf("  Done:        %d\n", done)
		fmt.Printf("  Failed:      %d\n", failed)
	}

	return nil
}

// PrintCrawlProgress prints a one-line crawl progress update.
func PrintCrawlProgress(state *CrawlState) {
	fmt.Printf("\r  done=%-6d  pending=%-6d  failed=%-4d  in-flight=%-3d  rps=%.1f    ",
		state.Done, state.Pending, state.Failed, len(state.InFlight), state.RPS)
}
