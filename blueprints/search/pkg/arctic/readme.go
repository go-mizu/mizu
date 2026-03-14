package arctic

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"time"
)

type ReadmeData struct {
	LatestMonth      string
	CommentMonths    int
	SubmissionMonths int
	CommentRows      int64
	SubmissionRows   int64
	CommentSize      int64
	SubmissionSize   int64
	GrowthChart      string
	GeneratedAt      string
}

func GenerateREADME(rows []StatsRow) ([]byte, error) {
	data := buildReadmeData(rows)
	funcMap := template.FuncMap{
		"commentRows":    func(d ReadmeData) string { return fmtCount(d.CommentRows) },
		"commentSize":    func(d ReadmeData) string { return fmtBytes(d.CommentSize) },
		"submissionRows": func(d ReadmeData) string { return fmtCount(d.SubmissionRows) },
		"submissionSize": func(d ReadmeData) string { return fmtBytes(d.SubmissionSize) },
	}
	tmpl, err := template.New("readme").Funcs(funcMap).Parse(readmeTmpl)
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
	var data ReadmeData
	data.GeneratedAt = time.Now().UTC().Format("2006-01-02")

	yearRows := make(map[int][2]int64) // year → [comments_rows, submissions_rows]
	latestYM := ""

	for _, r := range rows {
		ym := fmt.Sprintf("%04d-%02d", r.Year, r.Month)
		if ym > latestYM {
			latestYM = ym
		}
		yr := yearRows[r.Year]
		if r.Type == "comments" {
			data.CommentMonths++
			data.CommentRows += r.Count
			data.CommentSize += r.SizeBytes
			yr[0] += r.Count
		} else {
			data.SubmissionMonths++
			data.SubmissionRows += r.Count
			data.SubmissionSize += r.SizeBytes
			yr[1] += r.Count
		}
		yearRows[r.Year] = yr
	}
	if latestYM == "" {
		latestYM = "—"
	}
	data.LatestMonth = latestYM
	data.GrowthChart = buildGrowthChart(yearRows)
	return data
}

func buildGrowthChart(yearRows map[int][2]int64) string {
	if len(yearRows) == 0 {
		return ""
	}
	years := make([]int, 0, len(yearRows))
	for y := range yearRows {
		years = append(years, y)
	}
	sort.Ints(years)

	var maxRows int64
	for _, yr := range yearRows {
		if yr[0]+yr[1] > maxRows {
			maxRows = yr[0] + yr[1]
		}
	}
	if maxRows == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("```\n")
	for _, y := range years {
		yr := yearRows[y]
		total := yr[0] + yr[1]
		barLen := int(float64(total) / float64(maxRows) * 40)
		if barLen < 1 && total > 0 {
			barLen = 1
		}
		bar := strings.Repeat("█", barLen)
		sb.WriteString(fmt.Sprintf("%d  %-40s  %s\n", y, bar, fmtCount(total)))
	}
	sb.WriteString("```")
	return sb.String()
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
pretty_name: Arctic Shift Reddit Archive
size_categories:
- 100B<n<1T
---

# Arctic Shift Reddit Archive

Full Reddit dataset (comments + submissions) sourced from the
[Arctic Shift](https://github.com/ArthurHeitmann/arctic_shift) project,
covering all subreddits from 2005-06 through **{{.LatestMonth}}**.

Data is organized as monthly parquet shards by type, making it easy to load
specific time ranges or work with comments and submissions independently.

## Quick Start

` + "```python" + `
from datasets import load_dataset

# Stream all comments (recommended — dataset is very large)
comments = load_dataset("open-index/arctic", "comments", streaming=True)
for item in comments["train"]:
    print(item["author"], item["body"][:80])

# Load submissions for a specific year
subs = load_dataset("open-index/arctic", "submissions",
                    data_files="data/submissions/2020/**/*.parquet")
` + "```" + `

## Dataset Stats

| Type        | Months | Rows | Parquet Size |
|-------------|--------|------|--------------|
| comments    | {{.CommentMonths}} | {{commentRows .}} | {{commentSize .}} |
| submissions | {{.SubmissionMonths}} | {{submissionRows .}} | {{submissionSize .}} |

*Updated: {{.GeneratedAt}}*

## Growth (rows per year, comments + submissions combined)

{{.GrowthChart}}

## Schema

### Comments

| Column | Type | Description |
|--------|------|-------------|
| id | VARCHAR | Comment ID |
| author | VARCHAR | Username |
| subreddit | VARCHAR | Subreddit name |
| body | VARCHAR | Comment text |
| score | BIGINT | Net upvotes |
| created_utc | BIGINT | Unix timestamp |
| created_at | TIMESTAMP | Derived from created_utc |
| body_length | BIGINT | Character count of body |
| link_id | VARCHAR | Parent submission ID |
| parent_id | VARCHAR | Parent comment or submission ID |
| distinguished | VARCHAR | mod/admin/null |
| author_flair_text | VARCHAR | Author flair |

### Submissions

| Column | Type | Description |
|--------|------|-------------|
| id | VARCHAR | Submission ID |
| author | VARCHAR | Username |
| subreddit | VARCHAR | Subreddit name |
| title | VARCHAR | Post title |
| selftext | VARCHAR | Post body (self posts) |
| score | BIGINT | Net upvotes |
| created_utc | BIGINT | Unix timestamp |
| created_at | TIMESTAMP | Derived from created_utc |
| title_length | BIGINT | Character count of title |
| num_comments | BIGINT | Comment count |
| url | VARCHAR | External URL or permalink |
| over_18 | BOOLEAN | NSFW flag |
| link_flair_text | VARCHAR | Post flair |
| author_flair_text | VARCHAR | Author flair |

## Source & License

Repackaged from [Arctic Shift](https://github.com/ArthurHeitmann/arctic_shift) monthly dumps,
which re-process the [PushShift](https://pushshift.io) Reddit archive.
Original content by Reddit users.
`
