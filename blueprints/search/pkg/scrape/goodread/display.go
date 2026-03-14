package goodread

import (
	"fmt"
	"time"
)

// PrintStats prints a summary of the goodread database stats.
func PrintStats(db *DB, stateDB *State) error {
	stats, err := db.GetStats()
	if err != nil {
		return err
	}

	fmt.Println("── Goodreads Database Stats ──")
	fmt.Printf("  Books:    %d\n", stats.Books)
	fmt.Printf("  Authors:  %d\n", stats.Authors)
	fmt.Printf("  Series:   %d\n", stats.Series)
	fmt.Printf("  Lists:    %d\n", stats.Lists)
	fmt.Printf("  Reviews:  %d\n", stats.Reviews)
	fmt.Printf("  Quotes:   %d\n", stats.Quotes)
	fmt.Printf("  Users:    %d\n", stats.Users)
	fmt.Printf("  Genres:   %d\n", stats.Genres)
	fmt.Printf("  Shelves:  %d\n", stats.Shelves)
	fmt.Printf("  DB size:  %.1f MB\n", float64(stats.DBSize)/(1024*1024))
	fmt.Printf("  DB path:  %s\n", db.Path())

	if stateDB != nil {
		ms := stateDB.MemStats()
		total := ms.Pending + ms.InProgress + ms.Fetched + ms.Done + ms.Failed
		fmt.Println("── Queue (in-memory) ──")
		fmt.Printf("  Pending:     %d\n", ms.Pending)
		fmt.Printf("  In progress: %d\n", ms.InProgress)
		fmt.Printf("  Fetched:     %d  (HTML on disk, awaiting import)\n", ms.Fetched)
		fmt.Printf("  Done:        %d\n", ms.Done)
		fmt.Printf("  Failed:      %d\n", ms.Failed)
		fmt.Printf("  Total:       %d\n", total)
		if ms.DirtyItems > 0 {
			fmt.Printf("  Dirty:       %d  (pending checkpoint)\n", ms.DirtyItems)
		}
	}

	// Show recent books
	books, err := db.RecentBooks(5)
	if err == nil && len(books) > 0 {
		fmt.Println("── Recent Books ──")
		for _, b := range books {
			fmt.Printf("  [%.2f ★  %d ratings] %s — %s\n",
				b.AvgRating, b.RatingsCount, b.Title, b.AuthorName)
		}
	}

	return nil
}

// PrintCrawlProgress prints a one-line crawl progress update with ETA.
func PrintCrawlProgress(state *CrawlState) {
	inflight := len(state.InFlight)
	eta := ""
	if state.RPS > 0 && state.Pending > 0 {
		eta = "  eta=" + FormatETA(time.Duration(float64(state.Pending)/state.RPS)*time.Second)
	}
	fmt.Printf("\r  done=%-7d  pending=%-7d  failed=%-4d  in-flight=%-3d  rps=%.1f%s    ",
		state.Done, state.Pending, state.Failed, inflight, state.RPS, eta)
}

// PrintCrawlSummary prints a final summary after a crawl run.
func PrintCrawlSummary(metric CrawlMetric, stateDB *State) {
	fmt.Println() // end progress line
	fmt.Println("── Crawl summary ──")
	fmt.Printf("  Done:       %d\n", metric.Done)
	fmt.Printf("  Failed:     %d\n", metric.Failed)
	fmt.Printf("  Duration:   %s\n", metric.Duration.Round(time.Second))
	if metric.Duration.Seconds() > 0 {
		rps := float64(metric.Done) / metric.Duration.Seconds()
		fmt.Printf("  Throughput: %.1f req/s avg\n", rps)
	}
	if stateDB != nil {
		ms := stateDB.MemStats()
		fmt.Println("── Queue after ──")
		fmt.Printf("  Pending:     %d\n", ms.Pending)
		fmt.Printf("  Fetched:     %d\n", ms.Fetched)
		fmt.Printf("  Done:        %d\n", ms.Done)
		fmt.Printf("  Failed:      %d\n", ms.Failed)
		fmt.Printf("  Dirty items: %d  (will checkpoint shortly)\n", ms.DirtyItems)
	}
}

// FormatETA formats a duration as a human-readable ETA string.
func FormatETA(d time.Duration) string {
	d = d.Round(time.Second)
	if d >= 24*time.Hour {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	if d >= time.Hour {
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
	if d >= time.Minute {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
