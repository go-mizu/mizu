package kaggle

import (
	"fmt"
	"net/url"
	"strings"
)

func NormalizeDatasetURL(input string) string {
	ref := ExtractDatasetRef(input)
	if ref == "" {
		return input
	}
	return BaseURL + "/datasets/" + ref
}

func ExtractDatasetRef(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		s = strings.TrimPrefix(strings.TrimPrefix(s, "/"), "datasets/")
		if parts := strings.Split(s, "/"); len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		return ""
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	parts := splitPath(u.Path)
	if len(parts) >= 3 && parts[0] == "datasets" {
		return parts[1] + "/" + parts[2]
	}
	return ""
}

func NormalizeModelURL(input string) string {
	ref := ExtractModelRef(input)
	if ref == "" {
		return input
	}
	return BaseURL + "/models/" + ref
}

func ExtractModelRef(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		s = strings.TrimPrefix(strings.TrimPrefix(s, "/"), "models/")
		if parts := strings.Split(s, "/"); len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		return ""
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	parts := splitPath(u.Path)
	if len(parts) >= 3 && parts[0] == "models" {
		return parts[1] + "/" + parts[2]
	}
	return ""
}

func NormalizeCompetitionURL(input string) string {
	slug := ExtractCompetitionSlug(input)
	if slug == "" {
		return input
	}
	return BaseURL + "/competitions/" + slug
}

func ExtractCompetitionSlug(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		s = strings.TrimPrefix(strings.TrimPrefix(s, "/"), "competitions/")
		return strings.Trim(s, "/")
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	parts := splitPath(u.Path)
	if len(parts) >= 2 && parts[0] == "competitions" {
		return parts[1]
	}
	return ""
}

func NormalizeProfileURL(input string) string {
	handle := ExtractProfileHandle(input)
	if handle == "" {
		return input
	}
	return BaseURL + "/" + handle
}

func ExtractProfileHandle(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		s = strings.TrimPrefix(s, "@")
		s = strings.Trim(strings.TrimPrefix(s, "/"), "/")
		if s == "" || strings.Contains(s, "/") {
			return ""
		}
		switch s {
		case "datasets", "models", "competitions", "code":
			return ""
		default:
			return s
		}
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	parts := splitPath(u.Path)
	if len(parts) == 1 {
		switch parts[0] {
		case "datasets", "models", "competitions", "code":
			return ""
		default:
			return parts[0]
		}
	}
	return ""
}

func NormalizeNotebookURL(input string) string {
	ref := ExtractNotebookRef(input)
	if ref == "" {
		return input
	}
	return BaseURL + "/code/" + ref
}

func ExtractNotebookRef(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		s = strings.TrimPrefix(strings.TrimPrefix(s, "/"), "code/")
		if parts := strings.Split(s, "/"); len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		return ""
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	parts := splitPath(u.Path)
	if len(parts) >= 3 && parts[0] == "code" {
		return parts[1] + "/" + parts[2]
	}
	return ""
}

func splitPath(p string) []string {
	raw := strings.Split(strings.Trim(p, "/"), "/")
	out := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func splitOwnerSlug(ref string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(ref), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid owner/slug: %q", ref)
	}
	return parts[0], parts[1], nil
}
