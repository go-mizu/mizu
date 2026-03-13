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
	cb := "```" // fenced code block delimiter
	bt := "`"  // inline code delimiter
	return fmt.Sprintf(`---
license: odc-by
task_categories:
- text-generation
- feature-extraction
language:
- en
pretty_name: Open Index
size_categories:
- 1M<n<10M
tags:
- common-crawl
- web-crawl
- markdown
- text
configs:
- config_name: default
  data_files:
  - split: train
    path: data/*/*
- config_name: %[1]s
  data_files:
  - split: train
    path: data/%[1]s/*
---

# Open Index

> Clean markdown from the web, ready for training and retrieval

## What is it?

Open Index is a large-scale web text dataset built from [Common Crawl](https://commoncrawl.org). Every page goes through a pipeline that extracts the main content from raw HTML, converts it to clean Markdown using [trafilatura](https://github.com/adbar/trafilatura), and packages the result into Parquet files with full WARC metadata preserved.

The dataset currently includes crawl **%[1]s**. We plan to add more snapshots over time.

Open Index is released under the **Open Data Commons Attribution License (ODC-By) v1.0**, the same license used by Common Crawl.

## What is being released?

Each Common Crawl WARC file (~1 GB of compressed HTML) becomes one Parquet shard. The shards live under a crawl-specific directory so multiple snapshots can coexist:

%[2]s
data/
  %[1]s/
    00000.parquet
    00001.parquet
    ...
%[2]s

Every row in a Parquet file is one web page. Along with the markdown body, we preserve the original WARC headers as a JSON column so you can always trace a document back to its source record.

## How to download and use Open Index

### Using %[3]sdatasets%[3]s

%[2]spython
from datasets import load_dataset

# stream the entire dataset
ds = load_dataset("open-index/draft", name="%[1]s", split="train", streaming=True)
for doc in ds:
    print(doc["url"], len(doc["markdown"]))

# load a single shard into memory
ds = load_dataset(
    "open-index/draft",
    data_files="data/%[1]s/00000.parquet",
    split="train",
)
%[2]s

### Using %[3]shuggingface_hub%[3]s

%[2]spython
from huggingface_hub import snapshot_download

folder = snapshot_download(
    "open-index/draft",
    repo_type="dataset",
    local_dir="./open-index/",
    allow_patterns="data/%[1]s/*",
)
%[2]s

For faster downloads, install %[3]spip install huggingface_hub[hf_transfer]%[3]s and set %[3]sHF_HUB_ENABLE_HF_TRANSFER=1%[3]s.

### Using DuckDB

%[2]ssql
SELECT url, host, markdown_length
FROM read_parquet('hf://datasets/open-index/draft/data/%[1]s/*.parquet')
WHERE host = 'en.wikipedia.org'
LIMIT 10;
%[2]s

# Dataset card for Open Index

## Dataset Description

- **Homepage and Repository:** [https://huggingface.co/datasets/open-index/draft](https://huggingface.co/datasets/open-index/draft)
- **Point of Contact:** please create a discussion on the Community tab
- **License:** Open Data Commons Attribution License (ODC-By) v1.0

## Dataset Structure

### Data Instance

The following is an example row from the dataset:

%[2]sjson
{
  "doc_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "url": "https://example.com/article/interesting-topic",
  "host": "example.com",
  "crawl_date": "2026-02-06T18:14:58Z",
  "warc_type": "conversion",
  "warc_record_id": "<urn:uuid:a1b2c3d4-e5f6-7890-abcd-ef1234567890>",
  "warc_refers_to": "<urn:uuid:f9e8d7c6-b5a4-3210-fedc-ba0987654321>",
  "content_type": "text/markdown",
  "html_length": 48210,
  "markdown_length": 3847,
  "warc_headers_json": "{\"Content-Length\": \"3847\", ...}",
  "markdown": "# Interesting Topic\n\nThis is the main content of the page...",
  "source_warc_file": "00000.md.warc.gz",
  "source_file_index": 0
}
%[2]s

### Data Fields

- %[3]sdoc_id%[3]s (string): unique identifier derived from the WARC-Record-ID (UUID format)
- %[3]surl%[3]s (string): original URL of the crawled page
- %[3]shost%[3]s (string): lowercase hostname extracted from the URL
- %[3]scrawl_date%[3]s (string): RFC 3339 timestamp from the WARC record
- %[3]swarc_type%[3]s (string): WARC record type, typically "conversion" for markdown output
- %[3]swarc_record_id%[3]s (string): full WARC-Record-ID in the urn:uuid format
- %[3]swarc_refers_to%[3]s (string): WARC-Record-ID of the original HTTP response record this was converted from
- %[3]scontent_type%[3]s (string): content type of the converted record (text/markdown)
- %[3]shtml_length%[3]s (int64): byte length of the original HTML body before conversion
- %[3]smarkdown_length%[3]s (int64): byte length of the converted markdown body
- %[3]swarc_headers_json%[3]s (string): all WARC headers serialized as a JSON object with sorted keys, preserving every header from the packed record for full provenance
- %[3]smarkdown%[3]s (string): the cleaned markdown content extracted from the HTML page
- %[3]ssource_warc_file%[3]s (string): filename of the packed .md.warc.gz shard this record came from
- %[3]ssource_file_index%[3]s (int32): zero-based index of the source file in the crawl manifest

### Data Splits

The default subset includes all available data across all crawl snapshots. You can also load a specific crawl by using its ID as the config name (e.g. %[3]s%[1]s%[3]s).

## Dataset Creation

### Curation Rationale

Most open web datasets either release raw text without structure or keep the HTML and leave parsing to the user. Open Index sits in between: it converts every page to Markdown so the content is immediately usable for training, while preserving the full WARC headers so you can always go back to the source if you need to.

### Source Data

The source data consists of web pages crawled by the [Common Crawl](https://commoncrawl.org) foundation. Common Crawl archives billions of pages across the public web and makes the raw WARC files freely available on Amazon S3.

### Data Processing Steps

The processing pipeline runs in five stages:

1. **Download** raw .warc.gz files from Common Crawl S3 (each file is roughly 1 GB compressed)
2. **Filter** to keep only HTTP 200 responses with a text/html content type, discarding images, scripts, redirects, and error pages
3. **Convert** HTML to Markdown using [trafilatura](https://github.com/adbar/trafilatura), which extracts the main content and strips boilerplate, navigation, sidebars, footers, and ads
4. **Pack** converted records into seekable .md.warc.gz files where each record is wrapped in its own gzip member, matching Common Crawl's concatenated-gzip format
5. **Export** each shard to Apache Parquet with Zstd compression, 100,000 rows per row group, and an 8 MB page buffer

Empty conversions (pages where trafilatura could not extract meaningful content) are dropped.

### Personal and Sensitive Information

No additional PII filtering is applied beyond what Common Crawl provides. As the dataset is sourced from the public web, it is likely that some personally identifiable information is present. If you find your own PII in the dataset and would like it removed, please open an issue on the repository.

## Considerations for Using the Data

### Social Impact

By releasing both the dataset and the full processing pipeline, we aim to lower the barrier to training and evaluating language models on high quality web data. Researchers and practitioners who cannot afford to run their own Common Crawl processing pipelines can use Open Index directly.

### Discussion of Biases

Open Index inherits the biases present in Common Crawl and the public web at large. The trafilatura extraction step favors article-like pages and may underrepresent content from forums, social media, and non-standard page layouts. We have not applied any machine-learning-based quality or toxicity filters, as such filters have been shown to disproportionately remove content from certain dialects and communities.

### Known Limitations

Code-heavy pages may not convert well to Markdown. If you are training a model that needs strong code performance, consider supplementing Open Index with a dedicated code dataset such as [The Stack v2](https://huggingface.co/datasets/bigcode/the-stack-v2). Similarly, highly structured pages like Wikipedia may have better formatting in dedicated Wikipedia dumps than in their Common Crawl versions.

## Additional Information

### Licensing

The dataset is released under the **Open Data Commons Attribution License (ODC-By) v1.0**. The use of this dataset is also subject to [Common Crawl's Terms of Use](https://commoncrawl.org/terms-of-use). The original content remains subject to the rights and terms of its respective publishers.

### Contact

Please open a discussion on the [Community tab](https://huggingface.co/datasets/open-index/draft/discussions) for questions, feedback, or issues.
`, c, cb, bt)
}

func ccPublishLicense() string {
	return `Open Data Commons Attribution License (ODC-By) v1.0

Full text: https://opendatacommons.org/licenses/by/1-0/

You are free to copy, distribute, use, modify, transform, and build upon
this database, as long as you attribute the source.

Attribution: "Open Index, derived from Common Crawl (https://commoncrawl.org)"

Note: This dataset contains data derived from Common Crawl, which archives
third-party web content. The original content remains subject to the rights
of its respective publishers. You are responsible for complying with applicable
law including downstream licensing obligations, robots.txt restrictions, privacy
requirements, and content removal requests. See Common Crawl's Terms of Use:
https://commoncrawl.org/terms-of-use
`
}
