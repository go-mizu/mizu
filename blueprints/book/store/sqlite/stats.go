package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

// StatsStore implements store.StatsStore backed by SQLite.
type StatsStore struct {
	db *sql.DB
}

// GetStats returns reading statistics for a given year. It considers books that
// have a review with finished_at in the given year and are on the "Read" shelf.
func (s *StatsStore) GetStats(ctx context.Context, year int) (*types.ReadingStats, error) {
	startDate := fmt.Sprintf("%d-01-01", year)
	endDate := fmt.Sprintf("%d-01-01", year+1)

	stats := &types.ReadingStats{
		BooksPerMonth:  make(map[string]int),
		PagesPerMonth:  make(map[string]int),
		GenreBreakdown: make(map[string]int),
		RatingDist:     make(map[int]int),
	}

	// Books read this year (have finished_at in year range)
	rows, err := s.db.QueryContext(ctx, `
		SELECT b.*, r.rating, r.finished_at
		FROM books b
		JOIN reviews r ON r.book_id = b.id
		WHERE r.finished_at >= ? AND r.finished_at < ?
		ORDER BY r.finished_at ASC`, startDate, endDate)
	if err != nil {
		return stats, nil
	}
	defer rows.Close()

	return s.processStatsRows(rows, stats)
}

// GetOverallStats returns all-time reading statistics across all years.
func (s *StatsStore) GetOverallStats(ctx context.Context) (*types.ReadingStats, error) {
	stats := &types.ReadingStats{
		BooksPerMonth:  make(map[string]int),
		PagesPerMonth:  make(map[string]int),
		GenreBreakdown: make(map[string]int),
		RatingDist:     make(map[int]int),
	}

	// All reviewed books
	rows, err := s.db.QueryContext(ctx, `
		SELECT b.*, r.rating, r.finished_at
		FROM books b
		JOIN reviews r ON r.book_id = b.id
		ORDER BY r.finished_at ASC`)
	if err != nil {
		return stats, nil
	}
	defer rows.Close()

	return s.processStatsRows(rows, stats)
}

func (s *StatsStore) processStatsRows(rows *sql.Rows, stats *types.ReadingStats) (*types.ReadingStats, error) {
	var totalRating float64
	var ratedCount int

	for rows.Next() {
		var b types.Book
		var rating int
		var finishedAt sql.NullTime
		fields := append(scanFields(&b), &rating, &finishedAt)
		if err := rows.Scan(fields...); err != nil {
			continue
		}
		hydrateBook(&b)

		stats.TotalBooks++
		stats.TotalPages += b.PageCount

		if rating > 0 {
			totalRating += float64(rating)
			ratedCount++
			stats.RatingDist[rating]++
		}

		if finishedAt.Valid {
			month := finishedAt.Time.Format("2006-01")
			stats.BooksPerMonth[month]++
			stats.PagesPerMonth[month] += b.PageCount
		}

		for _, subj := range b.Subjects {
			stats.GenreBreakdown[subj]++
		}

		if b.PageCount > 0 {
			if stats.ShortestBook == nil || b.PageCount < stats.ShortestBook.PageCount {
				c := b
				stats.ShortestBook = &c
			}
			if stats.LongestBook == nil || b.PageCount > stats.LongestBook.PageCount {
				c := b
				stats.LongestBook = &c
			}
		}
		if rating > 0 && (stats.HighestRated == nil || float64(rating) > stats.HighestRated.AverageRating) {
			c := b
			c.AverageRating = float64(rating)
			stats.HighestRated = &c
		}
		if stats.MostPopular == nil || b.RatingsCount > stats.MostPopular.RatingsCount {
			c := b
			stats.MostPopular = &c
		}
	}

	if ratedCount > 0 {
		stats.AverageRating = totalRating / float64(ratedCount)
	}

	for _, bp := range []*types.Book{stats.ShortestBook, stats.LongestBook, stats.HighestRated, stats.MostPopular} {
		if bp != nil && bp.AuthorNames != "" {
			for _, name := range strings.Split(bp.AuthorNames, ", ") {
				bp.Authors = append(bp.Authors, types.Author{Name: name})
			}
		}
	}

	return stats, nil
}
