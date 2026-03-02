//go:build tantivy

// Package tantivy provides an embedded FTS engine backed by Tantivy via CGO.
//
// # Build requirements
//
// This package requires a prebuilt libtantivy_go.a static library placed under:
//
//	libs/<GOOS>-<GOARCH>/libtantivy_go.a
//
// relative to this source file.  Pre-built archives for darwin-arm64 are
// included in the repository.  For other platforms, build from source:
//
//	cd $(go env GOPATH)/pkg/mod/github.com/anyproto/tantivy-go@v1.0.6/rust
//	make build-darwin-arm64   # or the target for your platform
//	cp target/aarch64-apple-darwin/release/libtantivy_go.a \
//	   <this-dir>/libs/darwin-arm64/
//
// Build with:
//
//	go test -tags tantivy ./pkg/index/driver/tantivy-go/...
package tantivy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	tantivy_go "github.com/anyproto/tantivy-go"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

// libInitOnce ensures tantivy_go.LibInit is called at most once per process.
var libInitOnce sync.Once

func init() {
	index.Register("tantivy", func() index.Engine { return &Engine{} })
}

const (
	fieldID   = "id"
	fieldBody = "body"
)

// Engine is an embedded FTS engine backed by Tantivy via CGO.
type Engine struct {
	tc  *tantivy_go.TantivyContext
	dir string
}

func (e *Engine) Name() string { return "tantivy" }

// Open initialises the Tantivy index at dir.
func (e *Engine) Open(_ context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("tantivy mkdir: %w", err)
	}
	e.dir = dir

	// Initialise the Rust logging layer once per process.
	var libInitErr error
	libInitOnce.Do(func() {
		libInitErr = tantivy_go.LibInit(false, true, "error")
	})
	if libInitErr != nil {
		return fmt.Errorf("tantivy LibInit: %w", libInitErr)
	}

	sb, err := tantivy_go.NewSchemaBuilder()
	if err != nil {
		return fmt.Errorf("tantivy NewSchemaBuilder: %w", err)
	}

	// id field: stored, raw tokenizer (exact match for deletes), not full-text
	if err := sb.AddTextField(
		fieldID,
		true,  // stored
		false, // isText — use string (raw) not text
		false, // isFast
		tantivy_go.IndexRecordOptionBasic,
		tantivy_go.TokenizerRaw,
	); err != nil {
		return fmt.Errorf("tantivy AddTextField(id): %w", err)
	}

	// body field: stored, English stemming, with frequencies+positions for BM25
	if err := sb.AddTextField(
		fieldBody,
		true,  // stored
		true,  // isText — full text
		false, // isFast
		tantivy_go.IndexRecordOptionWithFreqsAndPositions,
		tantivy_go.TokenizerSimple,
	); err != nil {
		return fmt.Errorf("tantivy AddTextField(body): %w", err)
	}

	schema, err := sb.BuildSchema()
	if err != nil {
		return fmt.Errorf("tantivy BuildSchema: %w", err)
	}

	idxDir := filepath.Join(dir, "tantivy-index")
	tc, err := tantivy_go.NewTantivyContextWithSchema(idxDir, schema)
	if err != nil {
		return fmt.Errorf("tantivy NewTantivyContextWithSchema: %w", err)
	}

	// Register the simple (English stemming) tokenizer with a generous token limit.
	const tokenLimit = uintptr(1 << 20)
	if err := tc.RegisterTextAnalyzerSimple(tantivy_go.TokenizerSimple, tokenLimit, tantivy_go.English); err != nil {
		_ = tc.Close()
		return fmt.Errorf("tantivy RegisterTextAnalyzerSimple: %w", err)
	}

	// Register the raw tokenizer used for the id field.
	if err := tc.RegisterTextAnalyzerRaw(tantivy_go.TokenizerRaw); err != nil {
		_ = tc.Close()
		return fmt.Errorf("tantivy RegisterTextAnalyzerRaw: %w", err)
	}

	e.tc = tc
	return nil
}

// Close releases all resources held by the Tantivy writer.
func (e *Engine) Close() error {
	if e.tc == nil {
		return nil
	}
	err := e.tc.Close()
	e.tc = nil
	return err
}

// Stats returns the current document count and on-disk size.
func (e *Engine) Stats(_ context.Context) (index.EngineStats, error) {
	if e.tc == nil {
		return index.EngineStats{}, nil
	}
	n, err := e.tc.NumDocs()
	if err != nil {
		return index.EngineStats{}, fmt.Errorf("tantivy NumDocs: %w", err)
	}
	return index.EngineStats{
		DocCount:  int64(n),
		DiskBytes: index.DirSizeBytes(e.dir),
	}, nil
}

// Index ingests a batch of documents into the Tantivy index.
//
// Existing documents with the same DocID are replaced (delete then add).
func (e *Engine) Index(_ context.Context, docs []index.Document) error {
	if len(docs) == 0 || e.tc == nil {
		return nil
	}

	tdocs := make([]*tantivy_go.Document, 0, len(docs))
	deleteIDs := make([]string, 0, len(docs))

	for _, d := range docs {
		deleteIDs = append(deleteIDs, d.DocID)

		td := tantivy_go.NewDocument()
		if err := td.AddField(d.DocID, e.tc, fieldID); err != nil {
			return fmt.Errorf("tantivy AddField(id) for %s: %w", d.DocID, err)
		}
		if err := td.AddField(string(d.Text), e.tc, fieldBody); err != nil {
			return fmt.Errorf("tantivy AddField(body) for %s: %w", d.DocID, err)
		}
		tdocs = append(tdocs, td)
	}

	// Batch delete-then-add in a single commit for efficiency.
	_, err := e.tc.BatchAddAndDeleteDocumentsWithOpstamp(tdocs, fieldID, deleteIDs)
	if err != nil {
		return fmt.Errorf("tantivy BatchAddAndDelete: %w", err)
	}
	return nil
}

// docResult is used to deserialise a Tantivy document JSON string.
type docResult struct {
	ID         string  `json:"id"`
	Body       string  `json:"body"`
	Score      float64 `json:"score"`
	Highlights []struct {
		FieldName string `json:"field_name"`
		Fragment  struct {
			T string `json:"t"`
		} `json:"fragment"`
	} `json:"highlights"`
}

// Search executes a full-text query and returns ranked hits.
func (e *Engine) Search(_ context.Context, q index.Query) (index.Results, error) {
	if e.tc == nil {
		return index.Results{}, nil
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	// Offset is not supported by the tantivy-go API; pagination requires re-querying.

	sCtx := tantivy_go.NewSearchContextBuilder().
		SetQuery(q.Text).
		SetDocsLimit(uintptr(limit)).
		SetWithHighlights(true).
		AddFieldDefaultWeight(fieldBody).
		Build()

	sr, err := e.tc.Search(sCtx)
	if err != nil {
		return index.Results{}, fmt.Errorf("tantivy Search: %w", err)
	}
	defer sr.Free()

	size, err := sr.GetSize()
	if err != nil {
		return index.Results{}, fmt.Errorf("tantivy GetSize: %w", err)
	}

	hits := make([]index.Hit, 0, size)
	for i := uint64(0); i < size; i++ {
		doc, err := sr.Get(i)
		if err != nil {
			return index.Results{}, fmt.Errorf("tantivy get doc %d: %w", i, err)
		}
		jsonStr, err := doc.ToJson(e.tc, fieldID, fieldBody)
		doc.Free()
		if err != nil {
			continue
		}

		var dr docResult
		if err := json.Unmarshal([]byte(jsonStr), &dr); err != nil {
			continue
		}

		h := index.Hit{
			DocID: dr.ID,
			Score: dr.Score,
		}
		// Extract the first highlight snippet if present.
		for _, hl := range dr.Highlights {
			if hl.FieldName == fieldBody && hl.Fragment.T != "" {
				h.Snippet = hl.Fragment.T
				break
			}
		}
		hits = append(hits, h)
	}

	return index.Results{
		Hits:  hits,
		Total: len(hits),
	}, nil
}

var _ index.Engine = (*Engine)(nil)
