package git

import (
	"bufio"
	"context"
	"strings"
)

// Reference represents a branch or tag
type Reference struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // "branch" or "tag"
	SHA       string `json:"sha"`
	IsDefault bool   `json:"is_default"`
}

// ListBranches returns all branches in the repository
func (r *Repository) ListBranches(ctx context.Context) ([]*Reference, error) {
	defaultBranch, _ := r.GetDefaultBranch(ctx)

	out, err := r.git(ctx, "for-each-ref", "--format=%(refname:short) %(objectname)", "refs/heads/")
	if err != nil {
		return nil, err
	}

	refs := make([]*Reference, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		sha := parts[1]

		refs = append(refs, &Reference{
			Name:      name,
			Type:      "branch",
			SHA:       sha,
			IsDefault: name == defaultBranch,
		})
	}

	// Sort: default branch first, then alphabetically
	sortRefs(refs)

	return refs, nil
}

// ListTags returns all tags in the repository
func (r *Repository) ListTags(ctx context.Context) ([]*Reference, error) {
	out, err := r.git(ctx, "for-each-ref", "--format=%(refname:short) %(objectname)", "refs/tags/")
	if err != nil {
		return nil, err
	}

	refs := make([]*Reference, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		sha := parts[1]

		refs = append(refs, &Reference{
			Name: name,
			Type: "tag",
			SHA:  sha,
		})
	}

	// Sort by version (reverse order, newest first)
	sortTagsDesc(refs)

	return refs, nil
}

// GetBranch returns a specific branch by name
func (r *Repository) GetBranch(ctx context.Context, name string) (*Reference, error) {
	out, err := r.git(ctx, "rev-parse", "--verify", "refs/heads/"+name)
	if err != nil {
		return nil, ErrNotFound
	}

	sha := strings.TrimSpace(string(out))
	defaultBranch, _ := r.GetDefaultBranch(ctx)

	return &Reference{
		Name:      name,
		Type:      "branch",
		SHA:       sha,
		IsDefault: name == defaultBranch,
	}, nil
}

// GetTag returns a specific tag by name
func (r *Repository) GetTag(ctx context.Context, name string) (*Reference, error) {
	out, err := r.git(ctx, "rev-parse", "--verify", "refs/tags/"+name)
	if err != nil {
		return nil, ErrNotFound
	}

	sha := strings.TrimSpace(string(out))

	return &Reference{
		Name: name,
		Type: "tag",
		SHA:  sha,
	}, nil
}

// BranchCount returns the number of branches
func (r *Repository) BranchCount(ctx context.Context) (int, error) {
	out, err := r.git(ctx, "for-each-ref", "--format=%(refname)", "refs/heads/")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0, nil
	}
	return len(lines), nil
}

// TagCount returns the number of tags
func (r *Repository) TagCount(ctx context.Context) (int, error) {
	out, err := r.git(ctx, "for-each-ref", "--format=%(refname)", "refs/tags/")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0, nil
	}
	return len(lines), nil
}

// sortRefs sorts references: default branch first, then alphabetically
func sortRefs(refs []*Reference) {
	n := len(refs)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if shouldSwapRef(refs[j], refs[j+1]) {
				refs[j], refs[j+1] = refs[j+1], refs[j]
			}
		}
	}
}

func shouldSwapRef(a, b *Reference) bool {
	// Default branch comes first
	if a.IsDefault != b.IsDefault {
		return !a.IsDefault
	}
	// Alphabetical
	return strings.ToLower(a.Name) > strings.ToLower(b.Name)
}

// sortTagsDesc sorts tags in descending order (newest/highest version first)
func sortTagsDesc(refs []*Reference) {
	n := len(refs)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if compareVersions(refs[j].Name, refs[j+1].Name) < 0 {
				refs[j], refs[j+1] = refs[j+1], refs[j]
			}
		}
	}
}

// compareVersions compares two version strings
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func compareVersions(a, b string) int {
	// Strip 'v' prefix if present
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		var numA, numB int

		if i < len(partsA) {
			numA = parseVersionPart(partsA[i])
		}
		if i < len(partsB) {
			numB = parseVersionPart(partsB[i])
		}

		if numA < numB {
			return -1
		}
		if numA > numB {
			return 1
		}
	}

	return 0
}

func parseVersionPart(s string) int {
	// Extract leading number from version part
	num := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		} else {
			break
		}
	}
	return num
}
