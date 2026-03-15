package arctic

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"time"
)

// LiveSection holds pre-formatted strings for the README live-progress block.
type LiveSection struct {
	Phase       string
	MonthType   string
	ShardLine   string
	StartedAt   string
	ElapsedStr  string
	ProgressBar string
	Committed   int
	Skipped     int
	TotalRows   string
	TotalBytes  string
	UpdatedAt   string
	Done        bool
	CompletedAt string
}

// ReadmeData holds all template variables for the Arctic README.
type ReadmeData struct {
	// Date range
	FirstMonth  string // "2005-12"
	LatestMonth string // "2026-02"

	// Aggregate totals
	CommentMonths    int
	SubmissionMonths int
	CommentRows      int64
	SubmissionRows   int64
	CommentSize      int64
	SubmissionSize   int64
	TotalRows        int64
	TotalSize        int64

	// Pre-formatted
	CommentRowsFmt    string
	SubmissionRowsFmt string
	CommentSizeFmt    string
	SubmissionSizeFmt string
	TotalRowsFmt      string
	TotalSizeFmt      string

	// Charts
	CommentsChart    string
	SubmissionsChart string

	// Metadata
	GeneratedAt string

	// Live session (nil when no active session)
	Live *LiveSection
}

// GenerateREADME generates the README without a live session section.
func GenerateREADME(rows []StatsRow) ([]byte, error) {
	return GenerateREADMEWithLive(rows, nil)
}

// GenerateREADMEWithLive generates the README, optionally embedding a live progress section.
func GenerateREADMEWithLive(rows []StatsRow, snap *StateSnapshot) ([]byte, error) {
	data := buildReadmeData(rows)
	if snap != nil {
		data.Live = buildLiveSection(snap)
	}
	tmpl, err := template.New("readme").Parse(readmeTmpl)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildReadmeData(rows []StatsRow) ReadmeData {
	d := ReadmeData{}
	d.GeneratedAt = time.Now().UTC().Format("2006-01-02 15:04 UTC")

	yearRows := make(map[int][2]int64) // year → [comments_rows, submissions_rows]

	for _, r := range rows {
		ym := fmt.Sprintf("%04d-%02d", r.Year, r.Month)
		if d.FirstMonth == "" || ym < d.FirstMonth {
			d.FirstMonth = ym
		}
		if ym > d.LatestMonth {
			d.LatestMonth = ym
		}
		yr := yearRows[r.Year]
		if r.Type == "comments" {
			d.CommentMonths++
			d.CommentRows += r.Count
			d.CommentSize += r.SizeBytes
			yr[0] += r.Count
		} else {
			d.SubmissionMonths++
			d.SubmissionRows += r.Count
			d.SubmissionSize += r.SizeBytes
			yr[1] += r.Count
		}
		yearRows[r.Year] = yr
	}
	if d.FirstMonth == "" {
		d.FirstMonth = "-"
	}
	if d.LatestMonth == "" {
		d.LatestMonth = "-"
	}

	d.TotalRows = d.CommentRows + d.SubmissionRows
	d.TotalSize = d.CommentSize + d.SubmissionSize

	d.CommentRowsFmt = fmtCount(d.CommentRows)
	d.SubmissionRowsFmt = fmtCount(d.SubmissionRows)
	d.CommentSizeFmt = fmtBytes(d.CommentSize)
	d.SubmissionSizeFmt = fmtBytes(d.SubmissionSize)
	d.TotalRowsFmt = fmtCount(d.TotalRows)
	d.TotalSizeFmt = fmtBytes(d.TotalSize)

	d.CommentsChart = buildTypeChart(yearRows, 0)
	d.SubmissionsChart = buildTypeChart(yearRows, 1)
	return d
}

func buildTypeChart(yearRows map[int][2]int64, idx int) string {
	if len(yearRows) == 0 {
		return "  (no data yet)"
	}
	years := make([]int, 0, len(yearRows))
	for y := range yearRows {
		years = append(years, y)
	}
	sort.Ints(years)

	var maxRows int64
	for _, yr := range yearRows {
		if yr[idx] > maxRows {
			maxRows = yr[idx]
		}
	}
	if maxRows == 0 {
		return "  (no data yet)"
	}

	const barWidth = 30
	var sb strings.Builder
	for _, y := range years {
		yr := yearRows[y]
		count := yr[idx]
		if count == 0 {
			continue
		}
		width := int(float64(count) / float64(maxRows) * float64(barWidth))
		if width < 1 {
			width = 1
		}
		bar := strings.Repeat("█", width) + strings.Repeat("░", barWidth-width)
		sb.WriteString(fmt.Sprintf("  %d  %s  %s\n", y, bar, fmtCount(count)))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func buildLiveSection(snap *StateSnapshot) *LiveSection {
	ls := &LiveSection{
		Phase:      snap.Phase,
		Committed:  snap.Stats.Committed,
		Skipped:    snap.Stats.Skipped,
		TotalRows:  fmtCount(snap.Stats.TotalRows),
		TotalBytes: fmtBytes(snap.Stats.TotalBytes),
		UpdatedAt:  snap.UpdatedAt.UTC().Format("2006-01-02 15:04 UTC"),
		StartedAt:  snap.StartedAt.UTC().Format("2006-01-02 15:04 UTC"),
		Done:       snap.Phase == PhaseDone,
	}

	elapsed := snap.UpdatedAt.Sub(snap.StartedAt)
	ls.ElapsedStr = fmtElapsed(elapsed)

	if snap.Phase == PhaseDone {
		ls.CompletedAt = snap.UpdatedAt.UTC().Format("2006-01-02 15:04 UTC")
	}

	if snap.Current != nil {
		c := snap.Current
		ls.MonthType = fmt.Sprintf("**%s** - %s", c.YM, c.Type)
		switch c.Phase {
		case PhaseDownloading:
			if c.BytesTotal > 0 {
				pct := int(100 * c.BytesDone / c.BytesTotal)
				ls.ShardLine = fmt.Sprintf("%s / %s (%d%%)",
					fmtBytes(c.BytesDone), fmtBytes(c.BytesTotal), pct)
			} else {
				ls.ShardLine = "connecting to peers…"
			}
		case PhaseProcessing:
			if c.Shard > 0 {
				ls.ShardLine = fmt.Sprintf("shard %d in progress · %s rows", c.Shard, fmtCount(c.Rows))
			}
		case PhaseCommitting:
			ls.ShardLine = "committing to Hugging Face…"
		}
	}

	// Progress bar — 30 chars wide.
	total := snap.Stats.TotalMonths
	done := snap.Stats.Committed + snap.Stats.Skipped
	if total > 0 {
		filled := int(float64(done) / float64(total) * 30)
		if filled > 30 {
			filled = 30
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", 30-filled)
		pct := float64(done) / float64(total) * 100
		ls.ProgressBar = fmt.Sprintf("`%s`  %d / %d (%.1f%%)", bar, done, total, pct)
	}

	return ls
}

func fmtElapsed(d time.Duration) string {
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func fmtCount(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1e9)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1e3)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func fmtBytes(n int64) string {
	switch {
	case n >= 1<<40:
		return fmt.Sprintf("%.1f TB", float64(n)/(1<<40))
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

const readmeTmpl = `---
configs:
- config_name: comments
  data_files:
  - split: train
    path: "data/comments/**/*.parquet"
- config_name: submissions
  data_files:
  - split: train
    path: "data/submissions/**/*.parquet"
license: other
language:
- en
tags:
- reddit
- social-media
- arctic-shift
- pushshift
- comments
- submissions
- parquet
- community
pretty_name: Arctic Shift Reddit Archive
size_categories:
- 1B<n<10B
task_categories:
- text-generation
- text-classification
- feature-extraction
---

# Arctic Shift Reddit Archive

> Every Reddit comment and submission since 2005, organized as monthly Parquet shards

## Table of Contents

- [What is it?](#what-is-it)
- [What is being released?](#what-is-being-released)
- [Breakdown by type and year](#breakdown-by-type-and-year)
- [How to download and use this dataset](#how-to-download-and-use-this-dataset)
- [Dataset statistics](#dataset-statistics){{if .Live}}
- [Pipeline status](#pipeline-status){{end}}
- [Dataset card](#dataset-card-for-arctic-shift-reddit-archive)
  - [Dataset summary](#dataset-summary)
  - [Dataset structure](#dataset-structure)
  - [Dataset creation](#dataset-creation)
  - [Considerations for using the data](#considerations-for-using-the-data)
- [Additional information](#additional-information)

## What is it?

The full Reddit archive from [Arctic Shift](https://github.com/ArthurHeitmann/arctic_shift), converted to Parquet and hosted here for easy access. Covers every public subreddit from **{{.FirstMonth}}** through **{{.LatestMonth}}**.

Right now the archive has **{{.TotalRowsFmt}} items** ({{.CommentRowsFmt}} comments, {{.SubmissionRowsFmt}} submissions) in **{{.TotalSizeFmt}}** of compressed Parquet. Comments and submissions are stored as separate datasets, split into monthly shards you can load individually or stream together.

Reddit has been around since 2005. Millions of people use it to talk about everything - programming, sports, cooking, politics, niche hobbies. That makes it one of the best sources of natural conversation data for language model training, sentiment analysis, community research, and information retrieval. Most Reddit datasets only cover specific subreddits or time windows. This one covers all of it.

## What is being released?

Monthly Parquet files, split by type (comments vs submissions). Small months fit in one shard. Large months (post-2015 or so) get split into multiple ~200 MB shards.

` + "```" + `
data/
  comments/
    2005/12/000.parquet       earliest month with data
    2006/01/000.parquet
    ...
    2023/06/000.parquet
              001.parquet     large months get multiple shards
              002.parquet
  submissions/
    2005/12/000.parquet
    2006/01/000.parquet
    ...
stats.csv                     one row per committed (month, type) pair
states.json                   live pipeline state (updated every ~5 min)
` + "```" + `

` + "`stats.csv`" + ` tracks every committed (month, type) pair with row count, shard count, file size, processing time, and commit timestamp.

## Breakdown by type and year

**Comments**

` + "```" + `
{{.CommentsChart}}
` + "```" + `

**Submissions**

` + "```" + `
{{.SubmissionsChart}}
` + "```" + `

## How to download and use this dataset

Load comments or submissions separately, filter by year or month, or stream the whole thing. Standard Hugging Face Parquet layout, works with DuckDB, ` + "`datasets`" + `, ` + "`pandas`" + `, and ` + "`huggingface_hub`" + ` out of the box.

### Using DuckDB

DuckDB reads Parquet directly from Hugging Face - no download step needed.

` + "```sql" + `
-- Top 20 subreddits by comment volume (all time)
SELECT subreddit, count(*) AS comments
FROM read_parquet('hf://datasets/open-index/arctic/data/comments/**/*.parquet')
GROUP BY subreddit
ORDER BY comments DESC
LIMIT 20;
` + "```" + `

` + "```sql" + `
-- Monthly submission volume for 2023
SELECT
    strftime(created_at, '%Y-%m') AS month,
    count(*) AS submissions,
    sum(num_comments) AS total_comments
FROM read_parquet('hf://datasets/open-index/arctic/data/submissions/2023/**/*.parquet')
GROUP BY month
ORDER BY month;
` + "```" + `

` + "```sql" + `
-- Most active authors across all comments
SELECT author, count(*) AS comments, avg(score) AS avg_score
FROM read_parquet('hf://datasets/open-index/arctic/data/comments/**/*.parquet')
WHERE author != '[deleted]'
GROUP BY author
ORDER BY comments DESC
LIMIT 20;
` + "```" + `

` + "```sql" + `
-- Average comment length by year
SELECT
    extract(year FROM created_at) AS year,
    avg(body_length) AS avg_length,
    count(*) AS comments
FROM read_parquet('hf://datasets/open-index/arctic/data/comments/**/*.parquet')
GROUP BY year
ORDER BY year;
` + "```" + `

` + "```sql" + `
-- Top linked domains in submissions
SELECT
    regexp_extract(url, 'https?://([^/]+)', 1) AS domain,
    count(*) AS posts
FROM read_parquet('hf://datasets/open-index/arctic/data/submissions/**/*.parquet')
WHERE url IS NOT NULL AND url != ''
GROUP BY domain
ORDER BY posts DESC
LIMIT 20;
` + "```" + `

### Using ` + "`datasets`" + `

` + "```python" + `
from datasets import load_dataset

# Stream all comments without downloading everything
comments = load_dataset("open-index/arctic", "comments", split="train", streaming=True)
for item in comments:
    print(item["author"], item["subreddit"], item["body"][:80])

# Load submissions for a specific year
subs = load_dataset(
    "open-index/arctic", "submissions",
    data_files="data/submissions/2023/**/*.parquet",
    split="train",
)
print(f"{len(subs):,} submissions in 2023")
` + "```" + `

### Using ` + "`huggingface_hub`" + `

` + "```python" + `
from huggingface_hub import snapshot_download

# Download only 2023 comments
snapshot_download(
    "open-index/arctic",
    repo_type="dataset",
    local_dir="./arctic/",
    allow_patterns="data/comments/2023/**/*",
)
` + "```" + `

For faster downloads, install ` + "`pip install huggingface_hub[hf_transfer]`" + ` and set ` + "`HF_HUB_ENABLE_HF_TRANSFER=1`" + `.

### Using the CLI

` + "```bash" + `
# Download a single month of submissions
huggingface-cli download open-index/arctic \
    --include "data/submissions/2024/01/*" \
    --repo-type dataset --local-dir ./arctic/
` + "```" + `

## Dataset statistics

| Type | Months | Rows | Parquet Size |
|------|-------:|-----:|-------------:|
| comments | {{.CommentMonths}} | {{.CommentRowsFmt}} | {{.CommentSizeFmt}} |
| submissions | {{.SubmissionMonths}} | {{.SubmissionRowsFmt}} | {{.SubmissionSizeFmt}} |
| **Total** | **{{.CommentMonths}}** | **{{.TotalRowsFmt}}** | **{{.TotalSizeFmt}}** |

Query per-month stats directly:

` + "```sql" + `
SELECT year, month, type, shards, count, size_bytes
FROM read_csv_auto('hf://datasets/open-index/arctic/stats.csv')
ORDER BY year, month, type;
` + "```" + `

` + "`stats.csv`" + ` columns:

| Column | Description |
|--------|-------------|
| ` + "`year`" + `, ` + "`month`" + ` | Calendar month |
| ` + "`type`" + ` | ` + "`comments`" + ` or ` + "`submissions`" + ` |
| ` + "`shards`" + ` | Number of Parquet files for this (month, type) |
| ` + "`count`" + ` | Total rows across all shards |
| ` + "`size_bytes`" + ` | Total Parquet size across all shards |
| ` + "`dur_download_s`" + ` | Seconds to download the .zst source |
| ` + "`dur_process_s`" + ` | Seconds to decompress and convert to Parquet |
| ` + "`dur_commit_s`" + ` | Seconds to commit to Hugging Face |
| ` + "`committed_at`" + ` | ISO 8601 timestamp when this pair was committed |
{{if .Live}}
{{if .Live.Done}}
## Pipeline Complete

**Completed:** {{.Live.CompletedAt}} / **Duration:** {{.Live.ElapsedStr}}

| Metric | Result |
|--------|-------:|
| Months committed | {{.Live.Committed}} |
| Months skipped | {{.Live.Skipped}} |
| Rows processed | {{.Live.TotalRows}} |
| Data committed | {{.Live.TotalBytes}} |
{{else}}
## Pipeline Status

> The ingestion pipeline is running. This section updates every ~5 minutes.

**Started:** {{.Live.StartedAt}} / **Elapsed:** {{.Live.ElapsedStr}} / **Committed this session:** {{.Live.Committed}}
{{if .Live.MonthType}}
| | |
|:---|:---|
| Phase | {{.Live.Phase}} |
| Month | {{.Live.MonthType}} |{{if .Live.ShardLine}}
| Progress | {{.Live.ShardLine}} |{{end}}
{{end}}
{{.Live.ProgressBar}}

| Metric | This Session |
|--------|-------------:|
| Months committed | {{.Live.Committed}} |
| Rows processed | {{.Live.TotalRows}} |
| Data committed | {{.Live.TotalBytes}} |

*Last update: {{.Live.UpdatedAt}}*
{{end}}{{end}}

# Dataset card for Arctic Shift Reddit Archive

## Dataset summary

A repackaging of the [Arctic Shift](https://github.com/ArthurHeitmann/arctic_shift) monthly Reddit dumps into Parquet. Arctic Shift re-processes the [PushShift](https://pushshift.io) Reddit archive, which captured most public Reddit content from the early days through the 2023 API changes.

Covers every public subreddit, every month, both comments and submissions. Built for research, analysis, and training. People use it for:

- **Language model pretraining and fine-tuning** - one of the largest sources of natural conversation on the internet
- **Sentiment and trend analysis** - two decades of public opinion on just about everything
- **Community research** - thousands of subreddits, each with its own culture and moderation norms
- **Information retrieval** - real questions and answers from r/AskReddit, r/explainlikeimfive, and others
- **Content moderation research** - moderation signals are preserved in the data

## Dataset structure

### Data instances

Example comment:

` + "```json" + `
{
  "id": "c0001",
  "author": "spez",
  "subreddit": "reddit.com",
  "body": "Welcome to Reddit!",
  "score": 42,
  "created_utc": 1134028003,
  "created_at": "2005-12-08T10:06:43",
  "body_length": 19,
  "link_id": "t3_17",
  "parent_id": "t3_17",
  "distinguished": null,
  "author_flair_text": null
}
` + "```" + `

Example submission:

` + "```json" + `
{
  "id": "abc123",
  "author": "kn0thing",
  "subreddit": "reddit.com",
  "title": "The Downing Street Memo",
  "selftext": "",
  "score": 15,
  "created_utc": 1118895720,
  "created_at": "2005-06-16T01:02:00",
  "title_length": 23,
  "num_comments": 3,
  "url": "http://www.timesonline.co.uk/...",
  "over_18": false,
  "link_flair_text": null,
  "author_flair_text": null
}
` + "```" + `

### Data fields

#### Comments (` + "`data/comments/YYYY/MM/NNN.parquet`" + `)

| Column | Type | Description |
|--------|------|-------------|
| ` + "`id`" + ` | VARCHAR | Reddit's base-36 comment ID |
| ` + "`author`" + ` | VARCHAR | Username. ` + "`[deleted]`" + ` if account was removed |
| ` + "`subreddit`" + ` | VARCHAR | Subreddit name (no ` + "`r/`" + ` prefix) |
| ` + "`body`" + ` | VARCHAR | Comment text in Markdown |
| ` + "`score`" + ` | BIGINT | Net upvotes at time of archival |
| ` + "`created_utc`" + ` | BIGINT | Unix timestamp |
| ` + "`created_at`" + ` | TIMESTAMP | Derived from ` + "`created_utc`" + ` |
| ` + "`body_length`" + ` | BIGINT | Character count of ` + "`body`" + ` |
| ` + "`link_id`" + ` | VARCHAR | Parent submission ID (` + "`t3_...`" + ` format) |
| ` + "`parent_id`" + ` | VARCHAR | Parent comment or submission ID |
| ` + "`distinguished`" + ` | VARCHAR | ` + "`moderator`" + `, ` + "`admin`" + `, or null |
| ` + "`author_flair_text`" + ` | VARCHAR | Author's flair in this subreddit |

#### Submissions (` + "`data/submissions/YYYY/MM/NNN.parquet`" + `)

| Column | Type | Description |
|--------|------|-------------|
| ` + "`id`" + ` | VARCHAR | Reddit's base-36 submission ID |
| ` + "`author`" + ` | VARCHAR | Username of the poster |
| ` + "`subreddit`" + ` | VARCHAR | Subreddit name |
| ` + "`title`" + ` | VARCHAR | Post title |
| ` + "`selftext`" + ` | VARCHAR | Post body for text posts (empty for link posts) |
| ` + "`score`" + ` | BIGINT | Net upvotes at time of archival |
| ` + "`created_utc`" + ` | BIGINT | Unix timestamp |
| ` + "`created_at`" + ` | TIMESTAMP | Derived from ` + "`created_utc`" + ` |
| ` + "`title_length`" + ` | BIGINT | Character count of ` + "`title`" + ` |
| ` + "`num_comments`" + ` | BIGINT | Comment count on this post |
| ` + "`url`" + ` | VARCHAR | External URL for link posts, permalink for text posts |
| ` + "`over_18`" + ` | BOOLEAN | NSFW flag |
| ` + "`link_flair_text`" + ` | VARCHAR | Post flair text |
| ` + "`author_flair_text`" + ` | VARCHAR | Author's flair |

### Data splits

Two named configs: ` + "`comments`" + ` and ` + "`submissions`" + `. Each loads all monthly shards as a single ` + "`train`" + ` split.

You can also load specific years or months with ` + "`data_files`" + `:

` + "```python" + `
# Load just January 2020 comments
ds = load_dataset("open-index/arctic", data_files="data/comments/2020/01/*.parquet", split="train")

# Load all 2023 submissions
ds = load_dataset("open-index/arctic", data_files="data/submissions/2023/**/*.parquet", split="train")
` + "```" + `

## Dataset creation

### Why we built this

Reddit is one of the best sources of real human conversation on the internet, but getting at the full archive got a lot harder after Reddit locked down API access in 2023. The Arctic Shift project preserves the data as monthly .zst JSONL dumps. We convert those dumps to Parquet on Hugging Face so you can query with DuckDB, stream with ` + "`datasets`" + `, or bulk download without any special tooling.

### Source data

Everything comes from [Arctic Shift](https://github.com/ArthurHeitmann/arctic_shift) torrent archives, which re-process the [PushShift](https://pushshift.io) Reddit dumps. Source format is .zst-compressed JSONL, one JSON object per line.

- **2005-12 through 2023-12:** From the Arctic Shift bundle torrent
- **2024-01 onward:** Individual monthly torrents from Arctic Shift

### Processing steps

The pipeline is written in Go and uses [DuckDB](https://duckdb.org) for the Parquet conversion. For each (month, type) pair:

1. **Download** the .zst via BitTorrent with selective file priority (only the needed file from the bundle, not the whole archive)
2. **Stream** through a [klauspost/compress](https://github.com/klauspost/compress) zstd decoder with a 2 GB window
3. **Chunk** the JSONL into ~2 million line batches, writing each to a temp file
4. **Convert** each chunk to Parquet with DuckDB ` + "`read_json_auto`" + `, explicit column selection, ` + "`TRY_CAST`" + `, Zstandard compression, 131K-row row groups
5. **Delete** each temp chunk right after the shard is written (disk is tight)
6. **Commit** all shards plus updated ` + "`stats.csv`" + ` and ` + "`README.md`" + ` to Hugging Face
7. **Clean up** local shards after the commit goes through

The pipeline picks up where it left off - ` + "`stats.csv`" + ` tracks what has been committed, and those pairs get skipped on restart. Disk usage stays minimal: at most one .zst, one JSONL chunk, and the current month's shards on disk at a time.

No filtering, deduplication, or content changes. The data matches the Arctic Shift dumps exactly. All Parquet files use Zstandard compression.

### Personal and sensitive information

Usernames and user-generated text are included as they appeared publicly on Reddit. Deleted accounts show as ` + "`[deleted]`" + `, deleted content as ` + "`[removed]`" + `.

No PII scrubbing has been done. At this scale, the dataset almost certainly contains personal information that people posted publicly. If you find something that should be removed, open a discussion on the [Community tab](https://huggingface.co/datasets/open-index/arctic/discussions).

## Considerations for using the data

### Social impact

Making the full Reddit archive accessible in a standard format should help researchers study how online communities work, how language changes over time, and how one of the internet's biggest platforms has shaped public discourse.

### Biases

Reddit skews young, male, English-speaking, and North American/European. Subreddits vary wildly in culture, moderation, and toxicity. The voting system amplifies what each community already agrees with.

We did not filter, score, or assess the data in any way. Controversial, toxic, and NSFW content is all in there. Apply your own filtering for your use case.

### Known limitations

- **Completeness depends on PushShift.** PushShift missed some content, especially in the earliest months and during ingestion outages.
- **Scores are snapshots.** The ` + "`score`" + ` field is whatever PushShift captured at the time, not the final score.
- **Deleted content.** Posts deleted before PushShift got to them are gone. Posts deleted after capture may still have the original text.
- **No user profiles.** Just posts and comments. No karma, no account metadata.
- **Markdown and HTML.** Comment bodies use Reddit's Markdown variant. Some old content has raw HTML.

## Additional information

### Licensing

Reddit content is subject to [Reddit's Terms of Service](https://www.redditinc.com/policies/user-agreement). Arctic Shift distributes the archive under permissive research terms. This repackaging is provided as-is for research and education.

Not affiliated with or endorsed by Reddit, Inc. or Arctic Shift.

### Thanks

All the data here comes from [Arctic Shift](https://github.com/ArthurHeitmann/arctic_shift), which preserves and distributes the [PushShift](https://pushshift.io) Reddit archive through Academic Torrents. None of this would be practical without their work.

### Contact

Questions, feedback, or issues - open a discussion on the [Community tab](https://huggingface.co/datasets/open-index/arctic/discussions).

*Last updated: {{.GeneratedAt}}*
`
