package bleve

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	blevelib "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	index.Register("bleve", func() index.Engine { return &Engine{} })
}

// Engine is an embedded BM25 FTS engine backed by Bleve.
type Engine struct {
	idx blevelib.Index
	dir string
}

func (e *Engine) Name() string { return "bleve" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir
	dbPath := filepath.Join(dir, "bleve.db")

	idx, err := blevelib.Open(dbPath)
	if err != nil {
		// Index does not exist yet — create with English analyzer mapping
		mapping := blevelib.NewIndexMapping()
		docMapping := blevelib.NewDocumentMapping()

		textField := blevelib.NewTextFieldMapping()
		textField.Analyzer = en.AnalyzerName
		docMapping.AddFieldMappingsAt("Text", textField)

		mapping.DefaultMapping = docMapping

		idx, err = blevelib.New(dbPath, mapping)
		if err != nil {
			return fmt.Errorf("bleve create: %w", err)
		}
	}
	e.idx = idx
	return nil
}

func (e *Engine) Close() error {
	if e.idx == nil {
		return nil
	}
	return e.idx.Close()
}

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	if e.idx == nil {
		return index.EngineStats{}, nil
	}
	count, err := e.idx.DocCount()
	if err != nil {
		return index.EngineStats{}, err
	}
	return index.EngineStats{
		DocCount:  int64(count),
		DiskBytes: index.DirSizeBytes(e.dir),
	}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	b := e.idx.NewBatch()
	for _, doc := range docs {
		if err := b.Index(doc.DocID, struct{ Text string }{Text: string(doc.Text)}); err != nil {
			return fmt.Errorf("bleve batch index %s: %w", doc.DocID, err)
		}
	}
	return e.idx.Batch(b)
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	if e.idx == nil {
		return index.Results{}, nil
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	mq := blevelib.NewMatchQuery(q.Text)
	mq.SetField("Text")

	req := blevelib.NewSearchRequestOptions(mq, limit, q.Offset, false)
	req.Highlight = blevelib.NewHighlight()
	req.Highlight.AddField("Text")

	sr, err := e.idx.SearchInContext(ctx, req)
	if err != nil {
		return index.Results{}, fmt.Errorf("bleve search: %w", err)
	}

	results := index.Results{Total: int(sr.Total)}
	for _, hit := range sr.Hits {
		h := index.Hit{
			DocID: hit.ID,
			Score: hit.Score,
		}
		if frags, ok := hit.Fragments["Text"]; ok && len(frags) > 0 {
			h.Snippet = frags[0]
		}
		results.Hits = append(results.Hits, h)
	}
	return results, nil
}

var _ index.Engine = (*Engine)(nil)
