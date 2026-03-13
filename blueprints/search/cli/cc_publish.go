package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const ccPublishPythonScript = `
import json
import os
import sys

try:
    from huggingface_hub import CommitOperationAdd, HfApi
except Exception as exc:
    print(json.dumps({"error": f"missing huggingface_hub: {exc}"}))
    sys.exit(2)

payload = json.load(sys.stdin)
token = os.environ.get("HF_TOKEN", "").strip()
if not token:
    print(json.dumps({"error": "HF_TOKEN is not set"}))
    sys.exit(3)

api = HfApi(token=token)
api.create_repo(
    repo_id=payload["repo_id"],
    repo_type="dataset",
    exist_ok=True,
    private=payload.get("private", False),
)

paths = [item["path_in_repo"] for item in payload["files"]]
existing = set()
if paths:
    for start in range(0, len(paths), 100):
        chunk = paths[start:start+100]
        infos = api.get_paths_info(
            repo_id=payload["repo_id"],
            paths=chunk,
            repo_type="dataset",
            token=token,
        )
        existing.update(getattr(info, "path", "") for info in infos)

selected = []
skipped = []
republish = bool(payload.get("republish"))
for item in payload["files"]:
    if (not republish) and item["path_in_repo"] in existing:
        skipped.append(item["path_in_repo"])
        continue
    selected.append(item)

if not selected:
    print(json.dumps({
        "uploaded": [],
        "skipped": skipped,
        "commit_url": "",
    }))
    sys.exit(0)

operations = [
    CommitOperationAdd(path_in_repo=item["path_in_repo"], path_or_fileobj=item["local_path"])
    for item in selected
]
commit = api.create_commit(
    repo_id=payload["repo_id"],
    repo_type="dataset",
    operations=operations,
    commit_message=payload["commit_message"],
    token=token,
)
print(json.dumps({
    "uploaded": [item["path_in_repo"] for item in selected],
    "skipped": skipped,
    "commit_url": getattr(commit, "commit_url", ""),
}))
`

type ccPublishUploadFile struct {
	LocalPath  string `json:"local_path"`
	PathInRepo string `json:"path_in_repo"`
}

type ccPublishPayload struct {
	RepoID        string                `json:"repo_id"`
	Private       bool                  `json:"private"`
	Republish     bool                  `json:"republish"`
	CommitMessage string                `json:"commit_message"`
	Files         []ccPublishUploadFile `json:"files"`
}

type ccPublishResult struct {
	Error     string   `json:"error"`
	Uploaded  []string `json:"uploaded"`
	Skipped   []string `json:"skipped"`
	CommitURL string   `json:"commit_url"`
}

func newCCPublish() *cobra.Command {
	var (
		crawlID   string
		fileIdx   string
		repoRoot  string
		repoID    string
		republish bool
		private   bool
	)

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish exported Common Crawl parquet shards to Hugging Face",
		Long: `Publish $HOME/data/common-crawl/{crawl}/export/repo to a Hugging Face dataset repo.

The command creates the dataset repo if needed, ensures README.md and LICENSE
exist locally, uploads only missing parquet files by default, and supports
targeting one shard with --file 0.`,
		Example: `  search cc publish
  search cc publish --file 0
  search cc publish --crawl CC-MAIN-2026-08 --repo open-index/draft
  search cc publish --file 0 --republish`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCPublish(cmd.Context(), crawlID, fileIdx, repoRoot, repoID, republish, private)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&fileIdx, "file", "all", "File index, range (0-9), or all")
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "Local export repo root (default: $HOME/data/common-crawl/{crawl}/export/repo)")
	cmd.Flags().StringVar(&repoID, "repo", "open-index/draft", "Hugging Face dataset repo ID")
	cmd.Flags().BoolVar(&republish, "republish", false, "Upload even if the remote path already exists")
	cmd.Flags().BoolVar(&private, "private", false, "Create the Hugging Face dataset repo as private")
	return cmd
}

func runCCPublish(ctx context.Context, crawlID, fileIdx, repoRoot, repoID string, republish, private bool) error {
	resolvedID, note, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedID
	if note != "" {
		ccPrintDefaultCrawlResolution(crawlID, note)
	}

	if repoRoot == "" {
		repoRoot = ccDefaultExportRepoRoot(crawlID)
	}
	if err := ccEnsurePublishRepoFiles(repoRoot, crawlID); err != nil {
		return err
	}

	files, err := ccResolvePublishUploadFiles(repoRoot, crawlID, fileIdx)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no local parquet files selected under %s", filepath.Join(repoRoot, "data"))
	}

	payload := ccPublishPayload{
		RepoID:        repoID,
		Private:       private,
		Republish:     republish,
		CommitMessage: ccPublishCommitMessage(fileIdx, files),
		Files: append([]ccPublishUploadFile{
			{LocalPath: filepath.Join(repoRoot, "README.md"), PathInRepo: "README.md"},
			{LocalPath: filepath.Join(repoRoot, "LICENSE"), PathInRepo: "LICENSE"},
		}, files...),
	}

	stdout, err := ccRunPublishPython(ctx, payload)
	if err != nil {
		return err
	}

	var result ccPublishResult
	if err := json.Unmarshal(stdout, &result); err != nil {
		return fmt.Errorf("decode publish response: %w\nstdout: %s", err, string(stdout))
	}
	if result.Error != "" {
		return fmt.Errorf("publish failed: %s", result.Error)
	}

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Common Crawl Publish"))
	fmt.Println()
	fmt.Printf("  Crawl      %s\n", labelStyle.Render(crawlID))
	fmt.Printf("  Repo root  %s\n", labelStyle.Render(repoRoot))
	fmt.Printf("  HF repo    %s\n", infoStyle.Render(repoID))
	fmt.Printf("  Uploaded   %s\n", infoStyle.Render(ccFmtInt64(int64(len(result.Uploaded)))))
	fmt.Printf("  Skipped    %s\n", infoStyle.Render(ccFmtInt64(int64(len(result.Skipped)))))
	if result.CommitURL != "" {
		fmt.Printf("  Commit     %s\n", labelStyle.Render(result.CommitURL))
	}
	return nil
}

func ccDefaultExportRepoRoot(crawlID string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "common-crawl", crawlID, "export", "repo")
}

func ccEnsurePublishRepoFiles(repoRoot, crawlID string) error {
	if err := os.MkdirAll(filepath.Join(repoRoot, "data"), 0o755); err != nil {
		return fmt.Errorf("create repo root: %w", err)
	}
	files := map[string]string{
		filepath.Join(repoRoot, "README.md"): ccPublishREADME(crawlID),
		filepath.Join(repoRoot, "LICENSE"):   ccPublishLicense(),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", filepath.Base(path), err)
		}
	}
	return nil
}

func ccResolvePublishUploadFiles(repoRoot, crawlID, selector string) ([]ccPublishUploadFile, error) {
	dataDir := filepath.Join(repoRoot, "data")
	crawlDataDir := filepath.Join(dataDir, crawlID)
	if selector == "" || selector == "all" {
		// Walk all crawl subdirs under data/
		var files []ccPublishUploadFile
		_ = filepath.WalkDir(dataDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".parquet") {
				return nil
			}
			rel, _ := filepath.Rel(repoRoot, path)
			files = append(files, ccPublishUploadFile{
				LocalPath:  path,
				PathInRepo: filepath.ToSlash(rel),
			})
			return nil
		})
		sort.Slice(files, func(i, j int) bool { return files[i].PathInRepo < files[j].PathInRepo })
		return files, nil
	}

	indices, err := ccParseOpenFileSelector(selector)
	if err != nil {
		return nil, err
	}
	files := make([]ccPublishUploadFile, 0, len(indices))
	for _, idx := range indices {
		name := fmt.Sprintf("%05d.parquet", idx)
		localPath := filepath.Join(crawlDataDir, name)
		if !fileExists(localPath) {
			return nil, fmt.Errorf("selected parquet file not found: %s", localPath)
		}
		files = append(files, ccPublishUploadFile{
			LocalPath:  localPath,
			PathInRepo: filepath.ToSlash(filepath.Join("data", crawlID, name)),
		})
	}
	return files, nil
}

func ccParseOpenFileSelector(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "all" {
		return nil, nil
	}
	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		lo, err1 := strconv.Atoi(parts[0])
		hi, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || lo < 0 || hi < lo {
			return nil, fmt.Errorf("invalid range %q", s)
		}
		out := make([]int, hi-lo+1)
		for i := range out {
			out[i] = lo + i
		}
		return out, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return nil, fmt.Errorf("invalid file index %q", s)
	}
	return []int{n}, nil
}

func ccPublishCommitMessage(fileIdx string, files []ccPublishUploadFile) string {
	if len(files) == 1 {
		return "Publish " + files[0].PathInRepo
	}
	if fileIdx != "" && fileIdx != "all" {
		return "Publish Common Crawl shards " + fileIdx
	}
	return fmt.Sprintf("Publish %d Common Crawl parquet shards", len(files))
}

func ccRunPublishPython(ctx context.Context, payload ccPublishPayload) ([]byte, error) {
	input, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode publish payload: %w", err)
	}

	cmd := exec.CommandContext(ctx, "python3", "-c", ccPublishPythonScript)
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("python publish helper: %s", msg)
	}
	return bytes.TrimSpace(stdout.Bytes()), nil
}

func ccPublishREADME(crawlID string) string {
	c := crawlID
	return fmt.Sprintf(`---
license: odc-by
pretty_name: Open Index
language:
- en
tags:
- common-crawl
- web-crawl
- markdown
- text
size_categories:
- 10B<n<100B
---

# Open Index

**Open Index** is a large-scale web text dataset derived from [Common Crawl](https://commoncrawl.org) with HTML converted to clean Markdown. Designed for language model training, information retrieval research, and web-scale NLP.

This snapshot is built from crawl **%s**.

---

## Dataset Summary

| Property | Value |
|---|---|
| Source | Common Crawl (%s) |
| Format | Apache Parquet (Zstd compressed) |
| Content | Markdown-converted web pages |
| License | [ODC-By 1.0](https://opendatacommons.org/licenses/by/1-0/) |

---

## Dataset Structure

Parquet files are organised by crawl ID:

`+"`"+`
data/
└── %s/
    ├── 00000.parquet
    ├── 00001.parquet
    └── ...
`+"`"+`

Each file corresponds to one packed WARC shard (~1 GB source WARC).

### Data Fields

| Field | Type | Description |
|---|---|---|
| `+"`doc_id`"+` | string | UUID derived from the WARC-Record-ID |
| `+"`url`"+` | string | Original URL of the crawled page |
| `+"`host`"+` | string | Lowercase hostname extracted from the URL |
| `+"`crawl_date`"+` | string | RFC3339 timestamp from the WARC record |
| `+"`warc_type`"+` | string | WARC record type (conversion, response, …) |
| `+"`warc_record_id`"+` | string | Original `+"`<urn:uuid:…>`"+` WARC record identifier |
| `+"`warc_refers_to`"+` | string | Record ID of the source response record |
| `+"`content_type`"+` | string | HTTP Content-Type of the original response |
| `+"`html_length`"+` | int64 | Byte length of the original HTML body |
| `+"`markdown_length`"+` | int64 | Byte length of the converted Markdown body |
| `+"`warc_headers_json`"+` | string | All WARC headers as stable-key JSON |
| `+"`markdown_body`"+` | string | Clean Markdown text converted from HTML |
| `+"`source_warc_file`"+` | string | Source packed .md.warc.gz shard filename |
| `+"`source_file_index`"+` | int32 | Index of the source file in the crawl manifest |

---

## Usage

### Hugging Face datasets

`+"`"+`python
from datasets import load_dataset

# Stream the full snapshot
ds = load_dataset("open-index/draft", split="train", streaming=True)
for doc in ds:
    print(doc["url"], doc["markdown_body"][:200])

# Load a single shard
ds = load_dataset(
    "open-index/draft",
    data_files="data/%s/00000.parquet",
    split="train",
)
`+"`"+`

### DuckDB

`+"`"+`sql
SELECT url, host, markdown_length
FROM read_parquet('hf://datasets/open-index/draft/data/%s/*.parquet')
WHERE host LIKE '%%wikipedia.org'
LIMIT 10;
`+"`"+`

### pandas

`+"`"+`python
import pandas as pd

df = pd.read_parquet(
    "hf://datasets/open-index/draft/data/%s/00000.parquet",
    columns=["url", "host", "crawl_date", "markdown_body"],
)
`+"`"+`

---

## Data Processing Pipeline

1. **Download** — Raw .warc.gz files from Common Crawl S3.
2. **Filter** — HTTP 200 responses with text/html content only.
3. **Convert** — HTML → Markdown via [trafilatura](https://github.com/adbar/trafilatura) (removes boilerplate, navigation, ads).
4. **Pack** — Seekable .md.warc.gz files (one gzip member per record, CC-compatible format).
5. **Export** — Parquet with Zstd compression, 100K rows per row group.

---

## Source & License

- Common Crawl: [https://commoncrawl.org](https://commoncrawl.org)
- Terms of Use: [https://commoncrawl.org/terms-of-use](https://commoncrawl.org/terms-of-use)
- This dataset is released under the [Open Data Commons Attribution License (ODC-By) v1.0](https://opendatacommons.org/licenses/by/1-0/)
`, c, c, c, c, c, c)
}

func ccPublishLicense() string {
	return `Open Data Commons Attribution License (ODC-By) v1.0

This dataset is made available under the Open Data Commons Attribution License:
https://opendatacommons.org/licenses/by/1-0/

You are free to share, create, and adapt this data — even for commercial purposes —
as long as you attribute the source.

Attribution requirements:
- Cite "Open Index, derived from Common Crawl (https://commoncrawl.org)"
- Include a link to this dataset when used in publications or products

Additional notices:

1. This dataset contains data derived from Common Crawl, which archives third-party
   web content. The original content remains subject to the rights of its respective
   publishers and the Common Crawl Terms of Use: https://commoncrawl.org/terms-of-use

2. You are responsible for complying with applicable law including downstream licensing
   obligations, robots.txt restrictions, privacy requirements, and content removal
   requests from original publishers.
`
}
