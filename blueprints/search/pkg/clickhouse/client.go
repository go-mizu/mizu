package clickhouse

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client sends SQL queries to ClickHouse Cloud via the HTTP interface.
type Client struct {
	baseURL  string
	user     string
	password string
	http     *http.Client
}

// NewClient creates a ClickHouse HTTP client.
func NewClient(host string, port int, user, password string) *Client {
	if user == "" {
		user = "default"
	}
	return &Client{
		baseURL:  fmt.Sprintf("https://%s:%d", host, port),
		user:     user,
		password: password,
		http:     &http.Client{Timeout: 60 * time.Second},
	}
}

// Query runs SQL and returns rows as []map[string]any plus ordered column names.
// Uses FORMAT JSONEachRow (NDJSON).
func (c *Client) Query(sqlStr string) ([]map[string]any, []string, error) {
	fullSQL := strings.TrimRight(strings.TrimSpace(sqlStr), ";") + " FORMAT JSONEachRow"

	req, err := http.NewRequest("POST", c.baseURL+"/?query="+url.QueryEscape(fullSQL), nil)
	if err != nil {
		return nil, nil, err
	}
	req.SetBasicAuth(c.user, c.password)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("clickhouse error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rows []map[string]any
	var cols []string
	colSeen := map[string]bool{}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal(line, &row); err != nil {
			continue
		}
		// Capture column order from first row
		if len(rows) == 0 {
			// json.Unmarshal doesn't preserve order; decode again with decoder to get order
			dec := json.NewDecoder(strings.NewReader(string(line)))
			dec.UseNumber()
			var orderedRow map[string]json.RawMessage
			_ = json.Unmarshal(line, &orderedRow)
			// Re-decode preserving key order via json.Decoder token stream
			dec2 := json.NewDecoder(strings.NewReader(string(line)))
			tok, _ := dec2.Token() // {
			if delim, ok := tok.(json.Delim); ok && delim == '{' {
				for dec2.More() {
					key, _ := dec2.Token()
					var val json.RawMessage
					_ = dec2.Decode(&val)
					if k, ok := key.(string); ok && !colSeen[k] {
						cols = append(cols, k)
						colSeen[k] = true
					}
				}
			}
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("scan: %w", err)
	}
	return rows, cols, nil
}

// Ping runs SELECT 1 to verify connectivity.
func (c *Client) Ping() error {
	req, err := http.NewRequest("GET", c.baseURL+"/ping", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.user, c.password)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping failed: %d", resp.StatusCode)
	}
	return nil
}
