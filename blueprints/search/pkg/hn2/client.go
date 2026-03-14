package hn2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RemoteInfo describes the current state of the remote HN data source.
type RemoteInfo struct {
	Count     int64
	MaxID     int64
	MaxTime   string
	CheckedAt time.Time
}

// monthInfo describes a calendar month available in the remote source.
// It is internal to the package; callers see only the aggregated results.
type monthInfo struct {
	Year  int
	Month int
	Count int64
}

// remoteInfo queries the remote source for the current item count and max ID.
// Used by the live task as a cold-start watermark when no local state exists.
func (c Config) remoteInfo(ctx context.Context) (*RemoteInfo, error) {
	cfg := c.resolved()
	q := fmt.Sprintf(
		`SELECT toInt64(count()) AS c, toInt64(max(id)) AS max_id, toString(max(time)) AS max_time FROM %s FORMAT JSONEachRow`,
		cfg.fqTable(),
	)
	body, err := cfg.query(ctx, q)
	if err != nil {
		return nil, err
	}
	var row struct {
		Count   any    `json:"c"`
		MaxID   any    `json:"max_id"`
		MaxTime string `json:"max_time"`
	}
	if err := json.Unmarshal(body, &row); err != nil {
		return nil, fmt.Errorf("decode remote info: %w", err)
	}
	count, err := parseIntAny(row.Count)
	if err != nil {
		return nil, fmt.Errorf("parse remote count: %w", err)
	}
	maxID, err := parseIntAny(row.MaxID)
	if err != nil {
		return nil, fmt.Errorf("parse remote max_id: %w", err)
	}
	return &RemoteInfo{
		Count:     count,
		MaxID:     maxID,
		MaxTime:   row.MaxTime,
		CheckedAt: time.Now().UTC(),
	}, nil
}

// listMonths returns all calendar months with data in the remote source,
// excluding the current (incomplete) month.
func (c Config) listMonths(ctx context.Context) ([]monthInfo, error) {
	cfg := c.resolved()
	q := fmt.Sprintf(
		`SELECT toYear(time) AS y, toMonth(time) AS m, toInt64(count()) AS n `+
			`FROM %s WHERE time IS NOT NULL GROUP BY y, m ORDER BY y, m FORMAT JSONEachRow`,
		cfg.fqTable(),
	)
	body, err := cfg.query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list months: %w", err)
	}
	now := time.Now().UTC()
	curYear, curMonth := now.Year(), int(now.Month())
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	out := make([]monthInfo, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			Y any `json:"y"`
			M any `json:"m"`
			N any `json:"n"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("decode month row: %w", err)
		}
		y, _ := parseIntAny(row.Y)
		m, _ := parseIntAny(row.M)
		n, _ := parseIntAny(row.N)
		if int(y) == curYear && int(m) == curMonth {
			continue // current month is incomplete
		}
		out = append(out, monthInfo{Year: int(y), Month: int(m), Count: n})
	}
	return out, nil
}

// query executes a SQL statement against the ClickHouse endpoint and returns
// the response body. The limit is 16 MiB — enough for full JSONEachRow responses
// (ListMonths returns ~232 rows, each ~50 bytes).
func (c Config) query(ctx context.Context, q string) ([]byte, error) {
	req, err := c.newRequest(ctx, q)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("remote query: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("read remote response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("remote query HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// newRequest builds an authenticated HTTP POST request for the ClickHouse endpoint.
func (c Config) newRequest(ctx context.Context, q string) (*http.Request, error) {
	cfg := c.resolved()
	u, err := url.Parse(cfg.EndpointURL)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint URL: %w", err)
	}
	qp := u.Query()
	if cfg.User != "" {
		qp.Set("user", cfg.User)
	}
	if cfg.Database != "" {
		qp.Set("database", cfg.Database)
	}
	qp.Set("max_result_rows", "0")
	qp.Set("max_result_bytes", "0")
	qp.Set("result_overflow_mode", "throw")
	u.RawQuery = qp.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(q))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	return req, nil
}

// downloadHTTPClient returns a clone of the configured HTTP client with no
// timeout, suitable for long-running Parquet downloads. Clones to avoid
// mutating the shared client. Optionally configures a custom DNS resolver.
func (c Config) downloadHTTPClient() *http.Client {
	cfg := c.resolved()
	base := cfg.httpClient()
	// Clone to avoid mutating the shared client.
	clone := *base
	clone.Timeout = 0
	if cfg.DNSServer == "" {
		return &clone
	}
	var tr *http.Transport
	if bt, ok := base.Transport.(*http.Transport); ok && bt != nil {
		tr = bt.Clone()
	} else {
		tr = http.DefaultTransport.(*http.Transport).Clone()
	}
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "udp", cfg.DNSServer)
		},
	}
	tr.DialContext = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Resolver:  resolver,
	}).DialContext
	clone.Transport = tr
	return &clone
}
