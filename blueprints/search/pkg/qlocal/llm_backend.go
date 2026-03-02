package qlocal

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/llm/llamacpp"
)

type RerankDoc struct {
	File string
	Text string
}

type LLMBackend interface {
	Name() string
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	ExpandQuery(ctx context.Context, query string) ([]StructuredSubSearch, error)
	Rerank(ctx context.Context, query string, docs []RerankDoc) ([]float64, error)
}

type llmFactory func(context.Context) (LLMBackend, error)

var qlocalLLMFactory llmFactory = defaultLLMFactory

func defaultLLMFactory(ctx context.Context) (LLMBackend, error) {
	_ = ctx
	if disable := strings.TrimSpace(os.Getenv("QLOCAL_DISABLE_LLM")); disable == "1" || strings.EqualFold(disable, "true") {
		return nil, nil
	}
	provider := strings.TrimSpace(os.Getenv("QLOCAL_LLM_PROVIDER"))
	if provider == "" {
		if strings.TrimSpace(os.Getenv("QLOCAL_LLAMACPP_URL")) == "" {
			return nil, nil
		}
		provider = "llamacpp"
	}
	switch provider {
	case "llamacpp":
		baseURL := strings.TrimSpace(os.Getenv("QLOCAL_LLAMACPP_URL"))
		if baseURL == "" {
			baseURL = "http://localhost:8080"
		}
		timeoutSec := envInt("QLOCAL_LLM_TIMEOUT_SEC", 120)
		p, err := llm.New("llamacpp", llm.Config{
			BaseURL: baseURL,
			Timeout: timeoutSec,
		})
		if err != nil {
			return nil, err
		}
		return &llamaProvider{
			provider:    p,
			embedModel:  strings.TrimSpace(os.Getenv("QLOCAL_EMBED_MODEL")),
			expandModel: strings.TrimSpace(os.Getenv("QLOCAL_EXPAND_MODEL")),
			rerankModel: strings.TrimSpace(os.Getenv("QLOCAL_RERANK_MODEL")),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported QLOCAL_LLM_PROVIDER: %s", provider)
	}
}

func envInt(key string, fallback int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

type llamaProvider struct {
	provider    llm.Provider
	embedModel  string
	expandModel string
	rerankModel string
}

func (p *llamaProvider) Name() string { return p.provider.Name() }

func (p *llamaProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	req := llm.EmbeddingRequest{
		Model: p.embedModel,
		Input: texts,
	}
	resp, err := p.provider.Embedding(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make([][]float32, len(resp.Data))
	for i := range resp.Data {
		out[i] = resp.Data[i].Embedding
	}
	return out, nil
}

func (p *llamaProvider) ExpandQuery(ctx context.Context, query string) ([]StructuredSubSearch, error) {
	// Ask model for typed subqueries in strict JSON to mimic qmd behavior without requiring qmd's exact fine-tuned model.
	prompt := strings.TrimSpace(`
Return a JSON array of 1 to 3 objects for local search query expansion.
Each object must have:
- "type": one of "lex", "vec", "hyde"
- "query": string

Rules:
- Preserve the user's intent.
- Include at most one hyde query.
- Keep lex query keyword-oriented; vec/hyde natural language.
- Output JSON only, no markdown fences.
`)
	req := llm.ChatRequest{
		Model: p.expandModel,
		Messages: []llm.Message{
			{Role: "system", Content: prompt},
			{Role: "user", Content: query},
		},
		Temperature: 0.2,
		MaxTokens:   256,
	}
	resp, err := p.provider.ChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, nil
	}
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	return parseExpansionJSON(content)
}

func parseExpansionJSON(content string) ([]StructuredSubSearch, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	type item struct {
		Type  string `json:"type"`
		Query string `json:"query"`
	}
	var arr []item
	if err := json.Unmarshal([]byte(content), &arr); err != nil {
		// Try extracting first JSON array from noisy output.
		re := regexp.MustCompile(`(?s)\[[\s\S]*\]`)
		if m := re.FindString(content); m != "" {
			if err2 := json.Unmarshal([]byte(m), &arr); err2 != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	var out []StructuredSubSearch
	seen := map[string]struct{}{}
	for _, it := range arr {
		typ := strings.TrimSpace(it.Type)
		q := strings.TrimSpace(it.Query)
		if q == "" {
			continue
		}
		switch typ {
		case "lex", "vec", "hyde":
		default:
			continue
		}
		key := typ + "\x00" + q
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, StructuredSubSearch{Type: typ, Query: q})
		if len(out) >= 3 {
			break
		}
	}
	return out, nil
}

func (p *llamaProvider) Rerank(ctx context.Context, query string, docs []RerankDoc) ([]float64, error) {
	if len(docs) == 0 {
		return nil, nil
	}
	// Batched JSON scoring for latency. Falls back to per-doc zero score on parse issues.
	type payloadDoc struct {
		Index int    `json:"index"`
		Text  string `json:"text"`
	}
	plist := make([]payloadDoc, 0, len(docs))
	for i, d := range docs {
		plist = append(plist, payloadDoc{Index: i, Text: d.Text})
	}
	inputJSON, _ := json.Marshal(plist)
	prompt := strings.TrimSpace(`
Score each document chunk for relevance to the user query.
Return JSON array with objects: {"index": number, "score": number}
score range must be 0.0 to 1.0.
Output JSON only.
`)
	req := llm.ChatRequest{
		Model: p.rerankModel,
		Messages: []llm.Message{
			{Role: "system", Content: prompt},
			{Role: "user", Content: "QUERY:\n" + query + "\n\nDOCS:\n" + string(inputJSON)},
		},
		Temperature: 0,
		MaxTokens:   512,
	}
	resp, err := p.provider.ChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}
	scores := make([]float64, len(docs))
	if len(resp.Choices) == 0 {
		return scores, nil
	}
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	type item struct {
		Index int     `json:"index"`
		Score float64 `json:"score"`
	}
	var arr []item
	if err := json.Unmarshal([]byte(content), &arr); err != nil {
		re := regexp.MustCompile(`(?s)\[[\s\S]*\]`)
		m := re.FindString(content)
		if m == "" || json.Unmarshal([]byte(m), &arr) != nil {
			return scores, nil
		}
	}
	for _, it := range arr {
		if it.Index < 0 || it.Index >= len(scores) {
			continue
		}
		if it.Score < 0 {
			it.Score = 0
		}
		if it.Score > 1 {
			it.Score = 1
		}
		scores[it.Index] = it.Score
	}
	return scores, nil
}

func (a *App) llmBackend(ctx context.Context) (LLMBackend, error) {
	return qlocalLLMFactory(ctx)
}

func cacheKey(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (a *App) llmCacheGet(ctx context.Context, key string) (string, bool, error) {
	var v string
	err := a.DB.QueryRowContext(ctx, `SELECT result FROM llm_cache WHERE hash=?`, key).Scan(&v)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, err
	}
	return v, true, nil
}

func (a *App) llmCacheSet(ctx context.Context, key string, value string) error {
	_, err := a.DB.ExecContext(ctx, `
		INSERT OR REPLACE INTO llm_cache(hash, result, created_at) VALUES(?,?,?)
	`, key, value, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (a *App) embedTextsWithBackend(ctx context.Context, texts []string) ([][]float32, bool, error) {
	backend, err := a.llmBackend(ctx)
	if err != nil || backend == nil {
		return nil, false, err
	}
	vecs, err := backend.EmbedBatch(ctx, texts)
	if err != nil {
		return nil, false, err
	}
	return vecs, true, nil
}

func (a *App) expandQueryCached(ctx context.Context, query string) ([]StructuredSubSearch, bool, error) {
	backend, err := a.llmBackend(ctx)
	if err != nil || backend == nil {
		return nil, false, err
	}
	key := cacheKey("expand", backend.Name(), query)
	if cached, ok, err := a.llmCacheGet(ctx, key); err == nil && ok {
		var out []StructuredSubSearch
		if json.Unmarshal([]byte(cached), &out) == nil {
			return out, true, nil
		}
	}
	out, err := backend.ExpandQuery(ctx, query)
	if err != nil {
		return nil, false, err
	}
	if b, err := json.Marshal(out); err == nil {
		_ = a.llmCacheSet(ctx, key, string(b))
	}
	return out, true, nil
}

func (a *App) rerankCached(ctx context.Context, query string, docs []RerankDoc) ([]float64, bool, error) {
	backend, err := a.llmBackend(ctx)
	if err != nil || backend == nil {
		return nil, false, err
	}
	scores := make([]float64, len(docs))
	var missingDocs []RerankDoc
	var missingIdx []int
	for i, d := range docs {
		key := cacheKey("rerank", backend.Name(), query, d.File, d.Text)
		if cached, ok, err := a.llmCacheGet(ctx, key); err == nil && ok {
			if v, err := strconv.ParseFloat(cached, 64); err == nil {
				scores[i] = v
				continue
			}
		}
		missingDocs = append(missingDocs, d)
		missingIdx = append(missingIdx, i)
	}
	if len(missingDocs) > 0 {
		r, err := backend.Rerank(ctx, query, missingDocs)
		if err != nil {
			return nil, false, err
		}
		for j, v := range r {
			if j >= len(missingIdx) {
				break
			}
			i := missingIdx[j]
			scores[i] = v
			key := cacheKey("rerank", backend.Name(), query, docs[i].File, docs[i].Text)
			_ = a.llmCacheSet(ctx, key, strconv.FormatFloat(v, 'f', 6, 64))
		}
	}
	return scores, true, nil
}
