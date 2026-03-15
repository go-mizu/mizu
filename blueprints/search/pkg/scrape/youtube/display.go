package youtube

import "fmt"

func PrintStats(db *DB, stateDB *State) error {
	stats, err := db.GetStats()
	if err != nil {
		return err
	}
	fmt.Println("── YouTube Database Stats ──")
	fmt.Printf("  Videos:        %d\n", stats.Videos)
	fmt.Printf("  Channels:      %d\n", stats.Channels)
	fmt.Printf("  Playlists:     %d\n", stats.Playlists)
	fmt.Printf("  Playlist rows: %d\n", stats.PlaylistRows)
	fmt.Printf("  Related rows:  %d\n", stats.RelatedRows)
	fmt.Printf("  Captions:      %d\n", stats.CaptionRows)
	fmt.Printf("  DB size:       %.1f MB\n", float64(stats.DBSize)/(1024*1024))
	fmt.Printf("  DB path:       %s\n", db.Path())
	if stateDB != nil {
		pending, inProgress, done, failed := stateDB.QueueStats()
		fmt.Println("── Queue ──")
		fmt.Printf("  Pending:     %d\n", pending)
		fmt.Printf("  In progress: %d\n", inProgress)
		fmt.Printf("  Done:        %d\n", done)
		fmt.Printf("  Failed:      %d\n", failed)
	}
	if videos, err := db.RecentVideos(5); err == nil && len(videos) > 0 {
		fmt.Println("── Recent Videos ──")
		for _, v := range videos {
			fmt.Printf("  [%d views] %s — %s\n", v.ViewCount, v.Title, v.ChannelName)
		}
	}
	return nil
}

func PrintCrawlProgress(state *CrawlState) {
	fmt.Printf("\r  done=%-6d pending=%-6d failed=%-4d in-flight=%-3d rps=%.1f    ", state.Done, state.Pending, state.Failed, len(state.InFlight), state.RPS)
}
