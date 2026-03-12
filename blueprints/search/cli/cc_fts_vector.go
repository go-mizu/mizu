package cli

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/search/pkg/embed"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/embed/driver/llamacpp"
	"github.com/go-mizu/mizu/blueprints/search/pkg/vector"
	// Vector driver imports — register all drivers.
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/chroma"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/elasticsearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/meilisearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/milvus"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/opensearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/pgvector"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/qdrant"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/solr"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/typesense"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/weaviate"
)

func newCCFTSVector() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vector",
		Short: "Vector store operations: load embeddings and search",
		Long: `Load pre-computed embeddings into a vector store and run similarity search.

Workflow:
  1. Compute embeddings:  search cc fts embed run --input <dir>
  2. Load into store:     search cc fts vector load --input <embed-dir> --store qdrant
  3. Search:              search cc fts vector search "query" --store qdrant --driver llamacpp

Stores: ` + strings.Join(vector.List(), ", "),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newCCFTSVectorLoad())
	cmd.AddCommand(newCCFTSVectorSearch())
	cmd.AddCommand(newCCFTSVectorStores())
	return cmd
}

// ── vector load ─────────────────────────────────────────────────────────

func newCCFTSVectorLoad() *cobra.Command {
	var (
		input      string
		store      string
		addr       string
		collection string
		batchSize  int
	)

	cmd := &cobra.Command{
		Use:   "load",
		Short: "Load embeddings into a vector store",
		Long: `Read vectors.bin and meta.jsonl from an embed output directory and
index them into a vector store backend.

The input directory must contain:
  vectors.bin   Raw float32 vectors (N × dim × 4 bytes, little-endian)
  meta.jsonl    One JSON line per vector with id, file, chunk_idx, text_len, dim`,
		Example: `  # Load into Qdrant (default port 6333)
  search cc fts vector load --input ./embeddings/ --store qdrant

  # Load into pgvector with custom DSN
  search cc fts vector load --input ./embeddings/ --store pgvector \
    --addr "postgres://user:pass@localhost:5432/mydb?sslmode=disable"

  # Load into Elasticsearch
  search cc fts vector load --input ./embeddings/ --store elasticsearch --collection cc-vectors`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" {
				return fmt.Errorf("--input is required (path to embed output directory)")
			}
			return runVectorLoad(cmd.Context(), vectorLoadArgs{
				input: input, store: store, addr: addr,
				collection: collection, batchSize: batchSize,
			})
		},
	}

	cmd.Flags().StringVar(&input, "input", "", "Input directory containing vectors.bin + meta.jsonl (required)")
	cmd.Flags().StringVar(&store, "store", "qdrant", "Vector store driver: "+strings.Join(vector.List(), ", "))
	cmd.Flags().StringVar(&addr, "addr", "", "Store address/DSN (default: per driver)")
	cmd.Flags().StringVar(&collection, "collection", "cc-embed", "Collection/index name")
	cmd.Flags().IntVar(&batchSize, "batch-size", 500, "Items per indexing batch")
	return cmd
}

type vectorLoadArgs struct {
	input, store, addr, collection string
	batchSize                      int
}

func runVectorLoad(ctx context.Context, a vectorLoadArgs) error {
	vecPath := filepath.Join(a.input, "vectors.bin")
	metaPath := filepath.Join(a.input, "meta.jsonl")

	// Validate input files exist.
	for _, p := range []string{vecPath, metaPath} {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("missing %s — run 'search cc fts embed run' first", p)
		}
	}

	// Read all metadata to get count and dimension.
	metas, err := readMetaJSONL(metaPath)
	if err != nil {
		return fmt.Errorf("read meta: %w", err)
	}
	if len(metas) == 0 {
		return fmt.Errorf("meta.jsonl is empty")
	}
	dim := metas[0].Dim
	fmt.Fprintf(os.Stderr, "vector load: %d vectors, dim=%d, store=%s, collection=%s\n",
		len(metas), dim, a.store, a.collection)

	// Open vectors.bin for streaming reads.
	vecFile, err := os.Open(vecPath)
	if err != nil {
		return err
	}
	defer vecFile.Close()

	// Open vector store.
	cfg := vector.Config{Addr: a.addr}
	st, err := vector.Open(a.store, cfg)
	if err != nil {
		return fmt.Errorf("open store %q: %w", a.store, err)
	}
	if c, ok := st.(vector.Closer); ok {
		defer c.Close()
	}

	coll := st.Collection(a.collection)
	// Skip Init — Index() lazy-creates the collection with the correct
	// dimension from the first batch of vectors.

	// Stream vectors from disk and index in batches.
	start := time.Now()
	vecBuf := make([]byte, dim*4)
	var indexed int

	batch := make([]vector.Item, 0, a.batchSize)

	for i, m := range metas {
		// Read one vector (dim × 4 bytes).
		if _, err := io.ReadFull(vecFile, vecBuf); err != nil {
			return fmt.Errorf("read vector %d: %w", i, err)
		}

		vec := make([]float32, dim)
		for j := 0; j < dim; j++ {
			vec[j] = math.Float32frombits(binary.LittleEndian.Uint32(vecBuf[j*4:]))
		}

		batch = append(batch, vector.Item{
			ID:     m.ID,
			Vector: vec,
			Metadata: map[string]string{
				"_id":       m.ID,
				"file":      m.File,
				"chunk_idx": fmt.Sprintf("%d", m.ChunkIdx),
			},
		})

		if len(batch) >= a.batchSize {
			if err := coll.Index(ctx, batch); err != nil {
				return fmt.Errorf("index batch at %d: %w", i, err)
			}
			indexed += len(batch)
			batch = batch[:0]

			rate := float64(indexed) / time.Since(start).Seconds()
			fmt.Fprintf(os.Stderr, "\r  indexed %d / %d (%.0f items/s)", indexed, len(metas), rate)
		}
	}

	// Flush remaining.
	if len(batch) > 0 {
		if err := coll.Index(ctx, batch); err != nil {
			return fmt.Errorf("index final batch: %w", err)
		}
		indexed += len(batch)
	}

	elapsed := time.Since(start)
	rate := float64(indexed) / elapsed.Seconds()
	fmt.Fprintf(os.Stderr, "\r  indexed %d / %d (%.0f items/s) elapsed=%s\n",
		indexed, len(metas), rate, elapsed.Round(time.Millisecond))

	return nil
}

// readMetaJSONL reads all embedMeta entries from a meta.jsonl file.
func readMetaJSONL(path string) ([]embedMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var metas []embedMeta
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 256*1024), 256*1024)
	for sc.Scan() {
		var m embedMeta
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			return nil, fmt.Errorf("parse line %d: %w", len(metas)+1, err)
		}
		metas = append(metas, m)
	}
	return metas, sc.Err()
}

// ── vector search ───────────────────────────────────────────────────────

func newCCFTSVectorSearch() *cobra.Command {
	var (
		store      string
		addr       string
		collection string
		driver     string
		driverAddr string
		model      string
		modelDir   string
		k          int
	)

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Vector similarity search",
		Long: `Embed a query string and search for similar vectors in a store.

Requires an embedding driver (llamacpp or onnx) to encode the query,
and a vector store backend with previously loaded embeddings.`,
		Example: `  # Search Qdrant with llamacpp embeddings
  search cc fts vector search "machine learning" --store qdrant --driver llamacpp

  # Search pgvector with custom settings
  search cc fts vector search "neural networks" --store pgvector \
    --addr "postgres://localhost/mydb" --driver onnx -k 20`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVectorSearch(cmd.Context(), vectorSearchArgs{
				query: args[0], store: store, addr: addr, collection: collection,
				driver: driver, driverAddr: driverAddr, model: model,
				modelDir: modelDir, k: k,
			})
		},
	}

	cmd.Flags().StringVar(&store, "store", "qdrant", "Vector store driver: "+strings.Join(vector.List(), ", "))
	cmd.Flags().StringVar(&addr, "addr", "", "Store address/DSN (default: per driver)")
	cmd.Flags().StringVar(&collection, "collection", "cc-embed", "Collection name")
	cmd.Flags().StringVar(&driver, "driver", "llamacpp", "Embedding driver for query encoding")
	cmd.Flags().StringVar(&driverAddr, "driver-addr", "", "Embedding driver address (default: per driver)")
	cmd.Flags().StringVar(&model, "model", "", "Embedding model name")
	cmd.Flags().StringVar(&modelDir, "model-dir", "", "Model directory (default ~/data/models)")
	cmd.Flags().IntVarP(&k, "k", "k", 10, "Number of results")
	return cmd
}

type vectorSearchArgs struct {
	query, store, addr, collection string
	driver, driverAddr, model      string
	modelDir                       string
	k                              int
}

func runVectorSearch(ctx context.Context, a vectorSearchArgs) error {
	if a.modelDir == "" {
		a.modelDir = embed.DefaultModelDir()
	}

	// Open embedding driver for query encoding.
	drv, err := embed.New(a.driver)
	if err != nil {
		return fmt.Errorf("open embed driver: %w", err)
	}
	cfg := embed.Config{
		Addr:  a.driverAddr,
		Model: a.model,
		Dir:   embed.ModelFilesDir(a.modelDir, embed.ModelInfo{Driver: a.driver, Name: a.model}),
	}
	if err := drv.Open(ctx, cfg); err != nil {
		return fmt.Errorf("open embed driver: %w", err)
	}
	defer drv.Close()

	// Embed the query.
	vecs, err := drv.Embed(ctx, []embed.Input{{Text: a.query}})
	if err != nil {
		return fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) == 0 {
		return fmt.Errorf("embed returned no vectors")
	}

	// Open vector store.
	stCfg := vector.Config{Addr: a.addr}
	st, err := vector.Open(a.store, stCfg)
	if err != nil {
		return fmt.Errorf("open store %q: %w", a.store, err)
	}
	if c, ok := st.(vector.Closer); ok {
		defer c.Close()
	}

	coll := st.Collection(a.collection)

	// Search.
	results, err := coll.Search(ctx, vector.Query{
		Vector: vecs[0].Values,
		K:      a.k,
	})
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	// Display results.
	if len(results.Hits) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	fmt.Fprintf(os.Stderr, "query: %q → %d hits (dim=%d)\n\n", a.query, len(results.Hits), drv.Dimension())

	for i, hit := range results.Hits {
		displayID := hit.ID
		if origID, ok := hit.Metadata["_id"]; ok {
			displayID = origID
		}
		extra := ""
		if f, ok := hit.Metadata["file"]; ok && f != "" {
			extra = "  file=" + f
		}
		fmt.Printf("%2d. [%.4f] %s%s\n", i+1, hit.Score, displayID, extra)
	}

	return nil
}

// ── vector stores ───────────────────────────────────────────────────────

func newCCFTSVectorStores() *cobra.Command {
	return &cobra.Command{
		Use:   "stores",
		Short: "List available vector store drivers",
		RunE: func(cmd *cobra.Command, args []string) error {
			stores := vector.List()
			fmt.Fprintf(os.Stderr, "Available vector stores (%d):\n", len(stores))
			for _, s := range stores {
				fmt.Fprintf(os.Stderr, "  %s\n", s)
			}
			return nil
		},
	}
}
