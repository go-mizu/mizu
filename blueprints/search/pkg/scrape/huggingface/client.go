package huggingface

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Client struct {
	http       *http.Client
	delay      time.Duration
	userAgents []string
	mu         sync.Mutex
	lastReq    time.Time
}

func NewClient(cfg Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        16,
		MaxConnsPerHost:     cfg.Workers + 4,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return &Client{
		http: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		delay:      cfg.Delay,
		userAgents: userAgents,
	}
}

func (c *Client) ListPage(ctx context.Context, entityType string, nextURL string, pageSize int) ([]map[string]any, string, int, error) {
	rawURL := nextURL
	if rawURL == "" {
		u, err := url.Parse(BaseURL + c.listPath(entityType))
		if err != nil {
			return nil, "", 0, err
		}
		q := u.Query()
		if pageSize > 0 {
			q.Set("limit", fmt.Sprintf("%d", pageSize))
		}
		u.RawQuery = q.Encode()
		rawURL = u.String()
	}

	var items []map[string]any
	code, headers, _, err := c.FetchJSON(ctx, rawURL, &items)
	if err != nil {
		return nil, "", code, err
	}
	return items, parseNextLink(headers), code, nil
}

func (c *Client) GetModel(ctx context.Context, repoID string) (*Model, []RepoFile, []RepoLink, int, error) {
	path, err := repoAPIPath(EntityModel, repoID)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	var payload map[string]any
	code, _, body, err := c.FetchJSON(ctx, BaseURL+path, &payload)
	if err != nil {
		return nil, nil, nil, code, err
	}
	model := &Model{
		RepoID:               firstString(payload["id"], payload["modelId"]),
		Author:               stringValue(payload["author"]),
		SHA:                  stringValue(payload["sha"]),
		CreatedAt:            parseTime(stringValue(payload["createdAt"])),
		LastModified:         parseTime(stringValue(payload["lastModified"])),
		Private:              boolValue(payload["private"]),
		Gated:                boolValue(payload["gated"]),
		Disabled:             boolValue(payload["disabled"]),
		Likes:                int64Value(payload["likes"]),
		Downloads:            int64Value(payload["downloads"]),
		TrendingScore:        int64Value(payload["trendingScore"]),
		PipelineTag:          stringValue(payload["pipeline_tag"]),
		LibraryName:          stringValue(payload["library_name"]),
		TagsJSON:             marshalJSON(payload["tags"]),
		CardDataJSON:         marshalJSON(payload["cardData"]),
		ConfigJSON:           marshalJSON(payload["config"]),
		TransformersInfoJSON: marshalJSON(payload["transformersInfo"]),
		WidgetDataJSON:       marshalJSON(payload["widgetData"]),
		SpacesJSON:           marshalJSON(payload["spaces"]),
		RawJSON:              string(body),
		FetchedAt:            time.Now(),
	}
	return model, siblingsToFiles(EntityModel, model.RepoID, payload["siblings"]), modelSpacesToLinks(model.RepoID, payload["spaces"]), code, nil
}

func (c *Client) GetDataset(ctx context.Context, repoID string) (*Dataset, []RepoFile, int, error) {
	path, err := repoAPIPath(EntityDataset, repoID)
	if err != nil {
		return nil, nil, 0, err
	}
	var payload map[string]any
	code, _, body, err := c.FetchJSON(ctx, BaseURL+path, &payload)
	if err != nil {
		return nil, nil, code, err
	}
	ds := &Dataset{
		RepoID:        stringValue(payload["id"]),
		Author:        stringValue(payload["author"]),
		SHA:           stringValue(payload["sha"]),
		CreatedAt:     parseTime(stringValue(payload["createdAt"])),
		LastModified:  parseTime(stringValue(payload["lastModified"])),
		Private:       boolValue(payload["private"]),
		Gated:         boolValue(payload["gated"]),
		Disabled:      boolValue(payload["disabled"]),
		Likes:         int64Value(payload["likes"]),
		Downloads:     int64Value(payload["downloads"]),
		TrendingScore: int64Value(payload["trendingScore"]),
		Description:   stringValue(payload["description"]),
		TagsJSON:      marshalJSON(payload["tags"]),
		CardDataJSON:  marshalJSON(payload["cardData"]),
		RawJSON:       string(body),
		FetchedAt:     time.Now(),
	}
	return ds, siblingsToFiles(EntityDataset, ds.RepoID, payload["siblings"]), code, nil
}

func (c *Client) GetSpace(ctx context.Context, repoID string) (*Space, []RepoFile, []RepoLink, int, error) {
	path, err := repoAPIPath(EntitySpace, repoID)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	var payload map[string]any
	code, _, body, err := c.FetchJSON(ctx, BaseURL+path, &payload)
	if err != nil {
		return nil, nil, nil, code, err
	}
	space := &Space{
		RepoID:       stringValue(payload["id"]),
		Author:       stringValue(payload["author"]),
		SHA:          stringValue(payload["sha"]),
		CreatedAt:    parseTime(stringValue(payload["createdAt"])),
		LastModified: parseTime(stringValue(payload["lastModified"])),
		Private:      boolValue(payload["private"]),
		Disabled:     boolValue(payload["disabled"]),
		Likes:        int64Value(payload["likes"]),
		SDK:          stringValue(payload["sdk"]),
		Subdomain:    stringValue(payload["subdomain"]),
		TagsJSON:     marshalJSON(payload["tags"]),
		RuntimeJSON:  marshalJSON(payload["runtime"]),
		CardDataJSON: marshalJSON(payload["cardData"]),
		RawJSON:      string(body),
		FetchedAt:    time.Now(),
	}
	return space, siblingsToFiles(EntitySpace, space.RepoID, payload["siblings"]), spaceLinks(space.RepoID, payload), code, nil
}

func (c *Client) GetCollection(ctx context.Context, slug string) (*Collection, []CollectionItem, int, error) {
	parts := splitPath(slug)
	if len(parts) != 2 {
		return nil, nil, 0, fmt.Errorf("invalid collection slug: %s", slug)
	}
	rawURL := fmt.Sprintf("%s/api/collections/%s/%s", BaseURL, url.PathEscape(parts[0]), url.PathEscape(parts[1]))
	var payload map[string]any
	code, _, body, err := c.FetchJSON(ctx, rawURL, &payload)
	if err != nil {
		return nil, nil, code, err
	}
	collection := &Collection{
		Slug:        stringValue(payload["slug"]),
		Namespace:   parts[0],
		Title:       stringValue(payload["title"]),
		Description: stringValue(payload["description"]),
		OwnerJSON:   marshalJSON(payload["owner"]),
		Theme:       stringValue(payload["theme"]),
		Upvotes:     int64Value(payload["upvotes"]),
		Private:     boolValue(payload["private"]),
		Gating:      boolValue(payload["gating"]),
		LastUpdated: parseTime(stringValue(payload["lastUpdated"])),
		ItemsJSON:   marshalJSON(payload["items"]),
		RawJSON:     string(body),
		FetchedAt:   time.Now(),
	}
	items := collectionItems(collection.Slug, payload["items"])
	return collection, items, code, nil
}

func (c *Client) GetPaper(ctx context.Context, paperID string) (*Paper, int, error) {
	rawURL := fmt.Sprintf("%s/api/papers/%s", BaseURL, url.PathEscape(paperID))
	var payload map[string]any
	code, _, body, err := c.FetchJSON(ctx, rawURL, &payload)
	if err != nil {
		return nil, code, err
	}
	p := &Paper{
		PaperID:      stringValue(payload["id"]),
		Title:        stringValue(payload["title"]),
		Summary:      stringValue(payload["summary"]),
		AISummary:    stringValue(payload["ai_summary"]),
		PublishedAt:  parseTime(stringValue(payload["publishedAt"])),
		Upvotes:      int64Value(payload["upvotes"]),
		AuthorsJSON:  marshalJSON(payload["authors"]),
		GitHubRepo:   stringValue(payload["githubRepo"]),
		ProjectPage:  stringValue(payload["projectPage"]),
		ThumbnailURL: stringValue(payload["thumbnailUrl"]),
		RawJSON:      string(body),
		FetchedAt:    time.Now(),
	}
	return p, code, nil
}

func (c *Client) FetchJSON(ctx context.Context, rawURL string, out any) (int, http.Header, []byte, error) {
	body, headers, code, err := c.get(ctx, rawURL)
	if err != nil {
		return code, headers, nil, err
	}
	if code == http.StatusNotFound {
		return code, headers, nil, nil
	}
	if code < 200 || code >= 300 {
		return code, headers, body, fmt.Errorf("unexpected HTTP %d for %s", code, rawURL)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return code, headers, body, fmt.Errorf("decode JSON %s: %w", rawURL, err)
	}
	return code, headers, body, nil
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, http.Header, int, error) {
	c.rateLimit()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, nil, 0, err
	}
	req.Header.Set("User-Agent", c.userAgents[rand.Intn(len(c.userAgents))])
	req.Header.Set("Accept", "application/json,text/plain,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024))
	if err != nil {
		return nil, resp.Header, resp.StatusCode, err
	}
	return body, resp.Header, resp.StatusCode, nil
}

func (c *Client) rateLimit() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.delay <= 0 {
		return
	}
	if since := time.Since(c.lastReq); since < c.delay {
		time.Sleep(c.delay - since)
	}
	c.lastReq = time.Now()
}

func (c *Client) listPath(entityType string) string {
	switch entityType {
	case EntityModel:
		return "/api/models"
	case EntityDataset:
		return "/api/datasets"
	case EntitySpace:
		return "/api/spaces"
	case EntityCollection:
		return "/api/collections"
	case EntityPaper:
		return "/api/papers"
	default:
		return "/api/models"
	}
}

func parseNextLink(headers http.Header) string {
	link := headers.Get("Link")
	for _, part := range strings.Split(link, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, `rel="next"`) {
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start >= 0 && end > start {
				return part[start+1 : end]
			}
		}
	}
	return ""
}

func repoAPIPath(entityType, repoID string) (string, error) {
	parts := splitPath(repoID)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repo id: %s", repoID)
	}
	switch entityType {
	case EntityModel:
		return fmt.Sprintf("/api/models/%s/%s", url.PathEscape(parts[0]), url.PathEscape(parts[1])), nil
	case EntityDataset:
		return fmt.Sprintf("/api/datasets/%s/%s", url.PathEscape(parts[0]), url.PathEscape(parts[1])), nil
	case EntitySpace:
		return fmt.Sprintf("/api/spaces/%s/%s", url.PathEscape(parts[0]), url.PathEscape(parts[1])), nil
	default:
		return "", fmt.Errorf("unsupported repo entity: %s", entityType)
	}
}

func siblingsToFiles(entityType, repoID string, raw any) []RepoFile {
	items, _ := raw.([]any)
	out := make([]RepoFile, 0, len(items))
	for _, item := range items {
		m, _ := item.(map[string]any)
		if len(m) == 0 {
			continue
		}
		out = append(out, RepoFile{
			EntityType: entityType,
			RepoID:     repoID,
			Path:       stringValue(m["rfilename"]),
			Size:       int64Value(m["size"]),
			LFSJSON:    marshalJSON(m["lfs"]),
		})
	}
	return out
}

func modelSpacesToLinks(modelID string, raw any) []RepoLink {
	items, _ := raw.([]any)
	out := make([]RepoLink, 0, len(items))
	for _, item := range items {
		id := strings.TrimSpace(stringValue(item))
		if id == "" {
			continue
		}
		out = append(out, RepoLink{SrcType: EntityModel, SrcID: modelID, Rel: "used_by_space", DstType: EntitySpace, DstID: id})
	}
	return out
}

func spaceLinks(spaceID string, payload map[string]any) []RepoLink {
	var out []RepoLink
	for _, item := range anySlice(payload["models"]) {
		id := strings.TrimSpace(stringValue(item))
		if id != "" {
			out = append(out, RepoLink{SrcType: EntitySpace, SrcID: spaceID, Rel: "uses_model", DstType: EntityModel, DstID: id})
		}
	}
	for _, item := range anySlice(payload["datasets"]) {
		id := strings.TrimSpace(stringValue(item))
		if id != "" {
			out = append(out, RepoLink{SrcType: EntitySpace, SrcID: spaceID, Rel: "uses_dataset", DstType: EntityDataset, DstID: id})
		}
	}
	cardData, _ := payload["cardData"].(map[string]any)
	for _, key := range []struct {
		Field string
		Type  string
		Rel   string
	}{
		{Field: "models", Type: EntityModel, Rel: "uses_model"},
		{Field: "datasets", Type: EntityDataset, Rel: "uses_dataset"},
	} {
		for _, item := range anySlice(cardData[key.Field]) {
			id := strings.TrimSpace(stringValue(item))
			if id != "" {
				out = append(out, RepoLink{SrcType: EntitySpace, SrcID: spaceID, Rel: key.Rel, DstType: key.Type, DstID: id})
			}
		}
	}
	return dedupeLinks(out)
}

func collectionItems(collectionSlug string, raw any) []CollectionItem {
	items, _ := raw.([]any)
	out := make([]CollectionItem, 0, len(items))
	for _, item := range items {
		m, _ := item.(map[string]any)
		if len(m) == 0 {
			continue
		}
		out = append(out, CollectionItem{
			CollectionSlug: collectionSlug,
			ItemID:         stringValue(m["id"]),
			ItemType:       stringValue(m["type"]),
			Position:       int(int64Value(m["position"])),
			Author:         stringValue(m["author"]),
			RepoType:       stringValue(m["repoType"]),
			RawJSON:        marshalJSON(m),
		})
	}
	return out
}

func dedupeLinks(in []RepoLink) []RepoLink {
	seen := make(map[string]struct{}, len(in))
	out := make([]RepoLink, 0, len(in))
	for _, link := range in {
		key := link.SrcType + "\x00" + link.SrcID + "\x00" + link.Rel + "\x00" + link.DstType + "\x00" + link.DstID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, link)
	}
	return out
}

func firstString(values ...any) string {
	for _, v := range values {
		if s := stringValue(v); s != "" {
			return s
		}
	}
	return ""
}

func anySlice(v any) []any {
	if v == nil {
		return nil
	}
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

func stringValue(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func boolValue(v any) bool {
	b, _ := v.(bool)
	return b
}

func int64Value(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	default:
		return 0
	}
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func marshalJSON(v any) string {
	if v == nil {
		return ""
	}
	buf, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(buf)
}
