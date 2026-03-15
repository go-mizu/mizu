package spotify

import "fmt"

func PrintStats(db *DB, stateDB *State) error {
	stats, err := db.GetStats()
	if err != nil {
		return err
	}
	fmt.Println("── Spotify Database Stats ──")
	fmt.Printf("  Tracks:    %d\n", stats.Tracks)
	fmt.Printf("  Albums:    %d\n", stats.Albums)
	fmt.Printf("  Artists:   %d\n", stats.Artists)
	fmt.Printf("  Playlists: %d\n", stats.Playlists)
	fmt.Printf("  DB size:   %.1f MB\n", float64(stats.DBSize)/(1024*1024))
	fmt.Printf("  DB path:   %s\n", db.Path())

	if stateDB != nil {
		pending, inProgress, done, failed := stateDB.QueueStats()
		fmt.Println("── Queue ──")
		fmt.Printf("  Pending:     %d\n", pending)
		fmt.Printf("  In progress: %d\n", inProgress)
		fmt.Printf("  Done:        %d\n", done)
		fmt.Printf("  Failed:      %d\n", failed)
	}

	recent, err := db.RecentTracks(5)
	if err == nil && len(recent) > 0 {
		fmt.Println("── Recent Tracks ──")
		for _, t := range recent {
			fmt.Printf("  %s — %s\n", t.Name, t.AlbumName)
		}
	}
	return nil
}

func PrintCrawlProgress(state *CrawlState) {
	fmt.Printf("\r  done=%-6d  pending=%-6d  failed=%-4d  in-flight=%-3d  rps=%.1f    ",
		state.Done, state.Pending, state.Failed, len(state.InFlight), state.RPS)
}
