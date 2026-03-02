package qlocal

import (
	"context"
	"sync"
	"testing"
)

type fakeLLMBackend struct {
	mu          sync.Mutex
	embedCalls  int
	expandCalls int
	rerankCalls int
	lastExpand  string
}

func (f *fakeLLMBackend) Name() string { return "fake" }

func (f *fakeLLMBackend) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	f.mu.Lock()
	f.embedCalls++
	f.mu.Unlock()
	out := make([][]float32, len(texts))
	for i, s := range texts {
		out[i] = hashEmbed(s, 256)
	}
	_ = ctx
	return out, nil
}

func (f *fakeLLMBackend) ExpandQuery(ctx context.Context, query string) ([]StructuredSubSearch, error) {
	f.mu.Lock()
	f.expandCalls++
	f.lastExpand = query
	f.mu.Unlock()
	_ = ctx
	return []StructuredSubSearch{
		{Type: "lex", Query: "goroutine"},
		{Type: "vec", Query: "channels concurrency"},
	}, nil
}

func (f *fakeLLMBackend) expandState() (calls int, last string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.expandCalls, f.lastExpand
}

func (f *fakeLLMBackend) Rerank(ctx context.Context, query string, docs []RerankDoc) ([]float64, error) {
	f.mu.Lock()
	f.rerankCalls++
	f.mu.Unlock()
	_ = ctx
	_ = query
	out := make([]float64, len(docs))
	for i, d := range docs {
		if containsFold(d.Text, "goroutine") {
			out[i] = 0.95
		} else {
			out[i] = 0.05
		}
	}
	return out, nil
}

func containsFold(s, sub string) bool {
	return len(s) >= len(sub) && (indexFold(s, sub) >= 0)
}

func indexFold(s, sub string) int {
	ls := []rune(s)
	lsub := []rune(sub)
outer:
	for i := 0; i+len(lsub) <= len(ls); i++ {
		for j := range lsub {
			a := ls[i+j]
			b := lsub[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				continue outer
			}
		}
		return i
	}
	return -1
}

func TestQuery_UsesLLMExpansionAndRerank_WithCache(t *testing.T) {
	env := newTestEnv(t)
	env.writeFile(t, "kb/a.md", "# A\n\nGo routines and channels enable concurrency patterns.\n")
	env.writeFile(t, "kb/b.md", "# B\n\nThread pool sizing and locks discussion.\n")
	if _, err := env.App.CollectionAdd(env.RootDir+"/kb", "kb", "**/*.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Update(context.Background(), UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Embed(context.Background(), EmbedOptions{Force: true}); err != nil {
		t.Fatal(err)
	}

	fake := &fakeLLMBackend{}
	oldFactory := qlocalLLMFactory
	qlocalLLMFactory = func(context.Context) (LLMBackend, error) { return fake, nil }
	defer func() { qlocalLLMFactory = oldFactory }()

	res1, err := env.App.QueryContext(context.Background(), "concurrency in go", HybridOptions{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(res1) == 0 {
		t.Fatal("expected results")
	}
	if fake.expandCalls == 0 {
		t.Fatal("expected expansion call")
	}
	if fake.rerankCalls == 0 {
		t.Fatal("expected rerank call")
	}
	if res1[0].DisplayPath != "kb/a.md" {
		t.Fatalf("expected reranked top result kb/a.md, got %s", res1[0].DisplayPath)
	}

	// Repeat query should hit llm_cache for expand/rerank and not call fake again for those operations.
	prevExpand := fake.expandCalls
	prevRerank := fake.rerankCalls
	_, err = env.App.QueryContext(context.Background(), "concurrency in go", HybridOptions{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if fake.expandCalls != prevExpand {
		t.Fatalf("expected expansion cache hit, calls %d -> %d", prevExpand, fake.expandCalls)
	}
	if fake.rerankCalls != prevRerank {
		t.Fatalf("expected rerank cache hit, calls %d -> %d", prevRerank, fake.rerankCalls)
	}
}

func TestVectorSearch_UsesExpansionWithCache(t *testing.T) {
	env := newTestEnv(t)
	env.writeFile(t, "kb/a.md", "# A\n\nGo routines and channels enable concurrency patterns.\n")
	if _, err := env.App.CollectionAdd(env.RootDir+"/kb", "kb", "**/*.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Update(context.Background(), UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Embed(context.Background(), EmbedOptions{Force: true}); err != nil {
		t.Fatal(err)
	}

	fake := &fakeLLMBackend{}
	oldFactory := qlocalLLMFactory
	qlocalLLMFactory = func(context.Context) (LLMBackend, error) { return fake, nil }
	defer func() { qlocalLLMFactory = oldFactory }()

	res, err := env.App.VectorSearchContext(context.Background(), "concurrency", SearchOptions{Limit: 5, IncludeBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("expected vector search results")
	}
	if fake.expandCalls == 0 {
		t.Fatal("expected expansion call for vsearch")
	}
	prevExpand := fake.expandCalls
	_, err = env.App.VectorSearchContext(context.Background(), "concurrency", SearchOptions{Limit: 5, IncludeBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if fake.expandCalls != prevExpand {
		t.Fatalf("expected expansion cache hit, calls %d -> %d", prevExpand, fake.expandCalls)
	}
}

func TestQuery_ExplicitExpandPrefix_NormalizesBeforeExpansion(t *testing.T) {
	env := newTestEnv(t)
	env.writeFile(t, "kb/a.md", "# A\n\nGo routines and channels enable concurrency patterns.\n")
	if _, err := env.App.CollectionAdd(env.RootDir+"/kb", "kb", "**/*.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Update(context.Background(), UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Embed(context.Background(), EmbedOptions{Force: true}); err != nil {
		t.Fatal(err)
	}

	fake := &fakeLLMBackend{}
	oldFactory := qlocalLLMFactory
	qlocalLLMFactory = func(context.Context) (LLMBackend, error) { return fake, nil }
	defer func() { qlocalLLMFactory = oldFactory }()

	if _, err := env.App.QueryContext(context.Background(), "expand: concurrency in go", HybridOptions{Limit: 5}); err != nil {
		t.Fatal(err)
	}
	calls, last := fake.expandState()
	if calls == 0 {
		t.Fatal("expected expansion call")
	}
	if last != "concurrency in go" {
		t.Fatalf("expected explicit expand prefix to be stripped, got %q", last)
	}
}

func TestQuery_SingleLineTypedQuery_DoesNotUseExpansion(t *testing.T) {
	env := newTestEnv(t)
	env.writeFile(t, "kb/a.md", "# A\n\nRecursive descent parser and compiler notes.\n")
	if _, err := env.App.CollectionAdd(env.RootDir+"/kb", "kb", "**/*.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Update(context.Background(), UpdateOptions{}); err != nil {
		t.Fatal(err)
	}

	fake := &fakeLLMBackend{}
	oldFactory := qlocalLLMFactory
	qlocalLLMFactory = func(context.Context) (LLMBackend, error) { return fake, nil }
	defer func() { qlocalLLMFactory = oldFactory }()

	res, err := env.App.QueryContext(context.Background(), "LEX: parser", HybridOptions{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("expected results for single-line typed query")
	}
	calls, _ := fake.expandState()
	if calls != 0 {
		t.Fatalf("expected no expansion call for typed single-line query, got %d", calls)
	}
}
