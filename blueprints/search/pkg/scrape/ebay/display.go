package ebay

import "fmt"

// PrintStats prints a summary of the eBay database.
func PrintStats(db *DB, stateDB *State) error {
	stats, err := db.GetStats()
	if err != nil {
		return err
	}

	fmt.Println("── eBay Database Stats ──")
	fmt.Printf("  Items:        %d\n", stats.Items)
	fmt.Printf("  Search pages: %d\n", stats.SearchPages)
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

	items, err := db.RecentItems(5)
	if err == nil && len(items) > 0 {
		fmt.Println("── Recent Items ──")
		for _, item := range items {
			fmt.Printf("  [%s %.2f] %s — %s\n", item.Currency, item.Price, item.Title, item.SellerName)
		}
	}

	return nil
}

// PrintCrawlProgress prints a one-line crawl progress update.
func PrintCrawlProgress(state *CrawlState) {
	fmt.Printf("\r  done=%-6d  pending=%-6d  failed=%-4d  in-flight=%-3d  rps=%.1f    ",
		state.Done, state.Pending, state.Failed, len(state.InFlight), state.RPS)
}
