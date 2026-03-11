package embed

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ModelInfo describes a downloadable embedding model.
type ModelInfo struct {
	Name   string      // human-readable name
	Driver string      // "llamacpp" or "onnx"
	Dim    int         // embedding dimension
	SizeMB int         // approximate download size
	Desc   string      // one-line description
	Files  []ModelFile // files to download
}

// ModelFile is a single downloadable file.
type ModelFile struct {
	URL  string // HuggingFace download URL
	Name string // local filename
}

// DefaultModelName returns the default model name for a driver.
func DefaultModelName(driver string) string {
	switch driver {
	case "llamacpp":
		return "nomic-embed-text-v1.5"
	case "onnx":
		return "all-MiniLM-L6-v2"
	case "gemini":
		return "gemini-embedding-exp-03-07"
	default:
		return ""
	}
}

// DefaultModelDir returns the default model storage directory.
func DefaultModelDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "models")
}

// GeminiModels lists the supported Gemini API embedding models (no local download required).
// Free tier: 5 RPM and 100 RPD for gemini-embedding-exp-03-07.
var GeminiModels = []ModelInfo{
	{
		Name:   "gemini-embedding-exp-03-07",
		Driver: "gemini",
		Dim:    3072,
		SizeMB: 0, // API model — no local download
		Desc:   "Gemini Embedding Exp 03-07 (3072-dim, Matryoshka, best quality; free tier: 5 RPM / 100 RPD)",
	},
}

// Models lists all known downloadable embedding models.
var Models = []ModelInfo{
	// --- llamacpp (GGUF) models ---
	{
		Name:   "nomic-embed-text-v1.5",
		Driver: "llamacpp",
		Dim:    768,
		SizeMB: 137,
		Desc:   "Nomic Embed v1.5 (768-dim, 8K context, best quality for llamacpp)",
		Files: []ModelFile{
			{URL: "https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q8_0.gguf", Name: "nomic-embed-text-v1.5.Q8_0.gguf"},
		},
	},
	{
		Name:   "bge-small-en-v1.5",
		Driver: "llamacpp",
		Dim:    384,
		SizeMB: 67,
		Desc:   "BGE Small English v1.5 (384-dim, fast, compact)",
		Files: []ModelFile{
			{URL: "https://huggingface.co/CompendiumLabs/bge-small-en-v1.5-gguf/resolve/main/bge-small-en-v1.5-f16.gguf", Name: "bge-small-en-v1.5-f16.gguf"},
		},
	},
	{
		Name:   "all-MiniLM-L6-v2",
		Driver: "llamacpp",
		Dim:    384,
		SizeMB: 46,
		Desc:   "all-MiniLM-L6-v2 (384-dim, smallest, fastest)",
		Files: []ModelFile{
			{URL: "https://huggingface.co/leliuga/all-MiniLM-L6-v2-GGUF/resolve/main/all-MiniLM-L6-v2.Q8_0.gguf", Name: "all-MiniLM-L6-v2.Q8_0.gguf"},
		},
	},

	{
		Name:   "qwen3-embedding-0.6b",
		Driver: "llamacpp",
		Dim:    1024,
		SizeMB: 639,
		Desc:   "Qwen3 Embedding 0.6B (1024-dim, 32K context, multilingual, instruction-aware)",
		Files: []ModelFile{
			{URL: "https://huggingface.co/Qwen/Qwen3-Embedding-0.6B-GGUF/resolve/main/Qwen3-Embedding-0.6B-Q8_0.gguf", Name: "Qwen3-Embedding-0.6B-Q8_0.gguf"},
		},
	},

	// --- ONNX models ---
	{
		Name:   "all-MiniLM-L6-v2",
		Driver: "onnx",
		Dim:    384,
		SizeMB: 90,
		Desc:   "all-MiniLM-L6-v2 ONNX (384-dim, CPU-optimized, default)",
		Files: []ModelFile{
			{URL: "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx", Name: "model.onnx"},
			{URL: "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/vocab.txt", Name: "vocab.txt"},
		},
	},
}

// FindModel looks up a model by driver and name.
func FindModel(driver, name string) (ModelInfo, bool) {
	for _, m := range Models {
		if m.Driver == driver && m.Name == name {
			return m, true
		}
	}
	return ModelInfo{}, false
}

// ListModels returns all models for a given driver, or all models if driver is empty.
func ListModels(driver string) []ModelInfo {
	if driver == "" {
		return Models
	}
	var out []ModelInfo
	for _, m := range Models {
		if m.Driver == driver {
			out = append(out, m)
		}
	}
	return out
}

// ModelFilesDir returns the directory where model files should be stored.
// For llamacpp: {baseDir}/ (flat — matches docker volume mount)
// For onnx:     {baseDir}/onnx/{modelName}/
func ModelFilesDir(baseDir string, m ModelInfo) string {
	if baseDir == "" {
		baseDir = DefaultModelDir()
	}
	switch m.Driver {
	case "onnx":
		return filepath.Join(baseDir, "onnx", m.Name)
	default:
		return baseDir
	}
}

// GGUFFileName returns the GGUF filename for a llamacpp model.
// Returns empty string if the model has no GGUF files.
func (m ModelInfo) GGUFFileName() string {
	for _, f := range m.Files {
		if strings.HasSuffix(f.Name, ".gguf") {
			return f.Name
		}
	}
	return ""
}

// DownloadModel downloads all files for a model to the given base directory.
// Returns the directory where files were saved.
func DownloadModel(baseDir string, m ModelInfo) (string, error) {
	dir := ModelFilesDir(baseDir, m)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("embed: mkdir %s: %w", dir, err)
	}

	for _, f := range m.Files {
		dest := filepath.Join(dir, f.Name)
		if err := DownloadFile(f.URL, dest, f.Name); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// IsModelDownloaded checks if all files for a model exist locally.
func IsModelDownloaded(baseDir string, m ModelInfo) bool {
	dir := ModelFilesDir(baseDir, m)
	for _, f := range m.Files {
		if _, err := os.Stat(filepath.Join(dir, f.Name)); os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// DownloadFile downloads a URL to a local path if it doesn't already exist.
// Uses an atomic .tmp pattern to avoid partial files.
func DownloadFile(url, dest, desc string) error {
	if _, err := os.Stat(dest); err == nil {
		return nil // already exists
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "  downloading %s ...\n", desc)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", desc, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", desc, resp.StatusCode)
	}

	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create %s: %w", tmp, err)
	}

	written, err := io.Copy(f, &progressWriter{r: resp.Body, total: resp.ContentLength})
	f.Close()
	if err != nil {
		os.Remove(tmp)
		return fmt.Errorf("write %s: %w", desc, err)
	}

	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename %s: %w", desc, err)
	}

	fmt.Fprintf(os.Stderr, "  downloaded %s (%d MB)\n", desc, written/(1024*1024))
	return nil
}

// progressWriter wraps an io.Reader to print download progress.
type progressWriter struct {
	r       io.Reader
	total   int64
	current int64
	last    int // last printed percentage
}

func (pw *progressWriter) Read(p []byte) (int, error) {
	n, err := pw.r.Read(p)
	pw.current += int64(n)
	if pw.total > 0 {
		pct := int(pw.current * 100 / pw.total)
		if pct != pw.last && pct%5 == 0 {
			pw.last = pct
			fmt.Fprintf(os.Stderr, "\r  downloading ... %d%%", pct)
		}
	}
	return n, err
}
