package discord

import "fmt"

func PrintStats(db *DB, stateDB *State) error {
	stats, err := db.GetStats()
	if err != nil {
		return err
	}
	fmt.Println("── Discord Database Stats ──")
	fmt.Printf("  Guilds:   %d\n", stats.Guilds)
	fmt.Printf("  Channels: %d\n", stats.Channels)
	fmt.Printf("  Messages: %d\n", stats.Messages)
	fmt.Printf("  Users:    %d\n", stats.Users)
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

	recent, err := db.RecentMessages(5)
	if err == nil && len(recent) > 0 {
		fmt.Println("── Recent Messages ──")
		for _, msg := range recent {
			content := msg.Content
			if len(content) > 60 {
				content = content[:60] + "..."
			}
			fmt.Printf("  [#%s] %s: %s\n", msg.ChannelID, msg.AuthorUsername, content)
		}
	}
	return nil
}

func PrintCrawlProgress(state *CrawlState) {
	fmt.Printf("\r  done=%-6d  pending=%-6d  failed=%-4d  in-flight=%-3d  rps=%.1f    ",
		state.Done, state.Pending, state.Failed, len(state.InFlight), state.RPS)
}
