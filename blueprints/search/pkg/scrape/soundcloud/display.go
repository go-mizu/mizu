package soundcloud

import "fmt"

func PrintStats(db *DB, stateDB *State) error {
	stats, err := db.GetStats()
	if err != nil {
		return err
	}
	pending, inProgress, done, failed := stateDB.QueueStats()

	fmt.Printf("Users:      %d\n", stats.Users)
	fmt.Printf("Tracks:     %d\n", stats.Tracks)
	fmt.Printf("Playlists:  %d\n", stats.Playlists)
	fmt.Printf("Comments:   %d\n", stats.Comments)
	fmt.Printf("SearchRows: %d\n", stats.SearchRows)
	fmt.Printf("DB Size:    %d bytes\n", stats.DBSize)
	fmt.Println()
	fmt.Printf("Queue pending:     %d\n", pending)
	fmt.Printf("Queue in_progress: %d\n", inProgress)
	fmt.Printf("Queue done:        %d\n", done)
	fmt.Printf("Queue failed:      %d\n", failed)
	return nil
}
