package kaggle

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

	"github.com/PuerkitoBio/goquery"
)

type Client struct {
	http       *http.Client
	userAgents []string
	delay      time.Duration
	lastReq    time.Time
	mu         sync.Mutex
}

func NewClient(cfg Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxConnsPerHost:     cfg.Workers + 2,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableCompression:  false,
	}
	return &Client{
		http: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		userAgents: userAgents,
		delay:      cfg.Delay,
	}
}

func (c *Client) FetchJSON(ctx context.Context, rawURL string, out any) (int, []byte, error) {
	body, code, err := c.Fetch(ctx, rawURL, "application/json")
	if err != nil {
		return code, body, err
	}
	if code == 404 {
		return code, body, nil
	}
	if code != 200 {
		return code, body, fmt.Errorf("unexpected HTTP %d for %s", code, rawURL)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return code, body, fmt.Errorf("decode JSON: %w", err)
	}
	return code, body, nil
}

func (c *Client) FetchHTMLMeta(ctx context.Context, rawURL string) (*PageMeta, int, error) {
	body, code, err := c.Fetch(ctx, rawURL, "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	if err != nil {
		return nil, code, err
	}
	if code == 404 {
		return nil, code, nil
	}
	if code != 200 {
		return nil, code, fmt.Errorf("unexpected HTTP %d for %s", code, rawURL)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, code, fmt.Errorf("parse HTML: %w", err)
	}
	meta := &PageMeta{
		Title: strings.TrimSpace(doc.Find("title").First().Text()),
		URL:   rawURL,
		Meta:  make(map[string]string),
	}
	doc.Find("meta").Each(func(_ int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		prop, _ := s.Attr("property")
		content, _ := s.Attr("content")
		key := strings.TrimSpace(name)
		if key == "" {
			key = strings.TrimSpace(prop)
		}
		if key == "" || strings.TrimSpace(content) == "" {
			return
		}
		meta.Meta[key] = strings.TrimSpace(content)
	})
	meta.Description = firstNonEmpty(meta.Meta["description"], meta.Meta["twitter:description"])
	meta.OGTitle = meta.Meta["og:title"]
	meta.OGDesc = meta.Meta["og:description"]
	meta.OGImage = firstNonEmpty(meta.Meta["og:image"], meta.Meta["twitter:image"])
	return meta, code, nil
}

func (c *Client) Fetch(ctx context.Context, rawURL, accept string) ([]byte, int, error) {
	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		c.rateLimit()
		body, code, err := c.doGet(ctx, rawURL, accept)
		if err != nil {
			if attempt == maxAttempts {
				return nil, 0, err
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		if code == 404 {
			return nil, code, nil
		}
		if code == 429 {
			if attempt == maxAttempts {
				return nil, code, fmt.Errorf("rate limited (HTTP 429)")
			}
			time.Sleep(time.Duration(attempt*attempt) * 5 * time.Second)
			continue
		}
		if code >= 500 {
			if attempt == maxAttempts {
				return nil, code, fmt.Errorf("server error HTTP %d", code)
			}
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}
		return body, code, nil
	}
	return nil, 0, fmt.Errorf("all attempts failed for %s", rawURL)
}

func (c *Client) doGet(ctx context.Context, rawURL, accept string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", c.userAgents[rand.Intn(len(c.userAgents))])
	req.Header.Set("Accept", accept)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", BaseURL+"/")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func (c *Client) rateLimit() {
	if c.delay <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	since := time.Since(c.lastReq)
	if since < c.delay {
		time.Sleep(c.delay - since)
	}
	c.lastReq = time.Now()
}

func (c *Client) ListDatasets(ctx context.Context, page int, search string) ([]Dataset, int, error) {
	if page <= 0 {
		page = 1
	}
	values := url.Values{}
	values.Set("page", fmt.Sprintf("%d", page))
	if search = strings.TrimSpace(search); search != "" {
		values.Set("search", search)
	}
	var resp []datasetAPIResponse
	_, raw, err := c.FetchJSON(ctx, BaseURL+"/api/v1/datasets/list?"+values.Encode(), &resp)
	if err != nil {
		return nil, 0, err
	}
	items := make([]Dataset, 0, len(resp))
	for _, item := range resp {
		items = append(items, convertDataset(item, raw))
	}
	return items, len(items), nil
}

func (c *Client) ViewDataset(ctx context.Context, ref string) (*Dataset, error) {
	owner, slug, err := splitOwnerSlug(ref)
	if err != nil {
		return nil, err
	}
	var resp datasetAPIResponse
	_, raw, err := c.FetchJSON(ctx, fmt.Sprintf("%s/api/v1/datasets/view/%s/%s", BaseURL, owner, slug), &resp)
	if err != nil {
		return nil, err
	}
	item := convertDataset(resp, raw)
	return &item, nil
}

func (c *Client) ListModels(ctx context.Context, search, nextPageToken string) ([]Model, string, error) {
	values := url.Values{}
	if search = strings.TrimSpace(search); search != "" {
		values.Set("search", search)
	}
	if nextPageToken = strings.TrimSpace(nextPageToken); nextPageToken != "" {
		values.Set("nextPageToken", nextPageToken)
	}
	var resp modelListResponse
	_, raw, err := c.FetchJSON(ctx, BaseURL+"/api/v1/models/list?"+values.Encode(), &resp)
	if err != nil {
		return nil, "", err
	}
	items := make([]Model, 0, len(resp.Models))
	for _, item := range resp.Models {
		items = append(items, convertModel(item, raw))
	}
	return items, resp.NextPageToken, nil
}

func (c *Client) FindModel(ctx context.Context, ref string) (*Model, error) {
	_, slug, err := splitOwnerSlug(ref)
	if err != nil {
		return nil, err
	}
	next := ""
	for i := 0; i < 10; i++ {
		items, token, err := c.ListModels(ctx, slug, next)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item.Ref == ref {
				return &item, nil
			}
		}
		if token == "" {
			break
		}
		next = token
	}
	return nil, fmt.Errorf("model not found: %s", ref)
}

func convertDataset(in datasetAPIResponse, raw []byte) Dataset {
	item := Dataset{
		ID:                   in.ID,
		Ref:                  in.Ref,
		OwnerRef:             in.OwnerRef,
		OwnerName:            in.OwnerName,
		CreatorName:          in.CreatorName,
		CreatorURL:           in.CreatorURL,
		Title:                in.Title,
		Subtitle:             in.Subtitle,
		Description:          in.Description,
		URL:                  firstNonEmpty(in.URL, NormalizeDatasetURL(in.Ref)),
		LicenseName:          in.LicenseName,
		ThumbnailImageURL:    in.ThumbnailImageURL,
		DownloadCount:        in.DownloadCount,
		ViewCount:            in.ViewCount,
		VoteCount:            in.VoteCount,
		KernelCount:          in.KernelCount,
		TopicCount:           in.TopicCount,
		CurrentVersionNumber: in.CurrentVersionNumber,
		UsabilityRating:      in.UsabilityRating,
		TotalBytes:           in.TotalBytes,
		IsPrivate:            in.IsPrivate,
		IsFeatured:           in.IsFeatured,
		Tags:                 in.Tags,
		RawJSON:              string(raw),
		FetchedAt:            time.Now(),
	}
	item.LastUpdated = parseTime(in.LastUpdated)
	if b, err := json.Marshal(in.Versions); err == nil {
		item.VersionsJSON = string(b)
	}
	for _, f := range in.Files {
		item.Files = append(item.Files, DatasetFile{
			DatasetRef:   in.Ref,
			Name:         f.Name,
			TotalBytes:   f.TotalBytes,
			CreationDate: f.CreationDate,
		})
	}
	return item
}

func convertModel(in modelAPI, raw []byte) Model {
	item := Model{
		ID:             in.ID,
		Ref:            in.Ref,
		OwnerRef:       strings.SplitN(in.Ref, "/", 2)[0],
		Title:          in.Title,
		Subtitle:       in.Subtitle,
		Description:    in.Description,
		Author:         in.Author,
		AuthorImageURL: in.AuthorImageURL,
		URL:            firstNonEmpty(in.URL, NormalizeModelURL(in.Ref)),
		VoteCount:      in.VoteCount,
		UpdateTime:     parseTime(in.UpdateTime),
		IsPrivate:      in.IsPrivate,
		Tags:           in.Tags,
		RawJSON:        string(raw),
		FetchedAt:      time.Now(),
	}
	for _, inst := range in.Instances {
		rawInst, _ := json.Marshal(inst)
		item.Instances = append(item.Instances, ModelInstance{
			ModelRef:               in.Ref,
			InstanceID:             inst.ID,
			Slug:                   inst.Slug,
			Framework:              inst.Framework,
			FineTunable:            inst.FineTunable,
			Overview:               inst.Overview,
			Usage:                  inst.Usage,
			DownloadURL:            absolutizePath(inst.DownloadURL),
			VersionID:              inst.VersionID,
			VersionNumber:          inst.VersionNumber,
			URL:                    inst.URL,
			LicenseName:            inst.LicenseName,
			ModelInstanceType:      inst.ModelInstanceType,
			ExternalBaseModelURL:   inst.ExternalBaseModelURL,
			TotalUncompressedBytes: inst.TotalUncompressedBytes,
			RawJSON:                string(rawInst),
		})
	}
	return item
}

func absolutizePath(p string) string {
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	if strings.HasPrefix(p, "/") {
		return BaseURL + p
	}
	return p
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.9999999Z",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
