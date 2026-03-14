package hn2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
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

// MonthInfo describes a month available in the remote source.
type MonthInfo struct {
	Year  int
	Month int
	Count int64
}

// RemoteInfo queries the remote source for current item count and max ID.
func (c Config) RemoteInfo(ctx context.Context) (*RemoteInfo, error) {
	cfg := c.WithDefaults()
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

// ListMonths returns all months with data available in the remote source.
// The current calendar month is excluded (it is incomplete).
func (c Config) ListMonths(ctx context.Context) ([]MonthInfo, error) {
	cfg := c.WithDefaults()
	q := fmt.Sprintf(
		`SELECT toYear(time) AS y, toMonth(time) AS m, toInt64(count()) AS n FROM %s WHERE time IS NOT NULL GROUP BY y, m ORDER BY y, m FORMAT JSONEachRow`,
		cfg.fqTable(),
	)
	body, err := cfg.query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list months: %w", err)
	}
	now := time.Now().UTC()
	curYear, curMonth := now.Year(), int(now.Month())
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	out := make([]MonthInfo, 0, len(lines))
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
			continue
		}
		out = append(out, MonthInfo{Year: int(y), Month: int(m), Count: n})
	}
	return out, nil
}

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
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read remote response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("remote query HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func (c Config) newRequest(ctx context.Context, q string) (*http.Request, error) {
	cfg := c.WithDefaults()
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

func (c Config) downloadHTTPClient() *http.Client {
	cfg := c.WithDefaults()
	base := cfg.httpClient()
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
			d := &net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "udp", cfg.DNSServer)
		},
	}
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Resolver:  resolver,
	}
	tr.DialContext = dialer.DialContext
	clone.Transport = tr
	return &clone
}

func parseIntAny(v any) (int64, error) {
	switch x := v.(type) {
	case float64:
		return int64(x), nil
	case int64:
		return x, nil
	case int:
		return int64(x), nil
	case json.Number:
		return x.Int64()
	case string:
		return strconv.ParseInt(strings.TrimSpace(x), 10, 64)
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", v)
	}
}
