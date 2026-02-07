package cc

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"

	_ "github.com/duckdb/duckdb-go/v2"
)

// SeedStats holds summary stats about extracted seed URLs.
type SeedStats struct {
	TotalURLs     int
	UniqueDomains int
}

// ExtractSeedURLs queries the CC index and returns URLs as recrawler seeds.
// Applies the given filter (status, mime, language, domain, TLD, limit).
// Returns the seed URLs, unique domain count, and any error.
func ExtractSeedURLs(ctx context.Context, dbPath string, filter IndexFilter) ([]recrawler.SeedURL, int, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return nil, 0, fmt.Errorf("opening index db: %w", err)
	}
	defer db.Close()

	// Count unique domains first
	countQuery, countArgs := buildSeedCountQuery(filter)
	var uniqueDomains int
	if err := db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&uniqueDomains); err != nil {
		return nil, 0, fmt.Errorf("counting domains: %w", err)
	}

	// Extract URLs
	query, args := buildSeedQuery(filter)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying seeds: %w", err)
	}
	defer rows.Close()

	var seeds []recrawler.SeedURL
	for rows.Next() {
		var s recrawler.SeedURL
		if err := rows.Scan(&s.URL, &s.Domain); err != nil {
			return nil, 0, fmt.Errorf("scanning seed: %w", err)
		}
		seeds = append(seeds, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating seeds: %w", err)
	}

	return seeds, uniqueDomains, nil
}

// ExtractSeedStats returns summary stats without loading all URLs.
func ExtractSeedStats(ctx context.Context, dbPath string, filter IndexFilter) (*SeedStats, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return nil, fmt.Errorf("opening index db: %w", err)
	}
	defer db.Close()

	query, args := buildSeedStatsQuery(filter)
	var total, domains int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&total, &domains); err != nil {
		return nil, fmt.Errorf("querying stats: %w", err)
	}

	return &SeedStats{TotalURLs: total, UniqueDomains: domains}, nil
}

func buildSeedQuery(f IndexFilter) (string, []any) {
	var b strings.Builder
	var args []any

	b.WriteString(`SELECT url, COALESCE(url_host_registered_domain, '') as domain FROM ccindex`)

	conditions, condArgs := buildSeedConditions(f)
	args = append(args, condArgs...)

	if len(conditions) > 0 {
		b.WriteString(" WHERE ")
		b.WriteString(strings.Join(conditions, " AND "))
	}

	if f.Limit > 0 {
		b.WriteString(fmt.Sprintf(" LIMIT %d", f.Limit))
	}
	if f.Offset > 0 {
		b.WriteString(fmt.Sprintf(" OFFSET %d", f.Offset))
	}

	return b.String(), args
}

func buildSeedCountQuery(f IndexFilter) (string, []any) {
	var b strings.Builder
	var args []any

	b.WriteString(`SELECT COUNT(DISTINCT url_host_registered_domain) FROM ccindex`)

	conditions, condArgs := buildSeedConditions(f)
	args = append(args, condArgs...)

	if len(conditions) > 0 {
		b.WriteString(" WHERE ")
		b.WriteString(strings.Join(conditions, " AND "))
	}

	// Apply limit via subquery if limit is set
	if f.Limit > 0 {
		inner := b.String()
		b.Reset()
		b.WriteString(
			`SELECT COUNT(DISTINCT domain) FROM (SELECT url, COALESCE(url_host_registered_domain, '') as domain FROM ccindex`,
		)
		conditions2, condArgs2 := buildSeedConditions(f)
		args = condArgs2
		if len(conditions2) > 0 {
			b.WriteString(" WHERE ")
			b.WriteString(strings.Join(conditions2, " AND "))
		}
		b.WriteString(fmt.Sprintf(" LIMIT %d) sub", f.Limit))
		_ = inner
	}

	return b.String(), args
}

func buildSeedStatsQuery(f IndexFilter) (string, []any) {
	var b strings.Builder
	var args []any

	b.WriteString(`SELECT COUNT(*), COUNT(DISTINCT url_host_registered_domain) FROM ccindex`)

	conditions, condArgs := buildSeedConditions(f)
	args = append(args, condArgs...)

	if len(conditions) > 0 {
		b.WriteString(" WHERE ")
		b.WriteString(strings.Join(conditions, " AND "))
	}

	return b.String(), args
}

func buildSeedConditions(f IndexFilter) ([]string, []any) {
	var conditions []string
	var args []any

	// Always filter out NULL warc_filename (these have no content)
	conditions = append(conditions, "warc_filename IS NOT NULL")

	if len(f.StatusCodes) > 0 {
		placeholders := make([]string, len(f.StatusCodes))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		conditions = append(conditions, fmt.Sprintf("fetch_status IN (%s)", strings.Join(placeholders, ",")))
		for _, s := range f.StatusCodes {
			args = append(args, s)
		}
	}

	if len(f.MimeTypes) > 0 {
		placeholders := make([]string, len(f.MimeTypes))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		conditions = append(conditions, fmt.Sprintf("content_mime_detected IN (%s)", strings.Join(placeholders, ",")))
		for _, m := range f.MimeTypes {
			args = append(args, m)
		}
	}

	if len(f.TLDs) > 0 {
		placeholders := make([]string, len(f.TLDs))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		conditions = append(conditions, fmt.Sprintf("url_host_tld IN (%s)", strings.Join(placeholders, ",")))
		for _, t := range f.TLDs {
			args = append(args, t)
		}
	}

	if len(f.Domains) > 0 {
		placeholders := make([]string, len(f.Domains))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		conditions = append(conditions, fmt.Sprintf("url_host_registered_domain IN (%s)", strings.Join(placeholders, ",")))
		for _, d := range f.Domains {
			args = append(args, d)
		}
	}

	if len(f.ExcludeDomains) > 0 {
		placeholders := make([]string, len(f.ExcludeDomains))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		conditions = append(conditions, fmt.Sprintf("url_host_registered_domain NOT IN (%s)", strings.Join(placeholders, ",")))
		for _, d := range f.ExcludeDomains {
			args = append(args, d)
		}
	}

	for _, lang := range f.Languages {
		conditions = append(conditions, "content_languages LIKE ?")
		args = append(args, "%"+lang+"%")
	}

	return conditions, args
}
