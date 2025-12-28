package git

import (
	"bufio"
	"context"
	"strconv"
	"strings"
	"time"
)

// Commit represents a git commit
type Commit struct {
	SHA       string    `json:"sha"`
	ShortSHA  string    `json:"short_sha"`
	Message   string    `json:"message"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Author    Author    `json:"author"`
	Committer Author    `json:"committer"`
	Parents   []string  `json:"parents"`
	CreatedAt time.Time `json:"created_at"`
}

// Author represents a commit author or committer
type Author struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

// FileLastCommit holds the last commit info for a file
type FileLastCommit struct {
	Path        string  `json:"path"`
	Commit      *Commit `json:"commit"`
	RelativeAge string  `json:"relative_age"`
}

// GetCommit retrieves a commit by SHA
func (r *Repository) GetCommit(ctx context.Context, sha string) (*Commit, error) {
	resolved, err := r.ResolveRef(ctx, sha)
	if err != nil {
		return nil, err
	}

	// Get commit info using a custom format
	// Format: SHA%x00ParentSHAs%x00AuthorName%x00AuthorEmail%x00AuthorDate%x00CommitterName%x00CommitterEmail%x00CommitterDate%x00Subject%x00Body
	format := "%H%x00%P%x00%an%x00%ae%x00%at%x00%cn%x00%ce%x00%ct%x00%s%x00%b"
	out, err := r.git(ctx, "log", "-1", "--format="+format, resolved)
	if err != nil {
		return nil, ErrNotFound
	}

	return parseCommit(string(out))
}

// GetLatestCommit retrieves the latest commit for a ref
func (r *Repository) GetLatestCommit(ctx context.Context, ref string) (*Commit, error) {
	sha, err := r.ResolveRef(ctx, ref)
	if err != nil {
		return nil, err
	}
	return r.GetCommit(ctx, sha)
}

// GetCommitHistory retrieves commit history for a ref
func (r *Repository) GetCommitHistory(ctx context.Context, ref string, limit int) ([]*Commit, error) {
	if limit <= 0 {
		limit = 30
	}

	sha, err := r.ResolveRef(ctx, ref)
	if err != nil {
		return nil, err
	}

	format := "%H%x00%P%x00%an%x00%ae%x00%at%x00%cn%x00%ce%x00%ct%x00%s%x00%b%x00"
	out, err := r.git(ctx, "log", "-n", strconv.Itoa(limit), "--format="+format, sha)
	if err != nil {
		return nil, err
	}

	return parseCommits(string(out))
}

// GetFileLastCommit retrieves the last commit that modified a file
func (r *Repository) GetFileLastCommit(ctx context.Context, ref, path string) (*FileLastCommit, error) {
	if !IsValidPath(path) {
		return nil, ErrPathTraversal
	}

	sha, err := r.ResolveRef(ctx, ref)
	if err != nil {
		return nil, err
	}

	format := "%H%x00%P%x00%an%x00%ae%x00%at%x00%cn%x00%ce%x00%ct%x00%s%x00%b"
	out, err := r.git(ctx, "log", "-1", "--format="+format, sha, "--", path)
	if err != nil {
		return nil, ErrNotFound
	}

	commit, err := parseCommit(string(out))
	if err != nil {
		return nil, err
	}

	return &FileLastCommit{
		Path:        path,
		Commit:      commit,
		RelativeAge: relativeTime(commit.CreatedAt),
	}, nil
}

// GetTreeLastCommits retrieves the last commit for each entry in a tree
func (r *Repository) GetTreeLastCommits(ctx context.Context, ref, path string) (map[string]*FileLastCommit, error) {
	tree, err := r.GetTree(ctx, ref, path)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*FileLastCommit)

	for _, entry := range tree.Entries {
		lc, err := r.GetFileLastCommit(ctx, ref, entry.Path)
		if err == nil {
			result[entry.Name] = lc
		}
	}

	return result, nil
}

// GetCommitCount returns the total number of commits in a ref's history
func (r *Repository) GetCommitCount(ctx context.Context, ref string) (int, error) {
	sha, err := r.ResolveRef(ctx, ref)
	if err != nil {
		return 0, err
	}

	out, err := r.git(ctx, "rev-list", "--count", sha)
	if err != nil {
		return 0, err
	}

	count, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return count, nil
}

// GetContributorCount returns the number of unique contributors
func (r *Repository) GetContributorCount(ctx context.Context) (int, error) {
	out, err := r.git(ctx, "shortlog", "-sn", "--all")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return len(lines), nil
}

// parseCommit parses a single commit from formatted output
func parseCommit(s string) (*Commit, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrNotFound
	}

	parts := strings.Split(s, "\x00")
	if len(parts) < 9 {
		return nil, ErrNotFound
	}

	authorTime, _ := strconv.ParseInt(parts[4], 10, 64)
	committerTime, _ := strconv.ParseInt(parts[7], 10, 64)

	commit := &Commit{
		SHA:      parts[0],
		ShortSHA: shortSHA(parts[0]),
		Author: Author{
			Name:  parts[2],
			Email: parts[3],
			Date:  time.Unix(authorTime, 0),
		},
		Committer: Author{
			Name:  parts[5],
			Email: parts[6],
			Date:  time.Unix(committerTime, 0),
		},
		Title:     parts[8],
		CreatedAt: time.Unix(authorTime, 0),
	}

	if len(parts) > 9 {
		commit.Body = strings.TrimSpace(parts[9])
	}

	commit.Message = commit.Title
	if commit.Body != "" {
		commit.Message = commit.Title + "\n\n" + commit.Body
	}

	// Parse parents
	if parts[1] != "" {
		commit.Parents = strings.Fields(parts[1])
	}

	return commit, nil
}

// parseCommits parses multiple commits from formatted output
func parseCommits(s string) ([]*Commit, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	// Split by null byte at end of each commit
	records := strings.Split(s, "\x00\x00")
	commits := make([]*Commit, 0, len(records))

	for _, record := range records {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}
		// Add back the terminating null that was removed by split
		commit, err := parseCommit(record + "\x00")
		if err == nil {
			commits = append(commits, commit)
		}
	}

	return commits, nil
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// relativeTime formats a time as a relative string like "2 days ago"
func relativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return strconv.Itoa(mins) + " minutes ago"
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return strconv.Itoa(hours) + " hours ago"
	case diff < 48*time.Hour:
		return "yesterday"
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return strconv.Itoa(days) + " days ago"
	case diff < 14*24*time.Hour:
		return "last week"
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		return strconv.Itoa(weeks) + " weeks ago"
	case diff < 60*24*time.Hour:
		return "last month"
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return strconv.Itoa(months) + " months ago"
	default:
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return strconv.Itoa(years) + " years ago"
	}
}

// BlameHunk represents a section of a file with attribution
type BlameHunk struct {
	StartLine int      `json:"start_line"`
	EndLine   int      `json:"end_line"`
	Commit    *Commit  `json:"commit"`
	Lines     []string `json:"lines"`
}

// GetBlame retrieves blame information for a file
func (r *Repository) GetBlame(ctx context.Context, ref, path string) ([]*BlameHunk, error) {
	if !IsValidPath(path) {
		return nil, ErrPathTraversal
	}

	sha, err := r.ResolveRef(ctx, ref)
	if err != nil {
		return nil, err
	}

	out, err := r.git(ctx, "blame", "--porcelain", sha, "--", path)
	if err != nil {
		return nil, ErrNotFound
	}

	return parseBlame(ctx, r, string(out))
}

// parseBlame parses git blame --porcelain output
func parseBlame(ctx context.Context, repo *Repository, s string) ([]*BlameHunk, error) {
	hunks := make([]*BlameHunk, 0)
	commitCache := make(map[string]*Commit)

	scanner := bufio.NewScanner(strings.NewReader(s))
	var currentHunk *BlameHunk
	var currentSHA string

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this is a commit line (starts with SHA)
		if len(line) >= 40 && isHexString(line[:40]) {
			parts := strings.Fields(line)
			sha := parts[0]

			if sha != currentSHA {
				// Start new hunk
				if currentHunk != nil {
					hunks = append(hunks, currentHunk)
				}

				commit, ok := commitCache[sha]
				if !ok {
					commit, _ = repo.GetCommit(ctx, sha)
					commitCache[sha] = commit
				}

				currentHunk = &BlameHunk{
					StartLine: len(hunks) + 1,
					Commit:    commit,
					Lines:     make([]string, 0),
				}
				currentSHA = sha
			}
		} else if strings.HasPrefix(line, "\t") && currentHunk != nil {
			// Content line (starts with tab)
			currentHunk.Lines = append(currentHunk.Lines, line[1:])
			currentHunk.EndLine = currentHunk.StartLine + len(currentHunk.Lines) - 1
		}
	}

	if currentHunk != nil {
		hunks = append(hunks, currentHunk)
	}

	return hunks, nil
}

func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
