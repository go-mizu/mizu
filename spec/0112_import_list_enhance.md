# Spec 0112: Finewiki List & Import Enhancements

## Overview

Enhance the `finewiki list` and `finewiki import` commands with better UX, including:
1. Fix bug in `finewiki list` (currently shows 0 languages)
2. Enhance `--installed` flag with detailed information
3. Add download progress bar using charmbracelet/bubbles
4. Use `curl` for faster downloads with pure Go fallback

## Current Issues

### Bug: `finewiki list` Shows No Languages

**Root Cause**: The HuggingFace API response parsing is incorrect.

**Current Code** (list.go:72-78):
```go
var result struct {
    DatasetInfo struct {
        Configs []struct {
            ConfigName string `json:"config_name"`
        } `json:"config_names_with_splits,omitempty"`
    } `json:"dataset_info"`
}
```

**Actual API Response Structure**:
```json
{
  "dataset_info": {
    "en": {"config_name": "en", "splits": {...}, "download_size": 37722458405, ...},
    "vi": {"config_name": "vi", "splits": {...}, "download_size": 123456789, ...},
    ...
  }
}
```

The `dataset_info` field is a **map[string]ConfigInfo**, not a struct with an array. Each key is the language code.

**Fix**: Parse as `map[string]json.RawMessage` or a properly typed map.

### Missing Features in `--installed`

Current output:
```
installed languages (1):
  vi

data directory: /Users/apple/data/blueprint/finewiki
```

Desired output:
```
Installed Languages (1):

  LANG   FILES   SIZE       PAGES
  vi     1       245.3 MB   835,612

Data directory: /Users/apple/data/blueprint/finewiki
Total size: 245.3 MB
```

### Missing Download Progress

Current output:
```
downloading data.parquet (1/1)...
saved: /path/to/data.parquet (245.3 MB)
```

Desired output with bubbles progress bar:
```
Downloading Vietnamese Wikipedia (vi)
Source: huggingface.co/datasets/HuggingFaceFW/finewiki
Target: ~/data/blueprint/finewiki/vi/

 data.parquet  ████████████░░░░░░░░  58% │ 142.3/245.3 MB │ 12.4 MB/s │ ETA 8s
```

## Detailed Design

### 1. Fix `finewiki list` Command

**File**: `cli/list.go`

Update the response parsing:

```go
type datasetInfoResponse struct {
    DatasetInfo map[string]configInfo `json:"dataset_info"`
}

type configInfo struct {
    ConfigName   string `json:"config_name"`
    DownloadSize int64  `json:"download_size"`
    DatasetSize  int64  `json:"dataset_size"`
    Splits       map[string]splitInfo `json:"splits"`
}

type splitInfo struct {
    Name        string `json:"name"`
    NumBytes    int64  `json:"num_bytes"`
    NumExamples int64  `json:"num_examples"`
}
```

Extract languages from map keys instead of array:

```go
langs := make([]string, 0, len(result.DatasetInfo))
for lang := range result.DatasetInfo {
    langs = append(langs, lang)
}
sort.Strings(langs)
```

### 2. Enhance `--installed` Flag

**File**: `cli/list.go`

Add new `langInfo` struct and enhance output:

```go
type langInfo struct {
    Lang       string
    Files      int
    SizeBytes  int64
    Pages      int64  // estimated from parquet metadata if available
}

func listInstalled(dataDir string) error {
    // ... existing directory scanning ...

    var infos []langInfo
    for _, lang := range langs {
        info := gatherLangInfo(dataDir, lang)
        infos = append(infos, info)
    }

    // Print table with columns: LANG, FILES, SIZE, PAGES
    printInstalledTable(infos)

    // Print summary
    fmt.Printf("\nData directory: %s\n", dataDir)
    fmt.Printf("Total size: %s\n", formatBytes(totalSize))
}

func gatherLangInfo(dataDir, lang string) langInfo {
    langDir := filepath.Join(dataDir, lang)

    // Count parquet files and sum sizes
    var files int
    var sizeBytes int64

    single := filepath.Join(langDir, "data.parquet")
    if fi, err := os.Stat(single); err == nil {
        files = 1
        sizeBytes = fi.Size()
    } else {
        pattern := filepath.Join(langDir, "data-*.parquet")
        matches, _ := filepath.Glob(pattern)
        files = len(matches)
        for _, m := range matches {
            if fi, err := os.Stat(m); err == nil {
                sizeBytes += fi.Size()
            }
        }
    }

    // Pages count: could read from DuckDB if available, or estimate
    var pages int64
    dbPath := filepath.Join(langDir, "wiki.duckdb")
    if _, err := os.Stat(dbPath); err == nil {
        pages = countPagesFromDB(dbPath)  // SELECT COUNT(*) FROM titles
    }

    return langInfo{
        Lang:      lang,
        Files:     files,
        SizeBytes: sizeBytes,
        Pages:     pages,
    }
}
```

**Output Format**:
```
Installed Languages (2):

  LANG   FILES   SIZE       PAGES
  en     3       35.1 GB    6,614,655
  vi     1       245.3 MB   835,612

Data directory: /Users/apple/data/blueprint/finewiki
Total size: 35.3 GB
```

### 3. Add Download Progress with Bubbles

**File**: `cli/download.go` (new file)

Use `github.com/charmbracelet/bubbles/progress` for visual progress:

```go
package cli

import (
    "fmt"
    "io"
    "os"
    "os/exec"
    "time"

    "github.com/charmbracelet/bubbles/progress"
    tea "github.com/charmbracelet/bubbletea"
)

// Downloader handles file downloads with progress
type Downloader struct {
    useCurl bool  // true if curl is available
}

// NewDownloader creates a downloader, detecting curl availability
func NewDownloader() *Downloader {
    _, err := exec.LookPath("curl")
    return &Downloader{useCurl: err == nil}
}

// Download downloads a file with progress display
func (d *Downloader) Download(url, dst string, totalSize int64) error {
    if d.useCurl {
        return d.downloadWithCurl(url, dst)
    }
    return d.downloadWithGo(url, dst, totalSize)
}

func (d *Downloader) downloadWithCurl(url, dst string) error {
    tmp := dst + ".partial"

    // curl with progress bar: -# shows progress, -L follows redirects
    // -o outputs to file, --fail fails on HTTP errors
    cmd := exec.Command("curl", "-#", "-L", "-o", tmp, "--fail", url)

    // Pass through HF_TOKEN if set
    if token := os.Getenv("HF_TOKEN"); token != "" {
        cmd.Args = append(cmd.Args, "-H", "Authorization: Bearer "+token)
    }

    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Run(); err != nil {
        os.Remove(tmp)
        return fmt.Errorf("curl failed: %w", err)
    }

    return os.Rename(tmp, dst)
}

func (d *Downloader) downloadWithGo(url, dst string, totalSize int64) error {
    // Pure Go implementation with bubbles progress bar
    // ... implementation with tea.Program and progress.Model ...
}
```

**Progress Model for Pure Go**:

```go
type downloadModel struct {
    progress    progress.Model
    url         string
    dst         string
    totalSize   int64
    downloaded  int64
    speed       float64
    err         error
    done        bool
}

type tickMsg time.Time
type progressMsg int64
type doneMsg struct{ err error }

func (m downloadModel) Init() tea.Cmd {
    return tea.Batch(
        m.startDownload(),
        tickCmd(),
    )
}

func (m downloadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case progressMsg:
        m.downloaded = int64(msg)
        return m, nil
    case doneMsg:
        m.done = true
        m.err = msg.err
        return m, tea.Quit
    case tickMsg:
        // Update progress bar
        if m.totalSize > 0 {
            percent := float64(m.downloaded) / float64(m.totalSize)
            m.progress.SetPercent(percent)
        }
        return m, tickCmd()
    case tea.KeyMsg:
        if msg.String() == "ctrl+c" {
            return m, tea.Quit
        }
    }
    return m, nil
}

func (m downloadModel) View() string {
    if m.done {
        if m.err != nil {
            return fmt.Sprintf("Error: %v\n", m.err)
        }
        return fmt.Sprintf("Downloaded: %s\n", m.dst)
    }

    return fmt.Sprintf(
        "%s  %s │ %s/%s │ %s/s │ ETA %s\n",
        filepath.Base(m.dst),
        m.progress.View(),
        formatBytes(m.downloaded),
        formatBytes(m.totalSize),
        formatBytes(int64(m.speed)),
        formatETA(m.totalSize-m.downloaded, m.speed),
    )
}
```

### 4. Use Curl by Default, Fallback to Pure Go

**Rationale**:
- `curl` is faster due to optimized C implementation
- `curl` handles complex HTTP scenarios (redirects, compression) better
- Most systems have `curl` installed
- Pure Go fallback ensures portability

**Detection Logic**:
```go
func hasCurl() bool {
    _, err := exec.LookPath("curl")
    return err == nil
}
```

**Import Command Integration**:

```go
func runImport(ctx context.Context, dataDir, lang string) error {
    // Get parquet URLs and sizes
    files, err := getParquetFiles(ctx, lang)
    if err != nil {
        return err
    }

    if len(files) == 0 {
        return fmt.Errorf("no parquet files found for language: %s", lang)
    }

    // Print download info
    fmt.Printf("Downloading %s Wikipedia (%s)\n", langName(lang), lang)
    fmt.Printf("Source: huggingface.co/datasets/%s\n", hfDataset)
    fmt.Printf("Target: %s/\n\n", LangDir(dataDir, lang))

    // Print files to download
    var totalSize int64
    for _, f := range files {
        totalSize += f.Size
    }
    fmt.Printf("Files: %d (total: %s)\n\n", len(files), formatBytes(totalSize))

    // Create downloader
    dl := NewDownloader()
    if dl.useCurl {
        fmt.Println("Using curl for download")
    } else {
        fmt.Println("Using native Go downloader")
    }

    // Download each file
    langDir := LangDir(dataDir, lang)
    if err := os.MkdirAll(langDir, 0o755); err != nil {
        return err
    }

    for i, f := range files {
        var dst string
        if len(files) == 1 {
            dst = filepath.Join(langDir, "data.parquet")
        } else {
            dst = filepath.Join(langDir, fmt.Sprintf("data-%03d.parquet", i))
        }

        fmt.Printf("\n[%d/%d] %s\n", i+1, len(files), filepath.Base(dst))

        if err := dl.Download(f.URL, dst, f.Size); err != nil {
            return fmt.Errorf("download %s: %w", f.URL, err)
        }
    }

    fmt.Printf("\nImport complete. Run 'finewiki serve %s' to start.\n", lang)
    return nil
}
```

### 5. Enhanced API Response Types

**File**: `cli/hf_api.go` (new file)

```go
package cli

// parquetFileInfo contains URL and size for a parquet file
type parquetFileInfo struct {
    URL      string
    Filename string
    Size     int64
}

// getParquetFiles returns file info including sizes
func getParquetFiles(ctx context.Context, lang string) ([]parquetFileInfo, error) {
    url := fmt.Sprintf("%s/parquet?dataset=%s&config=%s", hfServerAPI, hfDataset, lang)

    // ... HTTP request ...

    var result struct {
        ParquetFiles []struct {
            URL      string `json:"url"`
            Filename string `json:"filename"`
            Size     int64  `json:"size"`
        } `json:"parquet_files"`
    }

    // ... parse and return ...
}
```

## File Changes Summary

| File | Change |
|------|--------|
| `cli/list.go` | Fix API parsing, enhance `--installed` output |
| `cli/import.go` | Integrate new downloader, show detailed info |
| `cli/download.go` | New file: curl/Go downloader with progress |
| `cli/hf_api.go` | New file: HuggingFace API helpers |
| `go.mod` | Add `github.com/charmbracelet/bubbles` |

## Dependencies

Add to `go.mod`:
```
github.com/charmbracelet/bubbles v0.20.0
github.com/charmbracelet/bubbletea v1.2.4
```

## Testing

1. **List Available**: `finewiki list` should show 300+ languages
2. **List Installed**: `finewiki list --installed` should show table with sizes
3. **Import with curl**: Install a small wiki (e.g., `finewiki import ay`)
4. **Import without curl**: Rename curl, test Go fallback
5. **Progress display**: Verify progress bar updates during download

## Example Session

```bash
$ finewiki list
Available Languages (334):

  ab, ace, ady, af, als, alt, am, ami, an, ang, anp, ar, arc, ary, arz, as,
  ast, atj, av, avk, awa, ay, az, azb, ba, ban, bar, bat_smg, bbc, bcl, be,
  ...
  vi, vls, vo, wa, war, wo, wuu, xal, xh, xmf, yi, yo, yue, za, zea, zh,
  zh_classical, zh_min_nan, zh_yue, zu

$ finewiki list --installed
Installed Languages (2):

  LANG   FILES   SIZE       PAGES
  en     3       35.1 GB    6,614,655
  vi     1       245.3 MB   835,612

Data directory: /Users/apple/data/blueprint/finewiki
Total size: 35.3 GB

$ finewiki import ay
Downloading Aymara Wikipedia (ay)
Source: huggingface.co/datasets/HuggingFaceFW/finewiki
Target: ~/data/blueprint/finewiki/ay/

Files: 1 (total: 4.2 MB)
Using curl for download

[1/1] data.parquet
######################################################################## 100.0%

Import complete. Run 'finewiki serve ay' to start.
```

## Future Enhancements

- Resume interrupted downloads
- Parallel downloads for sharded files
- Checksum verification
- Bandwidth limiting option
