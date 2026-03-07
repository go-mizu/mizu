package cli

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"sync"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	kgzip "github.com/klauspost/compress/gzip"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/embed"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/embed/driver/llamacpp"
)

func newCCFTSEmbed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "embed",
		Short: "Compute vector embeddings from markdown files",
		Long: `Generate vector embeddings from markdown files using llamacpp or onnx drivers.

Workflow:
  1. Download a model:    search cc fts embed download --driver llamacpp
  2. Start the server:    cd docker/llamacpp && docker compose up llamacpp-embed -d
  3. Compute embeddings:  search cc fts embed --input ~/data/common-crawl/.../markdown/00000

Drivers:
  llamacpp   HTTP client to a running llama.cpp server (default port 8086)
             Requires: llama.cpp server started with --embedding --pooling mean
             Models are GGUF files stored in ~/data/models/

  onnx       Local ONNX Runtime inference (build with -tags onnx)
             Requires: ONNX Runtime library (brew install onnxruntime)
             Models auto-download to ~/data/models/onnx/

Output:
  vectors.bin   Raw float32 vectors (N x dim x 4 bytes, little-endian)
  meta.jsonl    One JSON line per vector: {id, file, chunk_idx, text_len, dim}
  stats.json    Summary: {files, chunks, errors, dim, driver, elapsed_ms}`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newCCFTSEmbedRun())
	cmd.AddCommand(newCCFTSEmbedDownload())
	cmd.AddCommand(newCCFTSEmbedModels())
	return cmd
}

// ── embed run ──────────────────────────────────────────────────────────

func newCCFTSEmbedRun() *cobra.Command {
	var (
		crawlID      string
		fileIdx      string
		driver       string
		addr         string
		model        string
		batchSize    int
		embedWorkers int
		fileWorkers  int
		maxChars     int
		overlap      int
		modelDir     string
		input        string
		output       string
		download     bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Compute embeddings from markdown files",
		Example: `  # Embed from a local directory using llamacpp
  search cc fts embed run --input ~/data/common-crawl/CC-MAIN-2026-08/markdown/00000 --driver llamacpp

  # Embed using ONNX (auto-downloads model, needs -tags onnx build)
  search cc fts embed run --input ./my-docs/ --driver onnx

  # Embed CC data by WARC index (fetches manifest from CC API)
  search cc fts embed run --file 0 --driver llamacpp

  # Auto-download model before embedding
  search cc fts embed run --input ./docs/ --driver llamacpp --download

  # Custom output directory
  search cc fts embed run --input ./docs/ --output ./embeddings/ --driver llamacpp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" && !cmd.Flags().Changed("file") && !cmd.Flags().Changed("crawl") {
				fmt.Fprintln(os.Stderr, "error: specify --input <dir> or --file <idx>")
				fmt.Fprintln(os.Stderr)
				return cmd.Help()
			}
			return runCCFTSEmbedRun(cmd.Context(), embedRunArgs{
				crawlID: crawlID, fileIdx: fileIdx, driver: driver, addr: addr,
				model: model, batchSize: batchSize, embedWorkers: embedWorkers,
				fileWorkers: fileWorkers, maxChars: maxChars, overlap: overlap,
				modelDir: modelDir, input: input, output: output, download: download,
			})
		},
	}

	cmd.Flags().StringVar(&input, "input", "", "Input markdown directory (bypasses CC pipeline)")
	cmd.Flags().StringVar(&output, "output", "", "Output directory (default: auto)")
	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "0", "WARC file index or range (0-9)")
	cmd.Flags().StringVar(&driver, "driver", "llamacpp", "Embedding driver: "+strings.Join(embed.List(), ", "))
	cmd.Flags().StringVar(&addr, "addr", "", "Server address (for llamacpp, default http://localhost:8086)")
	cmd.Flags().StringVar(&model, "model", "", "Model name (default: auto per driver)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 64, "Inputs per embedding batch")
	cmd.Flags().IntVar(&embedWorkers, "embed-workers", 4, "Concurrent embedding workers (parallel HTTP calls)")
	cmd.Flags().IntVar(&fileWorkers, "file-workers", 0, "Parallel file readers (default: NumCPU)")
	cmd.Flags().IntVar(&maxChars, "max-chars", 500, "Max characters per text chunk (keep under model context limit)")
	cmd.Flags().IntVar(&overlap, "overlap", 200, "Chunk overlap in characters")
	cmd.Flags().StringVar(&modelDir, "model-dir", "", "Model storage directory (default ~/data/models)")
	cmd.Flags().BoolVar(&download, "download", false, "Download model before embedding if missing")
	return cmd
}

type embedRunArgs struct {
	crawlID, fileIdx, driver, addr, model    string
	batchSize, embedWorkers, fileWorkers     int
	maxChars, overlap                        int
	modelDir, input, output                  string
	download                                 bool
}

func runCCFTSEmbedRun(ctx context.Context, a embedRunArgs) error {
	if a.fileWorkers <= 0 {
		a.fileWorkers = runtime.NumCPU()
	}
	if a.embedWorkers <= 0 {
		a.embedWorkers = 4
	}
	if a.modelDir == "" {
		a.modelDir = embed.DefaultModelDir()
	}

	driverName := a.driver

	// Auto-download model if requested.
	if a.download {
		modelName := a.model
		if modelName == "" {
			modelName = embed.DefaultModelName(driverName)
		}
		m, ok := embed.FindModel(driverName, modelName)
		if !ok {
			return fmt.Errorf("unknown model %q for driver %q — run: search cc fts embed models", modelName, driverName)
		}
		if !embed.IsModelDownloaded(a.modelDir, m) {
			fmt.Fprintf(os.Stderr, "downloading model: %s (%s, ~%dMB)\n", m.Name, m.Driver, m.SizeMB)
			if _, err := embed.DownloadModel(a.modelDir, m); err != nil {
				return fmt.Errorf("download model: %w", err)
			}
			fmt.Fprintln(os.Stderr)
		}
	}

	// Open embedding driver.
	drv, err := embed.New(driverName)
	if err != nil {
		return err
	}
	cfg := embed.Config{
		Addr:      a.addr,
		Model:     a.model,
		BatchSize: a.batchSize,
		Dir:       embed.ModelFilesDir(a.modelDir, embed.ModelInfo{Driver: driverName, Name: a.model}),
	}
	if err := drv.Open(ctx, cfg); err != nil {
		return fmt.Errorf("open driver: %w", err)
	}
	defer drv.Close()

	dim := drv.Dimension()
	fmt.Fprintf(os.Stderr, "embed: driver=%s dim=%d batch=%d\n", drv.Name(), dim, a.batchSize)

	// Determine input directories and output paths.
	type embedJob struct {
		mdDir     string
		outputDir string
		label     string
	}
	var jobs []embedJob

	if a.input != "" {
		// Direct input mode — embed from a specific directory.
		mdDir := a.input
		if _, err := os.Stat(mdDir); os.IsNotExist(err) {
			return fmt.Errorf("input directory not found: %s", mdDir)
		}
		outDir := a.output
		if outDir == "" {
			outDir = filepath.Join(mdDir, "embed", driverName)
		}
		jobs = append(jobs, embedJob{mdDir: mdDir, outputDir: outDir, label: filepath.Base(mdDir)})
	} else {
		// CC pipeline mode — resolve WARC paths.
		crawlID := a.crawlID
		if crawlID == "" {
			crawlID = detectLatestCrawl()
		}
		homeDir, _ := os.UserHomeDir()
		baseDir := filepath.Join(homeDir, "data", "common-crawl", crawlID)

		paths, err := listCCWARCPaths(ctx, crawlID)
		if err != nil {
			return fmt.Errorf("manifest: %w", err)
		}
		selected, err := ccParseFileSelector(a.fileIdx, len(paths))
		if err != nil {
			return fmt.Errorf("--file: %w", err)
		}

		for _, idx := range selected {
			warcIdx := warcIndexFromPath(paths[idx], idx)
			mdDir := filepath.Join(baseDir, "markdown", warcIdx)
			if _, err := os.Stat(mdDir); os.IsNotExist(err) {
				return fmt.Errorf("markdown dir not found: %s", mdDir)
			}
			outDir := filepath.Join(baseDir, "embed", driverName, warcIdx)
			jobs = append(jobs, embedJob{mdDir: mdDir, outputDir: outDir, label: warcIdx})
		}
	}

	for _, job := range jobs {
		fmt.Fprintf(os.Stderr, "embed: %s → %s\n", job.mdDir, job.outputDir)
		if err := os.MkdirAll(job.outputDir, 0o755); err != nil {
			return err
		}
		if err := embedDir(ctx, drv, job.mdDir, job.outputDir, dim, a.batchSize, a.embedWorkers, a.fileWorkers, a.maxChars, a.overlap); err != nil {
			return fmt.Errorf("embed %s: %w", job.label, err)
		}
	}
	return nil
}

// ── embed download ─────────────────────────────────────────────────────

func newCCFTSEmbedDownload() *cobra.Command {
	var (
		driver   string
		model    string
		modelDir string
	)

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download embedding models",
		Long: `Download embedding model files to the local model directory.

For llamacpp: downloads GGUF files to ~/data/models/ (docker volume mount).
For onnx:     downloads ONNX model + vocab to ~/data/models/onnx/<model>/.

After downloading a llamacpp model, start the server:
  cd docker/llamacpp && docker compose up llamacpp-embed -d

Or manually:
  llama-server --model ~/data/models/<model>.gguf --embedding --pooling mean --port 8086`,
		Example: `  search cc fts embed download                          # download default llamacpp model
  search cc fts embed download --driver onnx             # download default ONNX model
  search cc fts embed download --driver llamacpp --model bge-small-en-v1.5
  search cc fts embed download --model-dir /tmp/models`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCFTSEmbedDownload(driver, model, modelDir)
		},
	}

	cmd.Flags().StringVar(&driver, "driver", "llamacpp", "Driver: llamacpp, onnx")
	cmd.Flags().StringVar(&model, "model", "", "Model name (default: auto per driver)")
	cmd.Flags().StringVar(&modelDir, "model-dir", "", "Model directory (default ~/data/models)")
	return cmd
}

func runCCFTSEmbedDownload(driver, modelName, modelDir string) error {
	if modelDir == "" {
		modelDir = embed.DefaultModelDir()
	}
	if modelName == "" {
		modelName = embed.DefaultModelName(driver)
	}

	m, ok := embed.FindModel(driver, modelName)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown model %q for driver %q\n\nAvailable models:\n", modelName, driver)
		for _, m := range embed.ListModels(driver) {
			fmt.Fprintf(os.Stderr, "  %-30s %4d-dim  %4dMB  %s\n", m.Name, m.Dim, m.SizeMB, m.Desc)
		}
		return fmt.Errorf("model not found")
	}

	if embed.IsModelDownloaded(modelDir, m) {
		dir := embed.ModelFilesDir(modelDir, m)
		fmt.Fprintf(os.Stderr, "model %q already downloaded at %s\n", m.Name, dir)
		printPostDownloadHelp(m)
		return nil
	}

	fmt.Fprintf(os.Stderr, "downloading: %s (%s, ~%dMB)\n", m.Name, m.Desc, m.SizeMB)
	dir, err := embed.DownloadModel(modelDir, m)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "\nmodel saved to: %s\n", dir)
	printPostDownloadHelp(m)
	return nil
}

func printPostDownloadHelp(m embed.ModelInfo) {
	switch m.Driver {
	case "llamacpp":
		gguf := m.GGUFFileName()
		fmt.Fprintf(os.Stderr, "\nTo start the llama.cpp server:\n")
		fmt.Fprintf(os.Stderr, "  cd docker/llamacpp && docker compose up llamacpp-embed -d\n")
		fmt.Fprintf(os.Stderr, "\nOr manually:\n")
		fmt.Fprintf(os.Stderr, "  llama-server --model ~/data/models/%s --embedding --pooling mean --port 8086\n", gguf)
		fmt.Fprintf(os.Stderr, "\nThen embed:\n")
		fmt.Fprintf(os.Stderr, "  search cc fts embed run --input <dir> --driver llamacpp\n")
	case "onnx":
		fmt.Fprintf(os.Stderr, "\nTo embed (requires ONNX Runtime: brew install onnxruntime):\n")
		fmt.Fprintf(os.Stderr, "  search cc fts embed run --input <dir> --driver onnx\n")
	}
}

// ── embed models ───────────────────────────────────────────────────────

func newCCFTSEmbedModels() *cobra.Command {
	var (
		driver   string
		modelDir string
	)

	cmd := &cobra.Command{
		Use:     "models",
		Short:   "List available embedding models",
		Example: `  search cc fts embed models
  search cc fts embed models --driver onnx`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCFTSEmbedModels(driver, modelDir)
		},
	}

	cmd.Flags().StringVar(&driver, "driver", "", "Filter by driver (llamacpp, onnx)")
	cmd.Flags().StringVar(&modelDir, "model-dir", "", "Model directory (default ~/data/models)")
	return cmd
}

func runCCFTSEmbedModels(driver, modelDir string) error {
	if modelDir == "" {
		modelDir = embed.DefaultModelDir()
	}

	models := embed.ListModels(driver)
	if len(models) == 0 {
		fmt.Fprintf(os.Stderr, "no models found for driver %q\n", driver)
		return nil
	}

	fmt.Fprintf(os.Stderr, "%-10s %-30s %6s %6s  %-8s  %s\n",
		"DRIVER", "MODEL", "DIM", "SIZE", "STATUS", "DESCRIPTION")
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("─", 100))

	for _, m := range models {
		status := "missing"
		if embed.IsModelDownloaded(modelDir, m) {
			status = "ready"
		}
		fmt.Fprintf(os.Stderr, "%-10s %-30s %6d %4dMB  %-8s  %s\n",
			m.Driver, m.Name, m.Dim, m.SizeMB, status, m.Desc)
	}
	return nil
}

// ── embed pipeline ─────────────────────────────────────────────────────

// embedMeta is one line of meta.jsonl.
type embedMeta struct {
	ID       string `json:"id"`
	File     string `json:"file"`
	ChunkIdx int    `json:"chunk_idx"`
	TextLen  int    `json:"text_len"`
	Dim      int    `json:"dim"`
}

type chunkItem struct {
	file     string
	chunkIdx int
	text     string
}

// embedResult is one completed batch ready for writing.
type embedResult struct {
	items []chunkItem
	vecs  []embed.Vector
}

func embedDir(ctx context.Context, drv embed.Driver, mdDir, outputDir string, dim, batchSize, embedWorkers, fileWorkers, maxChars, overlap int) error {
	start := time.Now()

	// Collect markdown files (.md.gz and .md).
	var files []string
	filepath.WalkDir(mdDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".md.gz") || strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return nil
	})

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "  no markdown files in %s\n", mdDir)
		return nil
	}
	fmt.Fprintf(os.Stderr, "  found %d markdown files, embed-workers=%d\n", len(files), embedWorkers)

	vecPath := filepath.Join(outputDir, "vectors.bin")
	metaPath := filepath.Join(outputDir, "meta.jsonl")

	vecFile, err := os.Create(vecPath)
	if err != nil {
		return err
	}
	defer vecFile.Close()

	metaFile, err := os.Create(metaPath)
	if err != nil {
		return err
	}
	defer metaFile.Close()

	var (
		metaEnc    = json.NewEncoder(metaFile)
		totalFiles atomic.Int64
		totalChunks atomic.Int64
		totalVecs  atomic.Int64
		totalErrs  atomic.Int64
	)

	// Pipeline channels.
	chunkCh  := make(chan chunkItem, batchSize*embedWorkers*2)
	batchCh  := make(chan []chunkItem, embedWorkers*2)
	resultCh := make(chan embedResult, embedWorkers*2)

	g, gctx := errgroup.WithContext(ctx)

	// ── Stage 1: Read files → chunkCh ──
	g.Go(func() error {
		defer close(chunkCh)
		rg, rgctx := errgroup.WithContext(gctx)
		rg.SetLimit(fileWorkers)

		for _, f := range files {
			path := f
			rg.Go(func() error {
				text, err := readMarkdownFile(path)
				if err != nil {
					totalErrs.Add(1)
					return nil
				}
				totalFiles.Add(1)

				cs := embed.ChunkText(text, maxChars, overlap)
				rel, _ := filepath.Rel(mdDir, path)
				for i, c := range cs {
					select {
					case chunkCh <- chunkItem{file: rel, chunkIdx: i, text: c}:
						totalChunks.Add(1)
					case <-rgctx.Done():
						return rgctx.Err()
					}
				}
				return nil
			})
		}
		return rg.Wait()
	})

	// ── Stage 2: Batcher — chunkCh → batchCh ──
	g.Go(func() error {
		defer close(batchCh)
		batch := make([]chunkItem, 0, batchSize)
		for item := range chunkCh {
			batch = append(batch, item)
			if len(batch) >= batchSize {
				// Copy to avoid race — hand off ownership.
				out := make([]chunkItem, len(batch))
				copy(out, batch)
				select {
				case batchCh <- out:
				case <-gctx.Done():
					return gctx.Err()
				}
				batch = batch[:0]
			}
		}
		// Flush remaining.
		if len(batch) > 0 {
			select {
			case batchCh <- batch:
			case <-gctx.Done():
				return gctx.Err()
			}
		}
		return nil
	})

	// ── Stage 3: Embed workers — batchCh → resultCh ──
	g.Go(func() error {
		defer close(resultCh)
		eg, egctx := errgroup.WithContext(gctx)
		eg.SetLimit(embedWorkers)

		for batch := range batchCh {
			batch := batch
			eg.Go(func() error {
				inputs := make([]embed.Input, len(batch))
				for i, c := range batch {
					inputs[i] = embed.Input{Text: c.text}
				}

				vecs, err := drv.Embed(egctx, inputs)
				if err != nil {
					// Batch failed — retry one-by-one to isolate bad inputs.
					for _, c := range batch {
						sv, serr := drv.Embed(egctx, []embed.Input{{Text: c.text}})
						if serr != nil {
							totalErrs.Add(1)
							continue
						}
						select {
						case resultCh <- embedResult{items: []chunkItem{c}, vecs: sv}:
						case <-egctx.Done():
							return egctx.Err()
						}
					}
					return nil
				}

				select {
				case resultCh <- embedResult{items: batch, vecs: vecs}:
				case <-egctx.Done():
					return egctx.Err()
				}
				return nil
			})
		}
		return eg.Wait()
	})

	// ── Stage 4: Writer — resultCh → disk (single goroutine, non-blocking) ──
	g.Go(func() error {
		vecBuf := make([]byte, 0, dim*4*batchSize)
		lastPrint := time.Time{}

		for res := range resultCh {
			// Serialize vectors to buffer.
			vecBuf = vecBuf[:0]
			for _, v := range res.vecs {
				for _, f := range v.Values {
					vecBuf = binary.LittleEndian.AppendUint32(vecBuf, math.Float32bits(f))
				}
			}

			// Single write call for the whole batch of vectors.
			if _, err := vecFile.Write(vecBuf); err != nil {
				return fmt.Errorf("write vectors: %w", err)
			}

			// Write metadata.
			for i := range res.vecs {
				c := res.items[i]
				metaEnc.Encode(embedMeta{
					ID:       fmt.Sprintf("%s:%d", c.file, c.chunkIdx),
					File:     c.file,
					ChunkIdx: c.chunkIdx,
					TextLen:  len(c.text),
					Dim:      dim,
				})
			}
			totalVecs.Add(int64(len(res.vecs)))

			// Throttled progress — max every 200ms.
			if now := time.Now(); now.Sub(lastPrint) >= 200*time.Millisecond {
				lastPrint = now
				rate := float64(totalVecs.Load()) / now.Sub(start).Seconds()
				fmt.Fprintf(os.Stderr, "\r  files=%d chunks=%d vectors=%d errors=%d %.0f vec/s",
					totalFiles.Load(), totalChunks.Load(), totalVecs.Load(), totalErrs.Load(), rate)
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	elapsed := time.Since(start)
	rate := float64(totalVecs.Load()) / elapsed.Seconds()
	fmt.Fprintf(os.Stderr, "\r  files=%d chunks=%d vectors=%d errors=%d %.0f vec/s elapsed=%s\n",
		totalFiles.Load(), totalChunks.Load(), totalVecs.Load(), totalErrs.Load(), rate, elapsed.Round(time.Millisecond))

	// Write stats.
	statsPath := filepath.Join(outputDir, "stats.json")
	stats := map[string]interface{}{
		"files":         totalFiles.Load(),
		"chunks":        totalChunks.Load(),
		"vectors":       totalVecs.Load(),
		"errors":        totalErrs.Load(),
		"dim":           dim,
		"driver":        drv.Name(),
		"embed_workers": embedWorkers,
		"batch_size":    batchSize,
		"elapsed_ms":    elapsed.Milliseconds(),
		"vec_per_sec":   int(rate),
	}
	statsJSON, _ := json.MarshalIndent(stats, "", "  ")
	os.WriteFile(statsPath, statsJSON, 0o644)

	fi, _ := os.Stat(vecPath)
	vecSize := int64(0)
	if fi != nil {
		vecSize = fi.Size()
	}
	fmt.Fprintf(os.Stderr, "  output: %s (%s)\n", vecPath, formatBytes(vecSize))
	fmt.Fprintf(os.Stderr, "  meta:   %s\n", metaPath)

	return nil
}

// ── file reading ───────────────────────────────────────────────────────

// gzReaderPool for reuse.
var embedGZPool sync.Pool

// readMarkdownFile reads a .md or .md.gz file and returns the text content.
func readMarkdownFile(path string) (string, error) {
	if strings.HasSuffix(path, ".md.gz") {
		return readMDGZ(path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func readMDGZ(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var gz *kgzip.Reader
	if cached := embedGZPool.Get(); cached != nil {
		gz = cached.(*kgzip.Reader)
		if err := gz.Reset(f); err != nil {
			gz, err = kgzip.NewReader(f)
			if err != nil {
				return "", err
			}
		}
	} else {
		gz, err = kgzip.NewReader(f)
		if err != nil {
			return "", err
		}
	}
	defer func() {
		gz.Close()
		embedGZPool.Put(gz)
	}()

	data, err := io.ReadAll(gz)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// listCCWARCPaths fetches the WARC paths manifest for the crawl.
func listCCWARCPaths(ctx context.Context, crawlID string) ([]string, error) {
	client := cc.NewClient("", 4)
	return client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
}
