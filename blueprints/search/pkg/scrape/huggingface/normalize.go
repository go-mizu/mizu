package huggingface

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

func NormalizeModelRef(ref string) (string, string, error) {
	return normalizeRepoRef(ref, "")
}

func NormalizeDatasetRef(ref string) (string, string, error) {
	return normalizeRepoRef(ref, "datasets")
}

func NormalizeSpaceRef(ref string) (string, string, error) {
	return normalizeRepoRef(ref, "spaces")
}

func NormalizeCollectionRef(ref string) (string, string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", fmt.Errorf("empty collection ref")
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		u, err := url.Parse(ref)
		if err != nil {
			return "", "", err
		}
		parts := splitPath(u.Path)
		if len(parts) < 3 || parts[0] != "collections" {
			return "", "", fmt.Errorf("invalid collection URL: %s", ref)
		}
		slug := parts[1] + "/" + parts[2]
		return slug, canonicalCollectionURL(slug), nil
	}
	ref = strings.TrimPrefix(ref, "/collections/")
	parts := splitPath(ref)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("collection ref must look like namespace/slug")
	}
	slug := parts[0] + "/" + parts[1]
	return slug, canonicalCollectionURL(slug), nil
}

func NormalizePaperRef(ref string) (string, string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", fmt.Errorf("empty paper ref")
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		u, err := url.Parse(ref)
		if err != nil {
			return "", "", err
		}
		parts := splitPath(u.Path)
		if len(parts) < 2 || parts[0] != "papers" {
			return "", "", fmt.Errorf("invalid paper URL: %s", ref)
		}
		ref = parts[1]
	}
	ref = strings.TrimPrefix(ref, "/papers/")
	ref = strings.Trim(path.Clean(ref), "/")
	if ref == "" || ref == "." {
		return "", "", fmt.Errorf("invalid paper ref")
	}
	return ref, canonicalPaperURL(ref), nil
}

func InferEntityType(rawURL string) string {
	switch {
	case strings.Contains(rawURL, "/datasets/"):
		return EntityDataset
	case strings.Contains(rawURL, "/spaces/"):
		return EntitySpace
	case strings.Contains(rawURL, "/collections/"):
		return EntityCollection
	case strings.Contains(rawURL, "/papers/"):
		return EntityPaper
	default:
		return EntityModel
	}
}

func canonicalRepoURL(entityType, repoID string) string {
	switch entityType {
	case EntityDataset:
		return BaseURL + "/datasets/" + repoID
	case EntitySpace:
		return BaseURL + "/spaces/" + repoID
	default:
		return BaseURL + "/" + repoID
	}
}

func canonicalCollectionURL(slug string) string {
	return BaseURL + "/collections/" + slug
}

func canonicalPaperURL(paperID string) string {
	return BaseURL + "/papers/" + paperID
}

func normalizeRepoRef(ref, prefix string) (string, string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", fmt.Errorf("empty ref")
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		u, err := url.Parse(ref)
		if err != nil {
			return "", "", err
		}
		parts := splitPath(u.Path)
		switch prefix {
		case "":
			if len(parts) < 2 || parts[0] == "datasets" || parts[0] == "spaces" || parts[0] == "collections" || parts[0] == "papers" {
				return "", "", fmt.Errorf("invalid model URL: %s", ref)
			}
			ref = parts[0] + "/" + parts[1]
		default:
			if len(parts) < 3 || parts[0] != prefix {
				return "", "", fmt.Errorf("invalid %s URL: %s", prefix, ref)
			}
			ref = parts[1] + "/" + parts[2]
		}
	} else {
		ref = strings.TrimPrefix(ref, "/")
		ref = strings.TrimPrefix(ref, prefix+"/")
	}
	parts := splitPath(ref)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("repo ref must look like namespace/name")
	}
	repoID := parts[0] + "/" + parts[1]
	entityType := EntityModel
	if prefix == "datasets" {
		entityType = EntityDataset
	} else if prefix == "spaces" {
		entityType = EntitySpace
	}
	return repoID, canonicalRepoURL(entityType, repoID), nil
}

func splitPath(s string) []string {
	var out []string
	for _, part := range strings.Split(strings.Trim(s, "/"), "/") {
		part = strings.TrimSpace(part)
		if part != "" && part != "." {
			out = append(out, part)
		}
	}
	return out
}
