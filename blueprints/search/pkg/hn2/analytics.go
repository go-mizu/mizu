package hn2

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Analytics holds rich dataset statistics queried from the ClickHouse source.
// Computed once at publish startup and passed to all README generations.
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
}

// NameCount is a generic name+count pair used for top-N lists.
type NameCount struct {
	Name  string
	Count int64
}

// QueryAnalytics fetches comprehensive dataset statistics from the ClickHouse source.
// This is called once at the start of the publish process.
func (c Config) QueryAnalytics(ctx context.Context) (*Analytics, error) {
	cfg := c.WithDefaults()
	a := &Analytics{}

	// Query 1: Type distribution + unique authors
	q1 := fmt.Sprintf(
		`SELECT countIf(type='story') AS stories, countIf(type='comment') AS comments, `+
			`countIf(type='job') AS jobs, countIf(type='poll') AS polls, `+
			`countIf(type='pollopt') AS pollopts, uniqExact(by) AS authors `+
			`FROM %s FORMAT JSONEachRow`, cfg.fqTable())
	body1, err := cfg.query(ctx, q1)
	if err != nil {
		return nil, fmt.Errorf("query type distribution: %w", err)
	}
	var r1 struct {
		Stories  any `json:"stories"`
		Comments any `json:"comments"`
		Jobs     any `json:"jobs"`
		Polls    any `json:"polls"`
		PollOpts any `json:"pollopts"`
		Authors  any `json:"authors"`
	}
	if err := json.Unmarshal(body1, &r1); err != nil {
		return nil, fmt.Errorf("decode type distribution: %w", err)
	}
	a.Stories, _ = parseIntAny(r1.Stories)
	a.Comments, _ = parseIntAny(r1.Comments)
	a.Jobs, _ = parseIntAny(r1.Jobs)
	a.Polls, _ = parseIntAny(r1.Polls)
	a.PollOpts, _ = parseIntAny(r1.PollOpts)
	a.UniqueAuthors, _ = parseIntAny(r1.Authors)

	// Query 2: Score and comment stats
	q2 := fmt.Sprintf(
		`SELECT round(avg(score),1) AS avg_score, round(median(score),0) AS med_score, `+
			`max(score) AS max_score, countIf(score>100) AS over100, countIf(score>1000) AS over1000, `+
			`round(avgIf(descendants, type='story' AND descendants>0),1) AS avg_desc, `+
			`max(descendants) AS max_desc, `+
			`round(100.0*countIf(url!='' AND type='story')/countIf(type='story'),1) AS url_pct `+
			`FROM %s FORMAT JSONEachRow`, cfg.fqTable())
	body2, err := cfg.query(ctx, q2)
	if err != nil {
		return nil, fmt.Errorf("query score stats: %w", err)
	}
	var r2 struct {
		AvgScore any `json:"avg_score"`
		MedScore any `json:"med_score"`
		MaxScore any `json:"max_score"`
		Over100  any `json:"over100"`
		Over1000 any `json:"over1000"`
		AvgDesc  any `json:"avg_desc"`
		MaxDesc  any `json:"max_desc"`
		URLPct   any `json:"url_pct"`
	}
	if err := json.Unmarshal(body2, &r2); err != nil {
		return nil, fmt.Errorf("decode score stats: %w", err)
	}
	a.AvgScore = parseFloatAny(r2.AvgScore)
	a.MedianScore, _ = parseIntAny(r2.MedScore)
	a.MaxScore, _ = parseIntAny(r2.MaxScore)
	a.StoriesOver100, _ = parseIntAny(r2.Over100)
	a.StoriesOver1000, _ = parseIntAny(r2.Over1000)
	a.AvgDescendants = parseFloatAny(r2.AvgDesc)
	a.MaxDescendants, _ = parseIntAny(r2.MaxDesc)
	a.StoriesWithURLPct = parseFloatAny(r2.URLPct)

	// Query 3: Top authors by story count
	q3 := fmt.Sprintf(
		`SELECT by, toInt64(count()) AS cnt FROM %s `+
			`WHERE type='story' AND by!='' GROUP BY by ORDER BY cnt DESC LIMIT 15 FORMAT JSONEachRow`,
		cfg.fqTable())
	body3, err := cfg.query(ctx, q3)
	if err != nil {
		return nil, fmt.Errorf("query top authors: %w", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(body3)), "\n") {
		if line == "" {
			continue
		}
		var r struct {
			By  string `json:"by"`
			Cnt any    `json:"cnt"`
		}
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}
		cnt, _ := parseIntAny(r.Cnt)
		a.TopAuthors = append(a.TopAuthors, NameCount{Name: r.By, Count: cnt})
	}

	// Query 4: Top domains
	q4 := fmt.Sprintf(
		`SELECT domain(url) AS d, toInt64(count()) AS cnt FROM %s `+
			`WHERE type='story' AND url!='' GROUP BY d ORDER BY cnt DESC LIMIT 10 FORMAT JSONEachRow`,
		cfg.fqTable())
	body4, err := cfg.query(ctx, q4)
	if err != nil {
		return nil, fmt.Errorf("query top domains: %w", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(body4)), "\n") {
		if line == "" {
			continue
		}
		var r struct {
			D   string `json:"d"`
			Cnt any    `json:"cnt"`
		}
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}
		cnt, _ := parseIntAny(r.Cnt)
		a.TopDomains = append(a.TopDomains, NameCount{Name: r.D, Count: cnt})
	}

	return a, nil
}

func parseFloatAny(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case json.Number:
		f, _ := x.Float64()
		return f
	case string:
		var f float64
		fmt.Sscanf(strings.TrimSpace(x), "%f", &f)
		return f
	default:
		n, _ := parseIntAny(v)
		return float64(n)
	}
}
