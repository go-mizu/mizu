package hn2

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const hnAPIBase = "https://hacker-news.firebaseio.com/v2"

// HNItem is an item returned by the HN Firebase API.
type HNItem struct {
	ID          int64   `json:"id"`
	Deleted     bool    `json:"deleted"`
	Type        string  `json:"type"` // "story","comment","job","poll","pollopt"
	By          string  `json:"by"`
	Time        int64   `json:"time"` // unix timestamp
	Text        string  `json:"text"`
	Dead        bool    `json:"dead"`
	Parent      int64   `json:"parent"`
	Poll        int64   `json:"poll"`
	Kids        []int64 `json:"kids"`
	URL         string  `json:"url"`
	Score       int32   `json:"score"`
	Title       string  `json:"title"`
	Parts       []int64 `json:"parts"`
	Descendants int32   `json:"descendants"`
}

// typeInt maps HN type strings to the integer used in the parquet schema.
func (item HNItem) typeInt() int8 {
	switch item.Type {
	case "story":
		return 1
	case "comment":
		return 2
	case "poll":
		return 3
	case "pollopt":
		return 4
	case "job":
		return 5
	default:
		return 0
	}
}

// FetchHNMaxItem returns the current highest item ID from the HN Firebase API.
func FetchHNMaxItem(ctx context.Context) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", hnAPIBase+"/maxitem.json", nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("maxitem: %w", err)
	}
	defer resp.Body.Close()
	var id int64
	if err := json.NewDecoder(resp.Body).Decode(&id); err != nil {
		return 0, fmt.Errorf("decode maxitem: %w", err)
	}
	return id, nil
}

// FetchHNItemRange fetches items with IDs in [fromID, toID] from the HN Firebase API.
// Uses up to 20 concurrent requests. Items are returned sorted by ID ascending.
// Deleted/missing items (null response) are included with Deleted=true and zero fields.
func FetchHNItemRange(ctx context.Context, fromID, toID int64) ([]HNItem, error) {
	n := int(toID - fromID + 1)
	if n <= 0 {
		return nil, nil
	}
	items := make([]HNItem, n)
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)
	sem := make(chan struct{}, 20)
	for i := 0; i < n; i++ {
		id := fromID + int64(i)
		wg.Add(1)
		go func(idx int, itemID int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			item, err := fetchHNItem(ctx, itemID)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			if item == nil {
				items[idx] = HNItem{ID: itemID, Deleted: true}
			} else {
				items[idx] = *item
			}
		}(i, id)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return items, nil
}

func fetchHNItem(ctx context.Context, id int64) (*HNItem, error) {
	url := fmt.Sprintf("%s/item/%d.json", hnAPIBase, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("item %d: %w", id, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil {
		return nil, fmt.Errorf("read item %d: %w", id, err)
	}
	if string(body) == "null" {
		return nil, nil // item not yet published or fully deleted
	}
	var item HNItem
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, fmt.Errorf("decode item %d: %w", id, err)
	}
	return &item, nil
}

// GroupHNItemsByWindow groups items into 5-min (interval) time buckets.
// Returns a map from window-start-time → items in that window, sorted by ID.
func GroupHNItemsByWindow(items []HNItem, interval time.Duration) map[time.Time][]HNItem {
	out := make(map[time.Time][]HNItem)
	for _, item := range items {
		if item.Time == 0 {
			continue // skip items with no timestamp (deleted nulls)
		}
		t := time.Unix(item.Time, 0).UTC().Truncate(interval)
		out[t] = append(out[t], item)
	}
	// Sort each bucket by ID ascending.
	for t := range out {
		sort.Slice(out[t], func(i, j int) bool { return out[t][i].ID < out[t][j].ID })
	}
	return out
}

// hnItemRow is the JSON row written to the temp NDJSON file for DuckDB ingestion.
type hnItemRow struct {
	ID          int64    `json:"id"`
	Deleted     int8     `json:"deleted"`
	Type        int8     `json:"type"`
	By          string   `json:"by"`
	TimeUnix    int64    `json:"time_unix"`
	Text        string   `json:"text"`
	Dead        int8     `json:"dead"`
	Parent      int64    `json:"parent"`
	Poll        int64    `json:"poll"`
	Kids        []int64  `json:"kids"`
	URL         string   `json:"url"`
	Score       int32    `json:"score"`
	Title       string   `json:"title"`
	Parts       []int64  `json:"parts"`
	Descendants int32    `json:"descendants"`
	Words       []string `json:"words"`
}

func toRow(item HNItem) hnItemRow {
	deleted := int8(0)
	if item.Deleted {
		deleted = 1
	}
	dead := int8(0)
	if item.Dead {
		dead = 1
	}
	kids := item.Kids
	if kids == nil {
		kids = []int64{}
	}
	parts := item.Parts
	if parts == nil {
		parts = []int64{}
	}
	return hnItemRow{
		ID:          item.ID,
		Deleted:     deleted,
		Type:        item.typeInt(),
		By:          item.By,
		TimeUnix:    item.Time,
		Text:        item.Text,
		Dead:        dead,
		Parent:      item.Parent,
		Poll:        item.Poll,
		Kids:        kids,
		URL:         item.URL,
		Score:       item.Score,
		Title:       item.Title,
		Parts:       parts,
		Descendants: item.Descendants,
		Words:       tokenizeHN(item.Title, item.Text),
	}
}

// tokenizeHN extracts lowercase word tokens from title+text for the words[] column.
func tokenizeHN(title, text string) []string {
	seen := make(map[string]bool)
	var buf strings.Builder
	add := func() {
		if buf.Len() >= 2 {
			seen[buf.String()] = true
		}
		buf.Reset()
	}
	for _, r := range title + " " + text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buf.WriteRune(unicode.ToLower(r))
		} else {
			add()
		}
	}
	add()
	words := make([]string, 0, len(seen))
	for w := range seen {
		words = append(words, w)
	}
	sort.Strings(words)
	return words
}

// WriteHNParquet writes HN items to a Parquet file at outPath using DuckDB.
// The schema matches the ClickHouse hacker_news table output (FORMAT Parquet).
func WriteHNParquet(ctx context.Context, items []HNItem, outPath string) (FetchResult, error) {
	start := time.Now()
	if len(items) == 0 {
		return FetchResult{Duration: time.Since(start)}, nil
	}

	// Write NDJSON to temp file.
	tmpf, err := os.CreateTemp(filepath.Dir(outPath), ".hn-api-*.ndjson")
	if err != nil {
		return FetchResult{}, fmt.Errorf("create ndjson tmp: %w", err)
	}
	tmpJSON := tmpf.Name()
	defer os.Remove(tmpJSON)

	enc := json.NewEncoder(tmpf)
	for _, item := range items {
		if err := enc.Encode(toRow(item)); err != nil {
			tmpf.Close()
			return FetchResult{}, fmt.Errorf("encode item %d: %w", item.ID, err)
		}
	}
	if err := tmpf.Close(); err != nil {
		return FetchResult{}, fmt.Errorf("close ndjson tmp: %w", err)
	}

	// Use DuckDB to convert NDJSON → Parquet.
	if err := ensureParentDir(outPath); err != nil {
		return FetchResult{}, err
	}
	tmpPq, err := os.CreateTemp(filepath.Dir(outPath), ".hn-pq-*.parquet")
	if err != nil {
		return FetchResult{}, fmt.Errorf("create parquet tmp: %w", err)
	}
	tmpPq.Close()
	tmpParquet := tmpPq.Name()
	defer os.Remove(tmpParquet)

	db, err := sql.Open("duckdb", "")
	if err != nil {
		return FetchResult{}, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	q := fmt.Sprintf(`
COPY (
    SELECT
        id::UINTEGER         AS id,
        deleted::UTINYINT    AS deleted,
        type::TINYINT        AS type,
        by                   AS by,
        to_timestamp(time_unix)::TIMESTAMP AS time,
        text                 AS text,
        dead::UTINYINT       AS dead,
        parent::UINTEGER     AS parent,
        poll::UINTEGER       AS poll,
        kids::UINTEGER[]     AS kids,
        url                  AS url,
        score::INTEGER       AS score,
        title                AS title,
        parts::UINTEGER[]    AS parts,
        descendants::INTEGER AS descendants,
        words                AS words
    FROM read_ndjson_auto('%s')
    ORDER BY id
) TO '%s' (FORMAT PARQUET, CODEC 'ZSTD', COMPRESSION_LEVEL 22)`,
		escapeSQLStr(tmpJSON), escapeSQLStr(tmpParquet))

	if _, err := db.ExecContext(ctx, q); err != nil {
		return FetchResult{}, fmt.Errorf("duckdb parquet write: %w", err)
	}

	if err := os.Rename(tmpParquet, outPath); err != nil {
		return FetchResult{}, fmt.Errorf("rename parquet: %w", err)
	}

	return Config{}.scanParquetResult(ctx, outPath, 0, time.Since(start))
}
