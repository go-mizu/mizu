package hn2

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Analytics holds rich dataset statistics queried from the ClickHouse source.
// It is computed once at publish startup and passed to all README generations.
type Analytics struct {
	// Content type counts
	Stories  int64
	Comments int64
	Jobs     int64
	Polls    int64
	PollOpts int64

	// Community
	UniqueAuthors int64
	TopAuthors    []NameCount

	// Engagement
	AvgScore        float64
	MedianScore     int64
	MaxScore        int64
	StoriesOver100  int64
	StoriesOver1000 int64
	AvgDescendants  float64
	MaxDescendants  int64

	// Content patterns
	StoriesWithURLPct float64
	TopDomains        []NameCount

	// Source freshness
	SourceMaxTime string // max(time) from the live ClickHouse table, e.g. "2026-03-14 15:30:00"
}

// NameCount is a name+count pair used for top-N ranked lists.
type NameCount struct {
	Name  string
	Count int64
}

// QueryAnalytics fetches comprehensive dataset statistics from the ClickHouse source.
// It makes four serial HTTP queries (ClickHouse public endpoint rate-limits concurrent
// connections). Called once at the start of the publish process.
func (c Config) QueryAnalytics(ctx context.Context) (*Analytics, error) {
	cfg := c.resolved()
	a := &Analytics{}

	// Query 1: type distribution, unique authors, and latest item time.
	q1 := fmt.Sprintf(
		`SELECT countIf(type='story') AS stories, countIf(type='comment') AS comments,`+
			` countIf(type='job') AS jobs, countIf(type='poll') AS polls,`+
			` countIf(type='pollopt') AS pollopts, uniqExact(by) AS authors,`+
			` toString(max(time)) AS max_time`+
			` FROM %s FORMAT JSONEachRow`,
		cfg.fqTable())
	if err := cfg.querySingleRow(ctx, q1, func(body []byte) error {
		var r struct {
			Stories  any    `json:"stories"`
			Comments any    `json:"comments"`
			Jobs     any    `json:"jobs"`
			Polls    any    `json:"polls"`
			PollOpts any    `json:"pollopts"`
			Authors  any    `json:"authors"`
			MaxTime  string `json:"max_time"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
		a.Stories, _ = parseIntAny(r.Stories)
		a.Comments, _ = parseIntAny(r.Comments)
		a.Jobs, _ = parseIntAny(r.Jobs)
		a.Polls, _ = parseIntAny(r.Polls)
		a.PollOpts, _ = parseIntAny(r.PollOpts)
		a.UniqueAuthors, _ = parseIntAny(r.Authors)
		a.SourceMaxTime = r.MaxTime
		return nil
	}); err != nil {
		return nil, fmt.Errorf("query type distribution: %w", err)
	}

	// Query 2: score and descendant stats.
	q2 := fmt.Sprintf(
		`SELECT round(avg(score),1) AS avg_score, round(median(score),0) AS med_score,`+
			` max(score) AS max_score, countIf(score>100) AS over100, countIf(score>1000) AS over1000,`+
			` round(avgIf(descendants, type='story' AND descendants>0),1) AS avg_desc,`+
			` max(descendants) AS max_desc,`+
			` round(100.0*countIf(url!='' AND type='story')/countIf(type='story'),1) AS url_pct`+
			` FROM %s FORMAT JSONEachRow`,
		cfg.fqTable())
	if err := cfg.querySingleRow(ctx, q2, func(body []byte) error {
		var r struct {
			AvgScore any `json:"avg_score"`
			MedScore any `json:"med_score"`
			MaxScore any `json:"max_score"`
			Over100  any `json:"over100"`
			Over1000 any `json:"over1000"`
			AvgDesc  any `json:"avg_desc"`
			MaxDesc  any `json:"max_desc"`
			URLPct   any `json:"url_pct"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
		a.AvgScore = parseFloatAny(r.AvgScore)
		a.MedianScore, _ = parseIntAny(r.MedScore)
		a.MaxScore, _ = parseIntAny(r.MaxScore)
		a.StoriesOver100, _ = parseIntAny(r.Over100)
		a.StoriesOver1000, _ = parseIntAny(r.Over1000)
		a.AvgDescendants = parseFloatAny(r.AvgDesc)
		a.MaxDescendants, _ = parseIntAny(r.MaxDesc)
		a.StoriesWithURLPct = parseFloatAny(r.URLPct)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("query score stats: %w", err)
	}

	// Query 3: top 15 authors by story count.
	q3 := fmt.Sprintf(
		`SELECT by, toInt64(count()) AS cnt FROM %s`+
			` WHERE type='story' AND by!='' GROUP BY by ORDER BY cnt DESC LIMIT 15 FORMAT JSONEachRow`,
		cfg.fqTable())
	if err := cfg.queryRows(ctx, q3, func(body []byte) error {
		var r struct {
			By  string `json:"by"`
			Cnt any    `json:"cnt"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			return err
		}
		cnt, _ := parseIntAny(r.Cnt)
		a.TopAuthors = append(a.TopAuthors, NameCount{Name: r.By, Count: cnt})
		return nil
	}); err != nil {
		return nil, fmt.Errorf("query top authors: %w", err)
	}

	// Query 4: top 10 domains by story count.
	q4 := fmt.Sprintf(
		`SELECT domain(url) AS d, toInt64(count()) AS cnt FROM %s`+
			` WHERE type='story' AND url!='' GROUP BY d ORDER BY cnt DESC LIMIT 10 FORMAT JSONEachRow`,
		cfg.fqTable())
	if err := cfg.queryRows(ctx, q4, func(body []byte) error {
		var r struct {
			D   string `json:"d"`
			Cnt any    `json:"cnt"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			return err
		}
		cnt, _ := parseIntAny(r.Cnt)
		a.TopDomains = append(a.TopDomains, NameCount{Name: r.D, Count: cnt})
		return nil
	}); err != nil {
		return nil, fmt.Errorf("query top domains: %w", err)
	}

	return a, nil
}

// querySingleRow executes q, reads the single-row JSONEachRow response, and calls fn.
func (c Config) querySingleRow(ctx context.Context, q string, fn func([]byte) error) error {
	body, err := c.query(ctx, q)
	if err != nil {
		return err
	}
	return fn(body)
}

// queryRows executes q, splits the multi-row JSONEachRow response, and calls fn per line.
func (c Config) queryRows(ctx context.Context, q string, fn func([]byte) error) error {
	body, err := c.query(ctx, q)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}
		if err := fn([]byte(line)); err != nil {
			continue // skip malformed rows
		}
	}
	return nil
}
