package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
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

	var totalRating float64
	var ratedCount int

	for rows.Next() {
		var b types.Book
		var rating int
		var finishedAt sql.NullTime
		err := rows.Scan(&b.ID, &b.OLKey, &b.GoogleID, &b.Title, &b.Subtitle, &b.Description,
			&b.AuthorNames, &b.CoverURL, &b.CoverID, &b.ISBN10, &b.ISBN13, &b.Publisher,
			&b.PublishDate, &b.PublishYear, &b.PageCount, &b.Language, &b.Format,
			&b.SubjectsJSON, &b.AverageRating, &b.RatingsCount, &b.CreatedAt, &b.UpdatedAt,
			&rating, &finishedAt)
		if err != nil {
			continue
		}
		json.Unmarshal([]byte(b.SubjectsJSON), &b.Subjects)

		stats.TotalBooks++
		stats.TotalPages += b.PageCount

		if rating > 0 {
			totalRating += float64(rating)
			ratedCount++
			stats.RatingDist[rating]++
		}

		// Per-month breakdown
		if finishedAt.Valid {
			month := finishedAt.Time.Format("2006-01")
			stats.BooksPerMonth[month]++
			stats.PagesPerMonth[month] += b.PageCount
		}

		// Genre breakdown
		for _, subj := range b.Subjects {
			stats.GenreBreakdown[subj]++
		}

		// Track extremes
		if b.PageCount > 0 {
			if stats.ShortestBook == nil || b.PageCount < stats.ShortestBook.PageCount {
				copy := b
				stats.ShortestBook = &copy
			}
			if stats.LongestBook == nil || b.PageCount > stats.LongestBook.PageCount {
				copy := b
				stats.LongestBook = &copy
			}
		}
		if rating > 0 && (stats.HighestRated == nil || float64(rating) > stats.HighestRated.AverageRating) {
			copy := b
			copy.AverageRating = float64(rating)
			stats.HighestRated = &copy
		}
		if stats.MostPopular == nil || b.RatingsCount > stats.MostPopular.RatingsCount {
			copy := b
			stats.MostPopular = &copy
		}
	}

	if ratedCount > 0 {
		stats.AverageRating = totalRating / float64(ratedCount)
	}

	// Populate authors for extreme books
	for _, bp := range []*types.Book{stats.ShortestBook, stats.LongestBook, stats.HighestRated, stats.MostPopular} {
		if bp != nil && bp.AuthorNames != "" {
			for _, name := range strings.Split(bp.AuthorNames, ", ") {
				bp.Authors = append(bp.Authors, types.Author{Name: name})
			}
		}
	}

	return stats, nil
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

	var totalRating float64
	var ratedCount int

	for rows.Next() {
		var b types.Book
		var rating int
		var finishedAt sql.NullTime
		err := rows.Scan(&b.ID, &b.OLKey, &b.GoogleID, &b.Title, &b.Subtitle, &b.Description,
			&b.AuthorNames, &b.CoverURL, &b.CoverID, &b.ISBN10, &b.ISBN13, &b.Publisher,
			&b.PublishDate, &b.PublishYear, &b.PageCount, &b.Language, &b.Format,
			&b.SubjectsJSON, &b.AverageRating, &b.RatingsCount, &b.CreatedAt, &b.UpdatedAt,
			&rating, &finishedAt)
		if err != nil {
			continue
		}
		json.Unmarshal([]byte(b.SubjectsJSON), &b.Subjects)

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
				copy := b
				stats.ShortestBook = &copy
			}
			if stats.LongestBook == nil || b.PageCount > stats.LongestBook.PageCount {
				copy := b
				stats.LongestBook = &copy
			}
		}
		if rating > 0 && (stats.HighestRated == nil || float64(rating) > stats.HighestRated.AverageRating) {
			copy := b
			copy.AverageRating = float64(rating)
			stats.HighestRated = &copy
		}
		if stats.MostPopular == nil || b.RatingsCount > stats.MostPopular.RatingsCount {
			copy := b
			stats.MostPopular = &copy
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
