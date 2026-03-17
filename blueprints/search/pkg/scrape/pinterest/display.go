package pinterest

import "fmt"

// PrintStats prints a summary of the Pinterest database.
func PrintStats(db *DB, stateDB *State) error {
	stats, err := db.GetStats()
	if err != nil {
		return err
	}

	fmt.Println("── Pinterest Database Stats ──")
	fmt.Printf("  Pins:    %d\n", stats.Pins)
	fmt.Printf("  Boards:  %d\n", stats.Boards)
	fmt.Printf("  Users:   %d\n", stats.Users)
	fmt.Printf("  DB size: %.1f MB\n", float64(stats.DBSize)/(1024*1024))
	fmt.Printf("  DB path: %s\n", db.Path())

	if stateDB != nil {
		pending, inProgress, done, failed := stateDB.QueueStats()
		fmt.Println("── Queue ──")
		fmt.Printf("  Pending:     %d\n", pending)
		fmt.Printf("  In progress: %d\n", inProgress)
		fmt.Printf("  Done:        %d\n", done)
		fmt.Printf("  Failed:      %d\n", failed)
	}

	pins, err := db.RecentPins(5)
	if err == nil && len(pins) > 0 {
		fmt.Println("── Recent Pins ──")
		for _, p := range pins {
			fmt.Printf("  [%d saves] %s — @%s\n", p.SavedCount, p.Title, p.Username)
		}
	}

	return nil
}

// PrintCrawlProgress prints a one-line crawl progress update.
func PrintCrawlProgress(state *CrawlState) {
	fmt.Printf("\r  done=%-6d  pending=%-6d  failed=%-4d  in-flight=%-3d  rps=%.1f    ",
		state.Done, state.Pending, state.Failed, len(state.InFlight), state.RPS)
}
