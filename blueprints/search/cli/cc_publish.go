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

	files, err := ccResolvePublishUploadFiles(repoRoot, fileIdx)
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

func ccResolvePublishUploadFiles(repoRoot, selector string) ([]ccPublishUploadFile, error) {
	dataDir := filepath.Join(repoRoot, "data")
	if selector == "" || selector == "all" {
		entries, err := os.ReadDir(dataDir)
		if err != nil {
			return nil, fmt.Errorf("read data dir: %w", err)
		}
		files := make([]ccPublishUploadFile, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".parquet") {
				continue
			}
			files = append(files, ccPublishUploadFile{
				LocalPath:  filepath.Join(dataDir, entry.Name()),
				PathInRepo: filepath.ToSlash(filepath.Join("data", entry.Name())),
			})
		}
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
		localPath := filepath.Join(dataDir, name)
		if !fileExists(localPath) {
			return nil, fmt.Errorf("selected parquet file not found: %s", localPath)
		}
		files = append(files, ccPublishUploadFile{
			LocalPath:  localPath,
			PathInRepo: filepath.ToSlash(filepath.Join("data", name)),
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
	return fmt.Sprintf(`---
license: other
pretty_name: Open Index Draft
---

# Open Index Draft

This dataset contains markdown exports derived from Common Crawl shard %s.

Layout:

- data/*.parquet: one parquet file per packed markdown WARC shard
- README.md: dataset description
- LICENSE: Common Crawl licensing and usage notice

Parquet schema:

- doc_id, url, host, crawl_date
- warc_type, warc_record_id, warc_refers_to
- content_type, content_length, markdown_length
- warc_headers_json: all WARC header metadata serialized as JSON
- markdown_body: markdown body extracted from the packed WARC record

Source:

- Common Crawl: [https://commoncrawl.org](https://commoncrawl.org)
`, crawlID)
}

func ccPublishLicense() string {
	return `Common Crawl License Notice

This repository contains data derived from Common Crawl.

Common Crawl makes its datasets publicly available subject to its Terms of Use:
https://commoncrawl.org/terms-of-use

Important:

1. Common Crawl is an archive of third-party web content.
2. The original content remains subject to the rights and terms of its respective publishers.
3. You are responsible for complying with applicable law, downstream licensing obligations,
   robots restrictions, privacy requirements, and content removal requests.

Refer to the Common Crawl Terms of Use for the governing terms for the crawl data itself.
`
}
